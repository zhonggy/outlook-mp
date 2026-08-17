# Outlook Manager

Hotmail/Outlook 账号自动化管理平台：长期运行于服务器，负责账号的
**集中管理、Token 自动刷新、健康检测、保活、收信与导出**。

- 后端：Go（Gin + GORM/SQLite + robfig/cron），`go:embed` 内嵌前端，单二进制部署
- 前端：Vue 3 + TypeScript + 自研 UI 组件 + Pinia + Vite（浅色/深色主题切换）
- 支持外部自动化系统经 API 上传账号，入库即纳入调度

## 功能

| 模块 | 说明 |
|------|------|
| 账号管理 | CRUD、分组/标签、搜索筛选、批量删除、一键清理失效、导入（文本/JSON/CSV）、导出（格式/状态/数量可选） |
| Token 刷新 | refresh_token 定时换新（含轮换回写），invalid_grant 自动标记失效 |
| 健康检测 | 刷新 + Graph `/me` 真实探测，状态机 unknown/healthy/dead/locked/error；支持一键全量检测 |
| 保活机制 | 定时调 Graph `/me` + 读一封信，维持账号活跃 |
| 收信机制 | 收件箱 + 垃圾邮件文件夹双拉取、合并展示，自动提取验证码 |
| 定时调度 | 各任务周期在线下拉可调（settings 覆盖 config），账号间错峰防风控 |
| 日志清理 | 保留期可配（默认 30 天），调度器每小时自动清理，支持手动清理过期/清空 |
| 主题切换 | 浅色/深色一键切换，跟随系统 + 本地记忆 |
| 自动化对接 | API Key 认证的上传接口，外部系统可直接推送账号入库 |
| 安全 | JWT 登录认证、bcrypt 密码、API Key 独立鉴权 |

## 快速开始

### 服务器部署（单二进制）

```bash
# 构建（需要 Go 1.25+ 与 Node 20+）
make build          # 前端 build + 后端打包，产物在 bin/outlook-manager

# 运行（首次启动自动生成 configs/config.yaml 与 data/）
./bin/outlook-manager
# 默认账号 admin，密码首次启动随机生成（控制台打印一次 + data/initial_admin_password.txt 备份）
# 配置项见 configs/config.example.yaml
```

浏览器访问 `http://服务器IP:18327`。

### 一键部署（Linux 服务器）

`deploy/deploy.sh` 自动完成：拉取源码 → 安装 Go/Node → 构建 → 配置 systemd 开机自启 → 启动服务。

```bash
# 方式一：源码在服务器上（已上传或 git clone）
sudo bash deploy/deploy.sh

# 方式二：从 git 仓库直接拉取源码
sudo bash deploy/deploy.sh --git https://github.com/zhonggy/outlook-mp.git

# 服务器无本地代理时加 --no-proxy
sudo bash deploy/deploy.sh --no-proxy
```

#### 参数说明

| 参数 | 说明 |
|------|------|
| `--git <URL>` | 从 git 仓库克隆/拉取源码后再部署 |
| `--dir /path` | 指定源码目录（默认 `/opt/outlook-manager`） |
| `--no-proxy` | 关闭 config 中的全局代理（服务器无代理时用） |
| `--skip-deps` | 跳过 Go/Node 安装（已装好时加速） |

#### 脚本特性

- 自动检测 Go ≥ 1.25 / Node ≥ 20，缺失才安装
- 幂等可重跑：更新代码后再次执行即可重新构建并重启服务
- systemd 服务 `Restart=always`，进程异常自动拉起
- 部署完成后输出访问地址与管理员初始密码

#### 常用命令

```bash
systemctl status outlook-manager       # 查看状态
journalctl -u outlook-manager -f       # 跟踪日志
systemctl restart outlook-manager      # 重启
```

### Docker 部署

```bash
# 构建并启动
docker compose up -d --build

# 查看日志
docker logs -f outlook-manager

# 重启
docker compose restart
```

首次启动时创建 `configs/` 和 `data/` 目录，参考 `configs/config.example.yaml` 配置。

### 开发模式

```bash
# 后端（18327）
make run

# 前端（5173，代理到 18327）
cd web && npm install && npm run dev
```

## 配置（configs/config.yaml）

```yaml
server:
  host: 0.0.0.0
  port: 18327
database:
  path: data/outlook-manager.db
auth:
  jwt_secret: ""            # 留空自动生成并持久化
  token_ttl_hours: 72
  admin_username: admin
  admin_password: ""       # 留空则首次启动随机生成；仅首次启动生效
scheduler:
  enabled: true
  refresh_interval: 12h     # token 刷新周期
  health_interval: 6h       # 健康检测周期
  keepalive_interval: 24h   # 保活周期
  mail_interval: 30m        # 收信周期
proxy:
  enabled: false
  url: ""                   # 全局代理，如 http://127.0.0.1:7897
microsoft:
  token_url: https://login.microsoftonline.com/common/oauth2/v2.0/token
  graph_url: https://graph.microsoft.com/v1.0
  scope: https://graph.microsoft.com/.default offline_access
```

环境变量覆盖：`OM_PORT` `OM_DB_PATH` `OM_JWT_SECRET` `OM_ADMIN_PASSWORD` `OM_PROXY` `OM_CONFIG`。

## 对接自动化上传

1. 管理平台「系统设置 → API 密钥」创建密钥（形如 `omk_xxx`）
2. 上传端以 `X-API-Key` 头调用 `POST /api/v1/ingest/accounts`，Body 为账号 JSON 数组：

```bash
curl -X POST http://your-server:18327/api/v1/ingest/accounts \
  -H "X-API-Key: omk_xxx" \
  -H "Content-Type: application/json" \
  -d '[{"email":"a@hotmail.com","password":"密码","client_id":"xxx","refresh_token":"xxx"}]'
```

按 email 幂等 upsert：已存在则更新凭据，不存在则新建。字段说明与示例在
「系统设置 → 上传接口说明」里也有，可直接复制。

也可手动导入存量账号：管理平台「账号管理 → 导入」直接粘贴
`email----密码----client_id----refresh_token`（每行一个）等格式的文本。

## API 概览（/api/v1）

```
POST /auth/login                     登录
GET  /accounts                       列表（keyword/status/tag/group 筛选 + 分页）
POST /accounts/import                导入（文本/JSON）
GET  /accounts/export?format=txt     导出（format/status/limit 可选）
POST /accounts/batch-delete          批量删除
POST /accounts/delete-by-status      按状态清理（如全部失效账号）
POST /accounts/check-all             一键全量健康检测
POST /accounts/refresh-all           一键全量刷新
POST /accounts/:id/refresh           单账号刷新 token
POST /accounts/:id/check             单账号健康检测
GET  /accounts/:id/mails             邮件列表（收件箱 + 垃圾邮件）
POST /accounts/:id/mails/fetch       在线收信
GET  /mails/:id                      邮件全文（含验证码提取）
GET  /tasks/logs                     任务日志
POST /tasks/logs/cleanup             日志清理（{"all":true} 清空）
GET|PUT /tasks/schedule              调度配置（周期/日志保留期）
GET  /stats/dashboard                仪表盘统计
GET|POST|DELETE /apikeys             API 密钥管理
POST /ingest/accounts                自动化上传（X-API-Key 认证）
```

## 项目结构

```
outlook-manager/
├── cmd/server/          # 入口（main + 静态资源挂载）
├── deploy/              # 一键部署脚本（deploy.sh）
├── Dockerfile           # Docker 构建
├── docker-compose.yml   # Docker Compose
├── internal/
│   ├── config/          # 配置加载（yaml + env 覆盖）
│   ├── model/           # GORM 模型与状态常量
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务层（账号/token/健康/保活/收信）
│   ├── scheduler/       # 定时调度（settings 可热改周期）
│   ├── msgraph/         # 微软 OAuth + Graph 客户端
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # JWT / API Key / CORS
│   ├── router/          # 路由注册
│   └── pkg/             # 工具（JWT/bcrypt/验证码提取）
├── web/                 # Vue3 前端（dist 由 go:embed 内嵌）
├── configs/             # 配置文件
├── data/                # SQLite 与日志（运行时生成）
└── Makefile
```

## 状态机说明

```
unknown ──刷新成功──► healthy ──invalid_grant──► dead（需重新登录取 token）
   │                    │
   └──网络故障──► error（下轮自动重试）
```

`dead` 账号不再参与保活/收信；重新上传有效 refresh_token 后自动回到 unknown 重新判定。
