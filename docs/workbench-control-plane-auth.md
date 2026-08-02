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
- Ticket validation, renewal, and revocation require an audience-specific
  server credential and are server-to-server only.
- Password/security token-version changes or an inactive user immediately
  reject access and refresh tokens. Explicit revocation removes the refresh
  grant; an already issued access token expires within five minutes.

The Workbench backend keeps both credentials in its server-side session. They
must never be sent to the iframe URL, browser JavaScript, local storage, logs,
or error responses.

## Server-to-Server Contract

Each enabled home tab with `workbench_sso=true` is an independent SSO audience.
The audience is the tab URL origin (`scheme://host[:port]`), and its credential
is Base64URL-without-padding encoded
`HMAC-SHA256(WORKBENCH_SSO_INTERNAL_SECRET, "ximoai-workbench-sso-audience:" + audience)`.
The master secret remains on the main site. Each child site receives only its
derived credential, so it cannot authenticate as another audience.

`POST /api/v1/workbench/sso-ticket/validate` sends that derived credential as
the Bearer token. Its request body contains the one-time ticket and the exact
audience origin. The response keeps the existing flat identity fields and adds
only this top-level field:

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

The child server sends `accessToken` as the Bearer token to the five native GET
endpoints. It rotates the refresh token with
`POST /api/v1/workbench/control-token/refresh` and revokes it with
`POST /api/v1/workbench/control-token/revoke`. Both lifecycle endpoints require
the same audience-specific Bearer credential and this JSON body:

```json
{
  "refreshToken": "..."
}
```

Refresh grants are bound to that audience in Redis. A different child cannot
refresh or revoke the grant. A successful revoke returns HTTP 204.

## Desktop SSO Broker Credential

The desktop shell does not receive an audience-derived server credential. It
authenticates its existing device-bound Desktop Session and requests a short-
lived broker credential instead:

```http
POST /api/v1/desktop/sso-broker-credential
Authorization: DPoP <desktop_access_token>
DPoP: <device proof>
Content-Type: application/json

{"workbench_id":"image"}
```

The server resolves the configured Workbench audience from `workbench_id` and
returns a five-minute Bearer credential bound to the user, parent Desktop
Session, device key thumbprint, and Workbench ID. The client cannot submit an
audience or URL.

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "credential": "...",
    "token_type": "Bearer",
    "expires_in": 300,
    "workbench_id": "image",
    "audience": "https://image.ximoai.cn"
  }
}
```

The existing ticket validation, control-token refresh, and control-token
revocation endpoints accept either their existing audience-derived Bearer
credential or this broker credential. Broker-authorized operations revalidate
the parent Desktop Session and current Workbench permission, and may consume
only tickets or refresh grants belonging to the same user. Revoking the parent
Desktop Session immediately invalidates outstanding broker credentials.

## Model Calls

Catalog reads do not proxy generation requests. Workbench selects a user API
key returned by the existing key endpoint and calls the gateway with the
selected native protocol. The main site does not infer protocols or reasoning
settings from model names and does not transform payloads, responses, or SSE
for this control-plane integration.
