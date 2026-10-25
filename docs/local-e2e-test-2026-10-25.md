# 本地端到端测试记录 — 网关 ( synp ) × 后端 ( hermet )

> 测试日期: 2026-10-25
> 环境: 网关与后端容器跑在开发机 ( Docker Desktop, jrmarcco_net );
> 中间件 ( kafka×3 mTLS / etcd×3 认证已启用 / redis / pgsql 分片×2 / es / debezium connect×3 ) 部署在 192.168.3.3。
> 测试账号: A = `jrmarcco@gmail.com` ( uid 134605228232032256 ),
> B = `test-user-1@example.com` ( uid 240070963854417920, 经 `POST /api/v1/admin/user` 创建, 默认密码 `hermet@2026` )。

## 一、结果总览

### 后端 REST API

| 接口 | 结果 | 备注 |
|---|---|---|
| POST /api/v1/auth/sign-in | ✅ | 双账号; 错误密码 401 ( 修复后 ) |
| POST /api/v1/auth/refresh-token | ✅ | |
| POST /api/v1/auth/sign-out | ✅ | |
| 无效 token 访问受保护接口 | ✅ | 401 ( 响应体为空, 见观察项 4 ) |
| POST /api/v1/admin/user | ✅ | 创建 B 账号 |
| POST /api/v1/contact/apply | ✅ | B → A 申请 |
| GET /api/v1/contact/applications | ✅ | 仅返回待审, 无 status 字段 |
| POST /api/v1/contact/review | ✅ | approved 后经 CDC 建立双向联系人 |
| GET /api/v1/contact/contacts | ✅ | 双向均可见, 含读侧宽表字段 |
| POST /api/v1/group/create | ✅ | 修复后群主也入群 ( 见修复 2 ) |
| GET /api/v1/channel/channels | ✅ | single + group 频道均出现 |
| POST /api/v1/admin/message | ✅ | 走 kafka 上行链路, ES 可检索 |
| GET /api/v1/message/search | ✅ | conversationId 格式 `s:<uid小>:<uid大>` |

### 网关 WS 协议 ( 11/11 )

| 用例 | 结果 |
|---|---|
| 双端登录 + AUTH → AUTH_ACK success | ✅ |
| HEARTBEAT 回显 ( messageId 一致 ) | ✅ |
| 未鉴权先发非 AUTH 帧 → 连接被关闭 | ✅ |
| UPSTREAM 单聊 B→A → UPSTREAM_ACK isSuccess | ✅ |
| A 在线收 NOTIFY ( latestSeq ) | ✅ |
| A SYNC(cursor=0) → SYNC_ACK 含新消息 | ✅ |
| UPSTREAM 群聊 A→群 → UPSTREAM_ACK | ✅ |
| B 收群消息 NOTIFY | ✅ |
| B SYNC 收到群消息内容 | ✅ |

WS 回执帧为 binary opcode ( protojson ), 客户端读帧需用通用读取而非仅 text。

## 二、修复的问题

### 1. [backend] sign-in 密码错误返回 500 而非 401
`internal/api/auth_handler.go`: `errs.ErrInvalidUser` 未映射到 `ErrUnauthorized`,
落入 xgin 默认 500。已补映射, 回归 ✅ ( 错误密码 → 401 )。

### 2. [backend] 建群漏掉群主自身的成员关系
`internal/service/group_service.go` Create 只把 `MemberIDs` 送入异步成员管线,
群主无 group_member/user_group/channel 行, 群不会出现在自己的频道列表。
修复: 群主并入成员批次 ( 去重 )。回归 ✅。

### 3. [backend] outbox 事件聚合粒度错误, 同群多成员事件互相覆盖
`internal/repo/group_member_repo.go` CreateView 写 outbox 时
`AggregateID: gmv.GroupID` —— 同群第二个成员的 `group_member_create` 事件
撞 `(aggregate_id, aggregate_type, event_type)` 唯一键被 `DO NOTHING` 静默吞掉,
该成员的 user_group/channel 读侧永不建立 ( 症状: 群消息后对方频道无未读/最后消息,
channel UPDATE rows=0 )。修复: `AggregateID` 改用成员行雪花 ID `gmv.ID`
( 消费侧只解析 payload, 不读 aggregate_id; 幂等仍由 redis `group_id-user_id` 键保证 )。
修复 2 暴露了此隐藏 bug ( 此前批次里只有一个成员时不会撞键 )。回归 ✅ ( 双方频道均建立 )。

### 4. [debezium] sidecar 自动注册从未成功过 ( 部署链 )
- `init_connectors.sh` shebang 为 bash 而 alpine 无 bash → compose 已加 apk add bash;
- scp 推送丢失可执行位 → compose command 改为 `bash /init_connectors.sh` 显式调用。
服务器重建后 sidecar 首次自动注册成功, 两个 connector + task 均 RUNNING
( 同时实战验证了 schema history TLS 显式配置与 `__PGSQL_HERMET_x_PASSWORD__`
占位符渲染链路 )。

## 三、观察项 ( 未修, 待决策 )

1. **心跳 topic 前缀配置未生效**: connector 配置 `heartbeat.topics.prefix=__debezium_heartbeat` ( 下划线 ),
   实际 topic 为默认 `__debezium-heartbeat.*` ( 连字符 )。心跳功能本身正常。
   将来接线 heartbeat_processor 时需按实际 topic 模式消费, 或修正配置属性名。
2. **gRPC NotifyService 直连通道无触发证据**: backend 配置了 notify client ( synp:///synp-gateway ),
   本次 NOTIFY 均经 kafka 通知链路送达, 未见 gRPC 直连日志。待定位其触发场景。
3. **group/create 响应字段为 PascalCase** ( `ID/GroupName/...` ), 与其余接口 camelCase 不一致。
4. **无效 token 401 响应体为空** ( Content-Length 0 ), 无 JSON 错误体。
5. **测试期数据**: pgsql 中存在 e2e-* 测试群与联系人、kafka 中存在 e2e 消息, 如需清理可按
   `e2e-group-*` / uid 240070963854417920 定位。

## 四、复现方式

- WS 端到端: 临时 Go 客户端 ( 登录→双连接 AUTH→心跳→单聊→SYNC→群聊 ),
  依赖 gateway 仓库 gobwas/ws; 仓库自带 `cmd/jointtest` 亦可做单聊端到端。
- 测试配置: 网关 `etc/config.docker.yaml`, 后端 `etc/config.docker.yaml`
  ( 真实凭据, 勿提交 ); 启动命令分别为
  `SYNP_CONFIG=... docker compose up -d` / `HERMET_CONFIG=... docker compose up -d`。
