#!/bin/bash
# Sub2API Sandbox Startup Script
set -e

export PATH=$PATH:/usr/local/go/bin

# Ensure Redis is running
if ! redis-cli ping > /dev/null 2>&1; then
    echo "Starting Redis..."
    redis-server --daemonize yes --port 6379
    sleep 1
fi

# Set data directory
export DATA_DIR=/workspace/projects/sub2api-src/data
mkdir -p "$DATA_DIR"

# Auto-setup environment variables (only used on first run when no config.yaml exists)
export AUTO_SETUP=true
export DATABASE_HOST=cp-fleet-ray-b2a01e96.pg4.aidap-global.cn-beijing.volces.com
export DATABASE_PORT=5432
export DATABASE_USER=postgres
export DATABASE_PASSWORD=Gukv079w7Y1x9WTdra
export DATABASE_DBNAME=sub2api
export DATABASE_SSLMODE=require
export REDIS_HOST=localhost
export REDIS_PORT=6379
export REDIS_PASSWORD=
export REDIS_DB=0
export SERVER_HOST=0.0.0.0
export SERVER_PORT=5000
export SERVER_MODE=release
export ADMIN_EMAIL=admin@sub2api.local
export ADMIN_PASSWORD=admin123456
export JWT_SECRET=sub2api-sandbox-jwt-secret-2026-fixed
export JWT_EXPIRE_HOUR=24
export TZ=Asia/Shanghai

echo "Starting Sub2API..."
echo "  Data Dir: $DATA_DIR"
echo "  Server: 0.0.0.0:5000"
echo "  Database: $DATABASE_HOST:$DATABASE_PORT/$DATABASE_DBNAME"
echo "  Redis: $REDIS_HOST:$REDIS_PORT"

cd /workspace/projects/sub2api-src
exec /workspace/projects/sub2api-src/bin/sub2api
