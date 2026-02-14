## What this project is

Navidrums is a download orchestrator and metadata browser.

It is NOT a streaming server.
The UI never downloads music directly.

All downloads happen asynchronously via jobs and workers.

---

## Mental Model

User action
→ HTTP handler
→ service layer
→ repository state
→ worker execution
→ provider fetch
→ filesystem write
→ tagging

Responsibilities:

Handlers: HTTP coordination only
Services: workflow orchestration
Repository: persistent state
Workers: long running execution
Providers: remote catalog access
Filesystem: local storage operations
Tagging: metadata writing

Handlers must never talk to providers directly.

---

## Architecture Constraints

Allowed dependencies:

handlers → services
services → repository
services → providers
services → filesystem
worker → services

Forbidden dependencies:

repository → services
providers → repository
handlers → repository directly
handlers → providers
ui → providers

Filesystem writes only inside filesystem package.

---

## Job Lifecycle (Invariant)

Jobs can only transition:

pending → processing → completed
pending → processing → failed
pending → processing → cancelled

Rules:

- A cancelled job must stop ongoing work
- A job cannot return to pending
- A job cannot skip processing
- Workers must persist state transitions

---

## Data Invariants

- A track file must exist before tagging
- Providers are stateless
- Provider responses are never stored raw in DB
- Artist downloads aggregate album tracks
- Partial downloads must still finalize job state
- Deleting a job does not delete files automatically

---

## Forbidden Changes

Do NOT:

- download files inside handlers
- spawn goroutines inside handlers
- access DB outside repository
- block HTTP requests waiting for downloads
- call providers from UI layer
- write files outside filesystem package
- mutate job state outside services

---

## Implementing Features

When implementing anything related to downloads:

1. Add or modify service method
2. Update repository state if needed
3. Extend worker behavior
4. Update handler last

Never start implementation from handlers.

---

## Error Handling Rules

- Services return domain errors
- Handlers map errors to HTTP responses
- Workers must never panic
- All external calls must be retry-safe

---

## Concurrency Rules

- Workers own execution
- Services own orchestration
- Context cancellation must stop downloads
- No shared mutable state across goroutines
- Repository operations must be safe for concurrent workers

---

## Verification Procedure

After modifying download logic:

1. Queue album download
2. Confirm job stored in DB
3. Worker picks job
4. Files appear in download directory
5. Tags written
6. Job finishes with correct state

If any step fails, the implementation is incorrect.
