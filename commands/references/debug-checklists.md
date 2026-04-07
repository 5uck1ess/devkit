# Debug Checklists

Domain-specific checklists for inclusion in debug agent prompts. Include only the relevant checklist based on the identified bug domain.

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
- Check for N+1 queries, missing indexes, unbounded loops
- Memory leak? Compare heap snapshots over time
- Connection pool or thread pool exhaustion?

### Quick Symptom Lookup

| Symptom | Likely Cause | Investigation |
|---------|--------------|---------------|
| Slow API response | N+1 queries | Log SQL count per request |
| Slow page render | Expensive recomputation | Profile render cycle |
| Gradual memory growth | Leak (listeners, connections) | Heap snapshots over time |
| Intermittent slowness | Lock contention / pool exhaustion | Connection pool metrics |
