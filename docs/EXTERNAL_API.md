# XimoAi External API Reference

本文档按当前代码实现整理外部可调用接口。覆盖范围包括用户登录、用户后台、管理员后台、支付回调、协议网关、模型列表、模型广场、平台管理、异步视频任务查询等入口。

## 1. 基本约定

### Base URL

生产环境示例：

```text
ximoai.cn
```

管理/用户 REST 接口默认前缀：

```text
/api/v1
```

协议网关接口默认前缀按协议区分：

```text
/v1
/v1beta
/antigravity
/antigravity/v1
/antigravity/v1beta
```

部分 OpenAI 兼容接口同时保留根路径别名，例如 `/responses`、`/chat/completions`、`/images/generations`、`/videos`。

### REST 响应包裹

后台和用户 REST 接口通常返回统一结构：

```json
{
  "code": 0,
  "message": "success",
  "reason": "",
  "metadata": {},
  "data": {}
}
```

分页接口的 `data` 通常为：

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "page_size": 20,
  "pages": 0
}
```

协议网关接口不使用这个包裹，返回对应协议的原生响应或上游原始响应。

### 时间和金额

时间字段使用 ISO/RFC3339 字符串或数据库序列化后的时间值。金额字段以 USD 计价，站内余额、消费、价格均使用浮点数。

### 管理端权限

`/api/v1/admin/*` 需要管理员 JWT。管理员路由还会经过合规确认保护，未确认时会被拦截。

支持的管理员鉴权：

```http
Authorization: Bearer <jwt_access_token>
```

部分 WebSocket 管理入口会兼容 `token` query 参数。

### 用户端权限

`/api/v1/user/*`、`/api/v1/keys/*`、`/api/v1/groups/*`、`/api/v1/channels/*` 等用户接口需要用户 JWT：

```http
Authorization: Bearer <jwt_access_token>
```

### 协议网关 API Key 鉴权

OpenAI/Anthropic/Antigravity `/v1` 类入口支持：

```http
Authorization: Bearer <api_key>
x-api-key: <api_key>
x-goog-api-key: <api_key>
```

普通 `/v1` 网关明确拒绝 query 中的 `key` 和 `api_key`。

Gemini `/v1beta` 和 `/antigravity/v1beta` 入口支持：

```http
x-goog-api-key: <api_key>
Authorization: Bearer <api_key>
x-api-key: <api_key>
```

Gemini 入口还兼容 query 参数：

```text
?key=<api_key>
```

Gemini 入口明确拒绝：

```text
?api_key=<api_key>
```

API Key 会校验 Key 状态、用户状态、分组可用性、IP 黑白名单、余额、订阅、限额和模型可售配置。

## 2. 通用公开接口

| 方法 | 路径 | 鉴权 | 响应 |
|---|---|---|---|
| `GET` | `/health` | 无 | `{"status":"ok"}` |
| `POST` | `/api/event_logging/batch` | 无 | `200` 空响应 |
| `GET` | `/setup/status` | 无 | `{code:0,data:{needs_setup:false,step:"completed"}}` |

## 3. 登录、注册和 OAuth

### 基础认证接口

| 方法 | 路径 | 请求字段 | 响应字段 |
|---|---|---|---|
| `POST` | `/api/v1/auth/register` | `email,password,verify_code,turnstile_token,promo_code,invitation_code,aff_code` | `AuthResponse` |
| `POST` | `/api/v1/auth/login` | `email,password,turnstile_token` | `AuthResponse` 或 `TotpLoginResponse` |
| `POST` | `/api/v1/auth/login/2fa` | `temp_token,totp_code` | `AuthResponse` |
| `POST` | `/api/v1/auth/send-verify-code` | `email,turnstile_token` | `message,countdown` |
| `POST` | `/api/v1/auth/refresh` | `refresh_token` | `access_token,refresh_token,expires_in,token_type` |
| `POST` | `/api/v1/auth/logout` | `refresh_token` | `message` |
| `POST` | `/api/v1/auth/validate-promo-code` | `code` | `valid,bonus_amount,error_code,message` |
| `POST` | `/api/v1/auth/validate-invitation-code` | `code` | `valid,error_code` |
| `POST` | `/api/v1/auth/forgot-password` | `email,turnstile_token` | `message` |
| `POST` | `/api/v1/auth/reset-password` | `email,token,new_password` | `message` |
| `GET` | `/api/v1/auth/me` | JWT | `User` |
| `POST` | `/api/v1/auth/revoke-all-sessions` | JWT | `message` |
| `POST` | `/api/v1/auth/oauth/bind-token` | JWT | OAuth 绑定 token 结果 |

### AuthResponse

```json
{
  "access_token": "jwt",
  "refresh_token": "token",
  "expires_in": 3600,
  "token_type": "Bearer",
  "user": {}
}
```

### TotpLoginResponse

```json
{
  "requires_2fa": true,
  "temp_token": "token",
  "user_email_masked": "u***@example.com"
}
```

### OAuth 入口

| Provider | 开始 | 回调 | 完成注册/绑定/创建 |
|---|---|---|---|
| LinuxDo | `GET /api/v1/auth/oauth/linuxdo/start`、`GET /api/v1/auth/oauth/linuxdo/bind/start` | `GET /api/v1/auth/oauth/linuxdo/callback` | `POST /oauth/linuxdo/complete-registration`、`POST /oauth/linuxdo/bind-login`、`POST /oauth/linuxdo/create-account` |
| GitHub | `GET /api/v1/auth/oauth/github/start` | `GET /api/v1/auth/oauth/github/callback` | `POST /oauth/github/complete-registration` |
| Google | `GET /api/v1/auth/oauth/google/start` | `GET /api/v1/auth/oauth/google/callback` | `POST /oauth/google/complete-registration` |
| WeChat | `GET /api/v1/auth/oauth/wechat/start`、`GET /api/v1/auth/oauth/wechat/bind/start`、`GET /api/v1/auth/oauth/wechat/payment/start` | `GET /oauth/wechat/callback`、`GET /oauth/wechat/payment/callback` | `POST /oauth/wechat/complete-registration`、`POST /oauth/wechat/bind-login`、`POST /oauth/wechat/create-account` |
| OIDC | `GET /api/v1/auth/oauth/oidc/start`、`GET /api/v1/auth/oauth/oidc/bind/start` | `GET /oauth/oidc/callback` | `POST /oauth/oidc/complete-registration`、`POST /oauth/oidc/bind-login`、`POST /oauth/oidc/create-account` |
| DingTalk | `GET /api/v1/auth/oauth/dingtalk/start`、`GET /api/v1/auth/oauth/dingtalk/bind/start` | `GET /oauth/dingtalk/callback` | `POST /oauth/dingtalk/complete-registration` |
| Pending OAuth | 无 | 无 | `POST /oauth/pending/exchange`、`POST /oauth/pending/send-verify-code`、`POST /oauth/pending/create-account`、`POST /oauth/pending/bind-login` |

OAuth 开始接口常用 query：`redirect_to,intent`。完成类接口按 Provider 返回的 pending token、邮箱、验证码、用户名等字段提交，最终返回 `AuthResponse` 或绑定结果。

## 4. 用户接口

### 用户资料与安全

| 方法 | 路径 | 请求字段 | 返回 |
|---|---|---|---|
| `GET` | `/api/v1/user/profile` | 无 | `UserProfile` |
| `PUT` | `/api/v1/user/password` | `old_password,new_password` | `message` |
| `PUT` | `/api/v1/user` | `username,avatar_url,balance_notify_enabled,balance_notify_threshold` | `User` |
| `GET` | `/api/v1/user/aff` | 无 | 邀请返利信息 |
| `POST` | `/api/v1/user/aff/transfer` | 转账金额等返利字段 | 转账结果 |
| `GET` | `/api/v1/user/platform-quotas` | 无 | 平台额度列表 |

`UserProfile` 在 `User` 基础上额外返回：

```text
avatar_url, avatar_source, username_source, display_name_source, nickname_source,
profile_sources, identities, auth_bindings, identity_bindings,
email_bound, linuxdo_bound, oidc_bound, wechat_bound, dingtalk_bound
```

账号绑定：

| 方法 | 路径 | 请求字段 |
|---|---|---|
| `POST` | `/api/v1/user/account-bindings/email/send-code` | `email` |
| `POST` | `/api/v1/user/account-bindings/email` | `email,verify_code,password` |
| `DELETE` | `/api/v1/user/account-bindings/:provider` | 路径参数 `provider` |
| `POST` | `/api/v1/user/auth-identities/bind/start` | `provider,redirect_to` |

通知邮箱：

| 方法 | 路径 | 请求字段 |
|---|---|---|
| `POST` | `/api/v1/user/notify-emails/send-code` | `email` |
| `POST` | `/api/v1/user/notify-emails/verify` | `email,code` |
| `PUT` | `/api/v1/user/notify-emails/toggle` | `email,disabled` |
| `DELETE` | `/api/v1/user/notify-emails` | `email` |

TOTP：

| 方法 | 路径 |
|---|---|
| `GET` | `/api/v1/user/totp/status` |
| `GET` | `/api/v1/user/totp/verification-method` |
| `POST` | `/api/v1/user/totp/send-code` |
| `POST` | `/api/v1/user/totp/setup` |
| `POST` | `/api/v1/user/totp/enable` |
| `POST` | `/api/v1/user/totp/disable` |

### 子网站登录会员资料

该接口用于子网站在用户完成本站登录后，读取当前登录用户的会员系统信息、可用分组和系统托管 API Key。它不接受 `user_id` 参数，只返回 JWT 所属用户的数据。

| 方法 | 路径 | 鉴权 | 返回 |
|---|---|---|---|
| `GET` | `/api/v1/membership/external-profile` | 用户 JWT | `ExternalMembershipProfile` |

请求示例：

```http
GET /api/v1/membership/external-profile HTTP/1.1
Host: ximoai.cn
Authorization: Bearer <jwt_access_token>
```

返回示例：

```json
{
  "code": 0,
  "message": "success",
  "reason": "",
  "metadata": {},
  "data": {
    "user_id": 123,
    "membership": {
      "level": {
        "id": 5,
        "name": "钻石会员",
        "code": "diamond",
        "color": "#0099ff",
        "discount_rate": 0.8,
        "enabled": true,
        "is_default": false,
        "sort_order": 4,
        "description": "",
        "groups": []
      },
      "starts_at": "2026-06-18T00:00:00Z",
      "expires_at": null,
      "levels": [],
      "groups": [
        {
          "id": 8,
          "name": "codex自用",
          "platform": "openai",
          "rate_multiplier": 1,
          "effective_rate_multiplier": 0.8,
          "is_exclusive": true,
          "status": "active",
          "subscription_type": "standard"
        }
      ],
      "managed_keys": [
        {
          "id": 1,
          "user_id": 123,
          "group_id": 8,
          "api_key_id": 99,
          "membership_level_id": 5,
          "status": "active",
          "disabled_reason": "",
          "group": {},
          "api_key": {
            "id": 99,
            "user_id": 123,
            "key": "sk-full-managed-key",
            "key_suffix": "d-key",
            "masked_key": "sk-...d-key",
            "name": "Membership Key - codex自用",
            "status": "active",
            "group_id": 8
          }
        }
      ]
    }
  }
}
```

字段说明：

```text
data.user_id: 当前 JWT 所属用户 ID
data.membership.level: 当前会员等级
data.membership.levels: 全部可展示会员等级及权益
data.membership.groups: 当前会员等级可用分组；effective_rate_multiplier 为分组倍率与会员折扣后的实际结算倍率
data.membership.managed_keys: 系统为该会员等级托管的 API Key 列表
data.membership.managed_keys[].api_key.key: 完整 API Key，供子网站代表当前用户调用协议网关
data.membership.managed_keys[].api_key.masked_key: 脱敏 Key，仅供展示
```

安全约定：

```text
该接口返回完整托管 API Key，只应由可信子网站服务端调用或在受控前端场景中使用。
不要把返回结果写入公开日志、浏览器长期存储或第三方分析系统。
会员中心展示接口 /api/v1/membership 仍只返回 masked_key/key_suffix，不返回完整 key。
```

### 用户 API Key

| 方法 | 路径 |
|---|---|
| `GET` | `/api/v1/keys` |
| `GET` | `/api/v1/keys/:id` |
| `POST` | `/api/v1/keys` |
| `PUT` | `/api/v1/keys/:id` |
| `DELETE` | `/api/v1/keys/:id` |

创建字段：

```text
name, group_id, custom_key, ip_whitelist, ip_blacklist, quota,
expires_in_days, rate_limit_5h, rate_limit_1d, rate_limit_7d
```

更新字段：

```text
name, group_id, status, ip_whitelist, ip_blacklist, quota, expires_at,
reset_quota, rate_limit_5h, rate_limit_1d, rate_limit_7d, reset_rate_limit_usage
```

列表 query：

```text
page, page_size, limit, sort_by, sort_order, search, status, group_id
```

### 分组、渠道、模型广场

| 方法 | 路径 | 返回 |
|---|---|---|
| `GET` | `/api/v1/groups/available` | 用户可用 `Group` |
| `GET` | `/api/v1/groups/rates` | 用户分组倍率 |
| `GET` | `/api/v1/channels/available` | 用户可见渠道，受功能开关影响 |
| `GET` | `/api/v1/channels/model-plaza` | 模型广场可售模型，不依赖 available channels 开关 |
| `GET` | `/api/v1/platforms` | 公开平台列表，隐藏 `base_url/auth_modes` |

用户可见模型广场字段：

```text
channel.name
channel.description
platforms[].platform
platforms[].groups[].id
platforms[].groups[].name
platforms[].groups[].platform
platforms[].groups[].subscription_type
platforms[].groups[].rate_multiplier
platforms[].groups[].is_exclusive
platforms[].supported_models[].name
platforms[].supported_models[].platform
platforms[].supported_models[].pricing.billing_mode
platforms[].supported_models[].pricing.input_price
platforms[].supported_models[].pricing.output_price
platforms[].supported_models[].pricing.cache_write_price
platforms[].supported_models[].pricing.cache_read_price
platforms[].supported_models[].pricing.image_output_price
platforms[].supported_models[].pricing.per_request_price
platforms[].supported_models[].pricing.intervals[].min_tokens
platforms[].supported_models[].pricing.intervals[].max_tokens
platforms[].supported_models[].pricing.intervals[].tier_label
platforms[].supported_models[].pricing.intervals[].input_price
platforms[].supported_models[].pricing.intervals[].output_price
platforms[].supported_models[].pricing.intervals[].cache_write_price
platforms[].supported_models[].pricing.intervals[].cache_read_price
platforms[].supported_models[].pricing.intervals[].per_request_price
```

### 使用记录、订阅、公告、兑换

| 方法 | 路径 |
|---|---|
| `GET` | `/api/v1/usage` |
| `GET` | `/api/v1/usage/errors` |
| `GET` | `/api/v1/usage/errors/:id` |
| `GET` | `/api/v1/usage/:id` |
| `GET` | `/api/v1/usage/stats` |
| `GET` | `/api/v1/usage/dashboard/stats` |
| `GET` | `/api/v1/usage/dashboard/trend` |
| `GET` | `/api/v1/usage/dashboard/models` |
| `POST` | `/api/v1/usage/dashboard/api-keys-usage`，字段 `api_key_ids` |
| `GET` | `/api/v1/user/api-keys/:id/usage/daily` |
| `GET` | `/api/v1/announcements` |
| `POST` | `/api/v1/announcements/:id/read` |
| `POST` | `/api/v1/redeem` |
| `GET` | `/api/v1/redeem/history` |
| `GET` | `/api/v1/subscriptions` |
| `GET` | `/api/v1/subscriptions/active` |
| `GET` | `/api/v1/subscriptions/progress` |
| `GET` | `/api/v1/subscriptions/summary` |
| `GET` | `/api/v1/channel-monitors` |
| `GET` | `/api/v1/channel-monitors/:id/status` |

## 5. 支付接口

### 用户支付

| 方法 | 路径 | 请求字段 |
|---|---|---|
| `GET` | `/api/v1/payment/config` | 无 |
| `GET` | `/api/v1/payment/checkout-info` | 无 |
| `GET` | `/api/v1/payment/plans` | 无 |
| `GET` | `/api/v1/payment/channels` | 无 |
| `GET` | `/api/v1/payment/limits` | 无 |
| `POST` | `/api/v1/payment/orders` | `amount,payment_type,openid,wechat_resume_token,return_url,payment_source,order_type,plan_id,is_mobile` |
| `POST` | `/api/v1/payment/orders/verify` | `out_trade_no` |
| `GET` | `/api/v1/payment/orders/my` | query 分页 |
| `GET` | `/api/v1/payment/orders/:id` | 路径参数 `id` |
| `POST` | `/api/v1/payment/orders/:id/cancel` | 无 |
| `POST` | `/api/v1/payment/orders/:id/refund-request` | `reason` |
| `GET` | `/api/v1/payment/orders/refund-eligible-providers` | 无 |

### 公开支付恢复

| 方法 | 路径 | 请求字段 |
|---|---|---|
| `POST` | `/api/v1/payment/public/orders/verify` | `out_trade_no` |
| `POST` | `/api/v1/payment/public/orders/resolve` | `resume_token` |

### 支付 Webhook

| 方法 | 路径 |
|---|---|
| `GET/POST` | `/api/v1/payment/webhook/easypay` |
| `POST` | `/api/v1/payment/webhook/alipay` |
| `POST` | `/api/v1/payment/webhook/wxpay` |
| `POST` | `/api/v1/payment/webhook/stripe` |
| `POST` | `/api/v1/payment/webhook/airwallex` |

支付订单返回字段：

```text
id, user_id, amount, pay_amount, fee_rate, currency, payment_type, out_trade_no,
status, order_type, created_at, expires_at, paid_at, completed_at,
refund_amount, refund_reason, refund_requested_at, refund_requested_by,
refund_request_reason, plan_id, provider_instance_id
```

支付计划字段：

```text
id, group_id, group_platform, group_name, rate_multiplier,
daily_limit_usd, weekly_limit_usd, monthly_limit_usd, supported_model_scopes,
name, description, price, original_price, validity_days, validity_unit,
features, product_name, for_sale, sort_order
```

## 6. 协议网关

### 模型列表规则

`GET /v1/models` 不直接暴露上游账号模型，也不暴露账号模型映射细节。它只返回当前 API Key 分组可见、渠道定价中存在正价格的可售模型。若分组启用了自定义模型列表，会在可售模型集合上再过滤。

OpenAI 平台返回：

```json
{
  "object": "list",
  "data": [
    {
      "id": "model",
      "object": "model",
      "created": 1704067200,
      "owned_by": "openai",
      "type": "model",
      "display_name": "model"
    }
  ]
}
```

非 OpenAI 平台返回：

```json
{
  "object": "list",
  "data": [
    {
      "id": "model",
      "type": "model",
      "display_name": "model",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

携带入口协议元数据：

```http
GET /v1/models?include_entry_protocols=1
Authorization: Bearer <api_key>
```

该参数开启后，模型条目使用精简结构，只返回模型 ID、固定 `object` 和 `ximoai` 扩展信息；不会返回 `created`、`owned_by`、`type`、`display_name`、`platform` 或 `supported_entry_protocols`。

```json
{
  "object": "list",
  "data": [
    {
      "id": "NanoBanana2",
      "object": "model",
      "ximoai": {
        "default_entry_protocol": "gemini",
        "default_endpoint": "/v1beta/models/NanoBanana2:generateContent",
        "model_type": "image",
        "execution_mode": "sync",
        "supports_stream": false,
        "supports_polling": false,
        "group": {
          "id": 12,
          "name": "梦工厂 gemini",
          "subscription_type": "standard",
          "is_exclusive": false,
          "rate_multiplier": 1,
          "effective_rate_multiplier": 1
        },
        "pricing": {
          "billing_mode": "image",
          "input_price": null,
          "output_price": null,
          "cache_write_price": null,
          "cache_read_price": null,
          "image_output_price": null,
          "per_request_price": 0.00006,
          "intervals": []
        }
      }
    }
  ]
}
```

`ximoai` 字段：

```text
default_entry_protocol: 推荐入口协议，例如 openai、anthropic、gemini
default_endpoint: 推荐入口端点
model_type: 公共模型类型，可能为 chat、image、video、audio、transcription、translation
execution_mode: 公共调用模式，sync 或 async
supports_stream: 该公共入口是否支持流式返回
supports_polling: 该公共入口是否需要/支持轮询查询结果
group.id: 当前 API Key 绑定分组 ID
group.name: 当前 API Key 绑定分组名称
group.subscription_type: standard 或 subscription
group.is_exclusive: 是否专属分组
group.rate_multiplier: 分组默认倍率
group.effective_rate_multiplier: 当前 API Key 所属用户在该分组的实际生效倍率；若未设置用户专属倍率，则等于分组默认倍率
pricing.billing_mode: token、per_request、image 或 video
pricing.input_price: 输入价格
pricing.output_price: 输出价格
pricing.cache_write_price: 缓存写入价格
pricing.cache_read_price: 缓存读取价格
pricing.image_output_price: 图片输出价格
pricing.per_request_price: 按次价格
pricing.intervals[]: 分层计价，字段同模型广场 pricing.intervals
```

调用方应以 `default_endpoint` 为准构造请求，不要只按 `model_type` 固定选择端点。例如 `model_type=audio`
既可能是 `/v1/audio/speech` 语音合成，也可能是 `/v1/chat/completions` 音频对话模型。

### `/v1/models?include_entry_protocols=1` 调用契约字段

开启 `include_entry_protocols=1` 后，每个模型的 `ximoai` 字段会额外返回调用契约，外部调用方应优先按这些字段构造请求和解析响应：

```json
{
  "id": "kling-audio",
  "object": "model",
  "ximoai": {
    "default_entry_protocol": "openai",
    "default_endpoint": "/v1/audio/speech",
    "model_type": "audio",
    "operation_type": "audio_tts",
    "execution_mode": "sync",
    "supports_stream": false,
    "supports_polling": false,
    "request_contract": {
      "required_fields": ["model", "input", "voice_id"],
      "optional_fields": ["voice_language", "voice_speed"],
      "field_notes": {
        "voice_id": "Must be a Kling voice id, not OpenAI voice names such as alloy or nova."
      },
      "examples": {
        "voice_id": "genshin_vindi2",
        "voice_language": "zh",
        "voice_speed": 1
      }
    },
    "response_contract": {
      "delivery": "json_url",
      "audio_url_path": "data.task_result.audios[0].url",
      "duration_path": "data.task_result.audios[0].duration",
      "task_id_path": "data.task_id"
    }
  }
}
```

`operation_type` 表示该模型在公共入口中的真实操作类型。常见值：

| operation_type | 说明 |
|---|---|
| `chat` | 文本/多模态对话 |
| `chat_audio` | OpenAI Chat Completions 音频对话，音频通常在 `choices[0].message.audio.data` |
| `audio_tts` | 文本转语音 |
| `audio_transcription` | 音频转写 |
| `audio_translation` | 音频翻译 |
| `image_generation` | 图像生成 |
| `video_generation` | 视频生成，通常为异步任务 |
| `voice_management` | 音色创建/查询，不是普通语音合成 |
| `voice_catalog` | 音色列表查询 |

外部调用方不要只按 `model_type` 固定选择请求格式。比如 `model_type=audio` 可能是 `/v1/audio/speech`，也可能是 `/v1/chat/completions` 的音频对话模型，还可能是可灵音色管理能力。应按 `operation_type`、`request_contract` 和 `response_contract` 决定请求字段和结果解析方式。

### Anthropic Messages 入口

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/v1/messages` | Anthropic Messages |
| `POST` | `/v1/messages/count_tokens` | token 计数 |
| `GET` | `/v1/models` | 模型列表 |
| `GET` | `/v1/usage` | 当前 Key 用量 |
| `POST` | `/antigravity/v1/messages` | Antigravity 强制平台 |
| `POST` | `/antigravity/v1/messages/count_tokens` | Antigravity token 计数 |
| `GET` | `/antigravity/v1/models` | Antigravity 模型列表 |
| `GET` | `/antigravity/v1/usage` | Antigravity 用量 |

Anthropic request 字段：

```text
model, max_tokens, system, messages, tools, stream, temperature, top_p,
stop_sequences, thinking, tool_choice, metadata, output_config
```

`messages[].content` 支持字符串或内容块数组。内容块字段：

```text
type, cache_control, text, thinking, source, id, name, input,
tool_use_id, content, is_error
```

`source` 字段：

```text
type, media_type, data
```

工具字段：

```text
type, name, description, input_schema, cache_control
```

非流式 Anthropic response 字段：

```text
id, type, role, content, model, stop_reason, stop_sequence, usage
```

`usage` 字段：

```text
input_tokens, output_tokens, cache_creation_input_tokens, cache_read_input_tokens
```

流式返回为 SSE，事件字段包括：

```text
type, message, index, content_block, delta, usage
```

`delta` 字段包括：

```text
type, text, partial_json, thinking, signature, stop_reason, stop_sequence
```

### OpenAI Responses 入口

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/v1/responses` | OpenAI Responses |
| `POST` | `/v1/responses/*subpath` | Responses 子路径透传 |
| `GET` | `/v1/responses` | Realtime WebSocket |
| `POST` | `/responses` | 根路径别名 |
| `POST` | `/responses/*subpath` | 根路径子路径别名 |
| `GET` | `/responses` | 根路径 Realtime WebSocket |
| `GET` | `/responses/*subpath` | Responses 子路径查询透传 |
| `DELETE` | `/responses/*subpath` | Responses 子路径删除透传 |
| `POST` | `/backend-api/codex/responses` | 仅 OpenAI 官方平台 |
| `POST` | `/backend-api/codex/responses/*subpath` | 仅 OpenAI 官方平台 |
| `GET` | `/backend-api/codex/responses` | 仅 OpenAI 官方平台 Realtime |

Responses request 字段：

```text
model, instructions, input, max_output_tokens, temperature, top_p, stream,
tools, include, store, parallel_tool_calls, reasoning, text, tool_choice,
service_tier, prompt_cache_key, previous_response_id
```

`reasoning` 字段：

```text
effort, summary
```

`text` 字段：

```text
verbosity
```

`input[]` 字段：

```text
type, role, content, call_id, name, arguments, id, output
```

`content[]` 字段：

```text
type, text, image_url
```

`tools[]` 字段：

```text
type, name, description, parameters, strict
```

Responses response 字段：

```text
id, object, model, status, output, usage, incomplete_details, error
```

`output[]` 字段：

```text
type, id, role, content, status, encrypted_content, summary,
call_id, name, arguments, action
```

`usage` 字段：

```text
input_tokens, output_tokens, total_tokens,
input_tokens_details, output_tokens_details
```

`input_tokens_details`：

```text
cached_tokens, audio_tokens
```

`output_tokens_details`：

```text
reasoning_tokens, audio_tokens, accepted_prediction_tokens, rejected_prediction_tokens
```

流式 Responses SSE 事件字段：

```text
type, response, usage, item, output_index, content_index, delta, text,
item_id, call_id, name, arguments, summary_index, part, code, param,
sequence_number
```

### OpenAI Chat Completions 入口

| 方法 | 路径 |
|---|---|
| `POST` | `/v1/chat/completions` |
| `POST` | `/chat/completions` |

请求字段：

```text
model, messages, instructions, max_tokens, max_completion_tokens,
temperature, top_p, stream, stream_options, tools, tool_choice,
reasoning_effort, service_tier, stop, functions, function_call
```

`messages[]` 字段：

```text
role, content, reasoning_content, name, tool_calls, tool_call_id, function_call
```

多模态内容块：

```text
type, text, image_url
```

`image_url`：

```text
url, detail
```

`tools[]`：

```text
type, function
```

`function`：

```text
name, description, parameters, strict
```

非流式响应字段：

```text
id, object, created, model, choices, usage, system_fingerprint, service_tier
```

`choices[]`：

```text
index, message, finish_reason
```

流式 chunk 字段：

```text
id, object, created, model, choices, usage, system_fingerprint, service_tier
```

`choices[].delta`：

```text
role, content, reasoning_content, tool_calls
```

### OpenAI Embeddings、Images、Audio

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/v1/embeddings` | OpenAI embeddings |
| `POST` | `/embeddings` | 根路径别名 |
| `POST` | `/v1/images/generations` | 图像生成 |
| `POST` | `/v1/images/edits` | 图像编辑 |
| `POST` | `/images/generations` | 根路径别名 |
| `POST` | `/images/edits` | 根路径别名 |
| `POST` | `/v1/audio/speech` | 语音合成 |
| `POST` | `/v1/audio/transcriptions` | 语音转写 |
| `POST` | `/v1/audio/translations` | 语音翻译 |
| `POST` | `/audio/speech` | 根路径别名 |
| `POST` | `/audio/transcriptions` | 根路径别名 |
| `POST` | `/audio/translations` | 根路径别名 |

Audio 请求可以是 JSON 或 multipart。本站必须能解析 `model`，否则返回错误。其余字段按 OpenAI 兼容上游透传或按平台适配。

### OpenAI Realtime

| 方法 | 路径 |
|---|---|
| `GET` | `/v1/realtime` |
| `GET` | `/realtime` |
| `GET` | `/v1/responses` |
| `GET` | `/responses` |

Realtime 使用 WebSocket。鉴权仍使用 API Key，平台需具备 realtime capability。

### OpenAI Videos 与异步查询

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/v1/videos` | 创建视频任务 |
| `GET` | `/v1/videos` | 查询当前 Key 可见任务列表 |
| `GET` | `/v1/videos/:id` | 查询任务状态 |
| `DELETE` | `/v1/videos/:id` | 删除任务或透传删除 |
| `GET` | `/v1/videos/:id/content` | 获取视频内容 |
| `POST` | `/v1/videos/edits` | 视频编辑 |
| `POST` | `/v1/videos/extensions` | 视频扩展 |
| `POST` | `/v1/videos/:id/remix` | remix |
| `POST` | `/v1/videos/characters` | 创建角色 |
| `GET` | `/v1/videos/characters/:id` | 查询角色 |
| `POST` | `/videos` | 根路径别名 |
| `GET` | `/videos` | 根路径别名 |
| `POST/GET/DELETE` | `/videos/*subpath` | 根路径子路径别名 |

创建类请求可以是 JSON 或 multipart。本站固定解析以下字段：

```text
model
```

为了关联 remix/edit/extend 的来源任务，会额外识别：

```text
video.id, video_id, source_video_id, input_video.id, input_video_id,
multipart: video, video_id, source_video_id, video[id]
```

对 OpenAI 兼容上游，其他字段按原 body 透传。对 Gemini 兼容上游，会按 Gemini 视频 endpoint 转换。用户必须自行调用查询入口获取异步结果，本站不主动替用户轮询。

### Gemini Native 入口

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/v1beta/models` | Gemini 模型列表 |
| `GET` | `/v1beta/models/:model` | Gemini 模型详情 |
| `GET` | `/v1beta/operations/*operation` | Gemini 异步任务查询 |
| `POST` | `/v1beta/models/*modelAction` | Gemini 模型动作 |

支持的 `modelAction` 形态包括：

```text
models/{model}:generateContent
models/{model}:streamGenerateContent
models/{model}:countTokens
models/{model}:generateVideos
```

Gemini 请求体为 Gemini 原生 JSON。常见字段：

```text
contents, systemInstruction, safetySettings, generationConfig, tools,
toolConfig, cachedContent
```

视频生成常见字段：

```text
prompt, image, video, config, negativePrompt, aspectRatio, durationSeconds,
personGeneration
```

无法明确映射的跨协议高级字段不会强行传递。

### Antigravity 入口

| 方法 | 路径 |
|---|---|
| `GET` | `/antigravity/models` |
| `POST` | `/antigravity/v1/messages` |
| `POST` | `/antigravity/v1/messages/count_tokens` |
| `GET` | `/antigravity/v1/models` |
| `GET` | `/antigravity/v1/usage` |
| `GET` | `/antigravity/v1beta/models` |
| `GET` | `/antigravity/v1beta/models/:model` |
| `POST` | `/antigravity/v1beta/models/*modelAction` |

Antigravity v1beta 不提供 `/operations/*operation` 查询入口。Gemini operations 查询仅在 `/v1beta/operations/*operation` 暴露。

### Header 透传白名单

OpenAI 兼容透传会保留以下请求头：

```text
accept, accept-language, content-type, conversation_id, openai-beta,
user-agent, originator, session_id, x-codex-turn-state,
x-codex-turn-metadata, idempotency-key, openai-organization,
openai-project, openai-version, x-request-id, x-stainless-arch,
x-stainless-connect-timeout, x-stainless-lang, x-stainless-os,
x-stainless-package-version, x-stainless-read-timeout,
x-stainless-retry-count, x-stainless-runtime,
x-stainless-runtime-version, x-stainless-timeout
```

不会全量透传任意自定义 Header。对 Gemini 视频上游，`authorization` 和 `x-goog-api-key` 会被替换为上游账号凭据。

## 7. 平台和渠道管理

### 平台管理

| 方法 | 路径 | 权限 |
|---|---|---|
| `GET` | `/api/v1/admin/platforms` | 管理员 |
| `POST` | `/api/v1/admin/platforms` | 管理员 |
| `PUT` | `/api/v1/admin/platforms/:slug` | 管理员 |
| `DELETE` | `/api/v1/admin/platforms/:slug` | 管理员 |
| `GET` | `/api/v1/platforms` | 用户 JWT，公开字段 |

管理端请求字段：

```text
slug, display_name, protocol, base_url, auth_modes, capabilities, color, enabled
```

管理端响应字段：

```text
slug, display_name, protocol, base_url, auth_modes, capabilities, color,
enabled, builtin, created_at, updated_at
```

公开端响应字段相同，但 `base_url` 固定为空，`auth_modes` 固定为空数组。

### 渠道管理

| 方法 | 路径 |
|---|---|
| `GET` | `/api/v1/admin/channels` |
| `GET` | `/api/v1/admin/channels/model-pricing` |
| `GET` | `/api/v1/admin/channels/pricing/sync-models` |
| `GET` | `/api/v1/admin/channels/:id` |
| `POST` | `/api/v1/admin/channels` |
| `PUT` | `/api/v1/admin/channels/:id` |
| `DELETE` | `/api/v1/admin/channels/:id` |

创建字段：

```text
name, description, group_ids, model_pricing, model_mapping,
billing_model_source, restrict_models, features, features_config,
apply_pricing_to_account_stats, account_stats_pricing_rules
```

更新字段：

```text
name, description, status, group_ids, model_pricing, model_mapping,
billing_model_source, restrict_models, features, features_config,
apply_pricing_to_account_stats, account_stats_pricing_rules
```

`model_pricing[]` 字段：

```text
platform, models, billing_mode, input_price, output_price,
cache_write_price, cache_read_price, image_output_price,
per_request_price, intervals
```

`intervals[]` 字段：

```text
min_tokens, max_tokens, tier_label, input_price, output_price,
cache_write_price, cache_read_price, per_request_price, sort_order
```

渠道响应字段：

```text
id, name, description, status, billing_model_source, restrict_models,
features, features_config, group_ids, model_pricing, model_mapping,
apply_pricing_to_account_stats, account_stats_pricing_rules,
created_at, updated_at
```

## 8. 管理后台接口清单

所有接口前缀均为 `/api/v1/admin`。

### 合规和风控

```text
GET /compliance
POST /compliance/accept
GET /risk-control/config
PUT /risk-control/config
POST /risk-control/api-keys/test
GET /risk-control/status
GET /risk-control/logs
POST /risk-control/users/:user_id/unban
DELETE /risk-control/hashes
DELETE /risk-control/hashes/all
PUT /api-keys/:id
```

### 运维 Ops

```text
GET /ops/concurrency
GET /ops/user-concurrency
GET /ops/account-availability
GET /ops/realtime-traffic
GET /ops/alert-rules
POST /ops/alert-rules
PUT /ops/alert-rules/:id
DELETE /ops/alert-rules/:id
GET /ops/alert-events
GET /ops/alert-events/:id
PUT /ops/alert-events/:id/status
POST /ops/alert-silences
GET /ops/email-notification/config
PUT /ops/email-notification/config
GET /ops/runtime/alert
PUT /ops/runtime/alert
GET /ops/runtime/logging
PUT /ops/runtime/logging
POST /ops/runtime/logging/reset
GET /ops/advanced-settings
PUT /ops/advanced-settings
GET /ops/settings/metric-thresholds
PUT /ops/settings/metric-thresholds
GET /ops/ws/qps
GET /ops/errors
GET /ops/errors/:id
PUT /ops/errors/:id/resolve
GET /ops/request-errors
GET /ops/request-errors/:id
GET /ops/request-errors/:id/upstream-errors
PUT /ops/request-errors/:id/resolve
GET /ops/upstream-errors
GET /ops/upstream-errors/:id
PUT /ops/upstream-errors/:id/resolve
GET /ops/requests
GET /ops/system-logs
POST /ops/system-logs/cleanup
GET /ops/system-logs/health
GET /ops/dashboard/snapshot-v2
GET /ops/dashboard/overview
GET /ops/dashboard/throughput-trend
GET /ops/dashboard/latency-histogram
GET /ops/dashboard/error-trend
GET /ops/dashboard/error-distribution
GET /ops/dashboard/openai-token-stats
```

### 仪表盘

```text
GET /dashboard/snapshot-v2
GET /dashboard/stats
GET /dashboard/realtime
GET /dashboard/trend
GET /dashboard/models
GET /dashboard/groups
GET /dashboard/api-keys-trend
GET /dashboard/users-trend
GET /dashboard/users-ranking
POST /dashboard/users-usage
POST /dashboard/api-keys-usage
GET /dashboard/user-breakdown
POST /dashboard/aggregation/backfill
```

### 用户、分组、账号

```text
GET /users
GET /users/:id
POST /users
PUT /users/:id
DELETE /users/:id
POST /users/:id/auth-identities
POST /users/:id/balance
GET /users/:id/api-keys
GET /users/:id/usage
GET /users/:id/balance-history
POST /users/:id/replace-group
GET /users/:id/rpm-status
POST /users/batch-concurrency
GET /users/:id/platform-quotas
PUT /users/:id/platform-quotas
POST /users/:id/platform-quotas/reset
GET /users/:id/attributes
PUT /users/:id/attributes

GET /groups
GET /groups/all
GET /groups/usage-summary
GET /groups/capacity-summary
PUT /groups/sort-order
GET /groups/:id/models-list-candidates
GET /groups/:id
POST /groups
PUT /groups/:id
DELETE /groups/:id
GET /groups/:id/stats
GET /groups/:id/rate-multipliers
PUT /groups/:id/rate-multipliers
DELETE /groups/:id/rate-multipliers
PUT /groups/:id/rpm-overrides
DELETE /groups/:id/rpm-overrides
GET /groups/:id/api-keys

GET /accounts
GET /accounts/:id
POST /accounts
PUT /accounts/:id
DELETE /accounts/:id
POST /accounts/check-mixed-channel
POST /accounts/import/codex-session
POST /accounts/sync/crs
POST /accounts/sync/crs/preview
POST /accounts/batch-test
POST /accounts/:id/test
POST /accounts/:id/recover-state
POST /accounts/:id/refresh
POST /accounts/:id/apply-oauth-credentials
POST /accounts/:id/set-privacy
POST /accounts/:id/refresh-tier
GET /accounts/:id/stats
POST /accounts/:id/clear-error
POST /accounts/:id/revert-proxy-fallback
GET /accounts/:id/usage
GET /accounts/:id/today-stats
POST /accounts/today-stats/batch
POST /accounts/:id/clear-rate-limit
POST /accounts/:id/reset-quota
GET /accounts/:id/temp-unschedulable
DELETE /accounts/:id/temp-unschedulable
POST /accounts/:id/schedulable
POST /accounts/models/sync-upstream-preview
GET /accounts/:id/models
POST /accounts/:id/models/sync-upstream
POST /accounts/batch
GET /accounts/data
POST /accounts/data
POST /accounts/batch-update-credentials
POST /accounts/batch-refresh-tier
POST /accounts/bulk-update
POST /accounts/batch-clear-error
POST /accounts/batch-refresh
GET /accounts/antigravity/default-model-mapping
POST /accounts/generate-auth-url
POST /accounts/generate-setup-token-url
POST /accounts/exchange-code
POST /accounts/exchange-setup-token-code
POST /accounts/cookie-auth
POST /accounts/setup-token-cookie-auth
```

账号创建字段：

```text
name, notes, platform, type, credentials, extra, proxy_id, concurrency,
priority, rate_multiplier, load_factor, group_ids, expires_at,
auto_pause_on_expired, confirm_mixed_channel_risk
```

账号响应核心字段：

```text
id, name, notes, platform, type, status, schedulable, credentials_status,
extra, proxy_id, concurrency, priority, rate_multiplier, load_factor,
group_ids, groups, expires_at, auto_pause_on_expired, privacy_mode,
created_at, updated_at, last_used_at, error_message, rate_limited_until,
quota_reset_at, usage_info
```

敏感凭据不会作为明文普通返回字段对外暴露。

### OAuth 管理、代理、卡密、优惠码

```text
POST /openai/generate-auth-url
POST /openai/exchange-code
POST /openai/refresh-token
POST /openai/accounts/:id/refresh
POST /openai/create-from-oauth

POST /gemini/oauth/auth-url
POST /gemini/oauth/exchange-code
GET /gemini/oauth/capabilities

POST /antigravity/oauth/auth-url
POST /antigravity/oauth/exchange-code
POST /antigravity/oauth/refresh-token

GET /proxies
GET /proxies/all
GET /proxies/data
POST /proxies/data
GET /proxies/:id
POST /proxies
PUT /proxies/:id
DELETE /proxies/:id
POST /proxies/:id/test
POST /proxies/:id/quality-check
GET /proxies/:id/stats
GET /proxies/:id/accounts
POST /proxies/batch-delete
POST /proxies/batch

GET /redeem-codes
GET /redeem-codes/stats
GET /redeem-codes/export
GET /redeem-codes/:id
POST /redeem-codes/create-and-redeem
POST /redeem-codes/generate
DELETE /redeem-codes/:id
POST /redeem-codes/batch-delete
POST /redeem-codes/batch-update
POST /redeem-codes/:id/expire

GET /promo-codes
GET /promo-codes/:id
POST /promo-codes
PUT /promo-codes/:id
DELETE /promo-codes/:id
GET /promo-codes/:id/usages
```

### 系统设置、数据、备份、系统管理

```text
GET /settings
PUT /settings
POST /settings/test-smtp
POST /settings/send-test-email
GET /settings/email-templates
POST /settings/email-template-preview
GET /settings/email-templates/:event/:locale
PUT /settings/email-templates/:event/:locale
POST /settings/email-templates/:event/:locale/restore-official
GET /settings/admin-api-key
POST /settings/admin-api-key/regenerate
DELETE /settings/admin-api-key
GET /settings/overload-cooldown
PUT /settings/overload-cooldown
GET /settings/rate-limit-429-cooldown
PUT /settings/rate-limit-429-cooldown
GET /settings/stream-timeout
PUT /settings/stream-timeout
GET /settings/rectifier
PUT /settings/rectifier
GET /settings/beta-policy
PUT /settings/beta-policy
GET /settings/web-search-emulation
PUT /settings/web-search-emulation
POST /settings/web-search-emulation/test
POST /settings/web-search-emulation/reset-usage

GET /data-management/agent/health
GET /data-management/config
PUT /data-management/config
GET /data-management/sources/:source_type/profiles
POST /data-management/sources/:source_type/profiles
PUT /data-management/sources/:source_type/profiles/:profile_id
DELETE /data-management/sources/:source_type/profiles/:profile_id
POST /data-management/sources/:source_type/profiles/:profile_id/activate
POST /data-management/s3/test
GET /data-management/s3/profiles
POST /data-management/s3/profiles
PUT /data-management/s3/profiles/:profile_id
DELETE /data-management/s3/profiles/:profile_id
POST /data-management/s3/profiles/:profile_id/activate
POST /data-management/backups
GET /data-management/backups
GET /data-management/backups/:job_id

GET /backups/s3-config
PUT /backups/s3-config
POST /backups/s3-config/test
GET /backups/schedule
PUT /backups/schedule
POST /backups
GET /backups
GET /backups/:id
DELETE /backups/:id
GET /backups/:id/download-url
POST /backups/:id/restore

GET /system/version
GET /system/check-updates
POST /system/update
POST /system/rollback
POST /system/restart
```

### 订阅、用量、属性、监控、返利

```text
GET /subscriptions
GET /subscriptions/:id
GET /subscriptions/:id/progress
POST /subscriptions/assign
POST /subscriptions/bulk-assign
POST /subscriptions/:id/extend
POST /subscriptions/:id/reset-quota
DELETE /subscriptions/:id
GET /groups/:id/subscriptions
GET /users/:id/subscriptions

GET /usage
GET /usage/stats
GET /usage/search-users
GET /usage/search-api-keys
GET /usage/cleanup-tasks
POST /usage/cleanup-tasks
POST /usage/cleanup-tasks/:id/cancel

GET /user-attributes
POST /user-attributes
POST /user-attributes/batch
PUT /user-attributes/reorder
PUT /user-attributes/:id
DELETE /user-attributes/:id

POST /scheduled-test-plans
PUT /scheduled-test-plans/:id
DELETE /scheduled-test-plans/:id
GET /scheduled-test-plans/:id/results
GET /accounts/:id/scheduled-test-plans

GET /error-passthrough-rules
GET /error-passthrough-rules/:id
POST /error-passthrough-rules
PUT /error-passthrough-rules/:id
DELETE /error-passthrough-rules/:id

GET /tls-fingerprint-profiles
GET /tls-fingerprint-profiles/:id
POST /tls-fingerprint-profiles
PUT /tls-fingerprint-profiles/:id
DELETE /tls-fingerprint-profiles/:id

GET /channel-monitors
POST /channel-monitors
GET /channel-monitors/:id
PUT /channel-monitors/:id
DELETE /channel-monitors/:id
POST /channel-monitors/:id/run
GET /channel-monitors/:id/history
GET /channel-monitor-templates
POST /channel-monitor-templates
GET /channel-monitor-templates/:id
PUT /channel-monitor-templates/:id
DELETE /channel-monitor-templates/:id
GET /channel-monitor-templates/:id/monitors
POST /channel-monitor-templates/:id/apply

GET /affiliates/invites
GET /affiliates/rebates
GET /affiliates/transfers
GET /affiliates/users
GET /affiliates/users/lookup
POST /affiliates/users/batch-rate
GET /affiliates/users/:user_id/overview
PUT /affiliates/users/:user_id
DELETE /affiliates/users/:user_id
```

## 9. 页面接口

| 方法 | 路径 | 鉴权 | 返回 |
|---|---|---|---|
| `GET` | `/api/v1/pages/:slug` | JWT，且页面可见 | `text/markdown` |
| `GET` | `/api/v1/pages/:slug/images/*filename` | 无 JWT，但页面必须用户可见 | 文件 |
| `GET` | `/api/v1/pages` | 管理员 | `string[]` |

`slug` 只允许字母、数字、下划线、横线，长度不超过 64。

## 10. 核心对象字段全集

### User

```text
id, email, username, role, balance, concurrency, status, allowed_groups,
last_active_at, created_at, updated_at, deleted_at,
balance_notify_enabled, balance_notify_threshold_type,
balance_notify_threshold, balance_notify_extra_emails,
total_recharged, rpm_limit, api_keys, subscriptions
```

AdminUser 额外字段：

```text
notes, last_used_at, group_rates
```

### APIKey

```text
id, user_id, key, name, group_id, status, ip_whitelist, ip_blacklist,
last_used_at, quota, quota_used, expires_at, created_at, updated_at,
rate_limit_5h, rate_limit_1d, rate_limit_7d,
usage_5h, usage_1d, usage_7d,
window_5h_start, window_1d_start, window_7d_start,
reset_5h_at, reset_1d_at, reset_7d_at,
user, group
```

### Group

```text
id, name, platform, rate_multiplier, status, is_exclusive,
subscription_type, daily_limit_usd, weekly_limit_usd, monthly_limit_usd,
image_output_price, image_cache_read_price, image_cache_write_price,
claude_code_enabled, fallback_group_ids, fallback_mode,
allow_messages_dispatch, require_oauth_only, require_privacy_set,
rpm_limit, created_at, updated_at
```

AdminGroup 额外字段：

```text
model_routing, model_routing_enabled, mcp_xml_inject,
default_mapped_model, messages_dispatch_model_config,
models_list_config, supported_model_scopes, account_groups,
account_count, active_account_count, rate_limited_account_count,
sort_order
```

### Account

```text
id, name, notes, platform, type, status, schedulable, credentials_status,
extra, proxy_id, concurrency, priority, rate_multiplier, load_factor,
group_ids, groups, expires_at, auto_pause_on_expired, privacy_mode,
created_at, updated_at, last_used_at, error_message, rate_limited_until,
quota_reset_at, usage_info
```

账号列表带并发状态时额外返回：

```text
current_concurrency, current_window_cost, active_sessions, current_rpm
```

### Proxy

```text
id, name, type, host, port, username, status, created_at, updated_at,
expires_at, fallback_mode, backup_proxy_id, expiry_warn_days
```

带统计代理额外字段：

```text
account_count, latency_ms, latency_status, latency_message, ip_address,
country, country_code, region, city, quality_status, quality_score,
quality_grade, quality_summary, quality_checked
```

管理员代理可额外返回：

```text
password
```

### RedeemCode

```text
id, code, type, value, status, used_by, used_at, created_at, expires_at,
group_id, validity_days, notes, user, group
```

AdminRedeemCode 额外字段：

```text
notes
```

### UsageLog

```text
id, user_id, api_key_id, account_id, request_id, model,
service_tier, reasoning_effort, inbound_endpoint, upstream_endpoint,
group_id, subscription_id,
input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
cache_creation_5m_tokens, cache_creation_1h_tokens,
input_cost, output_cost, cache_creation_cost, cache_read_cost,
total_cost, actual_cost, rate_multiplier,
billing_type, request_type, stream, openai_ws_mode, duration_ms,
first_token_ms,
image_count, image_size, image_input_size, image_output_size,
image_output_tokens, image_output_cost, image_size_source,
image_size_breakdown, video_count, media_type,
user_agent, cache_ttl_overridden, billing_mode,
created_at, user, api_key, group, subscription
```

AdminUsageLog 额外字段：

```text
upstream_model, channel_id, model_mapping_chain, billing_tier,
account_rate_multiplier, account_stats_cost, ip_address, account
```

### UserSubscription

```text
id, user_id, group_id, starts_at, expires_at, status,
daily_window_start, weekly_window_start, monthly_window_start,
daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
created_at, updated_at, user, group
```

AdminUserSubscription 额外字段：

```text
assigned_by, assigned_at, notes, assigned_by_user
```

### Platform

```text
slug, display_name, protocol, base_url, auth_modes, capabilities,
color, enabled, builtin, created_at, updated_at
```

公开平台接口会隐藏：

```text
base_url, auth_modes
```

### Channel

```text
id, name, description, status, billing_model_source, restrict_models,
features, features_config, group_ids, model_pricing, model_mapping,
apply_pricing_to_account_stats, account_stats_pricing_rules,
created_at, updated_at
```

`ChannelModelPricing`：

```text
id, platform, models, billing_mode, input_price, output_price,
cache_write_price, cache_read_price, image_output_price,
per_request_price, intervals
```

`PricingInterval`：

```text
id, min_tokens, max_tokens, tier_label, input_price, output_price,
cache_write_price, cache_read_price, per_request_price, sort_order
```

`AccountStatsPricingRule`：

```text
id, name, group_ids, account_ids, pricing
```

## 11. 字段保留和协议转换原则

同协议直转时，标准字段和白名单 Header 会尽量保留。

跨协议转换时，仅映射当前代码明确支持的字段。无法安全映射的高级字段不会传给上游，以避免请求失败或计费混乱。

OpenAI 兼容上游使用账号配置的 Base URL 和 API Key/OAuth 凭据。自定义平台的能力由 `protocol` 与 `capabilities` 共同参与入口放行和前端展示，但实际路由最终会按平台协议与能力检查执行。

Gemini 视频已支持 Gemini native 入口和 OpenAI 视频入口到 Gemini 兼容上游的适配。Anthropic 不提供视频协议入口。

异步任务不会由站点主动轮询。用户必须调用对应协议的查询入口，例如：

```text
GET /v1/videos/:id
GET /v1/videos/:id/content
GET /v1beta/operations/*operation
```

## 12. 源码核对位置

本文档依据以下源码入口整理：

```text
backend/internal/server/routes/common.go
backend/internal/server/routes/auth.go
backend/internal/server/routes/user.go
backend/internal/server/routes/payment.go
backend/internal/server/routes/admin.go
backend/internal/server/routes/gateway.go
backend/internal/server/routes/ximoai.go
backend/internal/handler/dto/types.go
backend/internal/pkg/apicompat/types.go
backend/internal/handler/available_channel_handler.go
backend/internal/handler/admin/platform_handler.go
backend/internal/handler/admin/channel_handler.go
backend/internal/handler/gateway_handler.go
backend/internal/service/model_entry_protocol.go
backend/internal/service/openai_audio.go
backend/internal/service/openai_videos.go
backend/internal/service/openai_gateway_service.go
```
