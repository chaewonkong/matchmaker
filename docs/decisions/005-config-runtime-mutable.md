# ADR-005: Config 런타임 변경 가능 (DB 기반, GitOps 불채택)

## Status

Accepted (2026-05-23)

## Context

큐/풀/매칭 룰 설정의 저장 및 변경 메커니즘을 결정해야 한다.

선택지:

1. **GitOps 방식**: YAML 파일을 Git으로 관리, Helm/ConfigMap으로 배포, 변경 시 재배포
2. **DB 기반 런타임 변경**: API로 설정 변경, 즉시 반영

일반적인 OSS 인프라 도구(Prometheus, Argo Workflows 등)는 GitOps를 선호한다. 그러나 라이브 운영 게임의 매치메이커는 다른 특성을 가진다.

## Decision

**DB 기반 런타임 변경 API를 채택한다. GitOps는 부적합.**

- 설정은 별도 Postgres (Config Store, ADR-006)에 저장
- 운영자가 Config API로 변경 → 즉시 반영
- Config 변경 이력은 `config_history` 테이블에 보관 (audit, rollback)
- 변경은 atomic transaction + version 단조 증가
- Syncer가 polling으로 변경 감지 (ADR-002)

## Consequences

### 장점

- **라이브 튜닝 가능** — MMR 범위, 매칭 룰 등을 분 단위로 조정
- **롤백 자연스러움** — 이전 version으로 즉시 복원
- **canary 적용 가능** — 한 큐에만 먼저 적용 후 확장
- **재배포 사이클 회피** — 게임 서버 무중단 운영과 호환

### 단점

- **GUI/API 실수 위험** — 잘못된 룰이 즉시 운영에 영향 → 검증 단계 필요
- **버전 관리가 Git에 비해 약함** — config_history 테이블로 보강
- **외부 도구 통합 어려움** — Git 기반 워크플로우와 자연스럽지 않음

### 필수 검증 단계

운영자의 룰 변경 시 Config API는 다음 검증을 수행:

1. **문법 검증** — CEL 파싱
2. **타입 검증** — referenced attribute가 schema에 있는지
3. **시맨틱 검증** — dry-run으로 sample ticket에 적용
4. **canary 옵션** — 한 큐에만 먼저 적용
5. **atomic write + version 증가**

특히 dry-run과 canary는 라이브 운영의 안전성에 결정적.

## Alternatives Considered

### GitOps (YAML + Helm ConfigMap)

검토했으나 다음 이유로 부적합:

**이유 1: 큐 설정과 게임 서버 파라미터의 바인딩**

큐 설정은 게임 서버가 보내야 할 파라미터에 바인딩되는 경우가 많다. 큐 설정 롤백 시 게임 서버도 롤백해야 하는데, 게임 서버는 무중단 롤백이 현실적으로 어렵다. 한 번 적용한 설정의 롤백이 *기술적으로 가능해야* 한다.

**이유 2: 라이브 운영의 빠른 튜닝 사이클**

신규 게임 출시 초기 MMR 분포가 정규분포로 수렴하기까지 몇 주~몇 달 걸린다. 그 동안 지표 보며 분 단위로 룰을 튜닝해야 하는데, 배포 프로세스(PR → review → merge → deploy)는 이 사이클과 안 맞는다.

**이유 3: 운영자 권한 모델**

게임 운영팀이 PR 권한 + CI/CD 권한을 갖는 모델은 보안/책임 측면에서 부적합.

### 하이브리드 (DB + Git 백업)

DB가 primary, Git은 audit/backup 용도. 검토 가치 있으나 복잡도 증가 대비 이득 명확하지 않음.

## Notes

이 결정이 다른 결정들에 미치는 영향:

- **ADR-006 (DB 분리)**: Config가 런타임 critical이 되므로 runtime DB와 격리 필요
- **ADR-002 (Polling)**: Config 변경 반영 latency 1~10초가 운영적으로 충분
- **ADR-005-ME-version-propagation**: Config 변경 시 in-flight ticket 처리 정책 명확화 필요

Unity Matchmaker, AWS FlexMatch, PlayFab Matchmaking 등 상용 매치메이커가 모두 런타임 설정 변경을 지원하는 것으로 보인다. 라이브 게임 운영의 표준 요구사항.
