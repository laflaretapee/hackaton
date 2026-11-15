# Frontend implementation

React 19 + TypeScript + Vite, React Router, TanStack Query and Lucide. All product data and mutations use same-origin `/api` with cookie sessions. No frontend provider calls, secret persistence, mock data fallbacks or fabricated progress.

## Implemented

- Russian responsive workspace: compact rail, mobile navigation, clear headings, restrained blue actions/green progress, custom AI-buddy identity. Supports 360px layouts, semantic form labels, visible focus, skip link, accessible progress, loading/error/empty states, reduced motion, keyboard-trapped confirmation dialogs, and 44px mobile controls.
- Registration landing: server-controlled synthetic demo mode, name/team/role selection; signed Telegram Mini App initData passed unchanged to backend; browser bot link plus automatic token exchange polling. Unconfigured Telegram login is disabled with an explanation. Expired cached sessions return to login; user-scoped queries are cancelled and removed at expiry and before a new account enters. A login transition gate prevents stale profile rendering while the new `/me` resolves. Public config/teams and separate admin queries are preserved; initial network failures offer retry.
- Intern dashboard: real task progress, current tasks, mentor, nearest future meeting, next incomplete lesson, exam results and AI entry point.
- Task plan: filters, priority/deadline/material links, manual completion; access/exam tasks guide to relevant workflow and cannot be completed manually.
- Knowledge: server search, direction/category filters, real article sections and anchors, read completion, related exam, AI lesson context.
- Learning: lesson list, server-supplied five-question exam, complete-answer validation, real submission, result/feedback/recommendation and retry/AI explanation.
- AI: persisted history, questions, submission/loading/recovery, safe source links, clearly marked demo mode. Suggested access actions open an editable confirmation form. No mutation occurs until resource/reason and explicit confirmation are present. Failed answers stop typing indicators, preserve drafts and refresh persisted history.
- Requests: creation, status and history; mentor approve/reject mutations.
- Meetings: linked preparation material and server-created two-minute demo meeting; truthful notification statuses, including synthetic no-send status.
- Profile: notification settings, reminder interval, logout and confirmed demo deletion; shared seeded mentor offers a role switch and explains preserved shared data.
- Mentor: assigned intern list, progress/weak topics, detail with tasks, requests, meetings and exam scores/feedback/recommendations; assignment forms for tasks and meetings.
- Manager: server-scoped aggregate metrics and team progress table.
- Separate `/admin`: password login, masked server secret state, blank password fields preserving secrets, editable bot/model/provider/timeout/token-limit/app URL settings, explicit confirmed secret deletion, saved-configuration tests, audit and logout. Successful clear synchronizes integration switches before another save. Raw provider error bodies are never rendered.

## Commands

```sh
cd frontend
npm ci
npm run dev
npm run build
npm test
npm run test:coverage
npm run format:check
```

Development proxies `/api` and `/health` to `localhost:8080`. Production nginx listens on 80, proxies to `backend:8080`, preserves forwarded protocol, serves immutable assets, and falls back to the SPA. Multi-stage `Dockerfile` and `.dockerignore` included.

## Verification actually performed

Final local run on 2026-11-12:

- `npm run build`: passed strict TypeScript and Vite production build (390.68 kB JS, 117.58 kB gzip; 46.38 kB CSS).
- `npm run test:coverage`: **39 tests passed**, 2 files.
- Measured V8 coverage: **92.56% statements/lines, 86.75% branches, 90.41% functions**. Thresholds enforce at least 80% in each metric. All product pages are included; exclusions are tests/setup and entry wiring only. The type-only file is conservatively included as 0 runtime coverage.
- `npm run format:check`: passed.
- Tests mount actual React routes with an HTTP boundary fixture and assert displayed values and outgoing payloads. They cover five-answer exams, explicitly confirmed access requests, mentor decisions/assignments, manager scope/navigation, logout/reset, Mini App signed payload passthrough, source links, admin preservation/save/clear/test/logout, error recovery, and security regressions identified in review, including an expiry/account switch within the production 15-second cache window. That regression deliberately delays the new profile, observes the DOM throughout the switch, and resolves an old in-flight query after the new login to verify it cannot refill the cache. Regression tests were observed failing before the corresponding fixes.

These frontend tests do not prove Telegram delivery, live RouterAI behavior or production TLS. Parent project verification owns real API/database walkthroughs, public deployment and desktop/mobile browser screenshots. External integrations require credentials in `/admin`; the UI reports demo/configuration state honestly. The Google-hosted Golos Text font has a sans-serif fallback if unavailable.
