# AI-buddy implementation design

Build the supplied master spec as a Russian-language hackathon MVP. User explicitly delegates technical decisions and asks immediate implementation/deployment, so no design approval gate is needed. All records are demo records; real external integrations must report configuration and errors honestly.

## Approach
Use a Go modular monolith and PostgreSQL with React/Vite SPA; Docker Compose. This follows the requested stack, keeps operational complexity small, and supports protected server-side integrations. Alternatives of microservices or frontend-only demos do not fit reliable persistence and the requested scope.

## User experience
Light neutral product UI with dark text, blue primary actions, soft green progress, compact navigation. Intern: overview, onboarding, knowledge, chat, requests, learning, meetings, profile. Mentor: assigned interns, task/meeting assignment and request approval. Manager: team aggregates. Administrator: separate password login and encrypted Telegram/RouterAI configuration, connection tests, audit trail.

## Security
HttpOnly Secure sessions, same-origin mutation checks, signed Telegram initData with age check, one-use browser login via bot, server-side RBAC and ownership. Public demo role selection limited to synthetic/demo users. Admin is never selectable. Secrets encrypted using server key and omitted from JSON responses. Provider URL restricted to RouterAI origin to avoid SSRF. No real corporate data or access grants.

## Deployment
Existing VPS services preserved. Separate /opt/ai-buddy compose stack with database volume; backend bound to localhost; existing nginx gets a new host only. Use ai-buddy.85.159.231.99.sslip.io if DNS resolves and provision TLS. Polling bot single replica; scheduler can run in same process with DB guarded sends. External credentials entered later through admin UI.

## Completion evidence
Backend unit/integration tests, frontend build/tests, deployed health and API walkthrough including auth/roles, article completion, exam, request creation/approval/progress, admin masking and validation. Browser QA desktop/mobile. Report actual coverage without claiming unattained 80%. Docs and presentation material describe real implementation and remaining external configuration.
