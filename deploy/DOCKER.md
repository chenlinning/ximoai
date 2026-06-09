# Docker Deployment

This repository builds its own runtime image for the Sub2API service:

- `sub2api`

Do not deploy `weishaw/sub2api:latest` when using repository-local custom code, because that image will not contain the XimoAI customizations.

## Build And Start

```bash
cd /opt/ximoai-src
git pull

cd deploy
docker compose -f docker-compose.local.yml build sub2api
docker compose -f docker-compose.local.yml up -d sub2api
```

## Ports

- Sub2API web: `8080/tcp`
