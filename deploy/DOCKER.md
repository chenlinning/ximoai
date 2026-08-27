# Docker Deployment

Production deployments use the image built by this repository's GitHub Actions workflow. Do not build the application on the server, and do not deploy `weishaw/sub2api:latest`, because it does not contain the XimoAI customizations.

## Deploy A Verified Commit

1. Confirm `Security Scan`, `CI`, and `Docker Image` all passed for the target commit.
2. Set the immutable image tag in `deploy/.env`:

```dotenv
SUB2API_IMAGE=ilemon00/sub2api:sha-<12-char-commit>
```

3. Pull the image and recreate only the application container:

```bash
cd /opt/ximoai-src/deploy
docker compose -f docker-compose.local.yml pull sub2api
docker compose -f docker-compose.local.yml up -d --no-deps --no-build sub2api
docker compose -f docker-compose.local.yml ps sub2api
```

PostgreSQL, Redis, and persistent data directories are not recreated by these commands.

## Local Development

Use `docker-compose.dev.yml` when a local source build is intentionally required.

## Ports

- Sub2API web: `8080/tcp`
