# Debug Checklists

Domain-specific checklists for inclusion in debug agent prompts. Include only the relevant checklist based on the identified bug domain.

## Symptom Triage

When the bug domain isn't obvious, use the symptom to route to the right checklist:

| Symptom | Check first |
|---------|------------|
| "Cannot read property of undefined/null" | API Bugs (response shape), Auth Bugs (missing token) |
| "X is not a function" | API Bugs (import/module), Async Bugs (missing await) |
| Works sometimes, fails sometimes | Async/Concurrency Bugs |
| Works locally, fails in CI/prod | Auth Bugs (env config), Database Bugs (schema drift) |
| Wrong data displayed | Database Bugs (stale data), API Bugs (query params) |
| Timeout or hang | Database Bugs (pool exhaustion), Performance Bugs |
| Memory leak / growing resource usage | Async Bugs (listener leak), Performance Bugs |
| 401/403 after deploy | Auth Bugs (token/session mismatch) |
| Slow response | Performance Bugs |

## API Bugs
- Does the route exist and match the HTTP method?
- Is auth middleware applied and in the correct order?
- Does the request body parse correctly (Content-Type header)?
- Are 4xx vs 5xx responses distinguishable? Is error shape consistent?
- Are query parameters validated and typed?

## Database Bugs
- Is the query correct? Run it manually with `EXPLAIN ANALYZE`
- Are migrations up to date? Check for schema drift
- Connection pool exhaustion? Check pool size vs concurrent requests
- Transaction isolation — are reads seeing stale data?
- N+1 queries? Log SQL count per request

## Auth/Authorization Bugs
- Token expired vs invalid vs missing — which case?
- Middleware ordering — does auth run before the handler that needs it?
- Role/permission check — is the check on the right resource?
- Session store cleared on deploy while long-lived tokens persist? Token signing key rotated without invalidating existing tokens?

## Async/Concurrency Bugs
- Race condition? Can two operations interleave on shared state?
- Deadlock? Are locks acquired in inconsistent order?
- Unhandled promise rejection or missing `await`?
- Event listener leak? Check listener count over time

## Performance Bugs
- Profile first — the slow part is almost never where you think
- Slow API response? Log SQL count per request — likely N+1 queries
- Slow page render? Profile the render cycle — likely expensive recomputation
- Gradual memory growth? Heap snapshots over time — likely leaked listeners or connections
- Intermittent slowness? Check connection pool metrics — likely lock contention or pool exhaustion
- Check for missing indexes and unbounded loops
