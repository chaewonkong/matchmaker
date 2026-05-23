# Matchmaker 설계 토론 정리

> Postgres 기반 minimalistic 매치메이커 OSS 설계에 대한 토론 정리

---

## 1. 부하 산정과 RPS 추정

### 1.1 초기 가정으로 시작한 부하 계산

처음에는 다음 가정으로 부하를 추정함:

- 동접 20만 명
- 매치 생성 요청 + 폴링 (1초 1회)
- 게임 30분 진행
- 4인 매치, 1티켓 = 1명

**잘못된 모델**: 클라이언트 1인당 1Hz polling 가정 → 200,000 QPS라는 비현실적 수치.

### 1.2 폴링 모델 정정

폴링은 클라이언트별 상태 조회가 아니라 **컨슈머가 큐에서 매치 이벤트를 끌어가는 워커 패턴**임을 정정.

```
새 매치 생성률 = 200,000 / 1,800초 ≈ 111 ticket/s
완성 매치 = 111 / 4 ≈ 28 match/s
```

폴링 QPS는 동접 수가 아닌 *컨슈머 수 × 폴링 주기*로 결정됨. 실질 부하는 무시 가능.

### 1.3 결론: 부하는 문제가 아니다

| 동접 | 매치 생성률 |
|---|---|
| 20만 | ~28 match/s |
| 100만 | ~139 match/s |

이 수준의 부하는 단일 Go 프로세스 + 단일 Postgres로 충분히 처리 가능. **매치메이커 설계의 본질적 어려움은 throughput이 아닌 매칭 품질, 백프레셔, 운영성에 있음**.

---

## 2. 시스템 요구사항

### 2.1 목표 명세

- 동접 100만 가능, **500~600 RPS**
- 백필 지원
- 동적인 커스텀 어트리뷰트 (큐 설정 기반 payload 관리)
- expr language 기반 동적 매칭 룰 (cel-go 등)
- 관제 기능
- gRPC 기반 인터페이스
- Helm chart 설치형 배포
- **Minimalistic 원칙**: Redis 등 외부 의존성 없이 Postgres만으로 달성

### 2.2 Postgres-only 가능성 검토

가능. 600 RPS는 Postgres가 압도적으로 여유 있게 처리하는 수준. 다만 구현 패턴이 중요:

- **`SELECT ... FOR UPDATE SKIP LOCKED`**: 분산 컨슈머 워커 패턴
- **JSONB + GIN/generated column**: 동적 attribute 저장
- **LISTEN/NOTIFY**: 이벤트 통보 (best-effort, fallback polling 필요)
- **트랜잭션**: 매치 상태 전이의 일관성 보장

### 2.3 매치메이커의 진짜 설계 난점

부하가 아닌 다음 항목이 본질적 어려움:

1. **매칭 품질** — MMR, 핑, 파티, 역할 구성 등 다차원 최적화
2. **큐 체류 시간의 tail latency** — 평균보다 p99가 더 중요
3. **백프레셔와 부분 장애** — 다운스트림(DGS) 장애 대응
4. **파티 처리** — 솔로 vs 파티 우선순위
5. **운영성** — A/B 테스트, 룰 튜닝, 옵저버빌리티

---

## 3. 백프레셔와 Circuit Breaker

### 3.1 단순 차단 모델의 함정

"매치 큐 적체 → 티켓 차단"은 너무 단순. 적체 원인에 따라 대응이 달라야 함:

- **Case A**: 매칭 알고리즘이 느림 → 티켓 차단은 매칭 자체를 멈춤
- **Case B**: DGS 프로비저닝 막힘 → 티켓 차단이 정답
- **Case C**: 폴링 컨슈머 죽음 → 컨슈머 살리면 됨

증상은 같아도 원인과 대응이 다름.

### 3.2 매치메이킹 circuit breaker의 특수성

일반적인 HTTP Hystrix 패턴과 다름:

- **실패율이 아닌 큐 깊이 + age가 신호**: `ApproximateAgeOfOldestMessage`, consumer lag 등
- **Binary가 아닌 점진적 차단**: 수락률을 100% → 80% → 50%로 AIMD 방식
- **거절 비용이 큼**: 사용자가 게임 자체를 못 하게 됨 → 거절보다 긴 대기 + 다른 모드 유도 선호

### 3.3 계층적 대응 패턴

| 적체 정도 | 대응 |
|---|---|
| 정상 | 통상 동작 |
| 약간 (lag > 30s) | DGS fleet 확장 트리거, 알람 |
| 심각 (lag > 2min) | 신규 티켓 수락률 점진 감소 (50%~10%) |
| 위험 (lag > 5min) | 신규 티켓 완전 차단, drain 모드 |
| 복구 | 점진적 ramp-up (slow ramp으로 oscillation 방지) |

핵심:
- 신호는 **queue depth × age**
- 대응은 **계층적/점진적**
- 차단은 **mode/region별 isolated**
- 복구는 **slow ramp**

---

## 4. 아키텍처 진화

### 4.1 OpenMatch 스타일 출발

초기 구상:

- **API Gateway**: 모든 API 요청 처리
- **MatchingEngine (다수 병렬)**: pool별 매치 후보 생성
- **Evaluator**: 후보 중 품질 최고 매치 선별 + 중복 제거

### 4.2 다중 Pool 멤버십 모델

한 큐에 여러 pool이 있고, 한 ticket이 여러 pool에 동시 속할 수 있음 (Pool 범위 overlap).

이 모델에서:
- ticket이 여러 pool에 등장 → 같은 ticket이 여러 MatchingEngine의 후보가 됨 → **중복 제거가 구조적으로 필요** → Evaluator 필수

### 4.3 책임 분리: API Gateway는 큐/풀 설정을 모른다

**핵심 통찰**: 부하나 성능이 아닌 **bounded context 분리** 관점.

| Gateway가 알아야 함 | Gateway가 몰라도 됨 |
|---|---|
| RPC schema (gRPC proto) | Pool 정의 |
| Queue ID (라우팅용) | Pool 분류 룰 |
| 인증 정보 | Attribute schema |
| Ticket의 raw payload | CEL 룰 |

이점:
- 큐/풀 설정 변경이 Gateway 재배포 불필요
- Gateway는 진짜 stateless + dumb
- 멀티 테넌시 자연스러움
- 외부 노출 면에서 비즈니스 로직 격리 (보안)
- 설정 검증의 단일 소유자

### 4.4 Pull 모델 vs Push 모델

**Pull 모델 (DB-centric)**:
- ME가 DB에서 SELECT (with SKIP LOCKED)
- Pool 멤버십을 DB에 영속화 (`ticket_pool_membership` 테이블)
- 운영자 친화적, SQL로 디버깅

**Push 모델 (Director-centric)**:
- Director가 batch pull → 메모리에서 분류 → ME로 push
- Pool 멤버십이 Director의 in-memory 상태
- 설정 버전을 ticket과 함께 전파 가능 (일관성 우월)
- Director가 SPOF에 가까움

**최종 선택: Hybrid (push 모델 채택)**:
- Ticket lifecycle은 DB에 (관찰성)
- Pool 분류는 Director 메모리에 (유연성)
- Director가 dispatch에 config_version 명시

---

## 5. 최종 컴포넌트 구조

### 5.1 컴포넌트 목록

| 컴포넌트 | 역할 | 인스턴스 |
|---|---|---|
| **API Gateway** | gRPC API, 티켓 영속화, 큐/풀 무지 | 다수 (stateless) |
| **Director** | 큐 설정 캐싱, batch pull, pool 분류, ME로 dispatch | 1 active + standby |
| **MatchingEngine (ME)** | 자기 pool의 매치 후보 생성 | pool 수만큼 |
| **Evaluator** | 후보 aggregation, 중복 제거, 최종 매치 확정 | 1 active + standby |
| **Config Syncer** | Config Store polling, 컴포넌트에 config API 제공 | 1 active + standby |
| **Entity Store (Postgres)** | tickets, matches, ticket lifecycle | 단일 DB |
| **Config Store (Postgres)** | 큐/풀/룰 설정, 변경 이력 | 별도 DB |

### 5.2 데이터 흐름

```
[Client] → [Game Backend] → [API Gateway]
                                  ↓ Insert Tickets
                            [Entity Store]
                                  ↑ Polling (read tickets)
                            [Director] ← Sync Config ← [Config Syncer] ← Polling ← [Config Store]
                                  ↓ Distribute Tickets (by pool, with config_version)
                            [Matching Engine × N]
                                  ↓ Aggregate Match Candidates
                            [Evaluator]
                                  ↓ Matches
                            [Entity Store]
                                  ↑ Find Matches (polling, claim batch)
                            [External DGS Allocator]
                                  ↓
                            [DGS Fleet]
```

### 5.3 DB 분리 결정

**runtime DB와 config DB를 분리**:

이유:
- Config가 *런타임 critical path*에 들어옴 (라이브 튜닝)
- 변경 빈도와 패턴이 다름 (운영 DB는 빈번, config는 분~시간 단위)
- Config DB 장애 시 cached config로 매칭 계속 가능 (장애 격리)
- 다른 백업/보존 정책 (config는 long retention)
- 다른 권한 모델

둘 다 Postgres이므로 운영 부담 큰 차이 없음.

---

## 6. Config 관리

### 6.1 런타임 변경 요구사항

YAML/Git 기반은 부적합. 이유:

1. **큐 설정과 게임 서버 파라미터의 바인딩** — 롤백 시 게임 서버도 롤백 필요, 무중단 롤백 어려움
2. **라이브 운영 이슈 대응** — MMR 분포 튜닝 등 분 단위 사이클 필요

따라서 **DB 기반 런타임 변경 API**가 정답.

### 6.2 Config Syncer 도입

- Config Store와 컴포넌트 사이의 캐싱 + API 계층
- Config Store 장애 시 cached config로 매칭 계속 가능 (격리)
- 검증/변환 로직 단일화
- 컴포넌트가 Config Store 스키마를 몰라도 됨

### 6.3 동기화 방식: Polling 채택

**판단 근거**:
- Director cycle이 1초이므로 그보다 빠른 실시간성 무의미
- 실제 매치 대기 시간이 수십 초~수 분이므로 config 반영 1~2초 지연은 무시 가능
- 운영자가 변경 후 효과 관찰까지 분 단위 소요됨
- Polling이 단순함, 장애 복구 자명, 운영자 mental model 단순

**Stale cache 정책**:
- Syncer는 마지막 성공 config를 무기한 유지
- Config Store 장애 중에도 매칭 계속 동작
- 라이브 게임에서 매칭 중단보다 약간의 stale config가 훨씬 나음

### 6.4 ME의 lazy config fetch

ME 동기화 전략:
- ME는 단일 config version만 cache
- Director가 보낸 ticket의 config_version이 cache와 다르면 → Syncer에서 해당 version fetch → cache 교체
- 매칭은 항상 ticket에 명시된 version으로 진행

장점:
- Memory footprint 최소
- Director가 권위 있는 version 발급자 (일관성 보장)
- Syncer 부담 낮음 (변경 시에만 fetch)
- 자동 동기화 (별도 broadcast 불필요)
- Pull-based versioning 패턴 (K8s resource version, Git commit hash와 동일 원리)

### 6.5 Config version의 의미

큐별 version 권장:
- 큐 X가 변경되면 큐 X의 version++
- 다른 큐는 영향 없음
- ME가 큐별로 cache 가능

```
Config Syncer API:
  GET /config/queues/{queue_id}/versions/latest
  GET /config/queues/{queue_id}/versions/{version}

Ticket dispatch metadata:
  {
    ticket_id, pool_id, queue_id,
    config_version: { queue_id: 'rank-4v4', version: 124 }
  }
```

### 6.6 부분 업데이트 방지

Config 변경은 단일 트랜잭션으로:

```sql
begin;
  update queues set ... where queue_id = 'X';
  update pools set ... where queue_id = 'X' and pool_id = 'Y';
  insert into pools (...);
  update config_meta set version = version + 1 where queue_id = 'X';
commit;
```

Syncer는 version 기반 polling으로 atomic 변경 보장.

---

## 7. 통신 모델

### 7.1 외부 통신: Polling 강제

매치메이커의 외부 인터페이스(`API Gateway` 노출)는 **polling 기반**:

- gRPC streaming 대비 폴링의 장점:
  - 표준 HTTP/2 unary RPC (일반 LB로 충분)
  - L4/L7 로드밸런서 설정 단순
  - Keep-alive 튜닝 부담 없음
  - 클라이언트 재시도 시맨틱 자명
  - 디버깅 단순 (각 요청 독립)
- 설치형 OSS의 운영 부담을 최소화하는 결정

### 7.2 게임 서버 통보 모델

**중요**: 클라이언트는 매치메이커를 직접 호출하지 않음.

```
[Client] → [Game Backend] → [Matchmaker API Gateway]
                          ← [DGS Allocator (외부)] ← polling for matches
                          → [DGS Fleet]
```

- 게임 백엔드가 ticket 생성 요청
- DGS Allocator가 매치메이커를 polling하여 매치 가져감
- DGS Allocator가 fleet에 게임 서버 할당
- 게임 서버 정보가 ticket에 박힘 → 게임 백엔드가 ticket polling으로 확인 → 클라이언트에 통보

### 7.3 부하 재정리

매치메이커 시스템 전체 부하 (100만 동접 기준):

| 컴포넌트 | 부하 |
|---|---|
| API Gateway: CreateTicket | ~556 QPS |
| Director: Entity Store polling | 1 QPS (1초 cycle, batch) |
| Director → ME: dispatch | 1 cycle/s |
| ME → Evaluator: candidates | 1 cycle/s |
| Evaluator → Entity Store: match insert | ~139 매치/s |
| DGS Allocator: match polling | 1 QPS (batch dequeue ~139개) |
| Config Syncer: Config Store polling | ~0.2 QPS |

**유일한 의미 있는 부하는 CreateTicket의 ~556 QPS**. 단일 Postgres가 압도적으로 여유 있게 처리.

---

## 8. DGS Allocator 통합

### 8.1 Allocator-agnostic 설계

특정 Allocator(Agones, GameLift, PlayFab 등)에 바인딩하지 않음.

### 8.2 Pull-only 인터페이스

매치메이커는 외부 Allocator API를 호출하지 않음. Allocator가 매치메이커를 polling.

```protobuf
service MatchmakerBackend {
  rpc ClaimMatches(ClaimMatchesRequest) returns (ClaimMatchesResponse);
  rpc ReportAssignment(ReportAssignmentRequest) returns (ReportAssignmentResponse);
  rpc ReleaseMatch(ReleaseMatchRequest) returns (ReleaseMatchResponse);
}

message ClaimMatchesRequest {
  string allocator_id = 1;
  repeated string queue_ids = 2;
  int32 max_matches = 3;
  map<string, string> labels = 4;  // region, capability 등
}

message ClaimMatchesResponse {
  repeated Match matches = 1;
  string lease_token = 2;
  google.protobuf.Timestamp lease_expires_at = 3;
}
```

### 8.3 Lease 패턴

ClaimMatches로 가져간 매치는 임시 locked. lease 시간 내 `ReportAssignment` 없으면 자동 복원 (`state = 'ready'`로 reaper가 되돌림).

### 8.4 매치 dequeue 쿼리

```sql
update matches set 
  state = 'assigning',
  assigned_to_allocator = $allocator_id,
  claimed_at = now()
where id in (
  select id from matches
  where state = 'ready'
  order by created_at
  limit 500
  for update skip locked
)
returning *;
```

### 8.5 Adapter 패턴

OSS 배포 시 참조 구현 제공:

```
matchmaker/
├── helm/
├── proto/
├── server/                  # 매치메이커 본체
└── examples/
    ├── agones-adapter/
    ├── kubernetes-adapter/
    └── README.md
```

Adapter는 매치메이커 외부에서:
- `ClaimMatches` polling
- 자기 Allocator API 호출
- `ReportAssignment`로 결과 통보

---

## 9. 데이터 모델

### 9.1 핵심 테이블

```sql
-- Entity Store
create table tickets (
  id uuid primary key,
  queue_id text not null,
  payload jsonb not null,
  state text not null,            -- created, in_director, matched, assigned, ...
  created_at timestamptz default now(),
  expires_at timestamptz,
  matched_at timestamptz,
  config_version int               -- 분류된 시점의 version
);
create index on tickets (state, created_at);
create index on tickets (queue_id, state);

create table matches (
  id uuid primary key,
  queue_id text not null,
  ticket_ids uuid[] not null,
  quality_score float,
  state text not null,             -- ready, assigning, assigned, completed
  config_version int,
  created_at timestamptz default now(),
  claimed_at timestamptz,
  assigned_to_allocator text,
  dgs_info jsonb
);
create index on matches (state, created_at);

-- Config Store (별도 DB)
create table queues (
  queue_id text primary key,
  config jsonb not null,
  version int not null,
  updated_at timestamptz default now()
);

create table config_history (
  queue_id text not null,
  version int not null,
  config jsonb not null,
  changed_at timestamptz default now(),
  changed_by text,
  primary key (queue_id, version)
);
```

### 9.2 Ticket State Machine

```
created (Gateway INSERT)
  ↓
in_director (Director batch pull, lease)
  ↓
matched (Evaluator 확정)
  ↓
assigned (Allocator가 DGS 할당 완료)
  ↓
in_game / completed / cancelled / expired
```

### 9.3 Match State Machine

```
forming (Director가 매칭 중)
  ↓
ready (Evaluator 확정, Allocator 할당 대기)
  ↓
assigning (Allocator가 claim, lease 중)
  ↓
assigned (DGS 할당 완료)
  ↓
active → completed / failed / backfilling
```

---

## 10. Open Match와의 차별점

| 측면 | Open Match | 본 설계 |
|---|---|---|
| 의존성 | Redis + K8s | Postgres + Helm |
| Director | 사용자 구현 | 내장 |
| MMF (매칭 함수) | 사용자 구현 (gRPC) | CEL 룰 (config) |
| Evaluator | 사용자 구현 | 내장 |
| 백필 | 약함 | 일급 시민 |
| 라이브 룰 튜닝 | 재배포 필요 | 런타임 변경 API |
| 진입 장벽 | 높음 (Director/MMF 직접 구현) | 낮음 (config만) |
| 트랜잭션 일관성 | Lua 등 우회 필요 | Postgres native |

**포지셔닝**: Open Match가 못 채우는 시장 (라이브 운영 친화적, 백필 강함, 진입장벽 낮음) 공략.

---

## 11. 미해결 / 다음 단계

### 11.1 추가 설계 결정거리

1. **Director batch pull 전략** — limit N, lease timeout 튜닝
2. **ME dispatch 단위** — pool 단위 통째 vs 더 작은 분할
3. **Evaluator cycle 동기화** — Director cycle과의 맞물림
4. **Ticket TTL과 expiry** — 너무 오래 기다린 ticket 처리
5. **Region/label 기반 routing** — 다중 Allocator 운영 시
6. **백필 데이터 흐름** — 활성 게임에 ticket 추가 시 통보 경로

### 11.2 다이어그램 보강 필요

- Ticket lifecycle 상태 전이
- Match lifecycle 상태 전이
- 백필 데이터 흐름
- Director의 active-passive HA
- Config version 흐름
- Reaper / TTL cleanup 메커니즘

### 11.3 다음 산출물 후보

- 상태 머신 다이어그램 (Ticket, Match, Director lease)
- 시퀀스 다이어그램 (happy path + 장애 시나리오)
- gRPC proto 정의 초안
- Deployment 다이어그램 (인스턴스 다중성, leader election)

### 11.4 백필 통보 경로 (미정)

```
일반 매치: Director → ME → Evaluator → matches table → Allocator polling → DGS
백필 매치: Director → ME → Evaluator → backfill table → ??? → 기존 DGS
```

백필은 활성 게임 인스턴스로 직접 통보되어야 하므로 별도 데이터 흐름 설계 필요.

---

## 12. 핵심 원칙 요약

1. **Minimalistic 의존성**: Postgres 2대 (runtime + config) + 컴포넌트들. Redis, Kafka, NATS 없음.
2. **Polling 기반 통신**: 운영 견고성과 단순성 우선. Streaming은 필요한 곳만.
3. **Bounded context 분리**: API Gateway는 큐/풀 무지, Director가 큐/풀 권위자, ME는 매칭 로직, Evaluator는 일관성 수문장.
4. **Pull + lease 패턴**: SKIP LOCKED로 분산 컨슈머, lease로 장애 복구.
5. **Pull-based config versioning**: Director가 version 발급, 컴포넌트가 lazy fetch.
6. **외부 Allocator는 추상화**: Pull-only API로 매치메이커는 Allocator 모름.
7. **라이브 운영 first-class**: Config 런타임 변경, stale cache fallback, 점진적 backpressure.