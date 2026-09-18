# Docker VPS 快速部署指南（v0.37.1）

> 目标：30 分钟内把 **MicrossAPI v0.37.1** 部署到一台新 VPS，跑通核心功能。

---

## 0. 前置要求

| 项目 | 最低 | 推荐 |
|---|---|---|
| CPU | 1 核 | 2 核 |
| RAM | 2 GB | 4 GB |
| 磁盘 | 20 GB | 40 GB（PostgreSQL 日志会涨） |
| OS | Ubuntu 22.04+ / Debian 11+ | Ubuntu 22.04 LTS |
| 公网 | 1 个 TCP 端口 | 1 个 80 + 1 个 443 |
| 域名 | 可选 | 必须（生产 HTTPS） |

---

## 1. 装 Docker（5 分钟）

```bash
# Ubuntu / Debian 一键
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER  # 当前用户免 sudo
newgrp docker                  # 立即生效

# 验证
docker --version
docker compose version
```

> **防火墙**：记得 `sudo ufw allow 3000/tcp`（或 80/443）。

---

## 2. 选数据库（30 秒决策）

| 数据库 | 推荐场景 | v0.37.1 适配 |
|---|---|---|
| **SQLite**（默认）| 测试 / 小流量 / 个人 demo | ✅ 无需额外容器；数据落到 `./data/one-api.db` |
| **PostgreSQL** | 生产 / 中等流量 / 多节点 | ✅ 默认 docker-compose 用 PG |
| **MySQL** | 已有 MySQL 运维经验 | ✅ 注释解开切换 |

**第一次试**推荐 SQLite（最快验证）；**上生产**推荐 PostgreSQL。

---

## 3. 拉代码 + 改密码（2 分钟）

```bash
git clone https://github.com/dukaworks/micross-api.git
cd micross-api
git checkout v0.37.1    # 切到本版本

# ⚠️ 改密码（生产必改；本文演示用 'CHANGE_ME' 占位）
sed -i 's/123456/CHANGE_ME/g' docker-compose.yml
grep CHANGE_ME docker-compose.yml  # 确认改到
```

---

## 4. 启栈（3 分钟）

### 4A. SQLite 模式（最快）

```bash
# 1. 把 micross-api 服务改成 SQLite
cat > docker-compose.sqlite.yml <<'EOF'
services:
  micross-api:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        VERSION: v0.37.1
    image: dukaworks/micross-api:v0.37.1
    container_name: micross-api
    restart: always
    command: --log-dir /app/logs
    ports:
      - "3000:3000"
    volumes:
      - ./data:/data
      - ./logs:/app/logs
    environment:
      - SQL_DSN=sqlite:///data/one-api.db
      - REDIS_CONN_STRING=redis://:CHANGE_ME@redis:6379
      - TZ=Asia/Shanghai
      - NODE_NAME=micross-api-v0371
    depends_on:
      - redis
    networks:
      - micross-net
    healthcheck:
      test: ["CMD-SHELL", "wget -q -O - http://localhost:3000/api/status | grep -o '\"success\":\\s*true' || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:latest
    container_name: micross-redis
    restart: always
    command: ["redis-server", "--requirepass", "CHANGE_ME"]
    networks:
      - micross-net

networks:
  micross-net:
    driver: bridge
EOF

# 2. 启动
docker compose -f docker-compose.sqlite.yml up -d --build

# 3. 验证
docker compose -f docker-compose.sqlite.yml ps
docker logs micross-api 2>&1 | tail -20
```

### 4B. PostgreSQL 模式（推荐生产）

直接用项目自带的 `docker-compose.yml`：

```bash
docker compose up -d --build
docker compose ps
docker logs micross-api 2>&1 | tail -20
```

---

## 5. 首次启动验证（1 分钟）

### 5.1 服务健康

```bash
# 应该返 success: true
curl -s http://localhost:3000/api/status | jq .

# 关键日志应该看到：
docker logs micross-api 2>&1 | grep -E "(risk scan task started|account_ledger|AutoMigrate|server started)"
```

期望日志：
```
[INFO] commission risk scan task started: tick=6h0m0s batch_size=1000
[INFO] AutoMigrate: created/updated table account_ledger ...
[INFO] AutoMigrate: added columns reversed reversed_at reversed_by reverse_reason on commission_records
[INFO] server started on :3000
```

### 5.2 核心 API 健康

```bash
# 注册一个用户
curl -X POST http://localhost:3000/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"username":"smoke","password":"Smoke123!","email":"smoke@test.local","aff_code":""}'

# 登录拿 token
TOKEN=$(curl -s -X POST http://localhost:3000/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"smoke","password":"Smoke123!"}' | jq -r .data.token)

# 看钱包
curl -s -H "Authorization: $TOKEN" http://localhost:3000/api/user/self | jq .

# 看 commission 余额
curl -s -H "Authorization: $TOKEN" http://localhost:3000/api/user/aff/commission/balance | jq .
```

### 5.3 admin / 风控健康

```bash
# 用 root 账号登录（初始化时第一个注册的用户自动是 root）
ROOT_TOKEN=$(curl -s -X POST http://localhost:3000/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"<your-root>","password":"<pass>"}' | jq -r .data.token)

# 列出 commission_records（应该是空）
curl -s -H "Authorization: $ROOT_TOKEN" \
  http://localhost:3000/api/admin/commission/records | jq .

# 看日志确认 cron 没异常
docker logs micross-api 2>&1 | grep "risk scan done"
# 首次跑应该看到: scanned=0 reversed=0 duration_ms=...
```

---

## 6. v0.37.1 schema 变更确认（1 分钟）

```bash
# 进入容器
docker exec -it micross-api sh

# 看 schema（SQLite）
sqlite3 /data/one-api.db ".schema commission_records" | grep -E "reversed|reverse_"
# 应该看到:
# reversed bool DEFAULT 'false',
# reversed_at bigint DEFAULT '0',
# reversed_by integer DEFAULT '0',
# reverse_reason varchar(255) DEFAULT '',

# 看 schema（PostgreSQL）
docker exec -it micross-api-postgres psql -U root -d "micross-api" -c "\d commission_records" | grep -E "reversed|reverse_"
# 应该看到 4 个新列

# 看 account_ledger 表（v0.37.0 已建好，§20.1-§20.3 只写数据）
sqlite3 /data/one-api.db "SELECT event_type, COUNT(*) FROM account_ledger GROUP BY event_type;"
# 应该有 topup / refund / agent_quota_grant / commission / commission_reverse 几种 event_type
```

---

## 7. 升级到 v0.37.1（已跑 v0.37.0 时）

```bash
cd micross-api
git pull origin main          # 或 git fetch && git checkout v0.37.1
docker compose down
docker compose build --pull   # 重拉基础镜像
docker compose up -d
```

**回滚**：
```bash
git checkout v0.37.0
docker compose down && docker compose up -d --build
```
数据库**不需要回退**——v0.37.1 加列是前向兼容的。

---

## 8. 反向代理（Nginx 前置，HTTPS）

```nginx
# /etc/nginx/sites-available/micross-api
server {
    listen 80;
    server_name api.example.com;  # 改成你的域名

    client_max_body_size 100M;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 流模式超时
        proxy_http_version 1.1;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/micross-api /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# HTTPS（Let's Encrypt）
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d api.example.com
```

---

## 9. 监控脚本（5 条 SQL，建议每天跑）

```bash
# 加到 crontab: 0 9 * * * /opt/micross-monitor.sh
cat > /opt/micross-monitor.sh <<'BASH'
#!/bin/bash
cd /opt/micross-api  # 或你 clone 的路径

# 1. 风控命中率（健康 < 5%）
echo "[breach_pct]"
sqlite3 data/one-api.db "
SELECT
  DATE(settled_at, 'unixepoch') AS day,
  ROUND(100.0 * SUM(breach) / COUNT(*), 2) AS pct
FROM commission_records
WHERE settled_at > strftime('%s', 'now', '-1 day')
GROUP BY day;"

# 2. 自动撤销量（健康 < 100/天）
echo "[system_reversed]"
sqlite3 data/one-api.db "
SELECT DATE(reversed_at, 'unixepoch') AS day, COUNT(*)
FROM commission_records
WHERE reversed = 1 AND reversed_by = 0
  AND reversed_at > strftime('%s', 'now', '-1 day')
GROUP BY day;"

# 3. ledger 写入健康
echo "[ledger_events]"
sqlite3 data/one-api.db "
SELECT event_type, COUNT(*)
FROM account_ledger
WHERE created_at > strftime('%s', 'now', '-1 day')
GROUP BY event_type;"

# 4. cron 是否运行
echo "[risk_scan_last]"
docker logs micross-api 2>&1 | grep "risk scan done" | tail -1

# 5. 注册拦截触发数
echo "[self_invite_blocked]"
sqlite3 data/one-api.db "
SELECT DATE(created_at, 'unixepoch') AS day, COUNT(*)
FROM logs
WHERE content LIKE '%inviter%'
  AND created_at > strftime('%s', 'now', '-1 day')
GROUP BY day;"
BASH
chmod +x /opt/micross-monitor.sh
```

---

## 10. 常见问题

### Q1: 启动失败 `bind: address already in use`

```bash
sudo lsof -i :3000     # 查谁占着
# 或改 docker-compose.yml 端口映射：3001:3000
```

### Q2: `risk scan task started` 没出现

```bash
# 看完整启动日志
docker logs micross-api 2>&1 | head -50
# 可能是 main.go:130 没生效——git checkout v0.37.1 后重新 build
docker compose build --no-cache
```

### Q3: 第一次注册的用户不是 root

**第一个注册的用户自动是 root**。第二个开始是 common_user。
如需提升某用户为 admin：
```bash
sqlite3 data/one-api.db "UPDATE users SET role = 10 WHERE username = '<name>';"
```

### Q4: AutoMigrate 失败

```bash
docker logs micross-api 2>&1 | grep -i "migration\|auto migrate"
# 看具体错误。常见原因：
# - 数据库磁盘满：df -h
# - PostgreSQL 权限不够：docker exec -it micross-api-postgres psql -U root
```

### Q5: 反向代理后 `client_max_body_size` 不够

```nginx
client_max_body_size 100M;  # 改大
```

### Q6: 忘记了 root 密码

```bash
# 直接改数据库（SQLite）
docker exec -it micross-api sqlite3 /data/one-api.db
sqlite> UPDATE users SET password = '<new-hash>' WHERE username = 'root';
# 用 bcrypt hash（命令行 hash 一致即可）
```

---

## 11. 进阶：CI/CD 自动镜像

CI 已在 GitHub Actions 配好（项目根 `.github/workflows/`），每次打 tag 自动 publish `dukaworks/micross-api:v0.X.Y`。

部署机拉新版只需：
```bash
docker pull dukaworks/micross-api:v0.37.1
docker compose down && docker compose up -d
```

---

## 12. 完成 checklist

- [ ] 服务起得来（`docker compose ps` 全 healthy）
- [ ] `/api/status` 返 success
- [ ] root 用户能登录
- [ ] `commission_records` 表 4 个新列存在
- [ ] `account_ledger` 表可写可读
- [ ] 启动日志含 `risk scan task started`
- [ ] 至少 1 次 `risk scan done` 日志
- [ ] 备份策略：每日 `data/` + 数据库 dump
- [ ] HTTPS 已配（生产）
- [ ] 监控脚本已设 cron

---

**完成即用**。遇到问题看 [GitHub Issues](https://github.com/dukaworks/micross-api/issues) 或提交新 issue。