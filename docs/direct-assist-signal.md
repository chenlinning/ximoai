# DirectAssist Signal Service

DirectAssist signal 是独立进程和独立容器，只负责设备发现、连接信令、P2P 协商、UDP 公网地址探测和 P2P 失败原因记录。

它不是内容中继：不转发视频、音频、文件、剪贴板、键盘、鼠标数据；不做 hbbr、TURN、媒体 relay；也没有预留内容中继接口。

## 服务端口

- HTTP signal: `DIRECT_ASSIST_SIGNAL_ADDR=0.0.0.0:47880`
- UDP probe: `DIRECT_ASSIST_UDP_PROBE_ADDR=0.0.0.0:47822`

HTTP API 默认路径：

```text
/api/direct-assist/signal/v1/*
```

UDP probe 只接收 `udp_probe` JSON，校验 `deviceId`、`sessionId`、`token`，然后返回服务端观察到的来源公网 IP/端口。UDP probe 不保存、不转发任何远控内容。

## 设备签名

设备码不是密钥。设备首次 heartbeat 需要提交 `deviceId`、`deviceSecret` 和 `deviceSignature`，服务端只保存短期 `deviceSecret` hash，不返回 secret/signature。

签名算法：

```text
hex(HMAC-SHA256(deviceSecret, message))
```

支持把 `deviceSecret` 放在 JSON 字段 `deviceSecret`，或 HTTP Header `X-DirectAssist-Device-Secret`。支持把签名放在 JSON 字段 `deviceSignature`，或 HTTP Header `X-DirectAssist-Signature`。

当前 HTTP 操作的 `message`：

| 操作 | message |
|------|---------|
| heartbeat | `{deviceId}:{deviceCode}` |
| 创建 session | `{controllerDeviceId}:create_session` |
| long polling events | `{deviceId}:events` |
| answer | `{deviceId}:answer:{sessionId}` |
| candidates POST/GET | `{deviceId}:candidates:{sessionId}` |
| close | `{deviceId}:close:{sessionId}` |
| failure | `{deviceId}:failure:{sessionId}` |

UDP probe 使用 heartbeat 返回的短期 `udpProbeToken`，并校验 `sessionId` 中的参与设备身份。

## 云端部署

项目的 compose 已改为从当前仓库源码构建镜像，并用同一个镜像启动两个独立容器：

- `sub2api`: 网站主服务
- `direct-assist-signal`: DirectAssist signal/UDP probe 服务

推荐在服务器执行：

```bash
cd /opt/ximoai-src
git pull

cd deploy
docker compose -f docker-compose.local.yml build sub2api direct-assist-signal
docker compose -f docker-compose.local.yml up -d sub2api direct-assist-signal
docker compose -f docker-compose.local.yml ps
```

如果使用 named volume 版本：

```bash
cd /opt/ximoai-src
git pull

cd deploy
docker compose build sub2api direct-assist-signal
docker compose up -d sub2api direct-assist-signal
docker compose ps
```

这些命令不会删除 `data/`、`postgres_data/`、`redis_data/` 或 Docker named volumes，用户数据会保留。

## 阿里云安全组

如果 HTTP signal 直接暴露端口：

- 开放 `47880/tcp`
- 开放 `47822/udp`

如果 HTTP signal 通过 Nginx/Caddy 反代到域名：

- 公网只需开放 `80/tcp`、`443/tcp`
- 仍必须开放 `47822/udp`

UDP probe 不能使用普通 HTTP 反代。需要直接开放 UDP 端口，或使用支持 UDP 的 stream 代理。

## Nginx HTTP 反代示例

```nginx
location /api/direct-assist/signal/v1/ {
    proxy_pass http://127.0.0.1:47880;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 65s;
}
```

如果 HTTP signal 容器只在 Docker 网络内暴露，Nginx 也可以反代到 Docker host 映射端口 `127.0.0.1:47880`。

## Caddy HTTP 反代示例

```caddyfile
example.com {
    reverse_proxy /api/direct-assist/signal/v1/* 127.0.0.1:47880
}
```

## 本地 smoke 验收

后端测试已覆盖完整 smoke 流程：

1. signal health 正常。
2. 设备 A heartbeat。
3. 设备 B 查询 A 的 deviceCode。
4. B 创建 session。
5. A long polling 收到 session 请求。
6. A accepted。
7. A/B 提交 candidates。
8. A/B 拉取 candidates。
9. A/B 分别执行 UDP probe，拿到 publicIp/publicPort。
10. 上报一次 `udp_punch_failed`。
11. TTL 到期后设备自动离线。

运行：

```bash
cd backend
GOPROXY=https://goproxy.cn,direct go test ./internal/directassist ./cmd/direct-assist-signal
```

主站和 signal 服务是不同进程。停止 `direct-assist-signal` 后，网站主服务仍应正常：

```bash
cd deploy
docker compose -f docker-compose.local.yml stop direct-assist-signal
curl -fsS http://127.0.0.1:8080/health
docker compose -f docker-compose.local.yml start direct-assist-signal
```

## 云端 smoke 验收

HTTP signal：

```bash
curl -fsS https://你的域名/api/direct-assist/signal/v1/health
```

如果直接暴露端口：

```bash
curl -fsS http://服务器IP:47880/api/direct-assist/signal/v1/health
```

UDP probe 需要客户端先完成 heartbeat 并拿到短期 `udpProbeToken`，再对 `服务器IP:47822/udp` 发送 `udp_probe` JSON。返回中应包含服务端观察到的 `publicIp` 和 `publicPort`。

P2P 仍可能因为 NAT、运营商网络或防火墙失败。DirectAssist signal 不保证一定打洞成功，但会记录并投递 `offline`、`timeout`、`rejected`、`verify_failed`、`tcp_failed`、`udp_probe_failed`、`udp_punch_failed`、`nat_failed`、`firewall_blocked`、`unknown` 等失败原因。
