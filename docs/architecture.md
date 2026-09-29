# Architecture

API and worker are separate Go processes sharing one module. PostgreSQL holds streamers, clips, media metadata, assets, templates and jobs. The `media.Storage` boundary has an S3-compatible adapter, so MinIO, AWS S3 and R2 can be used without changing application logic. The worker claims pending jobs transactionally with `FOR UPDATE SKIP LOCKED` and executes Download → ExtractAudio → Transcribe → Render as restartable steps.

Pipeline updates are delivered as invalidation events rather than state snapshots. Row-level PostgreSQL triggers publish `NOTIFY` after actual changes to clips, processing jobs or media files; the API listens on a dedicated connection and fans those events out through `GET /api/events` over WebSocket. Row-level triggers are important here because the worker periodically runs claim statements that may affect zero jobs and must not cause a frontend refresh. This works across the separate API and worker processes. REST remains the source of truth: after an event, the frontend reloads the current clips, jobs and videos.

## Real-time UI rule

New screens with server-side state changed by workers or other asynchronous processes must follow the same event-driven pattern. Do not introduce polling every few seconds for job progress, media readiness, imports or similar state. Publish a narrow invalidation event, debounce related events in the client and fetch the authoritative state through the existing REST API. The client must reconnect with backoff and refresh immediately after reconnecting. A slow periodic refresh is acceptable only as recovery from a missed event, not as the primary update mechanism. Ordinary user-initiated reads and data that do not change in the background should continue to use REST without a persistent connection.

Video templates are declarative, versioned JSON. The API validates a template before a process job is created and stores an immutable snapshot with the job; the worker therefore renders the version selected by the user, not a later edit of the template.
