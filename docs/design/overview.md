# Matchmaker — System Overview

> 시스템 전체를 한 페이지로 요약. 상세 의사결정은 `docs/decisions/`, 컴포넌트별 디테일은 `docs/design/components/`에 분리.

## 1. 목표

Postgres 기반 minimalistic 매치메이커. Open Match의 대안 포지셔닝.

**기능 목표**:
- 동접 100만 명, 600 RPS 처리
- 백필 지원 (일급 시민)
- 큐별 동적 커스텀 어트리뷰트
- CEL 기반 동적 매칭 룰 (런타임 변경 가능)
- 관제 기능
- gRPC 인터페이스
- Helm chart 설치형 배포

**비기능 목표**:
- 의존성 최소화 (Postgres만, 외부 큐/캐시 없음)
- 라이브 운영 친화적 (재배포 없이 룰 튜닝)
- Allocator-agnostic (Agones, GameLift, PlayFab 등 모두 통합 가능)

## 2. 시스템 경계

```
┌─────────────────────────── 외부 ────────────────────────────┐
│                                                              │
│   Game Client → Game Backend ──┐                            │
│                                ├──gRPC──→ ┌──────────────┐  │
│   External DGS Allocator ──────┘          │ API Gateway  │  │
│                                            └──────┬───────┘  │
│                                                   │          │
└───────────────────────────────────────────────────┼──────────┘
                                                    │
┌─────────────────────────── 내부 ──────────────────┼──────────┐
│                                                   ▼          │
│                                          ┌──────────────┐    │
│                                          │ Entity Store │    │
│                                          │  (Postgres)  │    │
│                                          └──┬───────────┘    │
│                                             │ polling        │
│                                  ┌──────────▼─────────┐      │
│                                  │      Director      │◄─────┼─┐
│                                  └──────────┬─────────┘      │ │
│                                             │ dispatch       │ │
│                                  ┌──────────▼─────────┐      │ │
│                                  │ MatchingEngine × N │◄─────┼─┤
│                                  └──────────┬─────────┘      │ │
│                                             │ candidates     │ │
│                                  ┌──────────▼─────────┐      │ │
│                                  │     Evaluator      │◄─────┼─┤
│                                  └──────────┬─────────┘      │ │
│                                             │ matches        │ │
│                                  ┌──────────▼─────────┐      │ │
│                                  │   Entity Store     │      │ │
│                                  └────────────────────┘      │ │
│                                                              │ │
│                  ┌────────────────────┐                      │ │
│                  │   Config Syncer    │──────sync config─────┼─┘
│                  └──────────┬─────────┘                      │
│                             │ polling                        │
│                  ┌──────────▼─────────┐                      │
│                  │   Config Store     │                      │
│                  │    (Postgres)      │                      │
│                  └────────────────────┘                      │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**핵심 원칙**: 외부 시스템은 오직 API Gateway만 통해 매치메이커와 통신. 내부 컴포넌트는 외부에 노출되지 않음 (ADR-003).

## 3. 컴포넌트

| 컴포넌트 | 역할 | 인스턴스 | 상태 |
|---|---|---|---|
| **API Gateway** | 외부 진입점, 인증, ticket CRUD, 매치 dequeue | 다수 (stateless, 수평 확장) | stateless |
| **Director** | 큐 설정 캐싱, batch pull, pool 분류, ME로 dispatch | 1 active + standby | stateful (lease) |
| **MatchingEngine** | pool당 1개, 매치 후보 생성 | pool 수만큼 | stateless (config cache만) |
| **Evaluator** | 후보 aggregation, 충돌 해결, 최종 매치 확정 | 1 active + standby | stateless |
| **Config Syncer** | Config Store polling 캐싱, 컴포넌트에 config API 제공 | 1 active + standby | cache |
| **Entity Store** | tickets, matches 영속화 | Postgres 1대 | stateful |
| **Config Store** | 큐/풀/룰 설정 + 변경 이력 | Postgres 1대 (runtime DB와 분리) | stateful |

## 4. 외부 API

API Gateway가 노출하는 gRPC service:

```protobuf
service MatchmakerFrontend {
  // Game Backend가 호출
  rpc CreateTicket(CreateTicketRequest) returns (CreateTicketResponse);
  rpc GetTicket(GetTicketRequest) returns (GetTicketResponse);  // polling
  rpc CancelTicket(CancelTicketRequest) returns (CancelTicketResponse);
}

service MatchmakerBackend {
  // External DGS Allocator가 호출
  rpc ClaimMatches(ClaimMatchesRequest) returns (ClaimMatchesResponse);
  rpc ReportAssignment(ReportAssignmentRequest) returns (ReportAssignmentResponse);
  rpc ReleaseMatch(ReleaseMatchRequest) returns (ReleaseMatchResponse);
}

service MatchmakerAdmin {
  // 운영자가 호출 (라이브 룰 튜닝)
  rpc UpsertQueueConfig(UpsertQueueConfigRequest) returns (UpsertQueueConfigResponse);
  rpc GetQueueConfig(GetQueueConfigRequest) returns (GetQueueConfigResponse);
  rpc ListConfigHistory(ListConfigHistoryRequest) returns (ListConfigHistoryResponse);
  rpc RollbackConfig(RollbackConfigRequest) returns (RollbackConfigResponse);
}
```

모두 gRPC unary RPC (streaming 없음, ADR-002).

## 5. 데이터 모델 (핵심만)

### Ticket lifecycle

```
created (Gateway INSERT)
  ↓
in_director (Director batch pull, lease)
  ↓
matched (Evaluator 확정)
  ↓
assigned (Allocator가 DGS 할당 완료)
  ↓
in_game → completed / cancelled / expired
```

### Match lifecycle

```
forming (Director가 매칭 중, 메모리 only)
  ↓
ready (Evaluator 확정, Allocator 할당 대기)
  ↓
assigning (Allocator가 claim, lease 중)
  ↓
assigned (DGS 정보 박힘, ticket에 전파)
  ↓
active → completed / failed / backfilling
```

### 핵심 테이블 요약

```sql
-- Entity Store (runtime DB)
tickets         -- id, queue_id, payload (jsonb), state, config_version, timestamps
matches         -- id, queue_id, ticket_ids[], state, quality_score, 
                --   config_version, dgs_info (jsonb), assignment 메타

-- Config Store (별도 DB)
queues          -- queue_id, config (jsonb), version, updated_at
config_history  -- queue_id, version, config, changed_at, changed_by
```

상세는 `docs/design/data-model.md` 참조 (작성 예정).

## 6. 핵심 흐름

### 6.1 매칭 happy path

1. Game Backend → Gateway: `CreateTicket(queue_id, payload)`
2. Gateway → Entity Store: ticket INSERT (state=`created`)
3. Game Backend → Gateway: `GetTicket(ticket_id)` polling (1초)
4. Director (1초 cycle):
   - Entity Store에서 `created` ticket batch pull (state=`in_director`)
   - 메모리에서 큐별 config로 pool 분류
   - `{ticket, pool_id, config_version}`을 해당 ME로 push
5. MatchingEngine:
   - 받은 ticket batch를 자기 pool 후보로 누적
   - config_version이 cache와 다르면 Syncer에서 fetch
   - CEL 룰로 매치 후보 생성
   - Evaluator로 push
6. Evaluator:
   - 모든 ME의 후보 수집 (cycle 단위)
   - 중복 ticket 충돌 해결 (quality score 기반)
   - 트랜잭션: ticket state=`matched`, match INSERT (state=`ready`)
7. External DGS Allocator → Gateway: `ClaimMatches(...)` polling
8. Gateway → Entity Store: `SELECT ... FOR UPDATE SKIP LOCKED` (state=`assigning`, lease)
9. Allocator: 자기 fleet에서 DGS 인스턴스 시작
10. Allocator → Gateway: `ReportAssignment(match_id, dgs_address)`
11. Gateway → Entity Store: match state=`assigned`, ticket에 dgs_info 박음
12. Game Backend의 `GetTicket` polling이 dgs_info 받음 → 클라이언트에 통보

### 6.2 Config 변경 흐름

1. 운영자 → Gateway → Config Store: `UpsertQueueConfig(...)` 트랜잭션 (version++)
2. Config Syncer가 polling으로 변경 감지 → cache 갱신
3. Director가 Syncer에서 새 config fetch (다음 cycle)
4. Director가 새 config_version으로 ticket 분류
5. ME가 ticket의 config_version이 cache와 다르면 Syncer에서 lazy fetch

상세는 `docs/design/flows/` 참조 (작성 예정).

## 7. 부하 특성

100만 동접 기준 추정:

| 컴포넌트 | QPS | 비고 |
|---|---|---|
| Gateway CreateTicket | ~556 | 티켓 발급률 |
| Gateway GetTicket | ~수천 | Game Backend의 polling |
| Gateway ClaimMatches | 1~수십 | Allocator polling, batch |
| Director → Entity Store | 1 | 1초 cycle |
| Evaluator → matches insert | ~139 | 매치 생성률 |
| Config Syncer → Config Store | ~0.2 | 5초 polling |

**유일한 의미 있는 부하는 Gateway의 CreateTicket + GetTicket**. 단일 Postgres가 압도적으로 여유 있게 처리. 부하 측면에서 Postgres scaling 우려 없음.

## 8. 운영 특성

### 가용성

- **Gateway**: stateless, 다수 인스턴스 + LB
- **Director, Evaluator, Syncer**: active-passive + Postgres advisory lock 기반 leader election
- **ME**: pool당 1개 active + standby 또는 multi-active (큐 분할)
- **Entity Store/Config Store**: 사용자 운영 책임 (read replica는 선택)

### Stale config 정책

Config Store 일시 장애 시 Syncer는 마지막 성공 config를 무기한 유지. 매칭은 계속 동작. "최근 변경한 룰이 늦게 반영됨"이 "매칭 시스템 다운"보다 낫다는 원칙.

### Lease + Reaper 패턴

- Director가 ticket batch pull 시 `in_director` 상태 + leased_at
- Allocator가 ClaimMatches 시 `assigning` 상태 + claimed_at
- 별도 reaper job이 stale lease를 원래 상태로 복원 (Director/Allocator 다운 대응)

### 백프레셔

매치 큐 (`state='ready'`) depth와 oldest age를 모니터링:
- 약간 적체 → DGS fleet autoscale 트리거
- 심각 적체 → 신규 ticket 수락률 점진 감소
- 위험 적체 → ticket 수락 완전 차단 + drain 모드
- 복구 시 slow ramp (oscillation 방지)

## 9. 디렉토리 구조 (계획)

```
matchmaker/
├── CLAUDE.md
├── proto/                          # gRPC 정의
├── services/                       # 매치메이커 본체
│   ├── gateway/
│   ├── director/
│   ├── engine/
│   ├── evaluator/
│   └── syncer/
├── helm/                           # Helm chart
├── examples/
│   ├── agones-adapter/
│   └── kubernetes-adapter/
└── docs/
    ├── design/
    │   ├── overview.md             # 이 문서
    │   ├── data-model.md           # 스키마, 인덱스, 상태 머신 상세
    │   ├── api.md                  # gRPC proto + 시맨틱
    │   ├── components/             # 컴포넌트별 디테일
    │   ├── flows/                  # 시퀀스 다이어그램, 장애 시나리오
    │   └── operations/             # 배포, 모니터링, 튜닝
    ├── decisions/                  # ADR
    └── design-discussion.md        # 초기 토론 전문
```

## 10. 미해결 항목

- **백필 데이터 흐름**: 활성 게임 인스턴스로의 통보 경로 (일반 매치는 Allocator polling, 백필은 기존 DGS 직접 통보 필요)
- **Region/label 기반 routing**: 다중 Allocator 운영 시 매치 분배 정책
- **Evaluator cycle 동기화**: Director cycle과의 정확한 맞물림 (현재 1초 cycle 가정)
- **충돌 해결 알고리즘**: greedy vs maximum weight matching
- **Ticket TTL**: 너무 오래 기다린 ticket 처리 정책

## 11. 참조

- ADR-001: Postgres-only 의존성
- ADR-002: Polling 기반 통신
- ADR-003: API Gateway는 유일한 외부 진입점이자 큐/풀 설정 무지
- ADR-004: Director Push 모델
- ADR-005: Config 런타임 변경 (DB 기반)
- ADR-006: Runtime DB와 Config DB 분리
- ADR-007: Allocator-agnostic Pull-only API

Open Match와의 차별점은 ADR 전반에서 다룸. 요약하면: 의존성 최소(Postgres만), Director/Evaluator 내장, 라이브 룰 튜닝, 백필 일급, 진입 장벽 낮음.