# Finde Clip v2

This is a modular monolith for importing Twitch Clips and rendering vertical short videos.

- `back/` is Go. Keep feature/application code independent from Postgres, S3, Rod, FFmpeg and Whisper.
- `front/` is Vue 3 + TypeScript and follows FSD: app → pages → widgets → features → entities → shared.
- Prefer shadcn-vue components for standard interactive UI controls. If a needed shadcn-vue component is missing, install it through the project shadcn configuration before writing a bespoke replacement. Use Tailwind for layout and project-specific presentation; do not add unrelated UI libraries.
- Keep pages focused on functional content. Do not add generic page-title/description hero blocks such as “Settings” plus explanatory filler when navigation and section headings already provide that context.
- Database changes require a new `back/migrations` pair; never auto-sync schemas.
- Media is stored through the storage boundary, never in the repository or database blobs.
- Keep browser automation isolated behind `ClipDownloader`; do not leak Rod selectors.
- Long work runs only in the worker via PostgreSQL jobs, never in HTTP handlers.
- For UI state that changes asynchronously on the server (job progress, completed renders, imported media and similar background changes), use WebSocket invalidation events instead of short-interval polling. Emit events only for actual state changes; an empty worker claim/update must not invalidate the UI. Keep REST as the source of truth: a WebSocket event tells the client to refetch affected data. Add reconnect/backoff and, when recovery requires it, only a low-frequency fallback refresh.
- Use the shadcn-vue `Sonner` integration (`vue-sonner`) for success, error, warning and informational notifications. Notifications are persistent by default and must remain visible until the user closes them with the close button. Do not add inline `notice`/flash paragraphs or another toast library. Keep persistent page state, field validation and background-job status visible in the relevant screen instead of turning them into toasts.
- Read and update `docs/` after meaningful architectural changes. Never commit secrets, models or generated media.
