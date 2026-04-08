# Domain-Aware Probing Patterns

When brainstorming a feature, identify its domain and use the matching probes to surface gray areas before planning. Load only the relevant domain — not the full file.

## Authentication

| User mentions | Probe |
|---|---|
| "login" / "auth" | OAuth, email/password, magic link, or SSO? |
| "sign up" | Required fields? Email verification? |
| "MFA" / "2FA" | TOTP, SMS, or passkey? Recovery flow? |
| "session" | Duration? Refresh strategy? Multi-device? |
| "roles" / "permissions" | RBAC, ABAC, or simple admin/user? Granularity? |

## Real-Time / WebSockets

| User mentions | Probe |
|---|---|
| "real-time" / "live" | WebSocket, SSE, or polling? Acceptable latency? |
| "notifications" | In-app, push, email, or all three? Batching? |
| "collaboration" | Conflict resolution strategy? Presence indicators? |
| "chat" / "messaging" | Message persistence? Read receipts? Typing indicators? |

## Dashboard / Data Display

| User mentions | Probe |
|---|---|
| "dashboard" | Data sources? How many distinct views? |
| "charts" / "graphs" | Interactive or static? Drill-down? Export? |
| "metrics" / "KPIs" | Refresh — real-time, polling, or on-demand? Acceptable staleness? |
| "admin panel" | Role-based visibility? Actions beyond viewing? |
| "table" / "list" | Pagination, infinite scroll, or load-all? Sorting? Filtering? |

## API Design

| User mentions | Probe |
|---|---|
| "API" / "endpoints" | REST, GraphQL, or RPC? Versioning strategy? |
| "pagination" | Cursor, offset, or keyset? Default page size? |
| "rate limiting" | Per-user, per-key, or global? Limits? |
| "webhooks" | Retry policy? Signature verification? |
| "file upload" | Max size? Allowed types? Direct-to-storage or through API? |

## Database / Storage

| User mentions | Probe |
|---|---|
| "database" | SQL or NoSQL? Why? Expected scale? |
| "migration" | Zero-downtime required? Rollback strategy? |
| "caching" | What layer? TTL? Invalidation strategy? |
| "search" | Full-text, fuzzy, or exact? Dedicated search engine? |
| "file storage" | Local, S3, or CDN? Access control? |

## UI / Frontend

| User mentions | Probe |
|---|---|
| "form" | Validation — client, server, or both? Multi-step? |
| "mobile" | Responsive web, native, or PWA? Offline support? |
| "dark mode" / "theme" | System preference, user toggle, or both? |
| "loading" / "skeleton" | Skeleton screens, spinners, or progressive? |
| "empty state" | What shows when there's no data? CTA? |

## Testing

| User mentions | Probe |
|---|---|
| "tests" | Unit, integration, e2e, or all? Coverage target? |
| "CI" | Which provider? Required checks before merge? |
| "staging" | Separate environment? Data strategy (seed, copy, synthetic)? |

## Deployment

| User mentions | Probe |
|---|---|
| "deploy" | Manual, CI/CD, or GitOps? Rollback strategy? |
| "environment" | How many? (dev, staging, prod) Config management? |
| "Docker" / "container" | Orchestration? Health checks? Resource limits? |
| "serverless" | Cold start acceptable? Timeout limits? |
