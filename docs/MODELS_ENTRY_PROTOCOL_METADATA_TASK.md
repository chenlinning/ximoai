# /v1/models Entry Protocol Metadata Fix Task

## Background

`/v1/models?include_entry_protocols=1` is intended to return model metadata that clients can use directly to choose the correct protocol, endpoint, request fields, response shape, and streaming behavior.

Live testing against `https://ximoai.cn` showed that several returned contracts are not currently executable as-is. The main root cause is that chat models are using a shared OpenAI Chat Completions-style `messages` contract, while the actual entry protocols are different:

- OpenAI Responses uses `/v1/responses` with `input`.
- Anthropic Messages uses `/v1/messages` with `messages`, `max_tokens`, and optional `thinking`.
- Gemini native uses `/v1beta/models/{model}:generateContent` with `contents`, and streaming uses `streamGenerateContent?alt=sse`.

This task is only about making `/v1/models?include_entry_protocols=1` expose accurate metadata. It should not change the existing request forwarding behavior unless a failing test proves the forwarding path itself is wrong.

## Current Problems

### OpenAI Responses Chat

Affected models observed online:

- `gpt-5.4`
- `gpt-5.4-mini`
- `gpt-5.5`

Current returned metadata:

- `default_entry_protocol`: `openai`
- `default_endpoint`: `/v1/responses`
- `request_contract.required_fields`: `model,messages`

Problem:

- `/v1/responses` does not use `messages` as its primary request field.
- Live request with `messages` returned `HTTP 502`.
- Live request with native `input` returned `HTTP 200`.

Expected metadata:

- `required_fields`: `model,input`
- `optional_fields` should include at least `stream`, `tools`, `tool_choice`, `text`, `reasoning`, `previous_response_id`, `store`, `include`, `max_output_tokens`, `temperature`, `top_p`.
- `response_contract.delivery`: `openai_responses_json`
- `response_contract.stream_delivery`: `openai_responses_sse`
- Stream events should mention `response.output_text.delta`, `response.completed`, `response.failed`, and `response.error`.
- `reasoning` is a supported request parameter, but verified upstream responses did not expose visible reasoning content. Metadata must not claim a visible reasoning text path for OpenAI Responses.

### Anthropic Messages Chat

Affected models observed online:

- `claude-opus-4-6`
- `claude-opus-4-7`
- `claude-opus-4-8`
- `claude-sonnet-4-6`

Current returned metadata:

- `default_entry_protocol`: `anthropic`
- `default_endpoint`: `/v1/messages`
- `optional_fields`: `stream,tools,tool_choice,response_format`

Problem:

- `thinking` is not exposed.
- Live request with `thinking: { "type": "enabled", "budget_tokens": 1024 }` returned `HTTP 200`.
- Streaming response included a `content_block` with `type: thinking`.
- Usage included thinking token details.

Expected metadata:

- `required_fields`: `model,messages,max_tokens`
- `optional_fields` should include at least `stream`, `system`, `tools`, `tool_choice`, `thinking`, `temperature`, `top_p`, `metadata`, `stop_sequences`.
- `response_contract.delivery`: `anthropic_messages_json`
- `response_contract.stream_delivery`: `anthropic_messages_sse`
- Stream events should mention `message_start`, `content_block_start`, `content_block_delta`, `message_delta`, `message_stop`.
- Thinking paths should mention `content_block.type=thinking` and `usage.output_tokens_details.thinking_tokens`.

### Gemini Native Chat

Affected models observed online:

- `gemini-3.1-pro-preview`
- `gemini-3.5-flash`

Current returned metadata:

- `default_entry_protocol`: `gemini`
- `default_endpoint`: `/v1beta/models/{model}:generateContent`
- `supports_stream`: `true`
- `request_contract.required_fields`: `model,messages`

Problem:

- Gemini native request body requires `contents` or `prompt`, not OpenAI-style `messages`.
- Live request with the returned `messages` contract returned an error: `contents or prompt is required`.
- Live request with native `contents` returned `HTTP 200`.
- Live streaming requires `/v1beta/models/{model}:streamGenerateContent?alt=sse`, not `generateContent`.

Expected metadata:

- Non-stream endpoint: `/v1beta/models/{model}:generateContent`
- Stream endpoint: `/v1beta/models/{model}:streamGenerateContent?alt=sse`
- `required_fields`: `contents`
- `optional_fields` should include at least `systemInstruction`, `generationConfig`, `safetySettings`, `tools`, `toolConfig`.
- `response_contract.delivery`: `gemini_generate_content_json`
- `response_contract.stream_delivery`: `gemini_sse`
- Thinking usage path should mention `usageMetadata.thoughtsTokenCount`.

### OpenAI Image Generation

Affected model observed online:

- `gpt-image-2`

Current returned metadata:

- `default_endpoint`: `/v1/images/generations`
- `model_type`: `image`
- `supports_stream`: `true`

Problem:

- The current code marks image generation as streaming when endpoint contains `/images/generations`.
- The returned response contract is normal JSON and no stream endpoint is exposed.
- Unless image streaming is explicitly supported and verified, this should not advertise streaming.

Expected metadata:

- `supports_stream`: `false`
- `request_contract.required_fields`: `model,prompt`
- `request_contract.optional_fields`: `n,size,quality,response_format,background`
- `request_contract.size.values`: `1024x1024,1536x1024,1024x1536`
- `request_contract.size.aliases`: map common client intents such as `square`, `1:1`, `landscape`, `16:9`, `wide`, `portrait`, `9:16`, and `mobile_wallpaper` to the supported OpenAI sizes.
- `request_contract.response_format.values`: `b64_json`
- `response_contract.delivery`: `openai_image_json`
- `response_contract.image_data_path`: `data[].b64_json`
- No `stream_endpoint` unless a verified stream endpoint exists.
- Do not advertise `response_format: url` until it passes live verification. The verified usable image return format is base64 JSON.

Live validation:

- `POST /v1/images/generations` with `model: gpt-image-2`, `size: 1536x1024`, and `response_format: b64_json` returned `HTTP 200`.
- The response contained one `b64_json` image.

### Gemini Image Generation

Affected models observed online:

- `NanoBanana2`
- `NanoBananaPro`

Current returned metadata:

- `default_entry_protocol`: `gemini`
- `default_endpoint`: `/v1beta/models/{model}:generateContent`
- `model_type`: `image`
- `operation_type`: `image_generation`
- `request_contract.required_fields`: currently uses OpenAI-style `model,prompt`
- `request_contract.optional_fields`: currently includes OpenAI-style `size`

Problem:

- Gemini native image generation does not use OpenAI `size`.
- Gemini image generation uses `generationConfig.imageConfig.aspectRatio` and `generationConfig.imageConfig.imageSize`.
- Returning `size` for Gemini image models makes clients construct the wrong request shape.

Expected metadata:

- `supports_stream`: `false`
- `request_contract.required_fields`: `contents`
- `request_contract.optional_fields`: `generationConfig,safetySettings`
- `request_contract.generationConfig.responseModalities.values`: `TEXT,IMAGE`
- `request_contract.generationConfig.responseModalities.default`: `TEXT,IMAGE`
- `request_contract.generationConfig.imageConfig.aspectRatio.values`: `1:1,16:9,9:16,4:3,3:4`
- `request_contract.generationConfig.imageConfig.aspectRatio.aliases`: map `square`, `landscape`, `wide`, `portrait`, and `mobile_wallpaper` to the supported aspect ratios.
- `request_contract.generationConfig.imageConfig.imageSize.values`: `1K,2K,4K`
- `request_contract.generationConfig.imageConfig.imageSize.aliases`: map `standard`, `hd`, `high_definition`, `2k`, `4k`, and `ultra_hd` to the supported image sizes.
- `response_contract.delivery`: `gemini_generate_content_json`
- `response_contract.image_data_path`: `candidates[].content.parts[].inlineData.data`

Live validation:

- `POST /v1beta/models/NanoBanana2:generateContent` with `generationConfig.imageConfig.aspectRatio: 16:9` and `imageSize: 1K` returned `HTTP 200`.
- The response contained one inline JPEG image at `candidates[0].content.parts[].inlineData.data`.

### Image Contract Responsibility

The `/v1/models?include_entry_protocols=1` endpoint is responsible for exposing a machine-readable image contract that clients can follow directly. It does not need to parse user natural language, but it must expose:

- The protocol-specific field names that control image ratio and size.
- The supported enum values.
- Common aliases that map user-facing intents to valid enum values.
- The response path where generated image bytes or URLs can be read.

This means OpenAI image models may expose `size`, while Gemini image models must expose `generationConfig.imageConfig.aspectRatio` and `generationConfig.imageConfig.imageSize` instead of `size`.

### OpenAI Audio Chat

Affected models observed online:

- `gpt-4o-audio-preview`
- `gpt-4o-mini-audio-preview`

Current metadata is mostly correct for non-stream audio output:

- `default_endpoint`: `/v1/chat/completions`
- `required_fields`: `model,messages,modalities,audio`
- `response_contract.delivery`: `openai_chat_audio_base64`

Requirement:

- Keep `supports_stream: false` until streaming audio is tested and proven working.

## Implementation Plan

### 1. Make Entry Protocols Explicit

Keep existing fields for compatibility:

- `default_entry_protocol`
- `default_endpoint`
- `model_type`
- `operation_type`
- `supports_stream`
- `request_contract`
- `response_contract`

Add a new authoritative field:

- `entry_protocols`

Suggested shape:

```json
{
  "id": "openai_responses",
  "protocol": "openai",
  "endpoint": "/v1/responses",
  "stream_endpoint": "/v1/responses",
  "supports_stream": true,
  "request_contract": {},
  "response_contract": {}
}
```

The existing top-level `ximoai.request_contract` and `ximoai.response_contract` should mirror the default entry protocol for old clients.

### 2. Refactor Metadata Generation

Primary file:

- `backend/internal/handler/gateway_handler.go`

Current functions to update:

- `publicEntryMetadataForPricedModel`
- `defaultEntryProtocolForPricedModel`
- `publicRequestContractForPricedModel`
- `publicResponseContractForPricedModel`
- `supportsStreamForPublicEntry`

Required direction:

- Determine `protocol`, `endpoint`, `model_type`, and `operation_type` first.
- Generate request and response contracts from `protocol + endpoint + model_type + operation_type`, not from `operation_type` alone.
- Treat OpenAI Responses, Anthropic Messages, Gemini native, OpenAI Chat Completions audio, OpenAI images, audio speech, transcription, translation, video, Kling audio, and voice management as separate contract families.

### 3. Add Protocol-Specific Contract Helpers

Recommended helpers:

- `publicOpenAIResponsesChatContract`
- `publicAnthropicMessagesChatContract`
- `publicGeminiNativeChatContract`
- `publicOpenAIChatAudioContract`
- `publicOpenAIImageContract`
- `publicGeminiImageContract`
- `publicAudioSpeechContract`
- `publicVideoGenerationContract`

These helpers should return request and response contract maps with explicit delivery and stream delivery information.

### 4. Fix Stream Metadata

Required behavior:

- OpenAI Responses chat: `supports_stream: true`, `stream_endpoint: /v1/responses`.
- Anthropic Messages chat: `supports_stream: true`, `stream_endpoint: /v1/messages`.
- Gemini native chat: `supports_stream: true`, `stream_endpoint: /v1beta/models/{model}:streamGenerateContent?alt=sse`.
- OpenAI image generation: `supports_stream: false` unless a verified stream path exists.
- OpenAI audio chat: keep `supports_stream: false` until stream audio is separately verified.
- TTS, transcription, translation, sync image generation: `supports_stream: false`.
- Video generation: `supports_stream: false`, `supports_polling: true`, `execution_mode: async`.

### 5. Update Tests

Primary test file:

- `backend/internal/handler/gateway_models_test.go`

Required test changes:

- OpenAI Responses model returns `required_fields` containing `input` and not containing `messages`.
- OpenAI Responses model exposes `reasoning` as optional.
- Anthropic model returns `required_fields` containing `max_tokens`.
- Anthropic model exposes `thinking` as optional.
- Gemini chat model returns `required_fields` containing `contents` and not containing `messages`.
- Gemini chat model exposes `stream_endpoint` as `:streamGenerateContent?alt=sse`.
- OpenAI image model returns `supports_stream: false`.
- OpenAI image model exposes `size.values` and `size.aliases`.
- Gemini image model returns `required_fields` containing `contents` and not containing `prompt` or `size`.
- Gemini image model exposes `generationConfig.imageConfig.aspectRatio.values`.
- Gemini image model exposes `generationConfig.imageConfig.imageSize.values`.
- OpenAI audio preview contract remains `model,messages,modalities,audio`.
- Existing pricing scale assertions remain unchanged.

## Implementation Progress

Status as of 2026-06-21:

- Done: Added explicit `entry_protocols` metadata while keeping existing default fields for compatibility.
- Done: Added `default_entry_id` for the selected default entry protocol.
- Done: OpenAI Responses chat metadata now uses `model,input` instead of `model,messages`.
- Done: OpenAI Responses chat metadata exposes `reasoning` and Responses SSE delivery information.
- Done: Anthropic Messages metadata now exposes `max_tokens` and `thinking`.
- Done: Gemini native chat metadata now uses `contents` instead of `messages`.
- Done: Gemini native chat metadata now exposes `stream_endpoint` as `:streamGenerateContent?alt=sse`.
- Done: OpenAI image generation no longer advertises streaming by default.
- Done: OpenAI image generation metadata now exposes `size.values` and `size.aliases`.
- Done: Gemini image generation metadata now exposes `generationConfig.imageConfig.aspectRatio/imageSize` instead of OpenAI `size`.
- Done: Image response contracts now expose protocol-specific `image_data_path`.
- Done: OpenAI chat audio metadata remains non-stream and keeps the existing base64 audio contract.
- Done: Added `method`, `request_content_type`, `stream_contract`, `tool_contract`, `thinking_contract`, `media_contract`, `polling_contract`, and `unsupported_capabilities` to the default metadata and mirrored entry protocol metadata.
- Done: OpenAI Responses tool metadata advertises function tools only and explicitly marks `image_generation` as unsupported until verified.
- Done: OpenAI Responses thinking metadata advertises request support and reasoning token usage path, but marks visible reasoning content as unavailable.
- Done: OpenAI image metadata advertises `response_format: b64_json` only and records `response_format:url` plus SSE streaming as unsupported.
- Done: Account testing now routes `gpt-4o-audio-preview` style models to `/v1/chat/completions` instead of `/v1/audio/speech`.
- Done: Account testing now supports Gemini image JSON responses for `NanoBanana2` and `NanoBananaPro`.
- Done: Account create/edit UI now treats `openai-audio`, `grok`, and `kling_audio` as OpenAI-compatible API key platforms for Base URL/API key hints.

Changed files:

- `backend/internal/handler/gateway_handler.go`
- `backend/internal/handler/gateway_models_test.go`
- `backend/internal/service/account_test_service.go`
- `backend/internal/service/account_test_service_gemini_test.go`
- `backend/internal/service/account_test_service_openai_image_test.go`
- `backend/internal/service/antigravity_gateway_service.go`
- `backend/internal/service/antigravity_image_test.go`
- `frontend/src/components/account/AccountBatchTestModal.vue`
- `frontend/src/components/account/AccountTestModal.vue`
- `frontend/src/components/account/CreateAccountModal.vue`
- `frontend/src/components/account/EditAccountModal.vue`
- `docs/MODELS_ENTRY_PROTOCOL_METADATA_TASK.md`

Implementation notes:

- The metadata generation path is now protocol-aware instead of using a shared chat contract.
- The top-level `request_contract` and `response_contract` mirror the default entry protocol for backward compatibility.
- The new `entry_protocols` array mirrors the default entry protocol with the same endpoint, stream endpoint, request contract, response contract, stream contract, tool contract, thinking contract, media contract, and polling contract. Downstream clients can use either the top-level default fields or `entry_protocols[0]`; they contain the same default contract.

Current test expectations:

- Updated tests assert Gemini chat stream metadata via `:streamGenerateContent?alt=sse`.
- Updated tests assert `gpt-image-2` image generation does not advertise SSE streaming.
- Updated tests assert OpenAI chat audio uses `/v1/chat/completions`, not `/v1/audio/speech`.

## Acceptance Flow

### Unit Tests

Run targeted tests first:

```powershell
D:\Program Files\Go-1.26.4\bin\go.exe test -tags unit ./internal/handler -run "TestGatewayModels"
```

Then run relevant service tests if helper logic is moved into service code:

```powershell
D:\Program Files\Go-1.26.4\bin\go.exe test -tags unit ./internal/service
```

Pass criteria:

- All updated metadata tests pass.
- No pricing scale regression.
- No audio contract regression.
- No Gemini route test regression.

### Live Metadata Verification

Use the test account to fetch model metadata:

```http
GET https://ximoai.cn/v1/models?include_entry_protocols=1
Authorization: Bearer <test-api-key>
```

Pass criteria:

- OpenAI Responses models return `input`, not `messages`.
- Anthropic models include `thinking`.
- Gemini chat models return `contents`, not `messages`.
- Gemini chat models expose `stream_endpoint`.
- `gpt-image-2` does not advertise streaming unless verified.

### Live OpenAI Responses Verification

Request:

```http
POST https://ximoai.cn/v1/responses
Authorization: Bearer <openai-group-key>
Content-Type: application/json

{
  "model": "gpt-5.4-mini",
  "input": "只回答 ok",
  "max_output_tokens": 8
}
```

Pass criteria:

- HTTP status is `200`.
- Response object is `response`.
- Returned content is present.

Streaming request:

```json
{
  "model": "gpt-5.4-mini",
  "input": "只回答 stream ok",
  "max_output_tokens": 12,
  "stream": true
}
```

Pass criteria:

- HTTP status is `200`.
- Content-Type is `text/event-stream`.
- SSE contains `response.output_text.delta`.

### Live Anthropic Thinking Verification

Request:

```http
POST https://ximoai.cn/v1/messages
Authorization: Bearer <anthropic-group-key>
Content-Type: application/json
Accept: text/event-stream

{
  "model": "claude-opus-4-6",
  "max_tokens": 1500,
  "stream": true,
  "thinking": {
    "type": "enabled",
    "budget_tokens": 1024
  },
  "messages": [
    {
      "role": "user",
      "content": "请先思考再回答，但最终只输出：thinking stream ok"
    }
  ]
}
```

Pass criteria:

- HTTP status is `200`.
- Content-Type is `text/event-stream`.
- SSE contains Anthropic events.
- SSE includes `content_block.type=thinking` or usage includes thinking token details.
- Final text is returned.

### Live Gemini Native Verification

Non-stream request:

```http
POST https://ximoai.cn/v1beta/models/gemini-3.1-pro-preview:generateContent
Authorization: Bearer <gemini-group-key>
Content-Type: application/json

{
  "contents": [
    {
      "role": "user",
      "parts": [
        {
          "text": "请简短回答：gemini native ok"
        }
      ]
    }
  ]
}
```

Pass criteria:

- HTTP status is `200`.
- Response contains `candidates`.
- Usage may include `usageMetadata.thoughtsTokenCount`.

Streaming request:

```http
POST https://ximoai.cn/v1beta/models/gemini-3.1-pro-preview:streamGenerateContent?alt=sse
Authorization: Bearer <gemini-group-key>
Content-Type: application/json
Accept: text/event-stream
```

Use the same `contents` body.

Pass criteria:

- HTTP status is `200`.
- Content-Type is `text/event-stream`.
- SSE contains one or more `data:` events.
- Response text is returned.

### OpenAI Audio Regression Verification

Request:

```http
POST https://ximoai.cn/v1/chat/completions
Authorization: Bearer <openai-audio-group-key>
Content-Type: application/json

{
  "model": "gpt-4o-audio-preview",
  "messages": [
    {
      "role": "user",
      "content": "Say exactly: new key audio test ok"
    }
  ],
  "modalities": ["text", "audio"],
  "audio": {
    "voice": "alloy",
    "format": "wav"
  }
}
```

Pass criteria:

- HTTP status is `200`.
- Response object is `chat.completion`.
- `choices[0].message.audio.data` exists.
- Decoded audio header is `RIFF/WAVE`.
- Transcript matches the prompt requirement.

## Current Live Verification Results

These are the current observed results before this metadata fix:

### OpenAI Responses

- `/v1/models?include_entry_protocols=1` returned `/v1/responses` with `required_fields: model,messages`.
- Requesting `/v1/responses` with `messages` returned `HTTP 502`.
- Requesting `/v1/responses` with `input` returned `HTTP 200`.
- Streaming `/v1/responses` with `input` and `stream: true` returned `HTTP 200`, `text/event-stream`, and `response.output_text.delta`.

Result:

- Actual endpoint works.
- Returned metadata contract is wrong.

### Anthropic Messages

- `/v1/models?include_entry_protocols=1` did not expose `thinking`.
- Requesting `/v1/messages` with `thinking` returned `HTTP 200`.
- Streaming response included `content_block.type=thinking`.
- Usage included thinking token details.

Result:

- Actual endpoint supports thinking.
- Returned metadata omits supported thinking parameters.

### Gemini Native

- `/v1/models?include_entry_protocols=1` returned Gemini chat with `required_fields: model,messages`.
- Requesting Gemini native endpoint with `messages` returned an error: `contents or prompt is required`.
- Requesting `generateContent` with `contents` returned `HTTP 200`.
- Requesting `streamGenerateContent?alt=sse` with `contents` returned `HTTP 200` and `text/event-stream`.

Result:

- Actual endpoint works.
- Returned metadata contract and stream endpoint are wrong.

### OpenAI Image

- `/v1/models?include_entry_protocols=1` returned `gpt-image-2` with `supports_stream: true`.
- `POST /v1/images/generations` with `stream: true` returned normal JSON, not SSE.
- `POST /v1/images/generations` with `response_format: url` timed out/returned upstream 504 during long live verification.
- `POST /v1/images/generations` with `response_format: b64_json` returned `HTTP 200` and base64 image data.

Result:

- Base64 JSON image generation works.
- SSE streaming and URL response format are not verified usable and must not be advertised.

### OpenAI Audio

Previously verified result:

- `gpt-4o-audio-preview`
- HTTP status `200`
- Response object `chat.completion`
- Audio bytes returned
- Audio header `RIFF/WAVE`
- Transcript matched `new key audio test ok`

Result:

- Non-stream audio output works.
- Metadata should remain non-stream until streaming audio is separately verified.

## Final Acceptance Result Template

After implementation, record the final result here:

```text
Date:
Commit/Build:

Unit tests:
- handler metadata tests:
- service tests:

Live metadata:
- OpenAI Responses contract:
- Anthropic thinking contract:
- Gemini native contract:
- Image stream flag:
- Audio chat contract:

Live requests:
- OpenAI Responses non-stream:
- OpenAI Responses stream:
- Anthropic thinking stream:
- Gemini generateContent:
- Gemini streamGenerateContent:
- OpenAI audio non-stream:

Final result:
- PASS / FAIL
- Remaining risks:
```

## Current Acceptance Result

Date: 2026-06-21

Unit tests:

- PASS: `go test ./internal/handler -run 'TestGatewayModels|TestPublicEntryMetadata' -count=1`
- PASS: `go test ./internal/service -run 'TestAccountTestService_OpenAI|TestIsImageGenerationModel|TestPlatformService_BuiltinOpenAIAudioPlatform' -count=1`
- PASS: `go test -tags unit ./internal/service -run 'TestCreateGeminiTestPayload|TestProcessGemini' -count=1`
- PASS: `go test ./internal/handler ./internal/service -count=1`
- PASS: `vue-tsc --noEmit -p tsconfig.json`
- PASS: `vitest run src/components/account/__tests__/AccountTestModal.spec.ts src/components/account/__tests__/AccountBatchTestModal.spec.ts`
- PASS: `git diff --check`

Full backend sweep:

- PARTIAL: `go test ./... -count=1`
- Passed relevant packages including `internal/handler`, `internal/service`, `internal/handler/admin`, `internal/server/routes`, and related packages.
- Failed outside this task scope:
  - `ent/schema` attempted to download `golang.org/x/tools@v0.44.0` and failed due network access to `proxy.golang.org`. Retried with network permission and it still could not connect to Go proxy.
  - `internal/config` has existing default configuration expectation failures unrelated to this metadata change.

Verified by unit tests:

- PASS: OpenAI Responses metadata requires `input` and not `messages`.
- PASS: OpenAI Responses metadata exposes `reasoning`, `max_output_tokens`, and `openai_responses_sse`.
- PASS: OpenAI Responses metadata marks visible reasoning content as unavailable and `image_generation` tool as unsupported.
- PASS: Anthropic metadata requires `max_tokens` and exposes `thinking`.
- PASS: Anthropic metadata exposes tool and thinking response paths.
- PASS: Gemini metadata requires `contents` and not `messages`.
- PASS: Gemini metadata exposes `stream_endpoint` as `:streamGenerateContent?alt=sse`.
- PASS: Gemini metadata exposes function call and thinking response paths.
- PASS: OpenAI image metadata returns `supports_stream: false`.
- PASS: OpenAI image metadata exposes `request_contract.fields.size.values`, `size.aliases`, and `response_format: b64_json`.
- PASS: OpenAI image metadata marks `response_format:url` and `sse_streaming` as unsupported.
- PASS: Gemini image metadata exposes `generationConfig.imageConfig.aspectRatio/imageSize` instead of OpenAI `size`.
- PASS: Image response contracts expose protocol-specific `image_data_path`.
- PASS: OpenAI audio preview metadata remains non-stream with `openai_chat_audio_base64`.
- PASS: OpenAI audio account testing uses `/v1/chat/completions` with `modalities` and `audio`.
- PASS: Gemini image account testing handles non-stream JSON image responses for `NanoBanana2`/`NanoBananaPro`.
- PASS: OpenAI Audio account create/edit UI uses OpenAI-compatible Base URL/API key hints.

Live requests:

- Pending: Requires deployment of this local change before `/v1/models?include_entry_protocols=1` can be re-tested against `https://ximoai.cn`.
- Previous live baseline is recorded in `Current Live Verification Results`.
- PASS baseline: OpenAI `gpt-image-2` accepted `size: 1536x1024` on `/v1/images/generations` and returned one b64 image.
- PASS baseline: OpenAI `gpt-4o-audio-preview` returned `HTTP 200`, `chat.completion`, WAV audio bytes, and transcript.
- PASS baseline: Gemini `NanoBanana2` accepted `generationConfig.imageConfig.aspectRatio: 16:9` and `imageSize: 1K` on `/v1beta/models/NanoBanana2:generateContent` and returned one inline JPEG image.

Final result:

- Local metadata implementation: PASS.
- Live post-deploy verification: PENDING.
- Remaining risks: Live `/v1/models?include_entry_protocols=1` must be re-tested after deployment to confirm production returns the new contracts, including image enum/alias fields, stream/tool/thinking/media contracts, and unsupported capability markers.
