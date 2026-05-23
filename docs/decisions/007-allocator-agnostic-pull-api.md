# ADR-007: Allocator-agnostic Pull-only API

## Status

Accepted (2026-05-23)

## Context

매치메이커가 생성한 매치는 결국 DGS(Dedicated Game Server)에 할당되어야 한다. DGS fleet 관리 방식은 게임사마다 천차만별:

- Agones (K8s-native)
- AWS GameLift
- PlayFab Multiplayer Servers
- 자체 Kubernetes (StatefulSet/Deployment)
- 베어메탈 풀 + 자체 스케줄러

매치메이커가 이 다양한 시스템을 어떻게 통합할지가 OSS 도구의 도입 장벽을 결정한다.

## Decision

**DGS Allocator는 외부 시스템으로 두고, 매치메이커는 API Gateway를 통한 Pull-only API만 노출한다.**

- 매치메이커는 외부 Allocator API를 호출하지 않는다
- Allocator는 **API Gateway를 통해** 매치메이커를 polling한다 (Entity Store 직접 접근 불가, ADR-003)
- 매치메이커는 Allocator의 존재 자체를 모른다 (추상화 비용 zero)
- 참조 구현(adapter)을 examples로 제공

### API 명세

```protobuf
service MatchmakerBackend {
  rpc ClaimMatches(ClaimMatchesRequest) returns (ClaimMatchesResponse);
  rpc ReportAssignment(ReportAssignmentRequest) returns (ReportAssignmentResponse);
  rpc ReleaseMatch(ReleaseMatchRequest) returns (ReleaseMatchResponse);
}

message ClaimMatchesRequest {
  string allocator_id = 1;          // 어느 allocator인지
  repeated string queue_ids = 2;    // 관심 있는 큐 (필터링)
  int32 max_matches = 3;            // batch 크기
  map<string, string> labels = 4;   // region, capability 등
}

message ClaimMatchesResponse {
  repeated Match matches = 1;
  string lease_token = 2;
  google.protobuf.Timestamp lease_expires_at = 3;
}

message ReportAssignmentRequest {
  string match_id = 1;
  oneof result {
    AssignmentSuccess success = 2;  // dgs_address, dgs_port 등
    AssignmentFailure failure = 3;  // 실패 사유
  }
}
```

### Adapter 패턴

```
matchmaker/
├── services/                # 매치메이커 본체
└── examples/
    ├── agones-adapter/      # 참조 구현
    ├── kubernetes-adapter/
    └── README.md            # 사용자 정의 adapter 가이드
```

Adapter는 매치메이커 외부에서:
1. `ClaimMatches` polling
2. 자기 Allocator API 호출 (Agones Allocation API 등)
3. `ReportAssignment`로 결과 통보

## Consequences

### 장점

- **매치메이커 자체가 깨끗하게 유지** — Allocator 통합 코드 없음
- **모든 Allocator 지원 가능** — Adapter만 작성하면 됨
- **운영 환경 무관** — K8s, AWS, 베어메탈 등 어디든 OK
- **테스트 단순** — 매치메이커 테스트에 mock Allocator 불필요 (인터페이스만 검증)

### 단점

- **사용자 부담 약간 증가** — Adapter 작성 필요 (단, 참조 구현 제공)
- **Pull 모델의 latency** — Allocator polling interval만큼 지연 (보통 1초)

### Lease 패턴

ClaimMatches로 가져간 매치는 임시 locked (`state = 'assigning'`). lease 시간 내 `ReportAssignment`가 오지 않으면 reaper가 자동으로 `ready`로 복원.

```sql
-- Reaper job (주기 실행)
update matches set state = 'ready', assigned_to_allocator = null
where state = 'assigning' and claimed_at < now() - interval '1 minute';
```

### 매치 dequeue 쿼리

Allocator의 `ClaimMatches` gRPC 호출을 받은 **API Gateway가 Entity Store에 다음 쿼리를 실행**한다:

```sql
update matches set 
  state = 'assigning',
  assigned_to_allocator = $allocator_id,
  claimed_at = now()
where id in (
  select id from matches
  where state = 'ready'
    and ($2::text[] is null or queue_id = any($2))  -- queue 필터
  order by created_at
  limit $3
  for update skip locked
)
returning *;
```

SKIP LOCKED로 여러 Allocator가 동시에 polling해도 서로 다른 매치를 받음. Allocator는 Entity Store를 직접 알지 못한다.

## Alternatives Considered

### Push 모델 (매치메이커가 Allocator API 호출)

- 장점: latency 낮음
- 단점:
  - Allocator별 SDK/API 통합 필요 (Agones, GameLift, PlayFab 각각)
  - 매치메이커가 Allocator의 가용성에 의존
  - 의존성 폭발 (각 SDK 버전 관리)
- **OSS 도입 장벽이 매우 높아짐** → 부적합

### Allocator 내장 (Agones 전용)

- 장점: 단순, K8s 친화적
- 단점:
  - 특정 환경에 lock-in
  - 베어메탈, AWS 등 사용자 배제
- **시장 축소** → 부적합

### Webhook 콜백

- 매치메이커가 매치 생성 시 등록된 webhook URL 호출
- 장점: latency 낮음
- 단점:
  - Webhook 등록 메커니즘 필요
  - 신뢰성 보장 어려움 (재시도, 순서)
  - polling이 더 단순

## Notes

이 결정이 Open Match와 차별화되는 지점이다:

- Open Match: Director를 외부 사용자 구현 (매칭 알고리즘 + DGS 통합 모두 사용자 부담)
- 본 설계: Director, ME, Evaluator 내장, **외부 통합은 Allocator 한 곳만**

사용자 입장에서:
- 매칭 룰: CEL config로 정의
- DGS 통합: 작은 adapter 작성 (참조 구현 활용)

이게 인디 게임사도 도입 가능한 수준의 진입 장벽이다.

### 다중 Allocator + region routing

여러 Allocator가 동시 운영되는 경우 (region별 등) **매치에 region 라벨이 명확히 박혀있는 모델**을 권장:

- ticket의 region attribute → 매치의 region 라벨
- region별 Allocator는 자기 region 매치만 ClaimMatches
- region 간 분산은 Director의 분류 단계에서 처리 (region별 pool)

이게 fairness 문제와 routing 복잡도를 모두 단순화한다.
