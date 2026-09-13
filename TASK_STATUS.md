# MVP v2 — статус выполнения

Последнее обновление: 2026-09-13

## Готово

- [x] Новый корневой layout и правила для агентов.
- [x] Docker Compose: PostgreSQL и MinIO с persistent volumes и healthchecks.
- [x] Контракт переменных окружения.
- [x] Go module, Dockerfile и начальная миграция PostgreSQL.
- [x] Начальная схема: streamers, clips, media_files, banners, processing_jobs.
- [x] Partial unique index prevents duplicate active jobs of the same type per clip; migration applied to local DB.
- [x] Go API entrypoint, healthcheck, automatic migrations и CRUD streamers.
- [x] Streamer edit safely resets cached Twitch user ID when login changes.
- [x] Twitch API client, получение clips выбранного streamer и import с download job.
- [x] Automatic Twitch login-to-user-ID resolution on first clips request.
- [x] Storage abstraction и S3-compatible implementation для MinIO/S3/R2.
- [x] Typed not-found handling in S3 `Exists` keeps storage errors visible.
- [x] FFmpeg и Whisper CLI adapters через `exec.CommandContext`.
- [x] Render settings moved to validated environment config.
- [x] Whisper model is validated only by worker, so API can run before the model is installed.
- [x] Rod/Chromium downloader isolated behind `processing.Downloader`.
- [x] Idempotent pipeline runner with temporary OS files and S3 artifact keys.
- [x] PostgreSQL job repository with claim, step, complete and fail lifecycle operations.
- [x] Worker wired to S3, Rod, FFmpeg and Whisper adapters for download/process jobs.
- [x] Pipeline REST endpoints and frontend status/Process/Retry page.
- [x] Read-only jobs API: list and detail DTOs.
- [x] Pipeline UI polls active jobs and displays current step/progress.
- [x] Clips frontend select, Twitch list and Save/import flow.
- [x] Clips UI marks already imported Twitch clips; Pipeline and Ready Video cards show all required metadata/actions.
- [x] Ready Videos API with S3 presigned URLs and frontend page.
- [x] Worker persists source/audio/subtitle/render metadata in `media_files`.
- [x] Pipeline reports actual ExtractAudio/Transcribe/Render progress through `processing_jobs.current_step`.
- [x] Download completion preserves `downloaded` clip status, so the Pipeline UI exposes Process; a migration repairs older download-only clips.
- [x] Process jobs accept an optional banner UUID; an empty selection is stored as SQL NULL.
- [x] Runner idempotency unit test verifies retry skips existing source/audio/subtitle/render artifacts.
- [x] Current API and worker Docker images rebuilt after backend changes (2026-09-07).
- [x] S3-backed banner service foundation (list, upload, delete).
- [x] Banners REST API and frontend upload/list/delete page.
- [x] Go worker entrypoint и безопасное конкурентное claim jobs через `FOR UPDATE SKIP LOCKED`.
- [x] Graceful worker shutdown requeues interrupted jobs instead of incorrectly failing them.
- [x] Worker emits structured per-step logs with job, clip and progress context.
- [x] Worker logs database claim errors while keeping an empty queue quiet.
- [x] Документы архитектуры, БД и media pipeline.
- [x] README, backend и frontend local-development documentation.
- [x] Поиск клипов добавляет только в избранное, без автоматического download job.
- [x] Pipeline переделан в три колонки: избранное, скаченное и готовые видео; доступны кнопки и drag-and-drop для переходов.
- [x] Удаление избранного/скачанного/failed клипа без рендеров очищает объекты из S3 и связанные PostgreSQL-записи.
- [x] В готовых видео можно удалить конкретный render, не затрагивая исходник и другие результаты.
- [x] Стримеров можно добавлять списком: один Twitch-ник на строку.
- [x] Header переведён на shadcn-vue Dropdown Menu с Lucide-шестерёнкой.
- [x] Очередь process jobs отображается на карточке и блокирует повторный запуск.
- [x] Отдельная страница готовых видео удалена: плеер и ссылки находятся в третьей колонке Pipeline.
- [x] Добавлены PostgreSQL migrations для asset folders, assets, video templates и immutable template snapshots в process jobs.
- [x] Добавлены S3-backed API и страницы для загрузки, preview, создания/переименования/перемещения/удаления ассетов и папок.
- [x] Добавлен список шаблонов, duplicate/delete и визуальный редактор: слои, canvas 9:16, drag/resize, свойства, undo/redo и dirty warning.
- [x] Pipeline требует выбрать шаблон; выбранная конфигурация и S3 keys ассетов фиксируются в task snapshot.
- [x] Worker компилирует template layers в FFmpeg filter graph и сохраняет render по `jobID`, без перезаписи результатов других шаблонов.
- [x] API/worker Docker containers пересобраны, migration применена; `/api/templates` возвращает default template.
- [x] У шаблонов есть единственный default; базовый шаблон копируется в render editor и может быть изменён без сохранения перед enqueue.
- [x] Редактор показывает скачанный Twitch-клип и выбранные image/GIF/video assets, использует пять понятных типов слоёв и проводник выбора ассетов.
- [x] Слои video поддерживают Twitch-клип или video asset, text — имя канала, blur — силу, затемнение и прозрачность.
- [x] Скачанный клип остаётся доступен после completed и поддерживает повторные рендеры с отдельными immutable snapshots.
- [x] Добавлен Vue timeline editor: trim, split и удаление сегментов реально компилируются worker-ом в FFmpeg trim/concat graph.
- [x] Таймлайн расширен до многодорожечного монтажа: перестановка частей исходника, вставка video assets в позицию курсора, временные диапазоны для всех слоёв и корректный FFmpeg concat с тишиной для роликов без аудио.
- [x] Выделение синхронизировано между canvas, слоями и таймлайном; split/hide/delete работают с выбранным объектом, импортированные ролики видны отдельным типом, добавлены zoom и «вместить целиком».
- [x] Выделение блока больше не переносит курсор; разделённые части обычного слоя группируются по `trackId` и остаются на одной дорожке.
- [x] Верхняя дорожка явно обозначена как монтаж, «разделить монтаж и все слои» режет все пересекающие курсор объекты, Shift-выделение подсвечивается пунктиром и перемещается единой группой без сдвига интерфейса.
- [x] Обрезка левого или правого края одного Shift-выделенного фрагмента синхронно применяется ко всем выделенным фрагментам с общей проверкой границ и минимальной длительности.
- [x] Вставленный video asset связан с отдельным слоем через `timelineSegmentId`, виден на canvas, управляется по z-order и рендерится worker-ом как полноценное видео, а не только как звук.
- [x] Image/GIF в режиме `contain` сохраняют исходную прозрачность, а свободное место при сохранении пропорций дополняется прозрачным padding без чёрных полос.
- [x] Удаление карточки готового видео удаляет только render object и metadata; скачанный source остаётся доступен для нового рендера.
- [x] Повторные рендеры одного исходника отображаются отдельными карточками и удаляются независимо.
- [x] Удаление исходника сохраняет готовые рендеры и скрывает clip из «Скачанных» по признаку `hasSource`.

## В работе

- [x] Go API: jobs, banners и media URLs.
- [x] Go worker: Download/ExtractAudio/Transcribe/Render adapters.
- [x] Connect worker jobs to pipeline runner and persist media/status transitions.
- [x] Vue 3 + TypeScript + Tailwind frontend с FSD layers: app/pages/widgets/features/entities/shared.
- [x] Vue 3/Vite/FSD structure и Streamers UI с add/edit/delete.
- [x] Автоматические migrations, Go test и frontend build-проверки.
- [x] Go static analysis (`go vet ./...`).

## Local run note

- [x] Docker Compose syntax validated.
- [x] Local PostgreSQL and MinIO started successfully and passed healthchecks.
- [x] API smoke test against local PostgreSQL and MinIO: migrations, `/health` and Streamers JSON contract passed.
- [x] Final API Docker image starts and serves `/health` without `WHISPER_MODEL_PATH`; Whisper remains worker-only.
- [x] Jobs API smoke test against local PostgreSQL and MinIO.
- [x] ARM-compatible `whisper.cpp` Docker build stage and `whisper-cli --help` verified.
- [x] Local `models/ggml-tiny.bin` downloaded (ignored by Git) and loaded by production `whisper-cli`; a WAV-to-SRT smoke test passed.
- [x] Worker Docker runtime image includes Chromium, FFmpeg with the required subtitles/libass filter, and compiled whisper-cli; model stays a mounted volume. Image build and all three binaries were verified locally.
- [x] Production FFmpeg smoke test created an MP4 with an SRT overlay successfully inside the final Docker image.
- [ ] Existing local `.env` predates v2. Copy missing v2 variables from `.env.example` before running API/worker containers.

## Не начато

- [x] Полный local flow на настоящем Twitch clip: Rod/Chromium download → MinIO source → FFmpeg audio → Whisper SRT → FFmpeg vertical render → MinIO completed render. Проверено 2026-09-07 без баннера; render занял около 69 секунд.
- [ ] Production-ready README и остальные документы.
