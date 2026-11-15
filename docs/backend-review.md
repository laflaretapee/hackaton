# Backend security and functional review — 2026-11-12

Scope: current Go backend, `docs/api-contract.md`, relevant requirements in `docs/master-spec.md`, SQL schema, and proxy configuration. Source review only; no production credentials or private access files were inspected. Backend implementation was changing during review, so resolved observations below refer to the current files rather than the initial snapshot.

## Result

No new P0/P1 authorization or secret-disclosure blocker was found. The backend checks sessions and object ownership on protected operations; public demo users are separated from Telegram identities. Administrator authentication is distinct from user authentication. AI tool execution uses the authenticated user, checks request ownership, and only prepares access requests for explicit confirmation.

## Observations reported during review

1. **Resolved P2 — bootstrap environment could prevent restart despite persisted administrator settings.** `backend/seed.go` previously validated environment-derived integration configuration before its `INSERT ... ON CONFLICT DO NOTHING`. Reproduction: initialize valid settings, restart with `AI_MOCK_MODE=false` while model/key are blank in the environment because they are stored in the administrator database. Startup failed before loading the persisted settings. The new guard at `backend/seed.go:72` skips bootstrap configuration when settings exist. Reviewer independently ran `TestPersistentSettingsSurviveBlankEnvironment` against the isolated PostgreSQL test database; it passed (3.764 seconds).
2. **Resolved P2 — failed Telegram updates were acknowledged without retry.** `backend/telegram.go`, `botLoop`, previously advanced/persisted the update offset even when `handleUpdate` failed. Reproduction: a role-selection callback encounters a transient database failure in `saveBotState`; the next `getUpdates` uses the following offset, so Telegram does not redeliver the action. Current `backend/telegram.go:149` retries three times with backoff, preserves cancellation, and records an update-ID-only dead-letter audit after exhausting retries. Bot-state, existing-user, and browser-login-candidate query errors now propagate to that loop. After three failed attempts the user may still need `/start`; this is a deliberate bounded-retry policy, not unlimited guaranteed delivery. Source reviewed; the full failure-injection integration suite remains the implementer's verification responsibility.
3. **Resolved P2 — proxy timeout shorter than permitted AI timeout.** Administrator configuration allows 120 seconds, while the host proxy originally stopped reading after 90 seconds. A 95-second provider response could produce a gateway error despite being within the configured limit. Both proxy configurations now allow 190 seconds, exceeding the backend's 150-second write timeout and the chat's 120-second total deadline.

## Previously identified issues checked in current source

- Chat accepts up to 6,000 characters and truncates only the search query to 200 characters. The entire multi-tool chat shares one configured deadline.
- Request creation and exam completion use transactions. Request decisions lock the request row and reject repeated decisions. GitLab task completion is restricted to repository resources and the matching `gitlab-access` task; a subsequent rejection cannot regress a completed access task.
- Telegram HMAC includes the `signature` field and rejects duplicate fields and stale/future identities. Browser login tokens are random, stored as hashes, expire, require bot confirmation, and are consumed atomically.
- Polling offsets are keyed by bot-token hash rather than shared across token changes.
- Demo mentors/managers cannot access real Telegram users. Initial real-team mentor registration is an operational prerequisite for real approvals; later mentor registration adopts unassigned real interns. This isolation must remain intact.
- Stored integration credentials use authenticated AES-GCM encryption with random nonces. Settings responses expose only presence flags; blank credential inputs preserve the existing secret, explicit clearing is supported, and audit actions omit credentials. RouterAI destinations are restricted to the configured official endpoint and redirects are disabled.
- Session cookies are HttpOnly and Secure in production. Mutations reject cross-site fetch metadata and foreign Origin values. Telegram embedding can still make same-origin API requests. The supplied proxy replaces forwarded client IP headers, supporting administrator login throttling.
- SQL parameters are bound. User deletion cascades personal records, sessions, and login links. Reminder sends claim the pending record before sending, preserve retry state on provider errors, and recover interrupted sending states on restart.

## Verification and limits

Independent reviewer command passed on current source:

```sh
GOCACHE=/private/tmp/ai-buddy-go-cache GOPATH=/private/tmp/ai-buddy-gopath \
  /private/tmp/ai-buddy-go127/go/bin/go test ./... \
  -run 'TestHTTPAuthBoundary|TestEncryptRoundTripAndTampering|TestTelegramSignatureFreshnessAndTampering|TestScopeIsolation|TestSettingsValidationRejectsSSRFAndMissingSecrets|TestExamGradesServerAnswers|TestChatRateLimits|TestWeekendMeetingScheduling' \
  -count=1
```

Result: `ok ai-buddy/backend` (0.331 seconds, rerun after the Telegram changes). The focused persisted-settings PostgreSQL regression also independently passed (3.764 seconds), using `-run '^TestPersistentSettingsSurviveBlankEnvironment$' -count=1` with the isolated test DSN supplied outside the report. The implementer owns the full PostgreSQL suite and coverage measurement; this reviewer did not independently rerun the full mutable integration suite or call live Telegram/RouterAI providers.

Non-blocking differences from the full master specification: RouterAI transport failures have a clear error and bounded deadline but no automatic retry; recommendations use deterministic exam feedback rather than a separate AI recommendation pipeline; seed/schema migration evolution is minimal; notification delivery is at-least-once across a crash after provider success but before database acknowledgement. These do not invalidate the documented MVP API. Strict exactly-once external delivery would need provider-side idempotency or an explicit delivery protocol.
