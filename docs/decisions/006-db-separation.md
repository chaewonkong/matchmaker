# ADR-006: Runtime DB와 Config DB 분리

## Status

Accepted (2026-05-23)

## Context

매치메이커가 다루는 데이터는 크게 두 종류:

- **Runtime 데이터**: tickets, matches, ticket_pool_membership (운영 중 빈번한 read/write)
- **Config 데이터**: 큐/풀/룰 설정 + 변경 이력 (read-heavy, 가끔 write, 라이브 튜닝)

ADR-005에서 Config가 런타임 critical path에 들어왔다 (즉시 변경 반영 필요). 이 상태에서 한 DB에 둘지, 분리할지 결정 필요.

## Decision

**두 개의 Postgres 인스턴스로 분리한다.**

- `postgres-runtime`: inbox(없으면 생략), tickets, matches, ticket lifecycle
- `postgres-config`: queues, pools, rules, config_history, attribute_schemas

둘 다 Postgres이므로 운영 노하우는 공유된다.

## Consequences

### 장점

- **운영 격리** — Config DB 작업이 매칭 시스템에 영향 없음
  - 큰 트랜잭션, 마이그레이션, 운영자 실수가 운영 DB와 격리
- **다른 백업/보존 정책 가능** — Config는 장기 보존 + audit, Runtime은 짧게
- **다른 권한 모델** — Config DB는 운영자/관리자 접근, Runtime은 서비스 계정만
- **다른 scaling 전략** — Config는 read replica 적극 활용 가능
- **장애 격리** — Config DB 장애 시 cached config로 매칭 계속 동작 (ADR-002 stale cache 정책)

### 단점

- **운영 부담 증가** — Postgres 인스턴스 2개
- **Helm chart 복잡도 증가** — 의존성 명세가 길어짐
- **트랜잭션 분리** — Config 변경과 Runtime 상태 변경을 한 트랜잭션에 묶을 수 없음 (필요 없는 경우가 대부분이므로 큰 문제 아님)

## Alternatives Considered

### 단일 Postgres, schema로 분리

```sql
create schema runtime;
create schema config;
```

- 장점: 운영 단순, Helm chart 단순, 트랜잭션 자유
- 단점:
  - Config 변경 작업 (큰 트랜잭션, 마이그레이션)이 운영 DB에 영향
  - 백업 정책이 일률적 (transient runtime도 같이 백업)
  - Config DB 장애 = 매칭 시스템 장애

ADR-005에서 *Config가 런타임 critical이 되었기 때문에* 단일 DB는 부적합. 만약 Config가 GitOps 기반이었다면 단일 DB가 맞았을 것.

### 완전 분리 (3개 DB)

- runtime / config / inbox 각각 분리
- Over-engineering. 600 RPS 규모에서 정당화 어려움.

## Notes

이 결정의 핵심: **둘 다 Postgres라는 점**.

운영팀이 새로운 기술을 배우거나 새로운 운영 노하우를 쌓을 필요 없다. 백업, 모니터링, 튜닝 모두 같은 패턴. 그래서 "Postgres 2대"는 여전히 **minimalistic 원칙과 호환**된다.

만약 Config Store가 etcd, S3, Git 등 다른 시스템이었다면 운영 부담이 진짜 늘었을 것. Postgres-Postgres 분리는 부담 적음.

Sentry, Mattermost 같은 설치형 OSS도 비슷한 패턴: 운영 DB와 별개의 작은 DB(또는 schema)를 사용한다.

향후 multi-region 운영이 필요해지면:

- Runtime DB는 region-local primary
- Config DB는 global write + region read replica

자연스럽게 확장 가능한 구조.
