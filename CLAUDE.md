# Matchmaker

Postgres 기반 minimalistic 매치메이커 OSS. Open Match 대안.

## 핵심 원칙

- **의존성은 Postgres만** — Redis, Kafka, NATS 등 외부 큐/캐시 없음
- **Polling 기반 통신** — streaming 회피, 운영 단순성 우선
- **Config 런타임 변경** — 재배포 없이 라이브 튜닝 (API 기반)
- **Allocator-agnostic** — Agones/GameLift/PlayFab 등 모두 통합 가능
- **설치형 배포** — Helm chart 단일 명령으로 설치

## 컴포넌트

- **API Gateway** — gRPC API, stateless, 큐/풀 설정 무지 (단순 transport)
- **Director** — 1초 cycle, Entity Store batch pull, pool 분류, ME로 dispatch (1 active + standby)
- **MatchingEngine (ME)** — pool당 1개, 후보 매치 생성, lazy config fetch
- **Evaluator** — 후보 aggregation, 충돌 해결, 최종 매치 확정 (1 active + standby)
- **Config Syncer** — Config Store polling 캐싱, 컴포넌트에 config API 제공
- **Entity Store** — Postgres (tickets, matches)
- **Config Store** — Postgres (queues, pools, rules, history) — runtime DB와 분리

## 통신 경계

**외부 시스템(Game Backend, External DGS Allocator)은 오직 API Gateway를 통해서만 매치메이커와 통신한다.** Entity Store, Director, ME, Evaluator 등 내부 컴포넌트는 외부에 노출되지 않는다.

## 데이터 흐름

```
[외부 경계]
Client ──→ Game Backend ──┐
                          ├──gRPC──→ API Gateway ──┐
External DGS Allocator ───┘                        │
                          ↑ ClaimMatches/Report    │
                                                   ↓
[내부 컴포넌트]                              Entity Store
                                                   ↑ polling
                                                Director ← Config Syncer ← Config Store
                                                   ↓ dispatch (with config_version)
                                                Matching Engine × N
                                                   ↓ candidates
                                                Evaluator
                                                   ↓ matches insert
                                                Entity Store

[게임 서버 할당]
External DGS Allocator ──→ DGS Fleet (게임 서버 시작)
                       ──→ API Gateway (ReportAssignment)
```

외부 통신 경로 (모두 API Gateway 경유):

- **Game Backend → Gateway**: CreateTicket, GetTicket (매칭 결과 polling)
- **DGS Allocator → Gateway**: ClaimMatches (polling), ReportAssignment, ReleaseMatch
- **Gateway → 외부**: 없음 (Pull-only)

## 코드 컨벤션

- Go 1.22+
- gRPC 인터페이스는 `proto/` 하위 정의
- DB 접근은 pgx 사용
- CEL 룰 평가는 cel-go
- 컴포넌트 간 통신은 gRPC unary RPC (streaming 지양)

## 부하 기준

목표: 동접 100만, ~600 RPS (CreateTicket 기준)
- Director polling: 1 QPS
- Evaluator → matches insert: ~139 QPS
- DGS Allocator polling: 1 QPS
- 단일 Postgres가 압도적으로 여유 있게 처리

## 디렉토리

```
matchmaker/
├── proto/                  # gRPC 정의
├── services/               # 매치메이커 본체
│   ├── gateway/
│   ├── director/
│   ├── engine/
│   ├── evaluator/
│   └── syncer/
├── helm/                   # Helm chart
├── examples/
│   ├── agones-adapter/
│   └── kubernetes-adapter/
└── docs/
    ├── design/             # 컴포넌트별 상세
    ├── decisions/          # ADR
    └── design-discussion.md  # 초기 토론 전문
```

## 상세 문서

설계 상세: `docs/design/`
의사결정 이력: `docs/decisions/` (ADR 형식)
초기 토론: `docs/design-discussion.md`
