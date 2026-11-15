# Working on AI-buddy

- Requirements: `docs/master-spec.md`; implemented contract: `docs/api-contract.md` and `docs/openapi.json`.
- Frontend: React/TypeScript/Vite in `frontend/`. Backend: Go monolith in `backend/`, PostgreSQL migrations embedded at build time.
- Never expose stored Telegram/RouterAI secrets, passwords, initData or session tokens in client code, API errors, logs or Git.
- Preserve demo/Telegram user isolation and distinct administrator authentication. Check object ownership on every protected operation.
- AI prepares access requests; only explicit confirmation creates them. Only assigned mentors approve; unrelated resources and later rejection must not regress completed GitLab progress.
- Keep public mock labels honest. Real Telegram/RouterAI checks require credentials; fixtures do not prove live service delivery.
- Run frontend `npm run test:coverage` and `npm run build`. Backend full coverage needs a separate `TEST_DATABASE_URL`: `go test -race -coverprofile=coverage.out ./...`.
- Database integration tests mutate fixture settings and recreate seed users. Never point them at the running application database.
- Deployment is a separate `/opt/ai-buddy` Compose project. Preserve unrelated VPS applications and nginx hosts. Use one backend replica for polling/scheduling.
- Server secrets are in ignored `.env`; owner handoff is ignored `.deploy/access.txt`. Keep ENCRYPTION_KEY stable for existing encrypted credentials.
- Record behavior changes and actual verification in `docs/implementation-status.md`.
