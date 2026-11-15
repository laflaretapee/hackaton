# Backend implementation

Go 1.26+ HTTP modular monolith, PostgreSQL/pgx, one process for API, Telegram polling and reminders. Docker builds with Go 1.27 and runs as an unprivileged user with CA certificates and timezone data.

## Run

Required server environment:

- `DATABASE_URL`: PostgreSQL connection string.
- `ADMIN_PASSWORD`: at least 12 characters, distinct from user authentication.
- `ENCRYPTION_KEY`: at least 32 characters; 64 random hexadecimal characters are suitable. This key is SHA-256 derived into an AES-256-GCM key. Keep it stable across restarts and back it up securely with database recovery procedures.
- `APP_URL`: public application origin, e.g. `https://your-app.example`.
- `SECURE_COOKIES=true` in production. HTTPS cookies use HttpOnly and SameSite=None for Telegram embedding; mutation Origin/Sec-Fetch-Site checks enforce the configured application origin.
- `TRUST_PROXY=true` only behind a trusted proxy that overwrites incoming X-Forwarded-For; otherwise leave false. Backend should not be publicly exposed when trusting this header.
- `PORT=8080`, `TZ=Asia/Yekaterinburg`, `DEMO_MODE=true` optional defaults.

On initial empty-database initialization only, optional integration bootstrap values are `TELEGRAM_BOT_TOKEN`, `TELEGRAM_BOT_USERNAME`, `TELEGRAM_ENABLED=false`, `ROUTERAI_API_KEY`, `ROUTERAI_BASE_URL=https://routerai.ru/api/v1`, `ROUTERAI_MODEL`, `AI_MOCK_MODE=true`, `AI_REQUEST_TIMEOUT_SECONDS=60`, `AI_MAX_TOKENS=1500`. After initialization, saved admin settings take precedence. Blank secret edits preserve existing values; explicit clear flags remove them. Tokens and keys are AES-GCM encrypted in PostgreSQL and never included in settings responses or audit actions.

`go run .` starts the API, automatically applies the idempotent SQL migration and seed, then starts bot/scheduler goroutines. Use one replica; no distributed polling coordination is implemented.

## Implemented behavior

- Signed Telegram Mini App authorization validates bot HMAC including all fields except hash, checks 15-minute freshness, rejects duplicates and fabricated user IDs. Users register through role → direction → team bot steps. Browser login tokens are random, hashed at rest, expire after 10 minutes, require explicit bot confirmation, and are consumed atomically once.
- Separate admin sessions, constant-time password comparison, 10 failed login attempts per IP per 15 minutes, cookie revocation and same-origin mutation checks.
- Profile deletion atomically removes the user and Telegram registration state, including its pending browser-login token; foreign-key cascades revoke bound login links and sessions and remove tasks/progress/scheduled reminders. Unrelated Telegram registration states remain intact, and the Telegram ID can immediately register again. Shared synthetic mentor fixtures retain the existing session-only reset behavior.
- Public synthetic demo roles are limited to synthetic users. Shared seeded mentors see newly created demo interns in their own team; Telegram accounts use a separate scope. Managers see their own team. Mentors can mutate only assigned interns. Private AI conversations are never in mentor dashboards.
- Persistent tasks, articles/read marks, lessons, server-graded exams/feedback, messages, requests/history, meetings, settings/audit. Attempts and progress changes use one transaction; requests and approvals similarly update progress atomically. Access tasks cannot be manually completed. A later rejected request cannot revoke an already completed step. Only supported repository resource aliases affect the GitLab onboarding task; other resources retain their own request status without completing that task.
- 15 substantive articles, 10 lessons with five questions each, all eight requested directions, eight synthetic interns, eight shared mentors and two seeded managers. Saved initial results, varied task progress, pending/approved requests and upcoming meetings are seeded once; restarting preserves user progress.
- PostgreSQL Russian full-text search with title weighting and broader OR-term fallback. Server-generated links point to actual article section anchors.
- Real server-side RouterAI OpenAI-compatible chat, bounded history and actual profile/progress/knowledge context, tools with server-side ownership checks, up to four tool rounds, and explicit confirmation actions for access requests. The model alone cannot execute access mutations. AI requests have one total configured timeout, 12/user/minute, 120/global/minute, and eight concurrent calls. Mock responses explicitly identify themselves as excerpts without a model call.
- Admin Telegram getMe connection test; RouterAI catalog and an actual forced get_user_profile tool call for a configured model. Provider URLs are restricted to https://routerai.ru/api/v1, redirects are not followed, and upstream response bodies containing possible sensitive data are not returned as errors.
- Telegram single-instance long polling, per-token-hash durable offsets, removal of stale webhooks, three bounded processing attempts, audited dead letters without message content. New real mentors receive unassigned real interns in the same team.
- Durable reminders honor user settings and avoid sending already sent events. Failed sends retry after five minutes; unconfigured integrations wait. Users without Telegram receive a visibly marked `demo_reminder` status; no notification is claimed to have been delivered. Initial meetings use the next weekday at 11:00 server timezone; demo meetings are due in two minutes. Expired sessions/login links and old registration states are cleaned up.

## Storage design and limits

The MVP uses a hybrid schema: normalized relational users, teams, sessions, login links and configuration metadata, with indexed JSONB catalog/domain documents owned through a user foreign key. It does not claim a fully normalized domain schema or vector/embedding search. Database statements are parameterized. The schema and data live in `migrations/001_init.sql` and seed code.

Corporate access is an explicitly labeled mock, not a GitLab integration. Real RouterAI and Telegram code paths have deterministic HTTP-fixture tests; deployment with actual secrets and a human Telegram account must still be verified after configuration. Self-selected Telegram roles are a synthetic demonstration, not corporate IAM.

One process is required for polling/scheduling. A process crash after Telegram accepts a reminder but before the DB marks it sent can cause a repeated notification; Telegram sendMessage does not support an application idempotency key. Three failed bot-update processing attempts are audited then acknowledged to prevent a permanently malformed update blocking all registrations. There is no general demo-user retention policy; expired sessions are cleaned, and users can delete their own profiles. Deleting the public shared demo mentor ends that browser session rather than deleting the shared fixture. No email, calendar, corporate SSO or real access provisioning is integrated.

## Tests

`go test ./...` runs the unit and HTTP boundary tests. PostgreSQL tests skip unless `TEST_DATABASE_URL` is set. Use a **separate test database**: integration tests temporarily change integration settings and recreate fixed synthetic seed users.

```
TEST_DATABASE_URL='postgres://.../ai_buddy_test?sslmode=disable' go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go vet ./...
```

Tests cover crypto tampering, Telegram signatures/freshness, RBAC, deterministic grading, cookie boundaries, entire intern→mentor request/approval flow, persisted exam round trips and redaction, read marks, assignments, team scoping, admin masking/blank-secret preservation, live-provider HTTP fixtures/tool confirmation, bot registration/browser-token replay, reminders, token-specific poll offsets, retry/deadletter behavior, idempotent seed/settings restarts, and complete transactional Telegram profile erasure with unrelated-state preservation and repeat registration. Latest executed full PostgreSQL suite: `go test -race -coverprofile=coverage.out ./...` passed in 60.041 seconds with **80.2% statement coverage**. `go vet ./...` passed with Go 1.27.4. Coverage artifacts are `coverage.out` and `coverage-summary.txt`. Live production credentials were not used by these fixture tests.
