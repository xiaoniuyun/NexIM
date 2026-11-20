# NexIM 服务端（lite / standard 双形态；标准版消息队列为 NATS JetStream）

基于 `docs/01-业务需求文档.md` v2.0 与 `docs/02-技术架构文档.md` v1.12 §13.1 轻量形态实现的**可运行 IM 全栈系统**。

## 技术栈与形态

- 后端：Go 1.22 + Gin + gorilla/websocket + MySQL/GORM + Redis（模块化单体，JSON 协议）
- 网络：**HTTP REST `:8080` 与 WebSocket `:8081` 双端口**（见 `cmd/server/main.go` 与 `configs/config.yaml`）
- 前端：
  - H5：Vue 3 + Vite + Pinia（`web-h5/`）
  - 管理后台：Vue 3 + Vite + Pinia（`web-admin/`）
  - 桌面客户端：Electron + Vue 3，通过 `@h5` 别名复用 `web-h5/src` 的 API 层与 stores（`desktop/`）
  - 移动客户端：Flutter 独立实现（`mobile/`）
- 部署：H5/Admin 经 `go:embed` 打进单个 Go 二进制，HTTP `:8080` 同时提供 REST API 与 H5/Admin 静态资源；WS 独立监听 `:8081`。支持 lite / standard 双形态（`deployment.profile` 同一份代码切换）
- 标准版：NATS JetStream + Redis Cluster + 分片 + Meilisearch 3 副本 + RustFS（S3 兼容对象存储，替代 MinIO）+ Prometheus/Jaeger 完整配置，见 `configs/config-standard.yaml` 与 `docker-compose.standard.yml`
- 消息队列：standard 形态下消息落库后发布 `im.message.created` 事件到 JetStream 流 `IM_EVENT`（持久化、可按 `msgId` 去重、可回放），各网关节点消费后向本节点在线用户投递

> 设计目标（微服务 / Kafka / gRPC / 双原生移动端）与当前实现的差异，见 `docs/02-技术架构文档.md` 开头的「实现现状对照」。

## 目录结构

```
NexIM/
├── server/                       # Go 后端
│   ├── cmd/server/main.go        # 入口（HTTP API :8080 + WS 网关 :8081）
│   ├── configs/
│   │   ├── config.yaml           # lite 轻量版配置（默认）
│   │   └── config-standard.yaml  # standard 标准版配置
│   ├── internal/
│   │   ├── api/                  # REST 路由 + 鉴权 + 全部 handler
│   │   ├── gateway/              # WebSocket Hub 与命令字处理
│   │   ├── mq/                   # NATS JetStream 事件总线（建流/发布/消费）
│   │   ├── service/              # auth/message/contact/group/file/admin/call
│   │   ├── store/                # GORM 迁移 + 种子数据 + Redis
│   │   ├── storage/              # 本地 FS 与 RustFS（S3 兼容）对象存储
│   │   ├── model/ / config/ / pkg/ / webassets/
│   ├── webassets/                # go:embed 嵌入 H5/Admin 构建产物
│   │   ├── embed.go              # embed.FS + 静态服务 + SPA fallback
│   │   └── dist/{h5,admin}/      # 由 scripts/sync-web 从前端 dist 同步（构建产物）
│   ├── scripts/                  # sync-web.ps1 / sync-web.sh（同步前端产物）
│   ├── Dockerfile                # 多阶段：构建 H5→构建 Admin→Go embed 编译→alpine 运行
│   ├── docker-compose.yml        # lite：MySQL + Redis + NATS + RustFS + coturn + Server
│   ├── docker-compose.standard.yml # standard：完整中间件编排
│   └── nginx-web.conf            # 可选：独立 Nginx 托管前端（子路径 /h5/ /admin/）
├── web-h5/                       # H5 移动端（Vue 3 + Vite，hash 路由 base=/h5/）
├── web-admin/                    # 管理后台（Vue 3 + Vite，hash 路由 base=/admin/）
├── desktop/                      # Electron 桌面客户端（Vue 3，复用 web-h5/src）
├── mobile/                       # Flutter 移动客户端（Android + iOS）
├── docs/                         # 业务需求文档 / 技术架构文档
└── prototype/                    # 高保真交互原型（index.html）
```

## 一键启动（Docker Compose）

> 先启动 Docker Desktop。H5 / Admin 已通过 `go:embed` **打包进单个 Go 二进制**；
> HTTP `:8080` 提供 REST API 与前端静态资源，WebSocket 独立监听 `:8081`。
> 下面为 lite 轻量形态；**日常开发请使用标准版（含 NATS JetStream），见「标准版运行」**。

```bash
cd d:/workcode/NexIM/server
docker compose up -d --build
```

访问：

- **Vue H5 移动端**：`http://localhost:8080/h5/`（手机浏览器或开发者工具移动模式）
- **Vue 管理后台**：`http://localhost:8080/admin/`（管理员账号 linxiao 登录）
- 健康检查：`http://localhost:8080/health`
- WebSocket：`ws://localhost:8081/ws`（独立端口，非同源）
- 演示账号：`linxiao / 123456`（管理员，含后台入口）、`zhangwei / 123456`（普通用户）

### 本地单二进制运行（无需 Docker 前端容器）

```bash
# 1) 构建两个前端
cd web-h5   && npm install && npm run build
cd ../web-admin && npm install && npm run build
# 2) 同步产物到 server/webassets/dist（供 go:embed）
cd ../server && powershell -File scripts/sync-web.ps1   # Linux/Mac: bash scripts/sync-web.sh
# 3) 编译单二进制（已含 H5/Admin）
go build -o nexim-server ./cmd/server
# 4) 运行（MySQL/Redis/NATS/RustFS 需可达，地址可用环境变量覆盖）
./nexim-server
# 浏览器打开 http://localhost:8080/h5/ 或 /admin/；WS 为 ws://localhost:8081/ws
```

H5 / Admin 本地 dev 模式（热更新，独立端口，仍代理到后端）：

```bash
cd web-h5    && npm run dev   # http://localhost:5174/h5/，/api → :8080，/ws → :8081
cd web-admin && npm run dev   # http://localhost:5175/admin/
```

### 标准版（standard profile）运行 —— 开发默认形态

开发统一使用标准版：`deployment.profile=standard`，消息队列为 **NATS JetStream**，其余组件按架构文档 13.1 配置（Redis Cluster / 分片 / Meilisearch×3 / RustFS / Prometheus+Jaeger）。同一份代码通过 `deployment.profile` 切换形态。

方式一（推荐）：标准版中间件编排，开发/联调一键起：

```bash
cd d:/workcode/NexIM/server
docker compose -f docker-compose.standard.yml up -d --build
# 含 mysql / redis / nats(JetStream) / meilisearch / rustfs / coturn / server 等服务
```

对象存储（RustFS）：

- S3 API `http://localhost:9000`，控制台 `http://localhost:9001`（账号 `nexim / nexim123456`，对应 `RUSTFS_ACCESS_KEY/SECRET_KEY`）
- 服务端凭据来自 `storage.access_key/secret_key`（或 `STORAGE_ACCESS_KEY/STORAGE_SECRET_KEY` 环境变量），桶 `nexim-files` 不存在时首次写入自动创建
- 启动日志出现 `rustfs storage ready` 表示已接入对象存储；失败会回退本地 FS（lite）或仅告警（standard 下首次写入重试建桶）

只起核心链路（消息收发 + MQ，最快）：

```bash
docker compose -f docker-compose.standard.yml up -d --build mysql redis nats server
```

方式二：本地以 standard 形态运行单二进制（需先启动 MySQL / Redis / NATS）：

```bash
cd d:/workcode/NexIM/server
$env:CONFIG_PATH="configs/config-standard.yaml"   # Linux: export CONFIG_PATH=...
$env:MQ_URL="nats://127.0.0.1:4222"               # 本机开发指向宿主机 NATS
./nexim-server
```

启动日志会输出形态摘要与 MQ 连接状态，例如：

```
deployment profile  summary="profile=standard mode=cluster services=microservice
  mq=jetstream cache=redis(cluster=true) db=mysql(shards=16,replicas=2)
  search=meilisearch(x3) storage=rustfs metrics=prometheus"  standard=true
nats jetstream ready  url=nats://nats:4222 stream=IM_EVENT node=<host>-<pid>
```

JetStream 约定：

- 流：`IM_EVENT`（FileStorage，`MaxAge` 取 `mq.retention`，默认 30d）
- 主题：`im.message.created`、`im.call.signal`
- 去重：以 `msgId` 作为 `Nats-Msg-Id`，WS/HTTP 双通道重复提交不会产生重复事件
- 消费：各节点独立 durable（`nexim-node-<nodeId>`，`DeliverNew`），消费后仅向本节点在线用户投递，不回发事件

`GET /health` 返回 `profile`、`mode`、`services`、`mq{driver,url,stream,retention}` 与 `summary`。显式环境变量（`MQ_DRIVER`、`MQ_URL`、`MQ_STREAM`、`DB_SHARDS`、`SEARCH_REPLICAS`）优先于 profile 派生值。

查看 JetStream 状态（NATS 监控端口 8222）：

```bash
curl "http://127.0.0.1:8222/jsz?streams=true"
```

查看日志 / 停止：

```bash
docker compose logs -f server
docker compose down -v      # 停止并清空数据卷
```

首次启动会自动建表并写入种子数据：6 个用户、5 个部门、1 个群聊、角色权限、2 条公告、好友关系。

## 已实现功能（对应文档与原型）

### 认证与安全
登录 / 注册 / 刷新 / 登出 / 权限 / 设备管理 / 验证码（演示码 `123456`）；JWT（含 `jti` 黑名单）、bcrypt、验证码 60s 频控。

### WebSocket 实时消息（`:8081/ws`）
`AUTH / HEARTBEAT / SEND / PUSH / SYNC / READ / REVOKE / TYPING / KICK / NOTICE_PUSH`；同设备互踢、在线路由、30s 心跳 / 90s 超时、下行多端推送。

### 消息与会话
`clientMsgId` 幂等、Redis `conv:seq` 单调序号、写扩散会话、已读 / 撤回 / 搜索 / 转发、历史漫游、置顶免打扰、一键已读。

### 联系人 / 组织
好友列表、好友申请与处理、用户搜索、部门树、部门成员、在线状态。

### 群聊
创建群、我的群、群详情、群成员、邀请 / 移除成员、解散群。

### 文件
上传 / 列表 / 下载（带 token）/ 删除。对象存储按 `storage.driver` 分流：lite 落本地 FS（`data/files`），standard 走 **RustFS**（S3 协议，SigV4 直连，桶 `nexim-files` 首次写入自动创建）；下载由服务端流式转发，`Content-Disposition` 兼容中文文件名。

### 实时音视频（WebRTC 1对1）
- 信令复用 WebSocket：`CALL_INVITE / CALL_ANSWER / CALL_ICE / CALL_HANGUP`，服务端校验后转发 `CALL_INVITE_PUSH / CALL_ANSWER_PUSH / CALL_ICE_PUSH / CALL_HANGUP_PUSH`；standard 形态信令经 JetStream 主题 `im.call.signal` 跨网关节点投递
- 媒体：浏览器 `RTCPeerConnection` P2P，offer/answer + trickle ICE；ICE Servers（STUN/TURN）由 `GET /rtc/config` 下发，支持 coturn
- 通话页：本地/远端视频、静音、关摄像头、前后摄像头切换、通话计时、挂断；任意页面收到来电弹窗（接听/拒绝），忙线自动拒接
- 通话记录：`im_call` 表记录呼叫/接通/结束/时长，`GET /calls` 查询
- 注意：`getUserMedia` 需要安全上下文（`localhost` 或 HTTPS），局域网 IP 访问需配置 HTTPS

### 管理后台
工作台统计与趋势、用户 CRUD、重置密码 / 禁用、角色权限、审计日志、消息审计、公告管理、敏感词库。

## REST API 一览（前缀 `/api/v1`）

认证：`POST /auth/login|register|refresh|sms-code|logout|password/forgot`、`GET /auth/permissions|devices`、`POST /auth/devices/bind`、`DELETE /auth/devices/:deviceId`

消息：`GET /conversations`、`PATCH /conversations/:id/settings`、`POST /conversations/read-all`、`GET /conversations/:id/messages`、`POST /messages/send|read|revoke|search|forward`

实时音视频：`GET /rtc/config`（ICE Servers）、`GET /calls`（通话记录）

联系人：`GET /users/search|users/:id/profile`、`GET /friends`、`POST /friends/apply`、`GET /friends/applies`、`PUT /friends/applies/:id`、`GET /depts|/depts/tree|/depts/:id/members`

群聊：`GET /groups|/groups/:id|/groups/:id/members`、`POST /groups|/groups/:id/invite`、`DELETE /groups/:id/members/:uid|/groups/:id`

文件：`GET /files`、`POST /files/upload`、`GET /files/:id/download`、`DELETE /files/:id`

后台：`GET /admin/dashboard|users|roles|audit|audit/messages|notices|sensitive-words`、`POST /admin/users|roles/grant|notices|sensitive-words`、`PUT /admin/users/:id|notices/:id`、`DELETE /admin/users/:id`、`POST /admin/users/:id/reset-password`

公告：`GET /notices`

统一响应：`{ code, message, data, traceId, serverTime }`，`code=0` 成功，`code=2001` 触发刷新。

## 本地直连（无 Docker）

```bash
cd server
go build -o nexim-server.exe ./cmd/server
# 确保本机 MySQL(3306) / Redis(6379) / NATS(4222) 可用，修改 configs/config.yaml 对应地址
$env:CONFIG_PATH="configs/config.yaml"; .\nexim-server.exe
# HTTP API + 静态资源 :8080；WebSocket :8081/ws
# 前端 dev 模式见上文「H5 / Admin 本地 dev 模式」，Vite 已配置 /api 与 /ws 代理
```

## 测试

```bash
cd server
go test ./... -count=1                                   # 全量单测 + 集成测试
go test ./internal/api -run TestAdminIntegration -v    # 只跑端到端集成测试
```

- 单元测试：`internal/api/router_test.go`（gin 路由注册冲突）、`internal/storage/s3_test.go`（SigV4 签名、对象键编码、凭据校验）。
- 集成测试 `internal/api/integration_test.go`：自动创建独立库 `nexim_itest`（Redis 用 DB 15），经 `httptest` 驱动真实 Router 与服务端 RBAC 中间件，覆盖登录与权限下发、用户筛选/分页/角色部门关联、角色 CRUD 与权限版本、组织树 CRUD 与物化路径重算、授权 200 / 未授权 403 / 解绑后缓存失效、账号启停即时拦截、CSV 导出；结束后自动删除测试库。
- 中间件不可用时集成用例自动 `Skip`，纯代码环境 `go test ./...` 仍然通过；连接参数可用 `NEXIM_TEST_MYSQL_DSN`（服务器级 DSN，以 `/` 结尾）与 `NEXIM_TEST_REDIS_ADDR` 覆盖。

## 验证结果

- `go build ./...`、`go vet ./...`、`go test ./... -count=1`：通过（含 RBAC 端到端集成测试）
- `docker compose config`：通过（lite / standard 两套编排）
- Vue H5：`npm run build` 通过；Vue 管理后台：`npm run build` 通过
- 后端运行实测：MySQL / Redis / NATS 容器健康，服务 `:8080`（HTTP）与 `:8081`（WS）启动成功
- 管理 API 实测：登录 / dashboard / 用户 / 角色 / 审计 / 消息审计 / 公告 / 敏感词全部返回正常
- 单二进制 embed：`go:embed` 将 H5/Admin 构建产物嵌入 Go 二进制，HTTP `:8080` 提供 API 与静态资源、SPA fallback 正常；WS 独立 `:8081`
- 标准版适配：`deployment.profile` 驱动形态派生与校验；`config-standard.yaml`、`docker-compose.standard.yml` 已就绪；启动日志与 `/health` 输出形态摘要；standard 形态实测 H5/Admin 登录、KPI、14 导航正常
- NATS JetStream 接入：消息落库后发布 `im.message.created` 到流 `IM_EVENT`，节点消费后向本节点在线用户投递；以 `msgId` 去重；实测 HTTP/WS 双通道推送实时到达、重复提交仅落一条事件
- 实时音视频：信令 `CALL_INVITE/ANSWER/ICE/HANGUP` 及 PUSH 帧、在线校验（离线返回 `CALL_ACK 4041`）、standard 形态经 JetStream `im.call.signal` 跨节点转发；H5 `stores/call.js` 实现 RTCPeerConnection/offer-answer/trickle ICE/媒体控制；`im_call` 通话记录 + `GET /rtc/config|/calls` 实测通过
- 管理后台 14 页路由与渲染、H5 登录/4 Tab/WS `AUTH_ACK`+`HEARTBEAT_ACK`+`SEND_ACK`/消息落库/会话生成/聊天页渲染全链路实测通过
- Docker Desktop 守护进程需在部署机器上处于运行状态

## 安全提示

`config.yaml` / `docker-compose.yml` 中的 JWT 密钥与数据库密码仅用于本地演示，生产环境务必通过环境变量覆盖并启用 HTTPS/WSS。
