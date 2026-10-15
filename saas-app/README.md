# RAGFlow SaaS — vertical slice

A self-contained, dependency-light SaaS workspace built by studying RAGFlow's
architecture (`internal/` Go backend + `web/` React frontend) and extracting the
SaaS-specific pieces: multi-tenant workspaces (organizations), membership with
RBAC, invitations, and usage metering/quota.

This `saas-app/` directory is an **independent** Go module + Vite app. It does
not import the RAGFlow codebase and runs with **zero external Go dependencies**
(stdlib `net/http` + in-memory store), so you can build and run it offline.

## What's included (user workspace slice)

| Backend (Go) | Frontend (React + Vite) |
|---|---|
| Auth: register / login / logout / bearer sessions | Login / Register (invite-aware) |
| Organizations: create / list / switch | Onboarding · workspace org switcher |
| Membership RBAC: owner > admin > member > viewer | Members table + role editing |
| Invitations: create / list / accept (register-with-code) | Invite dialog + copy-link + list |
| Usage: current snapshot + events + quota consume | Overview usage cards · recent activity |
| Plans catalog (free / pro / enterprise) | Billing page · plan comparison |

## Architecture

```
saas-app/
├── cmd/server/            Go entrypoint (net/http, Go 1.22+ pattern routing)
├── internal/
│   ├── entity/            domain models (User, Organization, Membership, ...)
│   ├── dao/               data-access layer (in-memory Store)
│   ├── service/           business logic + RBAC + quota (Auth, Organization, Usage)
│   ├── app/               wiring + HTTP handlers + routes + CORS
│   └── crypto/            password hashing / tokens (stdlib)
└── web/                   React + TypeScript + Vite frontend
```

Layering mirrors RAGFlow: `handler(HTTP) → service(business) → dao(data)`.
The `dao.Store` is intentionally DB-shaped so a SQL/GORM backend (MySQL, like
RAGFlow) can be dropped in without touching the service layer.

## Run

Backend (terminal 1):

```bash
cd saas-app
go run ./cmd/server          # listens on :8080 (override with PORT)
```

Frontend (terminal 2):

```bash
cd saas-app/web
npm install
npm run dev                  # http://localhost:5173 (proxies /api → :8080)
```

Then open http://localhost:5173, register an account, create a workspace, and
invite a teammate from the Members page.

## Tests

```bash
cd saas-app
go test ./...
# web
cd web && npm run type-check && npm run build
```

## Production hardening notes

- **Password hashing** in `internal/crypto` uses salted+iterated SHA-256 so the
  slice runs offline. Swap for argon2id/bcrypt (`golang.org/x/crypto`) in
  production.
- The in-memory `dao.Store` is non-persistent; add a real DB backend before any
  production use.
- Quota `Consume` is atomic in the in-memory store; the parallel SQL design
  (`UPDATE ... WHERE used + ? <= ?`) is documented in `internal/service/usage.go`.