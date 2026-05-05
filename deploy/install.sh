#!/bin/bash
set -e

# =============================================================================
# XimoAi 一键部署脚本 (适用于低内存阿里云服务器)
# 原理: PostgreSQL + Redis 用 Docker 运行, XimoAi 直接运行预编译二进制
# =============================================================================

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}   XimoAi 一键部署脚本${NC}"
echo -e "${GREEN}========================================${NC}"

# 配置
INSTALL_DIR="/opt/ximoai"
CONFIG_FILE="$INSTALL_DIR/config.yaml"
BINARY_URL="https://github.com/chenlinning/ximoai/releases/download/v1.0.0/sub2api-linux-amd64"

# 检查是否 root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}请使用 root 用户运行此脚本${NC}"
  exit 1
fi

# --------------- 1. 安装 Docker ---------------
echo -e "${YELLOW}[1/7] 检查 Docker...${NC}"
if ! command -v docker &> /dev/null; then
  echo "安装 Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl enable docker && systemctl start docker
  echo -e "${GREEN}Docker 安装完成${NC}"
else
  echo -e "${GREEN}Docker 已安装${NC}"
fi

# --------------- 2. 创建 Swap (小内存服务器必须) ---------------
echo -e "${YELLOW}[2/7] 检查 Swap...${NC}"
if [ "$(swapon --show | wc -l)" -le 1 ]; then
  echo "创建 2GB Swap..."
  fallocate -l 2G /swapfile
  chmod 600 /swapfile
  mkswap /swapfile
  swapon /swapfile
  echo '/swapfile none swap sw 0 0' >> /etc/fstab
  echo -e "${GREEN}Swap 创建完成${NC}"
else
  echo -e "${GREEN}Swap 已存在${NC}"
fi

# --------------- 3. 生成配置 ---------------
echo -e "${YELLOW}[3/7] 生成配置文件...${NC}"
mkdir -p $INSTALL_DIR

# 生成随机密码
PG_PASSWORD=$(openssl rand -hex 16)
ADMIN_PASSWORD=$(openssl rand -hex 12)
JWT_SECRET=$(openssl rand -hex 32)

cat > $INSTALL_DIR/.env << EOF
POSTGRES_USER=ximoai
POSTGRES_PASSWORD=$PG_PASSWORD
POSTGRES_DB=ximoai
REDIS_PASSWORD=
EOF

cat > $CONFIG_FILE << EOF
server:
  host: 0.0.0.0
  port: 8080
  mode: release

database:
  host: 127.0.0.1
  port: 5432
  user: ximoai
  password: "$PG_PASSWORD"
  dbname: ximoai
  sslmode: disable

redis:
  host: 127.0.0.1
  port: 6379
  password: ""
  db: 0

jwt:
  secret: "$JWT_SECRET"
  expire_hour: 24

timezone: Asia/Shanghai

auto_setup:
  enabled: true
  admin_email: admin@ximoai.local
  admin_password: "$ADMIN_PASSWORD"
EOF

echo -e "${GREEN}配置文件已生成${NC}"

# --------------- 4. 下载二进制 ---------------
echo -e "${YELLOW}[4/7] 下载 XimoAi 二进制文件 (约 83MB)...${NC}"
curl -L -o $INSTALL_DIR/sub2api $BINARY_URL
chmod +x $INSTALL_DIR/sub2api
echo -e "${GREEN}下载完成${NC}"

# --------------- 5. 启动 PostgreSQL + Redis ---------------
echo -e "${YELLOW}[5/7] 启动 PostgreSQL + Redis (Docker)...${NC}"

# 停止旧的
docker compose -f $INSTALL_DIR/docker-compose.lite.yml down 2>/dev/null || true

# 复制 docker-compose 文件
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cp "$SCRIPT_DIR/docker-compose.lite.yml" $INSTALL_DIR/

# 启动
cd $INSTALL_DIR && docker compose -f docker-compose.lite.yml up -d

# 等待数据库就绪
echo "等待数据库启动..."
sleep 10

# --------------- 6. 启动 XimoAi ---------------
echo -e "${YELLOW}[6/7] 启动 XimoAi...${NC}"

# 创建 systemd 服务
cat > /etc/systemd/system/ximoai.service << EOF
[Unit]
Description=XimoAi Service
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/sub2api
Restart=always
RestartSec=5
EnvironmentFile=$INSTALL_DIR/.env

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable ximoai
systemctl restart ximoai

sleep 3

# --------------- 7. 完成 ---------------
echo -e "${YELLOW}[7/7] 检查服务状态...${NC}"
if systemctl is-active --quiet ximoai; then
  echo ""
  echo -e "${GREEN}========================================${NC}"
  echo -e "${GREEN}   部署成功!${NC}"
  echo -e "${GREEN}========================================${NC}"
  echo ""
  echo -e "  访问地址: ${GREEN}http://$(hostname -I | awk '{print $1}'):8080${NC}"
  echo -e "  管理员邮箱: ${GREEN}admin@ximoai.local${NC}"
  echo -e "  管理员密码: ${GREEN}$ADMIN_PASSWORD${NC}"
  echo ""
  echo -e "  配置文件: $INSTALL_DIR/config.yaml"
  echo -e "  服务管理: systemctl start/stop/restart ximoai"
  echo -e "  查看日志: journalctl -u ximoai -f"
  echo ""
  echo -e "${RED}请记好上面的管理员密码!${NC}"
  echo -e "${GREEN}========================================${NC}"
else
  echo -e "${RED}服务启动失败，查看日志:${NC}"
  journalctl -u ximoai -n 20
fi
