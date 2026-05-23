# ADR-003: API Gateway는 유일한 외부 진입점이자 큐/풀 설정 무지

## Status

Accepted (2026-05-23)

## Context

매치메이커의 외부 진입점은 API Gateway다. 두 가지 결정이 얽혀 있다:

**결정 1: 외부 진입점을 단일화할 것인가?**

매치메이커의 외부 시스템(Game Backend, External DGS Allocator)이 어떤 컴포넌트와 통신할 수 있는지 정의해야 한다. 옵션:

1. 외부가 Entity Store, Director 등 내부 컴포넌트를 직접 호출
2. 외부는 API Gateway만 통하고, 내부 컴포넌트는 외부에 노출되지 않음

**결정 2: Gateway가 어떤 책임을 가지는가?**

1. Gateway가 모든 비즈니스 로직 수행: 인증, 검증, attribute schema 체크, pool 분류, DB INSERT
2. Gateway는 transport만: 받아서 다음 컴포넌트로 넘기는 역할만

두 결정 모두 부하나 성능이 아닌 **bounded context 분리**의 문제다.

## Decision

**API Gateway는 매치메이커의 유일한 외부 진입점이며, 큐/풀 설정을 모르는 단순 transport 역할만 한다.**

### 외부 진입점 단일화

- Game Backend, External DGS Allocator 등 외부 시스템은 **오직 API Gateway를 통해서만** 매치메이커와 통신한다
- Entity Store, Director, MatchingEngine, Evaluator, Config Syncer 등 내부 컴포넌트는 **외부에 노출되지 않는다**
- Gateway가 외부에 노출하는 API:
  - `CreateTicket` (Game Backend → Gateway)
  - `GetTicket` (Game Backend → Gateway, 매칭 결과 polling)
  - `ClaimMatches` (DGS Allocator → Gateway)
  - `ReportAssignment` (DGS Allocator → Gateway)
  - `ReleaseMatch` (DGS Allocator → Gateway)

### Gateway의 큐/풀 무지

Gateway가 알아야 할 것 vs 몰라도 되는 것:

| Gateway가 알아야 함 | Gateway가 몰라도 됨 |
|---|---|
| RPC schema (gRPC proto) | Pool 정의 |
| Queue ID (라우팅용) | Pool 분류 룰 |
| 인증 정보 | Attribute schema |
| Ticket의 raw payload | CEL 룰 |
| Entity Store INSERT/SELECT 방법 | Matching 알고리즘 |

Gateway는 받은 ticket을 그대로 Entity Store에 INSERT만 한다. 분류는 Director가 책임진다. ClaimMatches 등 매치 dequeue도 Gateway가 Entity Store에 SQL 쿼리로 수행한다.

## Consequences

### 장점

**외부 진입점 단일화 효과**:

- **보안 경계 명확** — 내부 컴포넌트가 외부에 노출되지 않음, attack surface 최소화
- **인증/인가 단일 지점** — Gateway에서만 인증 처리, 내부 통신은 mTLS 등 단순화
- **rate limit / quota 단일 지점** — Gateway에서 전사적 정책 적용
- **관제 단일 지점** — 외부 트래픽 메트릭이 Gateway 하나에 집중
- **내부 리팩토링 자유** — Entity Store 스키마 변경 등이 외부 API에 영향 없음

**Gateway 큐/풀 무지 효과**:

- **큐/풀 설정 변경 시 Gateway 재배포 불필요** — Director만 hot-reload
- **Gateway가 진짜 stateless + dumb** — CEL 런타임, 룰 캐시 불필요, footprint 작음, 시작 빠름, 수평 확장 자유
- **멀티 테넌시 자연스러움** — 여러 게임 모드/큐가 한 인스턴스에 공존해도 Gateway는 모드별 차이 무지
- **비즈니스 로직 격리** — 외부 노출 면에 매칭 룰이 reverse engineering될 단서 없음
- **설정 검증의 단일 소유자** — Director 또는 Config Service에 집중

### 단점

- **추가 컴포넌트 필요** — Pool 분류 책임이 별도 컴포넌트(Director)로 이동
- **운영 시나리오에 따라 복잡도가 다른 곳으로 이동** — Gateway가 단순해진 대가로 Director가 무거워짐

## Alternatives Considered

### Gateway가 직접 분류 후 INSERT

- 장점: 컴포넌트 수 적음, write 한 번
- 단점:
  - 큐 설정 변경 시 Gateway 재배포 필요
  - Gateway가 stateful (CEL 룰 캐시)
  - 외부 노출 면에 비즈니스 로직 노출
  - 멀티 테넌시 시 복잡

### Inbox 테이블을 두고 Pool Assigner가 별도 처리

검토했으나 단일 테이블 + 상태 머신으로 표현 가능하므로 over-engineering.

## Notes

이 결정의 핵심 가치는 **외부 노출 면을 단순하게 유지하고 내부 복잡도를 응집된 컴포넌트에 모으는 것**이다.

Gateway가 단순해진 결과:
1. 신규 큐 추가 시 Gateway 무관
2. 매칭 룰 튜닝 시 Gateway 무관
3. 게임 추가 (멀티 테넌트) 시 Gateway는 game_id 라우팅만
4. 긴급 큐 정지 시 Director가 drain, Gateway는 여전히 수신

운영 시나리오 대부분이 Gateway 무지의 이점을 누린다.
