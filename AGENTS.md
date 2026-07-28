# AGENTS.md — Kartezya HR Backend

Tool-agnostic AI coding rules. Long architecture and endpoint lists live elsewhere; see conditional references.
Automatic loading depends on the tool; this is not guaranteed to work in every AI tool.

## A. Instruction hierarchy

- Root `AGENTS.md` is the project-wide normative source; a nearer scoped `AGENTS.md`, if present, only adds rules specific to its own folder.
- Adapter files must not define independent project rules and must not copy the main rules.
- User requests apply unless they violate security or repository policy; code and repository reality take precedence over stale documentation.

## B. Project overview

- **Language:** Go 1.25 · **HTTP:** Gin · **ORM:** GORM · **DB:** PostgreSQL
- **Auth:** JWT (+ Yandex OAuth) · **Authz:** capability-based (`internal/authz`) · **Entry:** `main.go` (manual DI)

## C. Architecture

```
HTTP Request → Handler → Service → Repository → PostgreSQL
```

- Preserve the layer chain; do not skip layers. Follow the existing `internal/` structure and constructor injection pattern.

## D. Task start and root cause

- Inspect the relevant flow end to end before editing; do not mask symptoms, find the root cause.
- Open only the files needed for the task; do not guess unknown behavior; state assumptions.
- Classify task risk (see K). For clearly low/medium bug fixes, do not wait for unnecessary approval.

## E. Authorization

- Do not assume a fixed ADMIN/EMPLOYEE model. Roles: `ADMIN`, `HR`, `FINANCIAL`, `EMPLOYEE`.
- Sources: `internal/authz/capabilities.go`, `BACKEND_API_ROLE_MATRIX.md`, `internal/middleware/auth_middleware.go`.
- On capability changes, check sync with FE `lib/authz/capabilities.ts`.
- UI/FE guards are not a security boundary; backend capability enforcement is mandatory.
- HR → ADMIN protection: `internal/authz/employee_protection.go`

## F. Database

- Do not hardcode `hr_` / `hr_test_` in table names; use `domain.GetTableName()` or the config prefix (including raw SQL).
- Hybrid migration: GORM AutoMigrate (`DB_AUTO_MIGRATE`) + `schema/` SQL; evaluate their combined impact together with production deploy.
- Evaluate transaction / partial-failure behavior and migration–API backward compatibility.
- High risk: data deletion, unique index, concurrency, production DB, cron/job (`internal/jobs/`).

## G. Locale, date and timezone

- Do not rely on system locale (developer/server); do not use implicit locale for date/time/number/currency/sorting.
- Do not parse display strings as if they were the storage/API format; use a stable locale-independent format in API/storage.
- Do not assume browser, backend, DB and scheduler share the same timezone.
- Do not hardcode `tr-TR` / `en-US` or another locale as a bug workaround; use a locale only if it is a product standard.
- Keep sorting/collation (e.g. Turkish characters) and BE/FE filter-date semantics consistent with product requirements.

## H. Production-first

- Produce production-deployable solutions; do not leave local-only or single-instance workarounds.
- Do not hardcode host/URL/DB prefix/SSL/credential/storage/callback; use config/environment.
- Do not treat local `.env`, config defaults or `.env.example` as production truth.
- Evaluate race, idempotency, transaction, retry, duplicate execution, locking and concurrency; assume multiple instances for jobs/cron.
- Do not treat timeout/sleep/delay/restart/manual refresh/cache clearing as a permanent fix; do not hide production errors with silent fallback.

## I. Git safety and user WIP

- Do not modify `main`/`master`; do not commit/push/PR/pull/fetch/merge/rebase without explicit request.
- Destructive Git (amend, force push, reset --hard, clean, branch deletion, stash pop/drop, loss via restore/checkout, cherry-pick/revert) is not allowed without explicit user approval.
- Do not alter user WIP under the pretext of revert/overwrite/format; do not touch out-of-scope files or do unnecessary refactors.

## J. Security

- Do not read or modify `.env` or secrets; do not log or output tokens/credentials/PII.
- Do not repeat secrets seen in prompt/terminal/files; redact them in reports. Do not make real service/API/DB calls.
- `.cursorignore` / `.geminiignore` are discovery filters, not a hard security deny.

## K. Task risk levels

| Level | Example | Approach |
|---|---|---|
| **Low** | Typo, type fix | Single pass; narrow context |
| **Medium** | Pagination, endpoint | Layer chain; package test |
| **High** | Auth, capability, migration, delete, job, prod DB | Read-only plan; apply after approval |

Do not write code without a plan for auth, migration, delete, job, concurrency and production DB.

## L. Validation

- Start from the validation closest to scope; widen as risk increases. `gofmt`; `go test ./internal/<package>/...`; if needed `go test ./...`; `git diff --check`.
- Brief PASS on success. Without an executed test/build, do not claim "works / fully resolved"; report skipped checks and remaining risk.

## M. Conditional references

Being listed does not mean reading it in every context. Adapters are not detailed guides.

- Open `docs/AI_CODING_GUIDE.md` only in these cases (read only the relevant section): (1) auth/capability, (2) migration/schema/data deletion, (3) background job/scheduler/concurrency, (4) production DB/config/deployment, (5) the user explicitly asks for a plan/workflow, (6) validation strategy cannot be decided from this file. "Cross-layer" alone is not sufficient.
- Open `docs/AI_TOKEN_OPTIMIZATION.md` only for AI instruction/token/tool compatibility/management reporting; do not open it during normal feature/bug fix/refactor/review.
- When needed: `BACKEND_API_ROLE_MATRIX.md`, `internal/config/config.go`, `.env.example`, `API_DOCUMENTATION.md`, `schema/schema.sql`, `JOB_MANAGEMENT.md`. `docs/project_analysis.md` may be stale.

> **Stale:** `README.md` / `docs/project_analysis.md` may describe an old ADMIN/EMPLOYEE model; for auth, rely on living code and capability sources.
