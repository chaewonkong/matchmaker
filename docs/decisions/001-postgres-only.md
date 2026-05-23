# ADR-001: Postgres-only 의존성

## Status

Accepted (2026-05-23)

## Context

매치메이커는 다음과 같은 데이터 처리가 필요하다:

- Ticket 영속화 및 lifecycle 관리
- 매칭 워커의 분산 컨슈머 패턴
- 매치 결과 저장
- 설정 (큐/풀/룰) 저장
- 컴포넌트 간 이벤트 통보

전통적으로 이런 시스템은 Redis (캐시/큐), Kafka/NATS (메시지 큐), Postgres (영속화)를 조합해서 구현한다. Open Match도 Redis를 핵심 의존성으로 사용한다.

설치형 OSS 도구로 배포할 경우, 사용자(게임사)가 직접 운영해야 한다. 의존성이 많을수록 도입 장벽과 운영 부담이 커진다.

## Decision

**외부 큐/캐시 의존성 없이 Postgres만 사용한다.**

- 분산 컨슈머 패턴: `SELECT ... FOR UPDATE SKIP LOCKED`
- 이벤트 통보: `LISTEN/NOTIFY` (선택적, polling이 기본)
- JSONB로 동적 attribute 저장
- 트랜잭션으로 복잡한 상태 전이 일관성 보장

단, runtime DB와 config DB는 분리 (ADR-006 참조).

## Consequences

### 장점

- **운영 부담 최소**: 사용자가 Postgres 하나만 관리하면 됨
- **Helm chart 단순**: 의존성 명세가 짧음
- **트랜잭션 일관성 자연스러움**: 매치 lifecycle의 복잡한 상태 전이를 native transaction으로 처리
- **JSONB로 동적 attribute 표현 가능**: 큐별 커스텀 schema 지원
- **운영자 친화적 디버깅**: SQL로 모든 상태 조회 가능

### 단점

- **Throughput 상한 존재**: 단일 Postgres는 결국 수만 QPS가 한계
- **Multi-region 시 복잡**: Logical replication 직접 설계 필요
- **GIN 인덱스의 range query 한계**: JSONB attribute의 SQL 쿼리 성능 제한

### 부하 검증

목표 100만 동접 + 600 RPS 환경에서 측정한 부하:

- CreateTicket: ~556 QPS
- Match insert: ~139 QPS
- 나머지 컴포넌트 polling: 1자리 QPS

단일 Postgres가 압도적으로 여유 있게 처리하는 영역.

## Alternatives Considered

### Redis Streams (Open Match 방식)

- 장점: 빠른 throughput, 분산 컨슈머 패턴 (XREADGROUP)
- 단점: 복잡한 상태 전이에 약함 (백필 등), 의존성 추가, Lua 스크립트 필요

### Kafka/NATS 등 메시지 큐

- 장점: 확실한 분리, 백프레셔 표현
- 단점: 트랜잭션 일관성 어려움, 운영 부담 큼

### Postgres + Redis 하이브리드

- 장점: 각자 잘하는 곳에 사용
- 단점: 두 시스템의 일관성 관리, 운영 부담

## Notes

부하 측면에서는 Redis가 빠르지만, 매치메이커는 **상태 전이가 많은 도메인**(ticket lifecycle, match lifecycle, 백필, 취소, 타임아웃)이므로 transaction이 일급 시민인 Postgres가 적합하다.

매치메이커의 본질적 어려움은 throughput이 아니라 매칭 품질, fairness, 다운스트림 통합에 있다 (ADR-003 참조).
