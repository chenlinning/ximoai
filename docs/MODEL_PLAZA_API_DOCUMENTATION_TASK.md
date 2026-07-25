# Model Plaza Dynamic API Documentation Task

## Status

- Design status: validated against the current gateway routes and platform metadata.
- Implementation status: completed and verified locally on 2026-07-25.
- Validation baseline: XimoAI branch `deploy/v0.1.164`, based on upstream sub2api `v0.1.164`.
- Prerequisite: the XimoAI Volcengine Agent Plan endpoints must be retained and tested before this task is completed.
- Browser validation: passed in regular-user and administrator modes at desktop and 390 px mobile widths, in light and dark themes.

## Objective

Add dynamic API documentation to each Model Plaza card without creating a separate documentation settings page.

The same documentation dialog must be used by all authenticated users:

- Regular users can view and copy documentation.
- Administrators can open an editor inside the same dialog.
- Administrators configure structured endpoint bindings instead of manually writing complete documents.
- The system generates the final document from the selected category, protocol, endpoint, transport, delivery format, and invocation mode.

The feature must support one model having multiple categories, multiple endpoints, and multiple invocation modes at the same time.

## Success Criteria

1. Clicking a Model Plaza card opens documentation for that exact visible platform and model.
2. A model can expose multiple categories, endpoint profiles, and invocation variants concurrently.
3. Synchronous, streaming, asynchronous, and bidirectional workflows render correctly.
4. HTTP JSON, HTTP SSE, HTTP binary, and WebSocket frame transports are described separately.
5. All six XimoAI Volcengine Agent Plan endpoints have first-class documentation profiles.
6. Custom platforms receive useful defaults from their configured entry protocol and capabilities.
7. Administrators edit bindings in the same dialog; no standalone settings page is added.
8. Regular users cannot create, update, or delete documentation bindings.
9. Documentation never exposes an upstream URL, upstream credential, user API key, account state, or model mapping target.
10. Upstream route changes cannot silently leave stale endpoint documentation in a successful build.
11. Model Plaza provides a platform-aware secondary category filter sourced from the effective documentation binding.

## Non-Goals

- Do not proxy model requests through the documentation API.
- Do not infer protocol, capability, or reasoning settings from a model name.
- Do not expose upstream platform Base URLs or account credentials.
- Do not add a general-purpose rich-text documentation CMS.
- Do not parse `docs/EXTERNAL_API.md` at runtime.
- Do not refactor upstream gateway route registration into a new framework.
- Do not modify payloads, responses, SSE events, or WebSocket frames used by model calls.

## Core Model

The relationship is intentionally one-to-many at every level:

```text
Model
  -> zero or more categories
    -> one or more endpoint profiles
      -> one or more invocation variants
        -> one or more workflow steps
```

Examples:

- One text model can support both OpenAI Responses and Chat Completions.
- One Responses endpoint can support synchronous JSON and streaming SSE.
- One multimodal model can expose both conversation and image documentation.
- One video model can expose create, status, and content workflow steps.
- `seed-tts-2.0` can expose HTTP synchronous TTS, WSS unidirectional streaming TTS, and WSS bidirectional TTS at the same time.

## Terminology

### Category

The user-facing model function:

- `conversation`
- `image`
- `video`
- `tts`
- `asr`

Categories are multi-select. A model is not restricted to one category.

### Entry Protocol

The client-facing request contract, not the upstream account implementation:

- `openai_responses`
- `openai_chat_completions`
- `anthropic_messages`
- `gemini_generate_content`
- `openai_images`
- `openai_videos`
- `openai_audio`
- `volcengine_agent_plan_native`

### Transport

- `http`
- `websocket`

### Delivery Format

- `json`
- `sse`
- `binary`
- `websocket_frames`

Transport and delivery format are separate. SSE is delivered over HTTP; it is not a separate transport.

### Invocation Mode

- `sync`
- `stream`
- `async`
- `bidirectional`

Invocation mode and transport are also separate. For example, Volcengine asynchronous ASR uses WebSocket transport.

### Endpoint Profile

An immutable, code-owned definition of a real public endpoint. It defines the allowed combinations of method, path, transport, delivery format, invocation mode, request template, response template, and workflow steps.

Administrators select endpoint profiles and variants. They do not enter arbitrary methods or paths.

## Endpoint Profile Structure

The backend registry should use a structure equivalent to:

```go
type ModelDocsEndpointProfile struct {
    ID           string
    Category     string
    Protocol     string
    Title        string
    Description  string
    Variants     []ModelDocsEndpointVariant
}

type ModelDocsEndpointVariant struct {
    ID             string
    Mode           string
    Transport      string
    Delivery       string
    Steps          []ModelDocsWorkflowStep
}

type ModelDocsWorkflowStep struct {
    ID              string
    Method          string
    PathTemplate    string
    ContentType     string
    RequestTemplate any
    ResponseTemplate any
}
```

The concrete DTO may differ, but it must preserve all dimensions. A flat document with one `path`, one `transport`, and one `mode` is not sufficient.

## Saved Binding Structure

Only administrator selections are persisted. Generated prose and examples are not stored.

```json
{
  "platform": "volcengine-agent-plan",
  "protocol": "native",
  "model": "seed-tts-2.0",
  "categories": [
    {
      "category": "tts",
      "endpoints": [
        {
          "profile": "volcengine_tts_unidirectional",
          "variants": ["sync"]
        },
        {
          "profile": "volcengine_tts_unidirectional_stream",
          "variants": ["stream"]
        },
        {
          "profile": "volcengine_tts_bidirectional",
          "variants": ["bidirectional"]
        }
      ]
    }
  ]
}
```

Bindings must be keyed by platform slug, current platform protocol, and public model name. Including the protocol in the key prevents an old OpenAI-compatible selection from being applied after the platform is changed to Anthropic or Gemini.

## Storage

Reuse the existing Settings repository. Do not add a new database table for this task.

Recommended internal key:

```text
ximoai_model_api_docs_bindings
```

The stored value is a versioned JSON object:

```json
{
  "version": 1,
  "bindings": {}
}
```

Requirements:

- Set a strict payload size limit.
- Reject unknown profile IDs, unknown variants, duplicate bindings, and invalid category/profile combinations.
- Treat malformed stored JSON as an internal configuration error for administrators and fall back to automatic defaults for regular users.
- Do not add this setting to public settings output.
- Do not log the full saved payload.

## Automatic Default Resolution

Automatic defaults are computed from structured runtime metadata:

```text
platform protocol
+ platform capabilities
+ model billing mode
+ registered endpoint profiles
```

Model names must not participate in protocol or capability inference.

### OpenAI-Compatible Custom Platforms

Default candidates are filtered by platform capabilities:

- `responses` -> OpenAI Responses profiles.
- `chat_completions` -> OpenAI Chat Completions profiles.
- `images` -> OpenAI image profiles.
- `videos` -> OpenAI video profiles.
- `audio` -> OpenAI TTS and ASR profiles.
- `realtime` -> realtime WebSocket profiles only when the capability is explicitly present.

### Anthropic Custom Platforms

Default candidates:

- Anthropic Messages synchronous HTTP JSON.
- Anthropic Messages streaming HTTP SSE.

### Gemini Custom Platforms

Default candidates:

- `generateContent` synchronous HTTP JSON.
- `streamGenerateContent?alt=sse` streaming HTTP SSE.
- Video profiles only when the platform advertises the relevant capability and the route is actually supported.

### Category Resolution

- `token` billing can default to `conversation` when the protocol is a text protocol.
- `image` billing can default to `image`.
- `video` billing can default to `video`.
- `per_request` is ambiguous and must not be guessed as TTS, ASR, image, or video without platform capability or an administrator selection.
- Built-in XimoAI native platforms use explicit code-owned defaults.

An administrator can replace automatic defaults with an explicit multi-profile binding and can reset the binding back to automatic mode.

## Standard Endpoint Matrix

The initial catalog must cover at least the following existing public contracts.

### Conversation

| Profile | Method and path | Transport | Delivery | Modes |
|---|---|---|---|---|
| OpenAI Responses | `POST /v1/responses` | HTTP | JSON or SSE | sync, stream |
| OpenAI Responses Realtime | `GET /v1/responses` | WebSocket | WebSocket frames | bidirectional |
| OpenAI Chat Completions | `POST /v1/chat/completions` | HTTP | JSON or SSE | sync, stream |
| Anthropic Messages | `POST /v1/messages` | HTTP | JSON or SSE | sync, stream |
| Gemini Content | `POST /v1beta/models/{model}:generateContent` | HTTP | JSON | sync |
| Gemini Streaming Content | `POST /v1beta/models/{model}:streamGenerateContent?alt=sse` | HTTP | SSE | stream |

Realtime documentation must only be offered when the selected platform explicitly supports it.

### Image

| Profile | Workflow | Mode |
|---|---|---|
| OpenAI Image Generation | `POST /v1/images/generations` | sync |
| OpenAI Image Edit | `POST /v1/images/edits` | sync |
| Async Image Generation | `POST /v1/images/generations/async` -> `GET /v1/images/tasks/{task_id}` | async |
| Async Image Edit | `POST /v1/images/edits/async` -> `GET /v1/images/tasks/{task_id}` | async |

### Video

| Profile | Workflow | Mode |
|---|---|---|
| Video Generation | `POST /v1/videos/generations` -> `GET /v1/videos/{request_id}` -> `GET /v1/videos/{request_id}/content` | async |
| Video Edit | `POST /v1/videos/edits` -> status -> content | async |
| Video Extension | `POST /v1/videos/extensions` -> status -> content | async |

### OpenAI-Compatible Audio

| Profile | Method and path | Transport | Delivery | Mode |
|---|---|---|---|---|
| Speech synthesis | `POST /v1/audio/speech` | HTTP | binary | sync |
| Transcription | `POST /v1/audio/transcriptions` | HTTP | JSON | sync |
| Translation | `POST /v1/audio/translations` | HTTP | JSON | sync |

## Volcengine Agent Plan Endpoint Matrix

All six existing XimoAI native entry points must be separate endpoint profiles.

| Category | Profile ID | Method and path | Transport | Delivery | Mode |
|---|---|---|---|---|---|
| Image | `volcengine_images_generations` | `POST /v1/volcengine/images/generations` | HTTP | JSON | sync |
| TTS | `volcengine_tts_unidirectional` | `POST /v1/volcengine/audio/tts/unidirectional` | HTTP | binary/provider response | sync |
| TTS | `volcengine_tts_unidirectional_stream` | `GET /v1/volcengine/audio/tts/unidirectional/stream` | WebSocket | WebSocket frames | stream |
| TTS | `volcengine_tts_bidirectional` | `GET /v1/volcengine/audio/tts/bidirection` | WebSocket | WebSocket frames | bidirectional |
| ASR | `volcengine_asr_bigmodel_async` | `GET /v1/volcengine/audio/asr/bigmodel_async` | WebSocket | WebSocket frames | async |
| ASR | `volcengine_asr_bigmodel_nostream` | `GET /v1/volcengine/audio/asr/bigmodel_nostream` | WebSocket | WebSocket frames | sync |

Required automatic bindings:

- `doubao-seedream-5.0-lite` -> image generation profile.
- `seed-tts-2.0` -> all three TTS profiles concurrently.
- `volc.seedasr.sauc.duration` -> both ASR profiles concurrently.

The generated WSS documentation must describe the actual handshake and provider frame lifecycle. A generic WebSocket text-message example is not sufficient.

## API Contract

### Read Complete Model Catalog

```text
GET /api/v1/channels/model-plaza
```

Requirements:

- Require the existing authenticated user session.
- Preserve the existing user-visible channel, platform, group, model, and pricing filtering path.
- Return platform display metadata and the current user's effective group rate in the same response.
- Return each model's public name, pricing, types, capabilities, protocols, invocation modes, and complete effective API documentation.
- Resolve all saved documentation bindings in one Settings load; never issue one repository read per model.
- Include administrator editor metadata only when the authenticated user is an administrator.
- Do not require separate platform, group-rate, category-summary, or per-model-documentation reads.
- Do not expose an upstream Base URL, credential, account, mapping target, internal channel ID, or management-only state.

Response shape:

```json
[
  {
    "name": "Public channel",
    "platforms": [
      {
        "platform": "example-platform",
        "display_name": "Example Platform",
        "color": "#2563eb",
        "protocol": "openai_compatible",
        "groups": [{ "id": 1, "name": "Default" }],
        "supported_models": [
          {
            "name": "public-model-name",
            "platform": "example-platform",
            "pricing": {
              "billing_mode": "token",
              "input_price": 0.000001,
              "output_price": 0.000002,
              "cache_write_price": null,
              "cache_read_price": null,
              "image_input_price": null,
              "image_output_price": null,
              "per_request_price": null
            },
            "types": ["conversation"],
            "capabilities": ["responses"],
            "protocols": ["openai_responses"],
            "invocation_modes": ["sync", "stream"],
            "api_documentation": {
              "source": "automatic",
              "binding": {},
              "profiles": []
            }
          }
        ]
      }
    ]
  }
]
```

### Save Administrator Binding

```text
PUT /api/v1/admin/model-plaza/docs
```

The body contains only platform, protocol, model, categories, profile IDs, variant IDs, and ordering. It must not contain arbitrary HTML or credentials.

### Reset to Automatic Documentation

```text
DELETE /api/v1/admin/model-plaza/docs
```

Use a JSON body containing platform, protocol, and model. Reset removes only the exact binding and immediately restores automatic defaults.

## UI Design

### Model Plaza Card

- Make the card body clickable.
- Keep the existing copy-model-name button.
- Use `click.stop` on copy and other child actions so they do not open the documentation dialog.
- Preserve keyboard accessibility with Enter and Space activation.
- Add an accessible label indicating that the card opens API documentation.
- Add a second filter row for All types, Conversation, Image, Video, Text to Speech, and Speech Recognition.
- Show only categories present in the currently selected platform while retaining multi-category membership.
- Combine category, platform, and search filters without changing the state of documentation bindings.
- Drive the category filter and documentation dialog directly from the complete catalog response.

### Shared Documentation Dialog

The dialog is the only documentation UI.

Regular-user mode:

- Read-only.
- Category tabs.
- Endpoint tabs inside each category.
- Invocation-mode tabs inside each endpoint.
- Copy buttons for base URL, endpoint, model name, headers, and examples.

Administrator mode:

- Shows the same rendered document.
- Adds an Edit action.
- Uses multi-select category controls.
- Uses endpoint-profile checkboxes filtered by protocol and capability.
- Uses variant checkboxes filtered by the selected endpoint profile.
- Supports ordering selected endpoints.
- Updates the preview immediately.
- Provides Save, Cancel, and Reset to Automatic actions.

Invalid combinations must be disabled rather than accepted and corrected later.

## Generated Document Content

Each selected variant must render:

1. Public base URL.
2. Method and path or WebSocket URL.
3. Authentication header using a placeholder.
4. Content type and delivery format.
5. Path, query, header, and body parameters.
6. Minimal request example using the current public model name.
7. Minimal success response or event/frame sequence.
8. Streaming termination behavior where applicable.
9. Async submit, poll, and content retrieval steps where applicable.
10. Copyable examples appropriate to the transport.

HTTP examples may include cURL, JavaScript, and Python. WSS profiles must use WebSocket-capable examples and must not render a fake cURL request as a complete client.

Examples must use placeholders such as:

```text
$XIMOAI_API_KEY
https://ximoai.cn
```

Never insert a real user API key into the DOM, URL, clipboard template, log, or response.

## Security and Visibility

- Reuse the existing authenticated Model Plaza route boundary.
- Reuse the existing administrator middleware for updates and resets.
- Validate model visibility before returning model-specific documentation.
- Return `404` for a model not visible to the current user instead of revealing whether a hidden model exists.
- Never return platform upstream Base URLs.
- Never return model mapping targets.
- Never return account IDs, credentials, concurrency, health, or scheduler state.
- Render examples as escaped code, not raw HTML.
- Apply strict length limits to descriptions and templates owned by the code registry.
- Do not include saved bindings in public settings.
- Audit administrator save and reset operations without logging full request bodies.

## Minimal-Mount Implementation Plan

Prefer new XimoAI-owned files:

```text
backend/internal/handler/ximoai_model_api_docs.go
backend/internal/handler/ximoai_model_api_docs_profiles.go
backend/internal/handler/ximoai_model_api_docs_test.go
backend/internal/service/ximoai_model_api_docs_settings.go
backend/internal/service/ximoai_model_api_docs_settings_test.go
frontend/src/extensions/model-plaza/ModelApiDocsDialog.vue
frontend/src/extensions/model-plaza/modelApiDocs.ts
frontend/src/extensions/model-plaza/__tests__/ModelApiDocsDialog.spec.ts
frontend/src/api/modelApiDocs.ts
```

Required existing-file hooks should remain small:

- Add only the administrator save/reset route registrations in `backend/internal/server/routes/ximoai.go`; reuse the existing Model Plaza read route.
- Add the card click and dialog mount in `frontend/src/extensions/model-plaza/ModelPlazaPage.vue`.
- Add localized strings through `frontend/src/i18n/ximoaiPatch.ts`.

Avoid changes to:

- `backend/internal/server/routes/gateway.go`
- official gateway handlers and protocol services
- upstream locale source files

The existing `AvailableChannelHandler` already has authentication-aware model visibility and `SettingService`. Add catalog enrichment in a separate receiver file. The only constructor/Wire change is injecting the existing `PlatformService` so the single response can include current protocol and public display metadata.

## Upstream Synchronization Strategy

### Automatically Synchronized Runtime Data

The following values must be resolved at request/render time:

- Public model name after channel mapping.
- Model Plaza visibility.
- Platform protocol.
- Platform capabilities.
- Billing mode and displayed pricing.
- Public site base URL.
- Active administrator binding for the exact platform/protocol/model key.

Changing any of those settings must update the generated document without editing the document template.

### Code-Level Upstream Changes

Use existing endpoint constants from `backend/internal/handler/endpoint.go` wherever possible. XimoAI-native endpoint paths should use XimoAI-owned constants.

This can automatically carry path changes when the shared constant changes, but the project has no complete machine-readable OpenAPI source for request semantics. The following changes therefore cannot be safely inferred:

- New or removed request fields.
- Changed field semantics.
- New SSE event types or termination behavior.
- Changed WebSocket frame schemas.
- Changed asynchronous workflow steps.
- Entirely new protocol endpoints.

Do not pretend these changes are automatic. Contract tests must block the build until the profile is reviewed and updated.

### Merge-Conflict Boundary

The current upstream does not contain XimoAI's `routes/ximoai.go` or Model Plaza extension. Keeping the feature in those custom files minimizes direct merge conflicts.

Upstream has changed `routes/gateway.go` frequently. This task must not add documentation logic to that file.

## Required Tests

Follow TDD: add failing tests before implementation.

### Backend Profile Tests

- Profile IDs and variant IDs are unique.
- Every profile has a valid category and protocol.
- Every variant has a valid mode, transport, and delivery format.
- HTTP SSE is represented as HTTP transport plus SSE delivery.
- WebSocket profiles use GET handshake routes.
- Invalid mode/transport combinations are rejected.
- Multi-category, multi-endpoint, and multi-variant bindings round-trip without loss.
- All six Volcengine profiles match the required matrix.
- The three Volcengine billing models receive the required automatic bindings.

### Route Contract Tests

- Every documented method/path is registered in the Gin router.
- Every documentable model gateway route is represented by a profile or an explicit exclusion.
- Removing or renaming a route makes the test fail.
- Adding a relevant upstream route without reviewing documentation coverage makes the test fail.
- XimoAI native endpoints remain separate from official built-in protocol behavior.

### Resolution Tests

- OpenAI-compatible capabilities produce the expected profile set.
- Anthropic produces synchronous and streaming Messages variants.
- Gemini produces synchronous and streaming content variants.
- Realtime is absent without the explicit capability.
- `per_request` billing does not guess TTS or ASR.
- Explicit administrator bindings override automatic defaults.
- Reset restores automatic defaults.
- A protocol change does not reuse a binding saved for the old protocol.
- One model can retain multiple categories and endpoint variants concurrently.

### Authorization and Visibility Tests

- Unauthenticated reads return `401`.
- An authenticated user receives documentation only for models already visible in the Model Plaza catalog.
- Hidden or unavailable models are absent from the catalog.
- A regular user cannot save or reset bindings.
- An administrator can save and reset bindings.
- The response does not expose upstream Base URLs, mappings, credentials, or account metadata.
- Logs do not include API keys or complete saved payloads.

### Frontend Tests

- Clicking a card opens the dialog.
- Clicking Copy does not open the dialog.
- Keyboard activation opens the dialog.
- Model Plaza performs one catalog read and does not fetch platform metadata, group rates, category summaries, or per-model documentation separately.
- Regular users see no edit controls.
- Administrators see structured edit controls.
- Categories, profiles, and variants are multi-select.
- Selecting one variant does not remove another selected variant.
- Invalid combinations are disabled.
- Sync, SSE stream, async workflow, WSS stream, and WSS bidirectional views render correctly.
- Switching platform protocol refreshes automatic defaults.
- Save updates the preview and persisted binding.
- Reset returns to automatic mode.
- No real API key is rendered into the page or clipboard template.

### Build and Regression Checks

Run at minimum:

```text
go test ./internal/server/routes
go test ./internal/handler
go test ./internal/service
go vet ./internal/server/routes ./internal/handler ./internal/service
pnpm test
pnpm typecheck
pnpm lint
pnpm build
git diff --check
```

Use the project-local Go, pnpm, and temporary caches required by `AGENTS.md`.

## Implementation Sequence

1. Add failing profile-schema, multi-binding, and Volcengine matrix tests.
2. Add the code-owned endpoint profile registry.
3. Add versioned Settings persistence and validation.
4. Add authenticated read, administrator save, and administrator reset handlers.
5. Add route parity and upstream merge-guard tests.
6. Add the shared dialog and card interaction tests.
7. Add automatic protocol/capability resolution.
8. Add structured administrator controls and live preview.
9. Run backend, frontend, security, and production build verification.
10. Perform browser checks at desktop and mobile widths in light and dark themes.

## Acceptance Scenarios

### Custom OpenAI-Compatible Text Model

- The card opens with Responses and Chat Completions based on current capabilities.
- Sync and stream variants appear together.
- Changing the platform protocol to Anthropic removes OpenAI defaults and shows Messages defaults.

### Volcengine TTS

- `seed-tts-2.0` displays HTTP sync, WSS unidirectional stream, and WSS bidirectional tabs concurrently.
- Each tab has transport-appropriate examples.
- Selecting or disabling one profile does not modify the others.

### Volcengine ASR

- `volc.seedasr.sauc.duration` displays asynchronous and non-streaming-result WSS profiles concurrently.
- The documentation does not mislabel WSS transport as synchronous HTTP.

### Async Video

- The generated document contains task submission, status polling, and content retrieval in order.
- The current public model name is used in the submit example.

### Upstream Route Change

- A changed or newly added relevant upstream route causes a contract-test failure unless the documentation profile or explicit exclusion is reviewed.
- No stale endpoint document reaches a successful release silently.

## Completion Report Requirements

When implementation is complete, report:

- Files added and modified.
- Endpoint profiles and automatic resolution rules implemented.
- Settings key and stored binding format.
- Authorization and model-visibility behavior.
- Backend and frontend test results.
- Browser verification results.
- Upstream conflict analysis.
- Any protocol semantics that still require manual template maintenance.
- Any production validation that could not be performed without real provider credentials.
