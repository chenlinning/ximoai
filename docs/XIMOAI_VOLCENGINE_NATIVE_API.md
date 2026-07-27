# XimoAI 火山原生接口字段文档

本文档描述 XimoAI 当前为火山原生平台公开的接口。接口按“模型 + 上游协议”分别整理，适用于：

- `doubao-seed-tts-2.0`
  - TTS HTTP 单向流式
  - TTS 单向 WebSocket 流式
  - TTS 双向 WebSocket 流式
- `doubao-seed-asr-2.0`
  - ASR 双向 WebSocket
  - ASR 双向 WebSocket 优化版
  - ASR 单向流式输入 WebSocket
- `doubao-seedream-5.0-lite`
  - 火山图像原生入口

## 1. 透传边界

XimoAI 的目标是把用户使用的火山原生协议接入到火山账号。除下面明确列出的鉴权和模型路由替换外，请求体、响应体、SSE 数据和 WebSocket 业务帧均按原协议透传：

| 项目 | XimoAI 行为 |
|---|---|
| 用户鉴权 | 使用用户的 XimoAI API Key；不会把用户 Key 发送给火山账号 |
| 上游鉴权 | 使用管理员配置在火山平台账号中的 API Key |
| TTS/ASR 模型头 | 将公开的 `X-Api-Resource-Id` 按渠道/账号映射改为上游 Resource ID |
| Seedream 模型字段 | 只在 JSON 的 `model` 字段应用显式渠道/账号映射 |
| 其他请求头 | 原样转发，但连接级、内部代理级和客户端鉴权头会按网关安全规则处理 |
| 请求体 | 不转换字段、不转换协议、不按模型名猜协议 |
| 响应体 | 不转换 JSON、Chunked JSON、SSE、音频二进制或 WebSocket 业务帧 |

### 1.1 公共地址和模型映射

以下是 XimoAI 对外地址。HTTP 和 WebSocket 必须分别使用 `https` 与 `wss`：

```text
HTTP_BASE=https://ximoai.cn
WS_BASE=wss://ximoai.cn
```

如果控制台公布的主站 API 域名是 `www.ximoai.cn`，将两个主机名同时替换为 `www.ximoai.cn`，不要依赖 HTTP 重定向完成 WebSocket Upgrade。

| 用户使用的模型名 | 用户请求位置 | 上游火山名称 | 上游协议 |
|---|---|---|---|
| `doubao-seed-tts-2.0` | TTS/WS 的 `X-Api-Resource-Id` | `seed-tts-2.0` | TTS 原生 |
| `doubao-seed-asr-2.0` | ASR/WS 的 `X-Api-Resource-Id` | `volc.seedasr.sauc.duration` | ASR 原生 |
| `doubao-seedream-5.0-lite` | 图像 JSON 的 `model` | `doubao-seedream-5.0-lite`，或显式映射后的名称 | 火山图像原生 |

两侧名称相同即可直接使用，不需要额外识别。模型映射只在配置了明确映射时发生；不存在映射时使用请求中的原名。模型名映射不改变协议。

### 1.2 XimoAI API Key

TTS 和 ASR 使用火山原生的 Key 头名：

```http
X-Api-Key: <XIMOAI_USER_API_KEY>
```

Seedream 使用火山图像原生的 Bearer 头名：

```http
Authorization: Bearer <XIMOAI_USER_API_KEY>
```

Key 必须属于使用内置 `volcengine-agent-plan` 平台的分组，并且满足 XimoAI 的余额、额度、账号可用性和并发检查。不要把 Key 放进 URL、查询参数、iframe URL、WebSocket 子协议或日志。

浏览器原生 `WebSocket` API 不能设置 `X-Api-Key` 等自定义握手头。TTS/ASR WebSocket 应由服务端客户端或支持自定义握手头的 WebSocket 库调用。

## 2. WebSocket V3 公共帧格式

TTS 和 ASR 都使用火山 V3 二进制帧。XimoAI 不改变这些帧。

每个 WebSocket message 的业务 payload 由下列部分组成：

```text
4-byte header
[optional extension fields]
4-byte payload size, unsigned int32, big-endian
payload
```

4-byte header 的高低位：

| 字节 | 高 4 位 | 低 4 位 |
|---|---|---|
| 0 | Protocol version，当前为 `0b0001` | Header size，当前为 `0b0001`，实际 header 为 4 字节 |
| 1 | Message type | Message type specific flags |
| 2 | Serialization：`0` raw，`1` JSON | Compression：`0` 无压缩，`1` gzip |
| 3 | 保留位，填 `0` | 保留位，填 `0` |

整数、事件号、sequence、长度和错误码均使用大端序。不同协议的 Message type：

| Message type | 十六进制 | TTS | ASR |
|---|---:|---|---|
| Full client request | `0b0001` | JSON 请求/事件 | 首个 JSON 元数据请求 |
| Audio-only client request | `0b0010` | 不使用 | 音频块 |
| Full server response | `0b1001` | JSON 事件/文本响应 | JSON 识别结果 |
| Audio-only server response | `0b1011` | 音频响应 | 不使用 |
| Error information | `0b1111` | 错误帧 | 错误帧 |

sequence/事件/连接 ID/session ID 等可选扩展字段由官方帧类型决定。XimoAI 只转发帧，不要求客户端改成 JSON 或 OpenAI 格式。

## 3. `doubao-seed-tts-2.0`：TTS HTTP

### 3.1 请求地址

```text
POST ${HTTP_BASE}/api/v3/plan/tts/unidirectional
```

这是 XimoAI 的公开入口。当前服务将其转发到已配置火山账号对应的原生 TTS HTTP 目标。

### 3.2 请求头

| Header | 类型 | 必填 | 说明 |
|---|---|---|---|
| `X-Api-Key` | string | 是 | XimoAI 用户 API Key；网关替换为上游账号 Key |
| `X-Api-Resource-Id` | string | 是 | 公开模型名 `doubao-seed-tts-2.0`；网关按映射替换为 `seed-tts-2.0` |
| `X-Api-Request-Id` | string | 是 | 每次请求唯一的 UUID 字符串，网关原样转发 |
| `X-Control-Require-Usage-Tokens-Return` | string | 否 | 使用 `*` 或 `text_words` 请求返回计费用量 |
| `Content-Type` | string | 是 | `application/json` |

旧版火山控制台的 `X-Api-App-Id`、`X-Api-Access-Key` 属于另一套鉴权方式。当前 XimoAI 账号接入使用新版 `X-Api-Key`，不要把 XimoAI Key 当作旧版火山 Access Key 发送。

### 3.3 请求体完整字段

请求体顶层必须包含 `req_params`：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `req_params` | object | 是 | TTS 参数对象 |
| `req_params.text` | string | 是 | 待合成文本 |
| `req_params.model` | string | 否 | 复刻音色使用的模型版本；常见 `seed-tts-2.0-standard`。普通 TTS 2.0 音色使用默认值 |
| `req_params.speaker` | string | 是 | 音色 ID，例如 `zh_female_vv_uranus_bigtts` |
| `req_params.ssml` | string | 否 | SSML 标记文本。仅官方支持的中英文音色可用 |
| `req_params.audio_params` | object | 是 | 音频输出参数 |
| `req_params.additions` | string | 否 | 官方定义为 JSON 字符串；字符串内容是 additions JSON 对象 |
| `req_params.tone_fidelity` | bool | 否 | 仅复刻 2.0 还原模式；不支持双向流式，普通 TTS 2.0 音色不使用 |

`audio_params`：

| 字段 | 类型 | 默认/范围 | 说明 |
|---|---|---|---|
| `format` | string | `mp3`；`mp3`/`pcm`/`ogg_opus`/`wav` | 流式场景推荐 `pcm`，不建议使用 `wav` |
| `sample_rate` | int | `8000`、`16000`、`22050`、`24000`、`32000`、`44100`、`48000` | 采样率，单位 Hz |
| `bit_rate` | int | 通常 `64000` 至 `160000` | MP3 比特率，其他格式不一定生效 |
| `speech_rate` | int | `-50` 至 `100`，默认 `0` | `100` 约为 2 倍速，`-50` 约为 0.5 倍速 |
| `loudness_rate` | int | `-50` 至 `100`，默认 `0` | 音量比例，`100` 约为 2 倍音量 |
| `enable_subtitle` | bool | `false` | TTS 2.0 中文/英文返回字级时间戳 |

`additions` JSON 字符串内的完整字段：

| 字段 | 类型 | 默认/范围 | 说明 |
|---|---|---|---|
| `max_length_to_filter_parenthesis` | int | `0` 至 `100` | 是否过滤括号内容 |
| `silence_duration` | int | `0`；`0` 至 `30000` | 文本末尾增加静音，单位 ms |
| `enable_language_detector` | bool | `false` | 自动识别语种；按当前音色/接口能力生效 |
| `disable_markdown_filter` | bool | `false` | 是否过滤 Markdown 语法 |
| `disable_emoji_filter` | bool | `false` | 是否过滤 Emoji |
| `disable_default_bit_rate` | bool | `false` | 允许显式使用更低 bit rate；仅对应格式支持时生效 |
| `enable_latex_tn` | bool | `false` | 是否播报 LaTeX 文本 |
| `latex_parser` | string | `v2` 或空 | 更强的 LaTeX 解析；通常需同时启用 `disable_markdown_filter` |
| `explicit_language` | string | 见下表 | 只朗读指定语种 |
| `context_language` | string | 由上游决定 | 为多语种前端提供参考语种 |
| `explicit_dialect` | string | `dongbei`/`shaanxi`/`sichuan` | 指定东北、陕西或四川方言，需匹配音色 |
| `unsupported_char_ratio_thresh` | float | 默认 `0.3`，最大 `1.0` | 不支持文本比例超过阈值时失败 |
| `aigc_watermark` | bool | `false` | 在音频结尾增加 AIGC 节奏标识 |
| `aigc_metadata` | object | 空 | 音频隐式元数据水印 |
| `aigc_metadata.enable` | bool | `false` | 是否启用元数据水印 |
| `aigc_metadata.content_producer` | string | 空 | 内容制作服务提供者 |
| `aigc_metadata.produce_id` | string | 空 | 内容制作编号 |
| `aigc_metadata.content_propagator` | string | 空 | 内容传播服务提供者 |
| `aigc_metadata.propagate_id` | string | 空 | 内容传播编号 |
| `cache_config` | object | 空 | 文本缓存配置 |
| `cache_config.text_type` | int | 与 `use_cache` 配合 | 开启缓存时使用官方要求的值 |
| `cache_config.use_cache` | bool | `false` | 是否使用缓存 |
| `cache_config.use_segment_cache` | bool | `false` | 双向流式分句缓存，具体以音色/接口能力为准 |
| `post_process` | object | 空 | 后处理配置 |
| `post_process.pitch` | int | `-12` 至 `12` | 音调 |
| `context_texts` | array[string] | 空 | TTS 2.0 语音指令；当前官方说明列表首项有效，不参与计费 |
| `section_id` | string | 空 | 多轮串行合成的上下文 ID，建议使用 UUID |
| `use_tag_parser` | bool | `false` | 复刻 2.0 表现力增强版本的语音标签/COT 解析 |
| `mute_cut_threshold` | string | 空 | 与 `mute_cut_remain_ms` 配合的静音阈值 |
| `mute_cut_remain_ms` | string | 空 | 保留静音时长；该组字段按官方要求使用字符串 |

`explicit_language` 的官方常用值包括：

```text
zh-cn, en, ja, es-mx, id, pt-br, pt, ko, it, de, fr,
th, vi, ru, fil, ms, ar, pl, tr, sv
```

下面是包含所有常用层级的请求示例。`additions` 内层对象需要作为字符串放入 `req_params.additions`：

```json
{
  "req_params": {
    "text": "你好，这是一次原生 TTS 测试。",
    "model": "seed-tts-2.0-standard",
    "speaker": "zh_female_vv_uranus_bigtts",
    "audio_params": {
      "format": "mp3",
      "sample_rate": 24000,
      "bit_rate": 128000,
      "speech_rate": 0,
      "loudness_rate": 0,
      "enable_subtitle": true
    },
    "additions": "{\"explicit_language\":\"zh-cn\",\"disable_markdown_filter\":true,\"aigc_metadata\":{\"enable\":false},\"post_process\":{\"pitch\":0},\"context_texts\":[\"请用自然、清晰的语气朗读\"],\"section_id\":\"00000000-0000-4000-8000-000000000001\"}"
  }
}
```

### 3.4 响应字段

HTTP 响应是 Chunked JSON，不是 SSE，也不保证整个响应是一个 JSON 对象。客户端必须按官方 Chunked JSON 方式连续解析每个 JSON 块。

| 位置 | 字段 | 类型 | 说明 |
|---|---|---|---|
| Header | `X-Tt-Logid` | string | 火山排查用 Log ID |
| JSON | `code` | int | 当前块/请求状态码 |
| JSON | `message` | string | 状态详情或错误信息 |
| JSON | `data` | string | Base64 音频数据块 |
| JSON | `sentence` | object | 句子或时间戳信息 |
| `sentence` | `phonemes` | object/array | 音素级时间戳信息，按上游返回 |
| `sentence` | `text` | string | 句子文本 |
| `sentence` | `words` | array | 字/词级时间戳列表 |
| `sentence.words[]` | `confidence` | float | 时间戳置信度，通常 `0` 至 `1` |
| `sentence.words[]` | `startTime` | float | 开始时间，单位秒 |
| `sentence.words[]` | `endTime` | float | 结束时间，单位秒 |
| `sentence.words[]` | `word` | string | 字或词 |
| JSON | `usage` | object | 用量统计 |
| `usage` | `text_words` | int | 计费文本字数，含标点 |

错误时保留火山的状态码、`code`、`message` 和诊断响应，XimoAI 不改写成 OpenAI error。

## 4. `doubao-seed-tts-2.0`：TTS 单向 WebSocket

### 4.1 地址和握手头

```text
GET ${WS_BASE}/api/v3/plan/tts/unidirectional/stream
```

```http
X-Api-Key: <XIMOAI_USER_API_KEY>
X-Api-Resource-Id: doubao-seed-tts-2.0
X-Api-Request-Id: <unique-request-id>
X-Api-Connect-Id: <unique-connection-id>
X-Control-Require-Usage-Tokens-Return: *
```

`X-Api-Request-Id` 是官方推荐的请求标识；`X-Api-Connect-Id` 用于连接追踪。每次新连接/新会话应使用新的 ID。XimoAI 会将 Resource ID 替换为上游 `seed-tts-2.0`，其他官方业务头和二进制帧保持原样。

### 4.2 上行 payload 字段

单向 WS 首个文本请求使用 `Full-client request` 帧，序列化为 JSON。payload 字段与 TTS HTTP 的 `req_params` 相同：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `user` | object | 否 | 日志追踪用户信息 |
| `user.uid` | string | 否 | 用户 ID |
| `event` | int | 否 | 单向发送文本阶段通常不带事件；关闭连接使用 `2` |
| `namespace` | string | 否 | 上游命名空间，按官方默认值传递 |
| `req_params` | object | 是 | TTS 参数 |
| `req_params.text` | string | 是 | 输入文本 |
| `req_params.model` | string | 否 | 复刻模型版本 |
| `req_params.ssml` | string | 否 | SSML，按音色能力使用 |
| `req_params.speaker` | string | 是 | 音色 ID |
| `req_params.audio_params` | object | 是 | `format`、`sample_rate`、`bit_rate`、`emotion`、`emotion_scale`、`speech_rate`、`loudness_rate`、`enable_timestamp`、`enable_subtitle` |
| `req_params.additions` | string | 否 | 官方定义为 JSON 字符串，包含第 3.3 节列出的 additions 字段 |
| `req_params.mix_speaker` | object | 否 | 仅混音能力适用；`speaker` 通常为 `custom_mix_bigtts` |
| `req_params.mix_speaker.speakers` | array | 否 | 最多三个混音源 |
| `req_params.mix_speaker.speakers[].source_speaker` | string | 否 | 混音源音色 |
| `req_params.mix_speaker.speakers[].mix_factor` | float | 否 | 混音因子，官方要求总和为 `1` |

完整字段以第 3.3 节的 TTS 字段表为准。单向 WS 的协议帧示例：

WS 相比 HTTP 还明确提供以下音频/附加字段：

| 字段 | 类型 | 默认/范围 | 说明 |
|---|---|---|---|
| `req_params.audio_params.emotion` | string | 空 | 音色情感，仅部分音色支持 |
| `req_params.audio_params.emotion_scale` | number | `4`；`1` 至 `5` | 情感强度，需配合 `emotion` |
| `req_params.audio_params.enable_timestamp` | bool | `false` | TTS 1.0/ICL 1.0 时间戳；TTS 2.0 使用 `enable_subtitle` |
| `req_params.audio_params.enable_subtitle` | bool | `false` | TTS 2.0/ICL 2.0 字幕事件 |
| `req_params.additions.enable_language_detector` | bool | `false` | 自动识别语种 |
| `req_params.additions.context_language` | string | 空 | 多语种参考语种 |
| `req_params.additions.unsupported_char_ratio_thresh` | float | `0.3`，最大 `1.0` | 不支持文本比例阈值 |
| `req_params.additions.cache_config.use_segment_cache` | bool | `false` | 双向场景分句缓存 |
| `req_params.additions.use_tag_parser` | bool | `false` | 复刻 2.0 表现力增强标签解析 |

```text
Full-client request:
  header + uint32(payload_size) + JSON payload

JSON payload:
{
  "user": {"uid": "user-1"},
  "req_params": {
    "text": "你好，这是单向流式 TTS。",
    "speaker": "zh_female_vv_uranus_bigtts",
    "audio_params": {"format": "pcm", "sample_rate": 24000}
  }
}
```

### 4.3 下行事件和响应字段

| 事件 | 编号 | 方向 | 内容 |
|---|---:|---|---|
| `SessionFinished` | `152` | 下行 | 一次合成完成 |
| `TTSSentenceStart` | `350` | 下行 | 句子开始；通常含 session ID 和句子文本 |
| `TTSSentenceEnd` | `351` | 下行 | 句子结束；可能含时间戳 JSON |
| `TTSResponse` | `352` | 下行 | 音频二进制帧 |
| `FinishConnection` | `2` | 上行 | 关闭连接 |
| `ConnectionFinished` | `52` | 下行 | 连接关闭成功 |

JSON 事件中按上游返回 `event`、`res_params.text`、字幕/时间戳字段；音频帧的 payload 是 `data` 二进制，不是 Base64。单向 WS 不要把它当作 HTTP Chunked JSON。

## 5. `doubao-seed-tts-2.0`：TTS 双向 WebSocket

### 5.1 地址和握手头

```text
GET ${WS_BASE}/api/v3/plan/tts/bidirection
```

```http
X-Api-Key: <XIMOAI_USER_API_KEY>
X-Api-Resource-Id: doubao-seed-tts-2.0
X-Api-Connect-Id: <unique-connection-id>
X-Control-Require-Usage-Tokens-Return: *
```

### 5.2 事件顺序

同一个连接可以复用多个 session，但同一时刻不能并发多个 session：

```text
StartConnection
  -> ConnectionStarted
StartSession
  -> SessionStarted
TaskRequest (可多次发送文本)
FinishSession
  -> SessionFinished
重复 StartSession ... 或 FinishConnection
  -> ConnectionFinished
```

| 事件 | 编号 | 方向 | 说明 |
|---|---:|---|---|
| `StartConnection` | `1` | 上行 | 创建连接阶段 |
| `FinishConnection` | `2` | 上行 | 关闭连接 |
| `ConnectionStarted` | `50` | 下行 | 建连成功 |
| `ConnectionFailed` | `51` | 下行 | 建连失败 |
| `ConnectionFinished` | `52` | 下行 | 连接关闭成功 |
| `StartSession` | `100` | 上行 | 创建 session；TTS 参数在这里生效 |
| `CancelSession` | `101` | 上行 | 主动取消 session |
| `FinishSession` | `102` | 上行 | 声明文本已发送完毕 |
| `SessionStarted` | `150` | 下行 | session 创建成功 |
| `SessionCanceled` | `151` | 下行 | session 已取消 |
| `SessionFinished` | `152` | 下行 | session 完成，可开始下一 session |
| `SessionFailed` | `153` | 下行 | session 失败 |
| `TaskRequest` | `200` | 上行 | 传输文本 |
| `TTSSentenceStart` | `350` | 下行 | 句子开始 |
| `TTSSentenceEnd` | `351` | 下行 | 句子结束 |
| `TTSResponse` | `352` | 下行 | 音频数据 |
| `TTSSubtitle` | 由上游版本定义 | 下行 | 启用字幕时可能出现 |

### 5.3 StartSession/TaskRequest payload

`StartSession` 的 JSON payload 需要包含 `event: 100` 和 TTS 配置；session ID 在二进制可选扩展字段中传输。`TaskRequest` 使用 `event: 200`，文本在 `req_params.text` 中传输。主要字段如下：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `user` | object | 否 | `uid` 等用户信息 |
| `user.uid` | string | 否 | 用户 ID |
| `event` | int | 是 | `100` 或 `200` |
| `namespace` | string | 否 | 官方默认 `BidirectionalTTS` |
| `req_params.text` | string | 是 | 当前 session 的输入文本 |
| `req_params.model` | string | 否 | 复刻模型版本 |
| `req_params.speaker` | string | 是 | 音色 ID |
| `req_params.audio_params` | object | 是 | 音频格式和音频控制字段 |
| `req_params.additions` | string | 否 | JSON 字符串；包含语种、方言、缓存、后处理、语音指令等 |
| `req_params.mix_speaker` | object | 否 | 仅混音音色适用 |

双向 WS 官方当前说明不支持 `req_params.ssml`。TTS 2.0 的 `context_texts`、`section_id`、`enable_subtitle` 等字段是否可用仍由所选音色和上游版本决定；XimoAI 不会替换、删除或改写这些字段。

## 6. `doubao-seed-asr-2.0`：ASR WebSocket

ASR 的公开模型名是 `doubao-seed-asr-2.0`，上游 Resource ID 是 `volc.seedasr.sauc.duration`。三个入口都使用同一套 ASR V3 二进制帧，区别只在 URL、结果时机和可用请求字段。

### 6.1 入口列表

| XimoAI 入口 | 上游入口 | 模式 |
|---|---|---|
| `GET ${WS_BASE}/api/v3/plan/sauc/bigmodel` | `wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel` | 标准双向流式，每个输入包尽快返回结果 |
| `GET ${WS_BASE}/api/v3/plan/sauc/bigmodel_async` | `wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel_async` | 双向流式优化版，仅结果变化时返回；支持二遍识别 |
| `GET ${WS_BASE}/api/v3/plan/sauc/bigmodel_nostream` | `wss://openspeech.bytedance.com/api/v3/plan/sauc/bigmodel_nostream` | 单向流式输入，通常在分句/负包后返回更完整结果 |

### 6.2 握手头

```http
X-Api-Key: <XIMOAI_USER_API_KEY>
X-Api-Resource-Id: doubao-seed-asr-2.0
X-Api-Request-Id: <unique-request-id>
X-Api-Connect-Id: <unique-connection-id>
```

可按官方客户端需要附带：

```http
X-Api-Sequence: -1
```

`X-Api-Key` 和 `X-Api-Resource-Id` 会被 XimoAI 替换为上游账号值；`X-Api-Request-Id`、连接 ID 和业务参数由客户端生成并透传。握手成功时应记录响应头：

```http
X-Api-Connect-Id: <provider-connection-id>
X-Tt-Logid: <provider-log-id>
```

### 6.3 Full client request 字段

WebSocket 建连后首先发送 `Full client request`，其 payload 通常是 JSON：

`user`：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `user` | object | 否 | 用户/设备信息 |
| `user.uid` | string | 否 | 用户标识 |
| `user.did` | string | 否 | 设备名称 |
| `user.platform` | string | 否 | `iOS`/`Android`/`Linux` 等 |
| `user.sdk_version` | string | 否 | SDK 版本 |
| `user.app_version` | string | 否 | 应用版本 |

`audio`：

| 字段 | 类型 | 必填 | 取值/说明 |
|---|---|---|---|
| `audio` | object | 是 | 音频配置 |
| `audio.language` | string | 条件 | `bigmodel_nostream` 支持；双向模式按官方限制使用 |
| `audio.format` | string | 是 | `pcm`/`wav`/`ogg`/`mp3` |
| `audio.codec` | string | 否 | `raw`/`opus`；`ogg` 通常必须为 `opus` |
| `audio.rate` | int | 否 | 当前常用 `16000` Hz |
| `audio.bits` | int | 否 | 当前常用 `16` bit |
| `audio.channel` | int | 否 | `1` 单声道或 `2` 双声道 |

音频内容必须与 `audio` 描述一致。PCM/WAV 内部应为 `pcm_s16le`。建议每个音频包约 100 至 200 ms，并保持稳定发送间隔。

`request`：

| 字段 | 类型 | 默认/范围 | 支持情况和说明 |
|---|---|---|---|
| `request` | object | 是 | ASR 请求配置 |
| `request.model_name` | string | `bigmodel` | 当前模型名 |
| `request.enable_nonstream` | bool | `false` | 仅 `bigmodel_async` 支持二遍识别 |
| `request.enable_itn` | bool | `true` | 数字归一化 |
| `request.enable_speaker_info` | bool | `false` | 说话人聚类；通常需 `ssd_version: "200"` |
| `request.ssd_version` | string | 空或 `200` | 大模型 SSD 版本 |
| `request.enable_punc` | bool | `true` | 标点 |
| `request.enable_ddc` | bool | `false` | 语义顺滑 |
| `request.output_zh_variant` | string | 空 | `traditional`、`tw`、`hk` |
| `request.enable_auto_lang` | bool | `false` | 仅 `bigmodel_nostream`，自动识别语种 |
| `request.show_utterances` | bool | `false` | 返回分句、停顿、分词 |
| `request.show_speech_rate` | bool | `false` | 仅 nostream/async，返回 `speech_rate` |
| `request.show_volume` | bool | `false` | 仅 nostream/async，返回 `volume` |
| `request.enable_lid` | bool | `false` | 仅 nostream/async，返回 `lid_lang` |
| `request.enable_emotion_detection` | bool | `false` | 仅 nostream/async，返回情绪 |
| `request.enable_gender_detection` | bool | `false` | 仅 nostream/async，返回性别 |
| `request.result_type` | string | `full` | `full` 全量结果或 `single` 增量结果 |
| `request.enable_accelerate_text` | bool | `false` | 首字返回加速，可能降低首字准确率 |
| `request.accelerate_score` | int | `0`；`0` 至 `20` | 配合首字加速 |
| `request.vad_segment_duration` | int | `3000` | 最大静音阈值，单位 ms |
| `request.end_window_size` | int | `800`；最小 `200` | 强制判停时间，单位 ms |
| `request.force_to_speech_time` | int | 空；最小 `1` | 配合 `end_window_size` 的最小语音时长 |
| `request.sensitive_words_filter` | string | 空 | 敏感词过滤配置，通常是序列化 JSON 字符串 |
| `request.enable_poi_fc` | bool | `false` | POI function call，通常需 nostream/async 二遍能力 |
| `request.enable_music_fc` | bool | `false` | 音乐 function call，通常需 nostream/async 二遍能力 |
| `request.corpus` | object | 空 | 热词、替换词或上下文 |

`corpus`：

| 字段 | 类型 | 说明 |
|---|---|---|
| `corpus.boosting_table_name` | string | 自学习平台热词表名称 |
| `corpus.boosting_table_id` | string | 自学习平台热词表 ID |
| `corpus.correct_table_name` | string | 替换词表名称 |
| `corpus.correct_table_id` | string | 替换词表 ID |
| `corpus.context` | string | 序列化 JSON；可传热词或上下文 |

`corpus.context` 常见 JSON 结构：

```json
{
  "context_type": "dialog_ctx",
  "context_data": [
    {"text": "用户正在和助手讨论北京天气"},
    {"image_url": "https://example.com/context.png"},
    {"text": "上一轮对话内容"}
  ]
}
```

其中 `context` 本身是字符串字段，因此发送时需要按上游要求转义。热词直传也使用 `context`，例如 `{"hotwords":[{"word":"热词1号"}]}`。

### 6.4 音频帧、结束帧和响应

发送音频时使用 `Audio-only client request`：

```text
header + [sequence] + uint32(compressed_audio_size) + audio_bytes
```

flags 含义：

| flags | 说明 |
|---:|---|
| `0` | 不带 sequence |
| `1` | 带正 sequence |
| `2` | 最后一包，不带 sequence |
| `3` | 带负 sequence，表示最后一包 |

`bigmodel` 标准双向模式会尽快返回识别字符；`bigmodel_async` 只在结果变化时返回，且可用 `enable_nonstream` 做二遍识别；`bigmodel_nostream` 通常在音频超过约 15 秒或收到负包后返回更完整的分句结果。

服务端 `Full server response` 的 JSON 字段：

| 位置 | 字段 | 类型 | 说明 |
|---|---|---|---|
| JSON | `audio_info` | object | 音频信息 |
| `audio_info` | `duration` | int | 音频时长，单位 ms |
| JSON | `result` | object | 识别结果 |
| `result` | `text` | string | 整段识别文本 |
| `result` | `utterances` | array | 开启 `show_utterances` 后的分句 |
| `result.utterances[]` | `text` | string | 分句文本 |
| `result.utterances[]` | `start_time` | int | 分句开始，单位 ms |
| `result.utterances[]` | `end_time` | int | 分句结束，单位 ms |
| `result.utterances[]` | `definite` | bool | 是否确定分句 |
| `result.utterances[].words[]` | `text` | string | 字/词文本 |
| `result.utterances[].words[]` | `start_time` | int | 字/词开始，单位 ms |
| `result.utterances[].words[]` | `end_time` | int | 字/词结束，单位 ms |
| `result.utterances[].words[]` | `blank_duration` | int | 前后静音时长，按上游返回 |
| `result.utterances[].additions` | object | 否 | `speech_rate`、`volume`、`lid_lang`、`emotion`、`gender` 等附加结果 |

错误帧包含：4-byte header、4-byte error code、4-byte error message size、UTF-8 error message。常见业务码：

| Code | 含义 |
|---:|---|
| `20000000` | 成功 |
| `45000001` | 参数无效 |
| `45000002` | 空音频 |
| `45000081` | 等包超时 |
| `45000151` | 音频格式不正确 |
| `550xxxxx` | 服务内部错误 |

## 7. `doubao-seedream-5.0-lite`：火山图像原生入口

### 7.1 请求地址和鉴权

```text
POST ${HTTP_BASE}/api/plan/v3/images/generations
```

```http
Authorization: Bearer <XIMOAI_USER_API_KEY>
Content-Type: application/json
```

这是 XimoAI 的火山图像原生入口。请求体使用图像原生 JSON，XimoAI 只根据显式渠道/账号映射替换 `model` 字段；提示词、图片、尺寸、流式开关和其他参数不转换。

### 7.2 请求字段

| 字段 | 类型 | 必填 | 取值/说明 |
|---|---|---|---|
| `model` | string | 是 | `doubao-seedream-5.0-lite` 或配置后的上游 Model ID/Endpoint ID |
| `prompt` | string | 是 | 文生图或图生图提示词 |
| `image` | array[string] | 否 | URL 或 Base64 图片；具体数量由模型版本决定 |
| `size` | string | 否 | 输出宽高或官方尺寸别名，例如 `2K`、`2048x2048` |
| `seed` | int32 | 否 | 随机种子；相同 seed 只能提高结果稳定性，不能保证完全一致 |
| `sequential_image_generation` | string | 否 | `auto` 组图，`disabled` 单图 |
| `sequential_image_generation_options` | object | 否 | 仅 `auto` 时生效 |
| `sequential_image_generation_options.max_images` | int32 | 条件 | 组图最大图片数，由模型上限约束 |
| `stream` | bool | 否 | 是否流式输出；只有模型版本支持时才可用 |
| `guidance_scale` | float | 否 | Prompt 一致性/自由度，官方范围 `[1, 10]` |
| `response_format` | string | 否 | `url` 或 `b64_json` |
| `watermark` | bool | 否 | 是否添加图像水印 |
| `optimize_prompt_options` | object | 否 | 提示词优化配置，仅部分模型支持 |
| `optimize_prompt_options.mode` | string | 条件 | 当前官方接口按模型开放值使用；常见为 `standard` 或 `fast` |

请求示例，展示完整字段层级：

```json
{
  "model": "doubao-seedream-5.0-lite",
  "prompt": "一座建在云海之上的未来城市，电影级光影，细节丰富",
  "image": ["https://example.com/reference.png"],
  "size": "2K",
  "seed": 123456,
  "sequential_image_generation": "disabled",
  "sequential_image_generation_options": {
    "max_images": 1
  },
  "stream": false,
  "guidance_scale": 5.5,
  "response_format": "url",
  "watermark": false,
  "optimize_prompt_options": {
    "mode": "standard"
  }
}
```

`doubao-seedream-5.0-lite` 是否接受每一个可选字段、图片数量、尺寸和流式输出，最终由上游该模型版本返回结果决定。XimoAI 不会为它增加 `n`、`size` 猜测、图片下载或响应转换等 OpenAI 兼容逻辑。

### 7.3 同步 JSON 响应

当 `stream` 为 `false` 时，响应保留火山图像原生 JSON：

| 字段 | 类型 | 说明 |
|---|---|---|
| `model` | string | 实际执行的模型名称/版本 |
| `created` | int32 | Unix 秒级创建时间 |
| `data` | array | 输出图片列表 |
| `data[].url` | string | 图片下载 URL，`response_format=url` 时返回 |
| `data[].b64_json` | string | Base64 图片，`response_format=b64_json` 时返回 |
| `data[].size` | string | 返回图片尺寸 |
| `usage` | object | 上游用量信息，具体字段按模型/版本返回 |
| `usage.generated_images` | int | 生成图片数量，按上游返回 |
| `usage.input_tokens` | int | 输入 token，用量存在时返回 |
| `usage.output_tokens` | int | 输出 token，用量存在时返回 |
| `usage.total_tokens` | int | 总 token，用量存在时返回 |
| `error` | object | 任务失败时的原生错误对象 |
| `error.code` | string | 错误码，按上游返回 |
| `error.message` | string | 错误信息 |

### 7.4 流式 SSE 响应

当 `stream` 为 `true` 且目标模型支持流式时，响应为火山原生 SSE。XimoAI 不改写：

```text
Content-Type: text/event-stream
event: <upstream event name>
data: <upstream JSON data>

...
data: [DONE]
```

事件中的 `type`、`image_index`、`url`、`b64_json`、`size`、`usage`、`error` 等字段按上游原样返回。模型不支持流式时，应使用 `stream: false` 并按上游错误处理。

## 8. 不要使用的入口

下面是 OpenAI/兼容平台入口，不是本组三个模型的原生入口：

```text
/v1/audio/speech
/v1/audio/transcriptions
/v1/audio/translations
/v1/chat/completions
/v1/images/generations
```

本组三个模型应使用本文第 3 至第 7 节的 `/api/v3/plan/...`、`/api/plan/v3/...` 和 WebSocket 原生帧。不要把火山原生请求改成 OpenAI payload，也不要期待 XimoAI 替客户端完成 TTS/ASR/图像协议适配。

## 9. 计费、失败和诊断

- `X-Api-Resource-Id` 是火山侧模型/计费资源选择的一部分；TTS 使用 `seed-tts-2.0`，ASR 使用 `volc.seedasr.sauc.duration`。
- TTS HTTP 的 `usage.text_words`、TTS WS 的 `SessionFinished` 用量和图像响应的 `usage` 会被保留；XimoAI 本地计费仍以渠道定价和可观察的上游用量为准。
- 音频中间帧、ASR 中间识别帧和图像 SSE 中间事件不是独立的模型请求，不应被客户端拆成多次 API 调用。
- 上游 HTTP 错误、Chunked JSON 错误、WebSocket 错误帧、SSE `error` 事件会保留原始诊断信息；同时网关会记录内部请求关联信息。
- 只有上游成功完成且满足当前渠道计费规则时，才会记录成功用量；鉴权失败、账号不可调度、协议错误和上游失败不会伪造成成功。

## 10. 官方字段参考

本文字段以以下官方页面在 2026-07-27 的内容为准；模型、音色、可选字段和限制可能随上游更新：

- [单向流式语音合成 HTTP](https://docs.volcengine.com/docs/6561/2528925?lang=zh)
- [WebSocket 双向流式 TTS V3](https://docs.volcengine.com/docs/6561/1329505?lang=zh)
- [WebSocket 单向流式 TTS V3](https://docs.volcengine.com/docs/6561/1719100?lang=zh)
- [大模型流式语音识别 WebSocket](https://docs.volcengine.com/docs/6561/1354869?lang=zh)
- [火山方舟 ImageGenerations](https://api.volcengine.com/api-explorer/debug?action=ImageGenerations&groupName=%E5%9B%BE%E7%89%87%E7%94%9F%E6%88%90API&serviceCode=ark&version=2024-01-01)
- [Seedream 5.0 lite、4.5、4.0 提示词与能力说明](https://www.volcengine.com/docs/82379/1829186)
