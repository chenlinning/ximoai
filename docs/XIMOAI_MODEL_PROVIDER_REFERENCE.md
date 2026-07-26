# XimoAI Model Provider and Protocol Reference

> Status: reference document only. This document does not add an API contract, does not change routing, and does not add a protocol conversion layer.
>
> Checked on: 2026-07-26 (Asia/Shanghai)

## 1. Purpose and Boundary

This document records which model providers can be used through the current XimoAI project, which entry protocol should be selected, and how reasoning/thinking information should be understood when users configure a model.

The document has three boundaries:

1. **Built-in platforms** are controlled by the upstream sub2api implementation and the current built-in platform definitions. Their entry behavior must follow the platform protocol and capability declared by the project.
2. **Custom platforms** use the protocol selected by the administrator. A custom platform does not become an official vendor platform merely because its model name has a familiar prefix.
3. **Model Plaza metadata** is reference and classification data. It does not transform requests, select a protocol, change pricing, or guarantee that an upstream account has access to a model.
4. **Public Model Plaza responses must not expose upstream details.** Upstream protocol fields, provider endpoint paths, request field names, upstream model IDs, provider documentation URLs, and internal mapping data are private registry data only. The public response may expose only abstract reasoning levels and a thinking-support flag for user integration reference.

The project can theoretically connect any provider that exposes a compatible OpenAI, OpenAI-compatible, Anthropic, or Gemini interface. Therefore, a permanently complete list of every provider is not possible. This document separates providers verified against current official documentation from providers that are only protocol-compatible candidates.

## 2. Current Project Entry Surface

### 2.1 Built-in platforms

| Platform | Protocol | Project entry surface | Declared capabilities | Classification expectation |
|---|---|---|---|---|
| `openai` | OpenAI | `/v1/responses`, `/v1/chat/completions`, `/v1/embeddings`, `/v1/images/*`, `/v1/audio/*`, `/v1/realtime`, `/v1/videos` | Responses, Chat Completions, Embeddings, Images, Audio, Realtime, Videos, Codex | Conversation, embedding, image, TTS/ASR, video |
| `anthropic` | Anthropic native | `/v1/messages` | Messages | Conversation |
| `gemini` | Gemini native | `/v1beta/models/*:generateContent`, `streamGenerateContent`, `countTokens`, video actions | Messages, native Gemini, Videos | Conversation, video |
| `antigravity` | Antigravity native | `/antigravity/v1/messages`, `/antigravity/v1beta/*` | Messages, native Gemini | Conversation |
| `grok` | OpenAI-compatible | `/v1/responses`, `/v1/chat/completions`, image/video routes | Responses, Chat Completions, Images, Videos | Conversation, image, video |
| `grok-video` | OpenAI-compatible specialized platform | Video generation/status routes | Videos | Video, async |
| `openai-audio` | OpenAI-compatible specialized platform | Chat Completions and Audio routes | Chat Completions, Audio | Conversation or TTS/ASR, depending on pricing |
| `kling-audio` | OpenAI-compatible specialized platform | Audio route | Audio | TTS |
| `volcengine-agent-plan` | Volcengine native specialized platform | Volcengine image, TTS, and ASR routes | Images, Audio | Image, TTS, ASR |

The source of the platform list is `backend/internal/service/platform_service.go`, especially `builtinPlatforms`, `defaultCustomPlatformCapabilities`, and `ensureRequiredPlatformCapabilities`.

### 2.2 Custom platform protocols

| Selected protocol | Main XimoAI entry | Suitable upstream interface | Important limitation |
|---|---|---|---|
| `openai` | `/v1/responses`, `/v1/chat/completions`, and capability-specific OpenAI routes | OpenAI native or an upstream that explicitly supports the selected OpenAI surface | Do not assume every OpenAI-compatible service implements Responses |
| `openai_compatible` | `/v1/responses`, `/v1/chat/completions`, and declared capability routes | OpenAI-compatible providers | The model name does not determine the protocol; the administrator's platform protocol does |
| `anthropic` | `/v1/messages` | Anthropic-compatible providers | Responses/Chat payloads are not automatically equivalent to native Anthropic features |
| `gemini` | Gemini native routes and the project's Gemini compatibility surface | Gemini-compatible providers | Advanced `thinkingConfig` fields are reliable only on native Gemini requests |

Custom platform account configuration must use the selected platform's URL and key rules. A custom OpenAI-compatible platform is not the same as the built-in `openai` or `grok` platform and must not inherit OAuth, Codex, Antigravity, Grok media, or other vendor-specific account behavior unless explicitly declared by the platform kind.

## 3. Model Plaza Behavior

### 3.1 Public metadata currently returned

The current model plaza metadata shape is:

```json
{
  "brand": "DeepSeek",
  "types": ["conversation"],
  "invocation_modes": ["sync", "stream"],
  "reasoning_levels": ["high", "max"],
  "thinking_supported": true
}
```

The current implementation returns only these public catalog fields:

- `brand`
- `types`
- `invocation_modes`
- `reasoning_levels` when the verified registry or administrator override supplies abstract levels
- `thinking_supported` when the verified registry or administrator override marks a thinking switch as supported

It does not return an upstream protocol field, an upstream request field name, or a guarantee that a model is enabled for a particular user's key.

Relevant implementation:

- `backend/internal/handler/ximoai_model_metadata.go`
- `backend/internal/service/ximoai_model_metadata_settings.go`
- `docs/EXTERNAL_API.md`, model plaza and platform sections

### 3.2 Automatic classification rules

| Billing/capability input | Automatic type | Automatic invocation modes |
|---|---|---|
| Token + Responses/Chat/Messages/Native Gemini | `conversation` | `sync`, `stream` |
| Token + Embeddings only | `embedding` | `sync` |
| Image billing | `image` | `sync`, `async` |
| Video billing | `video` | `async` |
| Per-request + Images | `image` | `sync`, `async` |
| Per-request + Videos | `video` | `async` |
| Per-request + Audio | `tts` | `sync` |
| Grok-video kind | `video` | `async` |
| Kling Audio kind | `tts` | `sync` |
| Volcengine Agent Plan Seedream | `image` | `sync`, `stream` |
| Volcengine Agent Plan TTS | `tts` | `sync`, `stream`, `bidirectional` |
| Volcengine Agent Plan ASR | `asr` | `stream` |

There is currently no independent `music` model type. Music models should not be described as TTS merely because both use audio-related billing.

Automatic metadata is derived from the configured channel pricing and platform capabilities. It is not fetched from the provider's model catalog and it cannot correct an incorrectly configured pricing mode. Administrators can override brand, types, and invocation modes for an individual model without changing the request path.

### 3.3 Public and internal data separation

The machine-readable registry should be treated as an internal reference source. It may contain complete per-model access profiles for administrator review and backend matching, but those fields must never be copied into the public Model Plaza response.

#### Public projection

The public `GET /api/v1/channels/model-plaza` response may contain only the existing non-sensitive catalog fields:

- Model display/request name: `name`
- Display brand: `brand`
- Model categories: `types`
- Project-level invocation labels: `invocation_modes`
- Abstract reasoning levels when verified: `reasoning_levels`
- Abstract thinking support when verified: `thinking_supported`
- Visible group ID and name
- User-visible pricing and pricing intervals

The public response must not add or expose:

- `access_profiles`
- Upstream protocol names when they are used to describe provider behavior
- Upstream endpoint paths or Base URLs
- Upstream request field names such as `reasoning_effort`, `enable_thinking`, `thinkingConfig`, or `output_config.effort`
- `upstream_model` or channel mapping targets
- Reasoning defaults, provider-specific capability flags, and any protocol-specific thinking field names
- Provider API documentation URLs, account credentials, internal channel IDs, or private notes

The existing section-level `platform`, `display_name`, `color`, and `protocol` fields should remain unchanged for backward compatibility only. They are XimoAI platform presentation/configuration fields, not a new upstream capability disclosure. No additional upstream information should be nested under them.

#### Internal registry

The internal registry may retain the complete fields described in the provider sections, including `access_profiles`, reasoning field paths, exact upstream endpoint semantics, and verification sources. Backend matching uses the built-in `platform + actual upstream model` pair first, then a unique actual model ID for custom platforms; the registry is never returned verbatim to normal users.

For channel mappings, `name` and the registry lookup name are intentionally different:

- `name` is the user-requested/display model name (the mapping key).
- The registry lookup uses the channel mapping target, which is the actual model name sent upstream.
- When no mapping exists, both names are the same.
- Administrator overrides are keyed by the display/requested name, so they remain stable even when a channel target changes.
- Custom platforms may reuse a registry record only when the actual upstream model ID uniquely matches one verified record. Ambiguous IDs are left for administrator correction.

The safe data flow is:

```text
internal model registry
  -> platform + actual upstream model exact lookup
  -> custom platform unique model-ID lookup when unambiguous
  -> admin override precedence
  -> public-field projection
  -> GET /api/v1/channels/model-plaza
```

If an administrator needs to inspect upstream details, that should be an administrator-only view or local maintenance document. It must not be added to the public model plaza contract.

## 4. Verified Provider Reference

The model lists below focus on text, reasoning, embedding, and speech-related models. Image and video families are intentionally not expanded here, except where the project has a built-in media entry that affects routing.

### 4.1 OpenAI

**Current model references:**

- Frontier: `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.6-luna`
- Current professional: `gpt-5.5`, `gpt-5.5-pro`, `gpt-5.4`, `gpt-5.4-pro`, `gpt-5.4-mini`, `gpt-5.4-nano`
- Coding and previous current families: `gpt-5.3-codex`, `gpt-5.2`, `gpt-5.2-pro`, `gpt-5.1`, `gpt-5`, `o3-pro`, `o3`
- Embeddings: `text-embedding-3-large`, `text-embedding-3-small`
- Current speech/realtime families: `gpt-realtime-2.1`, `gpt-realtime-2.1-mini`, `gpt-realtime-2`, `gpt-realtime-translate`, `gpt-realtime-whisper`, `gpt-audio-1.5`, `gpt-4o-transcribe`, `gpt-4o-mini-transcribe`

**Protocol and reasoning:**

- Recommended reasoning entry: OpenAI Responses API.
- Responses field: `reasoning.effort`.
- Current GPT-5.6 family levels: `none`, `low`, `medium`, `high`, `xhigh`, `max`.
- Chat Completions compatibility field: `reasoning_effort` where the selected model supports it.
- The project classifies ordinary token models as `conversation` with `sync` and `stream`; it does not expose OpenAI Batch as an automatic model-plaza mode.

Official references: [OpenAI models](https://developers.openai.com/api/docs/models), [OpenAI latest model guidance](https://developers.openai.com/api/docs/guides/latest-model), and [OpenAI all models](https://developers.openai.com/api/docs/models/all).

### 4.2 Anthropic

**Active model references checked against the current Anthropic model lifecycle pages:**

`claude-fable-5`, `claude-opus-5`, `claude-opus-4-8`, `claude-opus-4-7`, `claude-opus-4-6`, `claude-opus-4-5-20251101`, `claude-sonnet-5`, `claude-sonnet-4-6`, `claude-sonnet-4-5-20250929`, and `claude-haiku-4-5-20251001`.

**Protocol and reasoning:**

- Native entry: `/v1/messages`.
- Reasoning control: `thinking.type`, `thinking.budget_tokens`, and `output_config.effort` depending on the model generation.
- Native Messages supports synchronous and streaming responses.
- Responses/Chat to Anthropic conversion is not equivalent to the native Adaptive Thinking contract. Do not document it as a full native replacement.

Official references: [Claude models overview](https://platform.claude.com/docs/en/about-claude/models/overview) and [model deprecations](https://platform.claude.com/docs/en/about-claude/model-deprecations).

### 4.3 Google Gemini

**Current model references:**

- General text/multimodal: `gemini-3.6-flash`, `gemini-3.5-flash`, `gemini-3.5-flash-lite`, `gemini-3.1-flash-lite`
- Live/audio: `gemini-3.1-flash-live-preview`
- Embeddings: `gemini-embedding-2`, `gemini-embedding`
- Specialized current entries: `gemini-robotics-er-1.6-preview`, `gemini-deep-research-max-preview`
- Managed agent reference: `antigravity-preview-05-2026`

**Protocol and reasoning:**

- Native endpoint family: `/v1beta/models/{model}:generateContent` and `:streamGenerateContent`.
- Gemini 3 field: `generationConfig.thinkingConfig.thinkingLevel`.
- Older Gemini thinking models may use `generationConfig.thinkingConfig.thinkingBudget`.
- Thinking-level values depend on the model; do not copy Gemini 3 levels onto older models.
- The project should treat native Gemini as the reliable reasoning path. The Messages/Chat compatibility path does not promise complete conversion of `thinkingConfig`.

Official references: [Gemini model catalog](https://ai.google.dev/gemini-api/docs/models), [Gemini 3 thinking levels](https://ai.google.dev/gemini-api/docs/gemini-3), and [latest Gemini models](https://ai.google.dev/gemini-api/docs/latest-model).

### 4.4 xAI / Grok

**Current model references:**

`grok-4.5`, `grok-4.3`, `grok-4.20-multi-agent-0309`, `grok-4.20-0309-reasoning`, `grok-4.20-0309-non-reasoning`, and `grok-build-0.1`.

**Protocol and reasoning:**

- The built-in Grok platform uses OpenAI-compatible Responses and Chat Completions surfaces.
- `grok-4.5` uses `reasoning.effort` with `low`, `medium`, and `high`; reasoning cannot be disabled.
- `grok-4.20-multi-agent` uses `low`, `medium`, `high`, and `xhigh`, where the setting controls agent count rather than ordinary reasoning depth.
- Grok video is a separate specialized platform kind and should not be used to classify ordinary Grok text models.

Official references: [xAI reasoning](https://docs.x.ai/developers/model-capabilities/text/reasoning), [xAI Grok 4.5](https://docs.x.ai/developers/grok-4-5), and [xAI pricing/model catalog](https://docs.x.ai/developers/pricing).

### 4.5 DeepSeek

**Current model references:** `deepseek-v4-pro` and `deepseek-v4-flash`.

The legacy names `deepseek-chat` and `deepseek-reasoner` were compatibility aliases for V4 Flash modes and are retired after 2026-07-24 UTC. They must not be treated as the current model list.

**Protocol and reasoning:**

- OpenAI-compatible endpoint: Chat Completions.
- Anthropic-compatible endpoint: Messages.
- OpenAI fields: `thinking.type=enabled|disabled` and `reasoning_effort=high|max`.
- Anthropic field: `output_config.effort=high|max`; `thinking.budget_tokens` is ignored by DeepSeek.
- Both synchronous and streaming chat are supported.

Official references: [DeepSeek V4 pricing/models](https://api-docs.deepseek.com/quick_start/pricing/), [DeepSeek thinking mode](https://api-docs.deepseek.com/guides/thinking_mode), and [DeepSeek Anthropic API](https://api-docs.deepseek.com/guides/anthropic_api).

### 4.6 Moonshot AI / Kimi

**Current model references:** `kimi-k3`, `kimi-k2.7-code`, `kimi-k2.7-code-highspeed`, `kimi-k2.6`, `kimi-k2.5`, and the remaining `moonshot-v1` context variants where the account still exposes them.

**Protocol and reasoning:**

- Recommended project entry: custom `openai_compatible` with `/v1/chat/completions`.
- K3: `reasoning_effort=low|high|max`; reasoning is always enabled.
- K2.7 Code: thinking is always enabled; do not send a disabling thinking flag.
- K2.6/K2.5: `thinking.type=enabled|disabled`.
- K2.6 additionally supports `thinking.keep=null|all`.
- The project should not claim Kimi's model-specific fields are native Responses or Anthropic fields when the selected route is Chat Completions.

Official reference: [Kimi thinking models](https://platform.moonshot.cn/docs/guide/use-kimi-k2-thinking-model).

### 4.7 MiniMax

**Current text model references:** `MiniMax-M3`, `MiniMax-M2.7`, `MiniMax-M2.7-highspeed`, `MiniMax-M2.5`, `MiniMax-M2.5-highspeed`, `MiniMax-M2.1`, `MiniMax-M2.1-highspeed`, and `MiniMax-M2`.

**Protocol and reasoning:**

- OpenAI-compatible endpoint: `/v1/chat/completions`.
- Anthropic-compatible endpoint: `/anthropic/v1/messages`.
- MiniMax-M3 exposes a thinking switch (`adaptive` or `disabled`); M2.x models produce thinking content but cannot disable it.
- MiniMax models do not expose a universal `reasoning_effort` scale equivalent to OpenAI.
- OpenAI-compatible requests may use `reasoning_split=true` to separate reasoning into `reasoning_details`.
- Anthropic-compatible requests expose thinking blocks. Use the provider's documented response shape rather than guessing a generic field.
- Both synchronous and streaming text responses are supported.

Official references: [MiniMax OpenAI compatibility](https://platform.minimaxi.com/docs/api-reference/text-openai-api), [MiniMax Anthropic compatibility](https://platform.minimaxi.com/docs/api-reference/text-anthropic-api), and [MiniMax model releases](https://platform.minimaxi.com/docs/release-notes/models).

### 4.8 Zhipu AI / GLM

**Current model references:** `glm-5.2`, `glm-5.1`, `glm-5`, `glm-5-turbo`, `glm-4.7`, `glm-4.7-flashx`, `glm-4.7-flash`, `glm-4.6`, `glm-4.5-air`, and `glm-4.5-airx`.

**Protocol and reasoning:**

- Recommended project entry: custom `openai_compatible` Chat Completions.
- `thinking.type=enabled|disabled` is supported from GLM-4.5 and later.
- `reasoning_effort` is supported for GLM-5.2 and later.
- Current GLM-5.2 levels: `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, and `max`.
- GLM-4.7 is a forced-thinking model in the provider documentation; do not display it as freely switchable merely because a generic thinking switch exists.
- Synchronous and streaming Chat Completions are supported.

Official references: [GLM model overview](https://docs.bigmodel.cn/cn/guide/start/model-overview), [GLM thinking](https://docs.bigmodel.cn/cn/guide/capabilities/thinking), and [GLM-5.2](https://docs.bigmodel.cn/cn/guide/models/text/glm-5.2).

### 4.9 Alibaba Qwen / Model Studio

**Current model references:** `qwen3.8-max-preview`, `qwen3.7-max`, `qwen3.7-plus`, `qwen3.6-max-preview`, `qwen3.6-plus`, `qwen3.6-flash`, `qwen3.5-plus`, `qwen3.5-flash`, `qwen3-max`, `qwen-plus`, `qwen-flash`, `qwen-turbo`, `qwen3-next-80b-a3b-thinking`, and `qwen3-235b-a22b-thinking-2507`.

**Protocol and reasoning:**

- Model Studio provides OpenAI-compatible and Anthropic-compatible endpoints for supported models.
- Chat-compatible thinking control: `enable_thinking=true|false` for hybrid thinking models.
- Thinking-only models cannot be disabled.
- Responses-compatible Qwen uses `reasoning.effort` with `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, and `max` where supported.
- `enable_thinking` is a provider-specific field, not a universal OpenAI field. It must not be silently translated to another provider.
- Use Chat Completions or native compatible endpoints for the model family documented by Alibaba; do not infer availability only from the model prefix.

Official references: [Model Studio models](https://help.aliyun.com/en/model-studio/models), [deep thinking](https://help.aliyun.com/en/model-studio/deep-thinking), and [Qwen Responses API](https://help.aliyun.com/en/model-studio/qwen-api-via-openai-responses).

### 4.10 Doubao / Volcengine Ark

**Current model references:** `doubao-seed-2-0-pro-260215`, `doubao-seed-2-0-lite-260215`, `doubao-seed-2-0-mini-260215`, `Doubao-Seed-1.8`, `Doubao-Seed-1.6`, `Doubao-Seed-1.6-thinking`, `Doubao-Seed-1.6-vision`, and `Doubao-Seed-1.6-flash`.

**Protocol and reasoning:**

- Generic text access should use a custom OpenAI-compatible platform only when the selected Ark endpoint documents OpenAI-compatible Chat Completions.
- The `volcengine-agent-plan` built-in platform is a separate native image/audio integration. It is not a generic Doubao text gateway.
- The current Doubao 2.0 Responses entries expose an abstract thinking switch in the registry; the exact request field is kept internal.
- `Doubao-Seed-1.6-thinking` is fixed-thinking; `Doubao-Seed-1.6` supports dynamic thinking modes; other model generations must be checked against their own endpoint documentation.
- Do not represent every Doubao model with OpenAI `reasoning_effort` unless that exact Ark endpoint explicitly supports it.
- Text models are normally synchronous or streaming; image/video/TTS/ASR models have separate endpoint semantics and are not covered by this text table.

Official reference: [Volcengine Ark product and model catalog](https://www.volcengine.com/product/ark).

## 5. Other Compatible Providers

The following providers are candidates for custom platform configuration, not built-in XimoAI platforms. Their current model IDs and feature availability must be checked against the selected account, region, and endpoint before publishing them in Model Plaza.

| Provider family | Usual custom protocol | Do not assume |
|---|---|---|
| Mistral | OpenAI-compatible | Native Responses support |
| Cohere | OpenAI-compatible or provider-specific | Chat Completions field parity |
| Perplexity | OpenAI-compatible | Search/tool fields are universally portable |
| Groq | OpenAI-compatible | Every hosted model supports reasoning controls |
| Together AI | OpenAI-compatible | Provider model IDs equal upstream owner IDs |
| SiliconFlow | OpenAI-compatible | One model ID has the same behavior across regions |
| Baidu Qianfan | OpenAI-compatible or provider-specific | `enable_thinking` has one universal shape |
| Tencent Hunyuan | OpenAI-compatible or provider-specific | Native Tencent fields are preserved by generic routes |
| Huawei Pangu/ModelArts | Provider-specific or compatible gateway | Standard OpenAI reasoning fields |
| StepFun | OpenAI-compatible | All model families support both sync and stream |
| Baichuan | OpenAI-compatible or provider-specific | Legacy model names are still active |
| 01.AI Yi | OpenAI-compatible | Model access is independent of account tier |
| OpenRouter and similar gateways | OpenAI-compatible | Gateway model IDs describe a single fixed upstream behavior |

For these providers, the safe publishing rule is: verify the provider's model list, endpoint, request field, response shape, stream behavior, and account access first; then configure channel pricing and model mapping. Do not classify a model solely by its name prefix.

## 6. Practical Configuration Rules

1. Select the protocol from the upstream endpoint contract, not from the model brand.
2. Use `openai_compatible` plus `/v1/chat/completions` for providers that explicitly document OpenAI Chat compatibility.
3. Use custom `anthropic` only when the upstream exposes `/v1/messages` semantics; do not use it merely because a model is good at reasoning.
4. Use custom `gemini` only when the upstream exposes Gemini native request objects; an OpenAI-compatible provider that happens to host a Gemini model remains OpenAI-compatible.
5. Configure channel model mapping when the public model name differs from the upstream model ID.
6. Set billing mode and capability correctly because the model plaza type and invocation modes are derived from them.
7. Use metadata overrides for an incorrect brand/type/mode display; do not use metadata overrides to change routing or model mapping.
8. Treat thinking and reasoning information as internal reference only. Do not return it from the public Model Plaza API and do not use it as an automatic conversion instruction.

## 7. Known Gaps and Non-Goals

- There is no universal reasoning-level field shared by OpenAI, Anthropic, Gemini, DeepSeek, Kimi, GLM, Qwen, MiniMax, and Doubao.
- A model that supports thinking on its native endpoint may not support thinking through a converted Messages, Chat, or Responses route.
- Model Plaza has no independent `music` type at present.
- Model Plaza does not automatically import the complete upstream model catalog.
- Account availability, regional availability, subscription plans, and provider retirement schedules can make a documented model unavailable to a particular key.
- This document intentionally does not add a reasoning conversion layer, model hardcoding, or public upstream fields.

## 8. Maintenance Checklist

When updating this document after an upstream release:

1. Check the official model catalog and deprecation page.
2. Check the exact API endpoint for the selected model.
3. Check thinking/reasoning fields and whether the model is fixed-thinking, hybrid, or non-thinking.
4. Check sync, stream, async, batch, or bidirectional behavior.
5. Check whether the current XimoAI platform capability allows that route.
6. Check channel pricing and model mapping before publishing the model in Model Plaza.
7. Record uncertainty instead of padding the list with retired aliases or unsupported model names.
