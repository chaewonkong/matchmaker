# ADR-002: Polling 기반 통신 채택

## Status

Accepted (2026-05-23)

## Context

매치메이커 시스템의 통신 경로:

- 외부 → 매치메이커: 게임 백엔드의 ticket 생성, DGS Allocator의 매치 조회
- 매치메이커 내부: Director ↔ Entity Store, Config Syncer ↔ Config Store, Syncer ↔ 각 컴포넌트
- 매치메이커 → 외부: 없음 (Pull-only API, ADR-007 참조)

각 경로마다 gRPC streaming과 polling 사이 선택지가 있다.

## Decision

**외부 인터페이스 및 컴포넌트 간 통신의 기본은 polling으로 한다.**

구체적으로:

- 외부 → 매치메이커: gRPC unary RPC polling (1초 또는 그 이상)
- Director → Entity Store: SQL polling (1초 cycle)
- Config Syncer → Config Store: SQL polling (5~10초)
- 각 컴포넌트 → Config Syncer: polling 또는 Director가 dispatch에 config_version 명시 (ADR-005 참조)

Streaming은 도입하지 않는다.

## Consequences

### 장점

- **로드밸런서 단순**: 표준 HTTP/2 unary RPC라 일반 L4/L7 LB로 충분
- **Keep-alive 부담 없음**: long-lived connection 관리 불필요
- **재시도 시맨틱 자명**: 각 요청 독립, 그냥 다시 요청하면 됨
- **장애 복구 자동**: 일시 단절 후 다음 cycle에 자연스럽게 catch-up
- **디버깅 단순**: 각 요청을 독립적으로 추적 가능
- **운영자 mental model 단순**: "N초마다 본다"는 한 줄로 설명

### 단점

- **Latency**: 평균 `interval/2`, 최악 `interval`의 지연
- **불필요한 query**: 변경 없어도 매 cycle 조회

### Latency 검증

- Director cycle: 1초 → 새 ticket이 매칭 시작까지 평균 0.5초
- Config polling: 5~10초 → 운영자 변경 반영까지 최대 ~10초
- 실제 매치 대기 시간이 수십 초~분 단위이므로 무시 가능

## Alternatives Considered

### gRPC streaming

- 장점: 낮은 latency (~수십 ms), 변경 즉시 전파
- 단점:
  - L4 LB의 connection 균등 분배 깨짐 → L7 (Envoy, gRPC-LB) 필요
  - HTTP/2 ping, TCP keep-alive, 프록시 idle timeout 모두 정렬 필요
  - Connection 재분배 (Gateway 추가 시 rebalancing)
  - 재시도 시맨틱 모호 (at-least-once vs at-most-once)
  - 조용한 connection drop 디버깅 어려움
- 설치형 OSS에서 사용자 운영 부담이 너무 큼

### LISTEN/NOTIFY

- Postgres 변경을 즉시 알림 받음
- 매치메이커 내부 (Syncer ↔ Config Store)에서는 보강으로 사용 가능
- 단, drop 가능성 있어 polling fallback 필요
- 컴포넌트 ↔ Syncer 사이는 polling이 더 단순

### Long polling

- 중간 지점이지만 단순 polling으로 부하 충분히 처리되므로 도입 가치 낮음

## Notes

게임 매칭 도메인의 시간 단위:

- Director 매칭 cycle: 1초
- 일반적인 매치 대기: 수 초 ~ 1~2분
- 운영자 룰 변경 → 효과 관찰: 분 단위
- MMR 분포 변화: 시간 ~ 일 단위

Polling 모델이 실시간성보다 우월하다고 판단할 근거가 없다. **시스템 전체의 자연 시간 단위가 polling과 잘 맞는다**.

부하 측면에서도 Director/Allocator/Syncer 모두 1자리 QPS 수준이므로 polling 비용 무시 가능.
