# ADR-004: Director Push 모델 채택 (Pool 멤버십을 메모리로)

## Status

Accepted (2026-05-23)

## Context

한 큐에 여러 pool이 있고 (Pool 범위 overlap), 한 ticket이 여러 pool에 동시 속할 수 있다. MatchingEngine은 pool당 1개씩 동작한다.

Pool 멤버십을 어디에 저장할지 두 가지 선택지:

### Pull 모델 (DB-centric)

- Pool 멤버십을 DB에 영속화 (`ticket_pool_membership` 테이블 또는 generated column)
- ME가 SQL로 자기 pool ticket 조회 (SKIP LOCKED)

### Push 모델 (Director-centric)

- Director가 batch pull → 메모리에서 pool 분류 → ME로 직접 push
- Pool 멤버십이 Director의 in-memory 상태

## Decision

**Push 모델 채택. Director가 분류 권위자 역할.**

흐름:

1. Ticket은 Entity Store에 영속화 (state = `created`)
2. Director가 1초 cycle로 batch pull (SKIP LOCKED, state → `in_director`)
3. Director가 메모리에서 pool 분류
4. Director가 ticket + pool_id + config_version을 ME로 push (gRPC)
5. ME가 매칭 후보 생성
6. Evaluator가 후보 수집 → 충돌 해결 → 최종 매치 확정

## Consequences

### 장점

- **JSONB 인덱스 문제 해소** — ME가 SQL로 attribute 쿼리할 필요 없음
- **Config version 전파 자연스러움** — Director가 권위 발급자, 일관성 보장 (ADR-005 참조)
- **Pool 정의 변경 시 즉시 반영** — Director 메모리만 갱신
- **ticket_pool_membership 테이블 불필요** — write 감소
- **ME는 stateless에 가까움** — 받은 ticket batch만 처리

### 단점

- **Director가 SPOF에 가까워짐** — single active + standby 필요
- **Push 모델의 coordination 복잡도** — ME discovery, health check, fan-out
- **ME 가용성 처리가 Director 책임** — 재dispatch 로직 필요
- **분산 컨슈머 패턴의 자연스러움 상실** — SKIP LOCKED 패턴 일부만 활용

### Lease 패턴

Director가 batch pull 시 `state = 'in_director'`로 변경하고 `leased_at`을 기록한다. Director 죽으면 별도 reaper가 stale lease를 `created`로 복원한다 (SQS visibility timeout과 동일한 패턴).

## Alternatives Considered

### Pull 모델 + DB에 영속화된 pool 멤버십

- 장점: ME가 자기 페이스로 동작, 단순한 분산 컨슈머, 운영자 친화적 디버깅
- 단점:
  - JSONB 인덱스로 동적 attribute 쿼리하는 문제
  - Pool 정의 변경 시 멤버십 재계산
  - Config version 전파 메커니즘 별도 필요
  - 컴포넌트 간 config 버전 불일치 가능

### Hybrid (DB에 ticket lifecycle + 메모리에 pool 멤버십)

채택한 모델이 이것에 해당. Director가 ticket lifecycle 상태는 DB에 기록하고, pool 분류만 메모리에서 처리.

## Notes

이 결정은 Open Match의 Director 패턴과 유사하지만 차이가 있다:

- Open Match: Director가 외부 사용자 구현 (Backend API 호출)
- 본 설계: Director가 내장 컴포넌트 (사용자는 CEL 룰만 정의)

내장하는 대신 외부 통합 부담을 매우 낮추는 trade-off.

부하 검증 (100만 동접 기준):

- 매치 생성률 ~139 match/s
- Director cycle 1초마다 batch pull (limit 500~1000)
- 단일 Director가 압도적으로 여유 있게 처리

향후 부하가 더 커지면 큐별로 Director 분리 가능 (multi-Director).
