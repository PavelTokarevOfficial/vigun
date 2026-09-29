# Finde Clip v2

This is a modular monolith for importing Twitch Clips and rendering vertical short videos.

- `back/` is Go. Keep feature/application code independent from Postgres, S3, Rod, FFmpeg and Whisper.
- `front/` is Vue 3 + TypeScript and follows FSD: app → pages → widgets → features → entities → shared.
- Do not add a UI library; use Tailwind and local base components.
- Database changes require a new `back/migrations` pair; never auto-sync schemas.
- Media is stored through the storage boundary, never in the repository or database blobs.
- Keep browser automation isolated behind `ClipDownloader`; do not leak Rod selectors.
- Long work runs only in the worker via PostgreSQL jobs, never in HTTP handlers.
- For UI state that changes asynchronously on the server (job progress, completed renders, imported media and similar background changes), use WebSocket invalidation events instead of short-interval polling. Emit events only for actual state changes; an empty worker claim/update must not invalidate the UI. Keep REST as the source of truth: a WebSocket event tells the client to refetch affected data. Add reconnect/backoff and, when recovery requires it, only a low-frequency fallback refresh.
- Read and update `docs/` after meaningful architectural changes. Never commit secrets, models or generated media.
