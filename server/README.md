# NexIM 全栈实现（lite / standard 双形态；标准版消息队列为 NATS JetStream）

基于 `docs/01-业务需求文档.md` v2.0 与 `docs/02-技术架构文档.md` v1.12 §13.1 轻量形态实现的**可运行 IM 全栈系统**：

- 后端：Go + Gin + gorilla/websocket + MySQL + Redis（模块化单体，JSON 协议）
- 桌面 Web：原生 HTML/CSS/JS 单页应用（消息、通讯录、文件、群聊、管理后台）
- H5 移动端：Vue 3 + Vite + Pinia + Vue Router 独立应用（对齐高保真原型 H5 视图）
- 管理后台：Vue 3 + Vite + Pinia 独立应用（对齐高保真原型 admin 视图，深色分组侧栏）
- 部署：H5/Admin 经 `go:embed` 打进单个 Go 二进制，Docker Compose 仅 MySQL + Redis + Server 三容器，单端口 8080 提供 API/WS/H5/Admin；支持 lite / standard 双形态（`deployment.profile` 同一份代码切换）
- 标准版：NATS JetStream + Redis Cluster + 分片 + Meilisearch 3 副本 + RustFS（S3 兼容对象存储，替代 MinIO）+ Prometheus/Jaeger 完整配置，见 `configs/config-standard.yaml` 与 `docker-compose.standard.yml`
- 消息队列：standard 形态下消息落库后发布 `im.message.created` 事件到 JetStream 流 `IM_EVENT`（持久化、可按 `msgId` 去重、可回放），各网关节点消费后向本节点在线用户投递

## 目录结构

```
NexIM/
├── server/                       # Go 后端
│   ├── cmd/server/main.go        # 入口（HTTP API + WS 网关）
│   ├── configs/config.yaml       # 配置（Viper + 环境变量覆盖）
│   ├── internal/
│   │   ├── api/                  # REST 路由 + 鉴权 + 全部 handler
│   │   ├── gateway/              # WebSocket Hub 与命令字处理
│   │   ├── mq/                   # NATS JetStream 事件总线（建流/发布/消费）
│   │   ├── service/              # auth/message/contact/group/file/admin
│   │   ├── store/                # GORM 迁移 + 种子数据 + Redis
│   │   ├── model/                # 数据模型
│   │   └── pkg/                  # 响应 / JWT / traceId
│   ├── webassets/                # go:embed 嵌入 H5/Admin 构建产物
│   │   ├── embed.go              # embed.FS + 静态服务 + SPA fallback
│   │   └── dist/{h5,admin}/      # 由 scripts/sync-web 从前端 dist 同步（构建产物）
│   ├── scripts/                  # sync-web.ps1 / sync-web.sh（同步前端产物）
│   ├── configs/
│   │   ├── config.yaml           # lite 轻量版配置（默认）
│   │   └── config-standard.yaml  # standard 标准版配置
│   ├── Dockerfile                # 4 阶段：构建 H5→构建 Admin→Go embed 编译→alpine 运行
│   ├── docker-compose.yml        # lite：MySQL + Redis + Server（单二进制含全部前端）
│   └── docker-compose.standard.yml # standard：+ NATS JetStream + Meilisearch + RustFS
│   └── nginx-web.conf            # 桌面 Web（可选，原生 web/ 用）
├── web/                          # 桌面 Web 单页应用（原生，可选独立 Nginx）
│   ├── index.html
│   ├── css/app.css
│   └── js/{api.js,app.js,pages.js}
├── web-h5/                       # H5 移动端（Vue 3 + Vite）
│   ├── src/
│   │   ├── views/                # 登录/会话/聊天/音视频通话/通讯录/好友/发起群聊/群聊/组织/文件/发现/收藏/我的/设置
│   │   ├── components/           # Avatar/TabBar/ToastHost/ModalBox/IncomingCall
│   │   ├── stores/               # auth / chat / call(Pinia)
│   │   ├── router.js             # hash 路由（base=/h5/）
│   │   ├── api.js / i18n.js
│   │   └── styles/h5.css
│   └── dist/                     # npm run build 产物（同步到 webassets/dist/h5）
└── web-admin/                    # 管理后台（Vue 3 + Vite）
    ├── src/
    │   ├── views/                # 工作台/用户/组织/群组/登录日志/角色/权限预览/操作日志/消息审计/内容安全/公告/通话记录/质量监控/设置（14页）
    │   ├── components/           # ModalBox / CmdPanel
    │   ├── stores/auth.js
    │   ├── router.js             # hash 路由（base=/admin/）
    │   ├── api.js
    │   └── styles/admin.css
    └── dist/                     # npm run build 产物（同步到 webassets/dist/admin）
```

## 一键启动（Docker Compose）

> 先启动 Docker Desktop。前端 H5 / Admin 已通过 `go:embed` **打包进单个 Go 二进制**，
> 由 server 容器在**同一端口 8080** 同时提供 REST API、WebSocket、H5 与管理后台。
> 下面为 lite 轻量形态；**日常开发请使用标准版（含 NATS JetStream），见「标准版运行」**。

```bash
cd d:/workcode/NexIM/server
docker compose up -d --build     # 仅 mysql / redis / server 三个容器
```

访问（全部同一端口 `8080`）：

- **Vue H5 移动端**：`http://localhost:8080/h5/`（手机浏览器或开发者工具移动模式）
- **Vue 管理后台**：`http://localhost:8080/admin/`（管理员账号 linxiao 登录）
- 健康检查：`http://localhost:8080/health`
- WebSocket：`ws://localhost:8080/ws`（与页面同源）
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
# 4) 运行（MySQL/Redis 需可达，地址可用环境变量覆盖）
./nexim-server
# 浏览器打开 http://localhost:8080/h5/ 或 /admin/
```

H5 / Admin 本地 dev 模式（热更新，独立端口，仍代理到后端）：

```bash
cd web-h5    && npm run dev   # http://localhost:5174/h5/
cd web-admin && npm run dev   # http://localhost:5175/admin/
```

### 标准版（standard profile）运行 —— 开发默认形态

开发统一使用标准版：`deployment.profile=standard`，消息队列为 **NATS JetStream**，其余组件按架构文档 13.1 配置（Redis Cluster / 分片 / Meilisearch×3 / RustFS / Prometheus+Jaeger）。同一份代码通过 `deployment.profile` 切换形态。

方式一（推荐）：标准版中间件编排，开发/联调一键起：

```bash
cd d:/workcode/NexIM/server
docker compose -f docker-compose.standard.yml up -d --build
# 含 mysql / redis / nats(JetStream) / meilisearch / rustfs / server(standard) 六个服务
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
- 主题：`im.message.created`
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

### WebSocket 实时消息（`/ws`）
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

统一响应：`{ code, message, data, traceId, serverTime }`，`code=0` 成功。

## 本地直连（无 Docker）

```bash
cd server
go build -o nexim-server.exe ./cmd/server
# 确保本机 MySQL(3306)/Redis(6379) 可用，修改 configs/config.yaml 的 db.dsn/redis.addr
$env:CONFIG_PATH="configs/config.yaml"; .\nexim-server.exe
# 另开终端启动静态服务托管 web/，并配置 /api 与 /ws 代理到 :8080/:8081
```

## 验证结果

- `go build ./...`、`go vet ./...`：通过
- `docker compose config`：通过（含 `h5` 服务）
- Vue H5：`npm run build` 通过（51 模块，产物 `web-h5/dist`）；浏览器实测渲染正常
- Vue 管理后台：`npm run build` 通过（44 模块，产物 `web-admin/dist`）
- 后端运行实测：MySQL/Redis 容器健康，`go run` 服务 `:8080/:8081` 启动成功
- 管理 API 实测：登录/dashboard/用户/角色/审计/消息审计/公告/敏感词全部返回正常
- Vue 管理后台浏览器实测：登录→工作台 KPI→用户管理全流程通过，新增用户真实写入 MySQL 并写入审计日志
- 原型功能对齐（本轮）：管理后台补齐至原型全部 14 个页面及全部核心交互——全局 Ctrl+K 命令面板/告警铃铛跳转/主题切换、用户管理筛选+编辑+启停+分页、组织成员表+数据范围示例+新增部门、群详情/冻结/解散弹窗、RBAC 权限树(24 节点)+全选清空保存+数据范围、消息审计检索卡+导出 MFA、内容安全命中策略+命中记录、公告表格+编辑/下线、通话回放 MFA+强制挂断确认、系统设置 6 参数+5 安全开关；H5 补齐音视频通话页、发起群聊页、设置页通知/隐私/通用三 Tab 与深色模式
- 浏览器实测（本轮）：工作台 7/30 天切换 + SVG 趋势图 + 6 项健康度、通话记录/质量监控/权限预览/登录日志/操作日志/消息审计全部路由与渲染正常；H5 通话页 4 控制键 + 4 项 MOS 质量指标渲染正常
- 运行验证（本轮）：后端 13 个 API + 鉴权拦截（401）全部通过；管理后台 14 页路由正确、新增用户真实写库并生成审计；H5 登录/4 Tab/WS `AUTH_ACK`+`HEARTBEAT_ACK`+`SEND_ACK`/消息落库/会话生成/聊天页渲染全链路通过
- Bug 修复（本轮）：User 模型补 `LastLoginAt` 字段（消除 `unknown column last_login_at`）；Conversation 的 `PeerName/PeerAvatar` 取消 `-:migration`（消除 `unknown column peer_name`，会话写扩散恢复正常）
- 单二进制 embed（本轮）：`go:embed` 将 H5/Admin 构建产物嵌入 Go 二进制（本地 18MB / 容器内 strip 后 12MB）；单端口 8080 同时提供 API/WS/H5/Admin；已修复静态资源路径前缀 bug；浏览器实测从容器化单二进制登录 H5/Admin、同源 WS `AUTH_ACK`、静态 JS/CSS MIME 正确、SPA fallback 正常
- 标准版适配（本轮）：Config 补齐 mq/cache/db(shards/replicas)/search/storage/observability/rtc 配置段；`deployment.profile` 驱动形态派生与校验；新增 `config-standard.yaml`、`docker-compose.standard.yml`；启动日志与 `/health` 输出形态摘要；standard 形态实测 H5/Admin 登录、KPI、14 导航全部正常，显式环境变量优先
- NATS JetStream 接入（本轮）：新增 `internal/mq`（连接/建流/发布/按节点 durable 消费），standard 形态 `mq.driver=jetstream`；消息落库后发布 `im.message.created` 到流 `IM_EVENT`，节点消费后向本节点在线用户投递；以 `msgId` 去重。实测 HTTP/WS 双通道推送均实时到达，事件写入 JetStream（0→1、2→3），重复提交仅落一条事件；`/health` 输出 `mq{driver=jetstream,url,stream,retention}`
- 实时音视频（本轮）：信令 `CALL_INVITE/ANSWER/ICE/HANGUP` 及对应 PUSH 帧、在线校验（离线返回 `CALL_ACK 4041`）、standard 形态经 JetStream `im.call.signal` 跨节点转发；H5 新增 `stores/call.js`（RTCPeerConnection/offer-answer/trickle ICE/媒体控制）、真实视频通话页与全局来电浮层；`im_call` 通话记录 + `GET /rtc/config|/calls`。实测：呼叫 ACK、来电推送（SDP/类型/主叫名）、应答推送、ICE 转发、挂断通知、通话记录状态、离线拒呼全部通过
- Docker Desktop 守护进程需在部署机器上处于运行状态

## 安全提示

`config.yaml` / `docker-compose.yml` 中的 JWT 密钥与数据库密码仅用于本地演示，生产环境务必通过环境变量覆盖并启用 HTTPS/WSS。
