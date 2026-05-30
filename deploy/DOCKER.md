# Docker Deployment

This repository builds its own runtime image for both containers:

- `sub2api`
- `direct-assist-signal`

Do not deploy `weishaw/sub2api:latest` when using repository-local custom code, because that image will not contain the DirectAssist signal binary.

## Build And Start

```bash
cd /opt/ximoai-src
git pull

cd deploy
docker compose -f docker-compose.local.yml build sub2api direct-assist-signal
docker compose -f docker-compose.local.yml up -d sub2api direct-assist-signal
```

## Ports

- Sub2API web: `8080/tcp`
- DirectAssist HTTP signal: `47880/tcp`
- DirectAssist UDP probe: `47822/udp`

If HTTP signal is proxied through Nginx or Caddy, expose the web ports publicly and proxy `/api/direct-assist/signal/v1/*` to `127.0.0.1:47880`. UDP probe still needs `47822/udp`; ordinary HTTP reverse proxy cannot proxy UDP probe.

See `docs/direct-assist-signal.md` for API, smoke testing, and cloud security group details.
