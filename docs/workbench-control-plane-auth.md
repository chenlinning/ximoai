# Workbench Model Control-Plane Authorization

## Boundary

The existing one-time SSO ticket establishes the Workbench user identity. The
server-to-server ticket validation response also carries a short-lived,
user-bound Workbench authorization grant. It must not carry model catalogs,
API keys, protocol decisions, inference settings, or `modelConfig`.

Workbench uses the access token only with these existing main-site endpoints:

- `GET /api/v1/keys`
- `GET /api/v1/groups/available`
- `GET /api/v1/groups/rates`
- `GET /api/v1/platforms`
- `GET /api/v1/channels/model-plaza`

No `/api/v1/workbench/model-access` or `/v1/workbench/catalog/*` compatibility
routes are retained.

## Token Lifecycle

- Access token: main-site JWT, `aud=workbench`, `token_use=workbench_control`,
  exact read-only scope, five-minute TTL.
- Refresh token: 32 random bytes, only its SHA-256 digest is stored in Redis,
  24-hour fixed session lifetime, atomically consumed and rotated.
- Renewal and revocation require the existing Workbench server secret and are
  server-to-server only.
- Password/security token-version changes or an inactive user immediately
  reject access and refresh tokens. Explicit revocation removes the refresh
  grant; an already issued access token expires within five minutes.

The Workbench backend keeps both credentials in its server-side session. They
must never be sent to the iframe URL, browser JavaScript, local storage, logs,
or error responses.

## Server-to-Server Contract

`POST /api/v1/workbench/sso-ticket/validate` keeps the existing flat identity
fields and adds only this top-level field:

```json
{
  "authorization": {
    "accessToken": "...",
    "refreshToken": "...",
    "tokenType": "Bearer",
    "expiresIn": 300,
    "refreshExpiresIn": 86400,
    "audience": "workbench",
    "scopes": ["workbench:model-control:read"]
  }
}
```

The Workbench server sends `accessToken` as the Bearer token to the five native
GET endpoints. It rotates the refresh token with
`POST /api/v1/workbench/control-token/refresh` and revokes it with
`POST /api/v1/workbench/control-token/revoke`. Both lifecycle endpoints require
the existing Workbench internal Bearer secret and a JSON body containing only
`{"refreshToken":"..."}`. A successful revoke returns HTTP 204.

## Model Calls

Catalog reads do not proxy generation requests. Workbench selects a user API
key returned by the existing key endpoint and calls the gateway with the
selected native protocol. The main site does not infer protocols or reasoning
settings from model names and does not transform payloads, responses, or SSE
for this control-plane integration.
