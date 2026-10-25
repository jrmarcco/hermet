# hermet 前端接入文档

> 版本: 2026-10 · 服务端: hermet ( backend ) + synp ( gateway )
> 契约源: `api/README.md` 与 `api/proto` ( 唯一事实源 )

## 1. 架构总览

```
                 HTTP ( 登录/业务/搜索 )
  frontend ─────────────────────────────> backend ( hermet, :10081 )
     │
     │  WebSocket ( Message 帧, protojson )
     └──────────────────────────────────> gateway ( synp, :10071 )
                                             │  上行: Kafka → backend 落库分配 seq
                                             │  下行: NOTIFY 触发拉取 ( 不含消息本体 )
```

核心模型——**落库即事实，推送即通知，游标驱动**:

1. 发消息经网关转发，**UPSTREAM_ACK 只表示网关收到**，不代表业务成功；
2. 消息由 backend 落库并分配单调递增 seq 后才成立；
3. 服务端 NOTIFY 只携带最新 seq，客户端比较本地游标，落后才发起 SYNC 拉取增量；
4. 链路 at-least-once: 一切按 `message_id` ( 帧 ) 与 `seq` ( 收件箱 ) 幂等去重。

## 2. 接入流程

```
1. POST /api/v1/auth/sign-in ──────────> 拿 accessToken / refreshToken
2. 建立 WebSocket ( ws://gateway:10071 )
3. 第一帧必须发 AUTH ( body = {"token": accessToken} )
   ├─ 成功: AUTH_ACK{success:true}
   └─ 失败: AUTH_ACK{success:false} 并被断开 ( 重新 sign-in )
4. 心跳: 周期发 HEARTBEAT ( 空 body )
5. 发消息: UPSTREAM ( body = 业务消息 JSON )
6. 收通知: NOTIFY {latest_seq} → 本地 cursor 落后时发 SYNC {cursor}
7. 收增量: SYNC_ACK ( InboxEntry[], 按 seq 升序，推进 cursor )
8. token 过期 ( 2h ): POST /api/v1/auth/refresh-token 换新;
   WS 连接不因 token 过期断开，但重连时需用新 token
```

## 3. HTTP API

**通用约定**

- Base URL: `http://<backend>:10081`
- 鉴权: 除登录/刷新/`/.well-known/jwks.json` 外，请求头带 `x-access-token: Bearer <accessToken>`
- 响应体: `{"code": <http status>, "message": "", "data": <any>}`；
  未登录返回 HTTP 401 ( 无 body )；服务端错误返回 HTTP 500 + `{"code":500,"msg":"..."}`
- 时间戳: Unix 毫秒值

### 3.1 认证

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/auth/sign-in` | 无 | 登录 |
| POST | `/api/v1/auth/refresh-token` | 无 | 刷新 token |
| POST | `/api/v1/auth/sign-out` | 是 | 登出 ( 使 session 失效 ) |

**sign-in**

```json
// request
{ "account": "user@example.com", "accountType": "email", "credential": "<密码>" }

// response data
{ "accessToken": "<JWT>", "refreshToken": "<JWT>" }
```

- accessToken: RS256 JWT ( iss=synp-auth, aud=synp-gateway, 有效期 2h )，
  同时用于 HTTP 鉴权与 WS AUTH
- refreshToken: 有效期 7 天，仅用于 HTTP 刷新

**refresh-token**

```json
// request
{ "refreshToken": "<refreshToken>" }
// response data: 同 sign-in
```

### 3.2 联系人

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/contact/apply` | 发起好友申请 |
| POST | `/api/v1/contact/review` | 审批好友申请 ( 通过后双向建立联系人 ) |
| GET  | `/api/v1/contact/applications` | 收到的好友申请列表 |
| GET  | `/api/v1/contact/contacts` | 联系人列表 |

```json
// apply
{ "contactId": 134605228232032256, "message": "hello", "source": "search" }
// source: search=搜索 / qrcode=扫码 / group=群聊添加；不能添加自己

// review
{ "applicationId": 143667005994565632, "status": "approved" }
// status: approved / rejected

// applications → data[]
{ "id", "applicantId", "applicantName", "applicantAvatar",
  "applicationMessage", "source", "createdAt" }

// contacts → data[]
{ "id", "userId", "contactId", "contactNickname", "contactAvatar",
  "contactGender", "contactMobile", "contactInfoVer", "remarkName",
  "source", "addedAt" }
```

> 好友关系建立后，会话 ( channel ) 由服务端 CDC 异步创建，会话列表稍后可见。

### 3.3 群聊

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/group/create` | 建群 ( 创建者为群主 ) |

```json
// request
{ "name": "群名", "avatar": "https://...", "memberIds": [10001, 10002] }

// response data: 群信息 ( 含群 ID )
```

> 建群异步生效 ( 成员分批写入 )，会话列表稍后可见。

### 3.4 会话列表

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/channel/channels` | 当前用户的会话列表 ( 按最后消息时间倒序 ) |

```json
// data[] 关键字段
{
  "id": 123,
  "destination": "10001:10002",     // 会话目的地 ( 见 §5 会话 ID 规则 )
  "channelType": "single",          // single=单聊 / group=群聊
  "targetId": 10002,
  "channelName": "对方昵称或群名",
  "channelAvatar": "...",
  "isMuted": false, "isPinned": false,
  "lastMessageId": 456,
  "lastMessageType": "text",
  "lastMessageContent": "最后一条消息摘要",
  "lastMessageSenderId": 10001,
  "lastMessageTime": 1761400000000,
  "unreadCount": 3, "mentionCount": 0
}
```

### 3.5 消息搜索 ( ES )

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/message/search` | 会话内关键词搜索 ( 需为会话参与者 ) |

Query 参数: `conversationId` ( 会话 ID，见 §5 )、`keyword`、`from` ( 默认 0 )、`size` ( 默认 20，上限 100 )

```json
// data
{ "items": [ { "messageId", "conversationId", "senderId", "contentType", "sentAt" } ],
  "total": 12 }
```

> 搜索为异步索引 ( 秒级延迟 )，且只返回定位信息 ( messageId/sentAt )，
> 消息本体经同步链路按 seq 拉取。

## 4. WebSocket 协议

**连接**: `ws://<gateway>:10071`，文本帧 ( protojson 编码 )

**帧结构** ( `Message` ):

```json
{
  "commandType": 4,          // CommandType 数字，见下表
  "messageId": "01JD...",    // 帧唯一 ID ( 前端建议 ULID )，*_ACK 原样回填
  "body": "..."              // 按 commandType 解释，JSON 序列化后的字符串/对象
}
```

| 指令 | 值 | 方向 | body |
|---|---|---|---|
| HEARTBEAT | 1 | C→G | 空 |
| AUTH | 2 | C→G | `{"token": "<accessToken>"}` |
| AUTH_ACK | 3 | G→C | `{"success": true}` 或 `{"success": false, "errorMessage": "..."}` |
| UPSTREAM | 4 | C→G | 业务消息 ( §5 ) |
| UPSTREAM_ACK | 5 | G→C | `{"isSuccess": true, "timestampMs": 1761400000000}` |
| SYNC | 6 | C→G | `{"cursor": 42, "limit": 0}` |
| SYNC_ACK | 7 | G→C | `{"isSuccess": true, "response": {...}}` |
| CONVERSATION_SYNC | 8 | C→G | `{"conversationId": "g:100", "cursor": 0, "limit": 0}` |
| CONVERSATION_SYNC_ACK | 9 | G→C | `{"isSuccess": true, "response": {...}}` |
| NOTIFY | 10 | G→C | `{"latestSeq": 43}` |
| REDIRECT / RATE_LIMIT_EXCEEDED | 11 / 12 | G→C | 预留 |

**规则**

- 连接后 **15s 内**必须完成 AUTH，否则被关闭；鉴权前其他指令一律拒绝；
- 心跳建议 30s 一次，连接空闲约 90s 会被服务端关闭；
- 客户端身份 ( businessId/userId/sessionId ) 由网关从 JWT 盖章，**前端自报身份不可信也不会被采用**。

## 5. 业务消息格式 ( UPSTREAM body )

body 是前端自定义 JSON，网关与后端不解释结构，但 backend 按以下字段读取目的地与内容:

```json
{
  "id": "01JD...",              // 可选，前端本地消息 ID
  "sid": 10001,                 // 可选，仅展示用途 ( 服务端以盖章身份为准 )
  "destinationId": 10002,       // 必填: 单聊=对方 userId；群聊=groupId
  "destinationType": "single",  // 必填: single=单聊 / group=群聊
  "content": "你好",             // 必填: 消息内容 ( 文本，UTF-8；搜索按全文索引 )
  "contentType": "text"         // 必填: text / image
}
```

发送流程:

```
UPSTREAM ( messageId=帧ID, body=上述 JSON )
  ← UPSTREAM_ACK ( 仅网关级回执，不代表业务成功 )
  ... backend 校验 ( 好友关系/群成员 ) + 落库 + 分配 seq ...
  ← NOTIFY { latestSeq }
  → SYNC { cursor: 本地游标 }
  ← SYNC_ACK { entries, latestSeq, hasMore, cursorExpired }
```

- 被拒绝的消息 ( 非好友/非群成员 ) **不会有任何回执**，静默丢弃 ( 后续将改为系统消息下发 )；
- 同一帧因重发产生重复投递是正常的，服务端按 messageId 幂等落库。

## 6. 会话 ID 规则 ( conversationId )

| 类型 | 格式 | 示例 | 用途 |
|---|---|---|---|
| 单聊 | `s:<minUid>:<maxUid>` | `s:10001:10002` | 搜索接口 |
| 群聊 | `g:<groupId>` | `g:500` | 搜索接口 / CONVERSATION_SYNC |

> 注意: `/api/v1/channel/channels` 返回的 `destination` 字段是**不带前缀**的历史格式
> ( 单聊 `minUid:maxUid`、群聊 `groupId` 裸数字 )，仅用于会话列表展示与本地组织；
> 搜索与 CONVERSATION_SYNC 请按上表带前缀格式传参。

## 7. 收件箱同步 ( SYNC / SYNC_ACK )

```
→ SYNC { "cursor": 42, "limit": 0 }        // limit=0 用服务端默认 ( 100 )

← SYNC_ACK { "isSuccess": true, "response": {
      "entries": [
        { "seq": 43,
          "message": {                      // 写扩散条目: 本体内联 ( 单聊/小群 )
            "conversationId": "s:10001:10002",
            "messageId": "01JD...",
            "senderId": 10001,
            "body": "<业务消息 JSON 原文>",
            "sentAt": "2026-10-25T10:00:00Z" } },
        { "seq": 44,
          "conversationUpdate": {           // 读扩散指针 ( 大群 ): 需再拉 timeline
            "conversationId": "g:500",
            "conversationLatestSeq": 128 } }
      ],
      "latestSeq": 44,
      "hasMore": false,
      "cursorExpired": false } }
```

客户端处理规则:

1. entries 按 seq **升序**应用；每端维护本地 `cursor`，应用后推进到最新 seq；
2. `hasMore=true` 时携带新 cursor 继续翻页；
3. `cursorExpired=true` 表示游标早于服务端保留窗口，需重置本地数据重新同步；
4. `body` 是发送者 UPSTREAM 帧 body 的**原文**，按 `contentType` 渲染；
5. 收到指针条目 ( conversationUpdate ) 时，对该会话发 CONVERSATION_SYNC 拉正文。

## 8. 群聊 timeline ( CONVERSATION_SYNC )

小群 ( 成员数 ≤ 500 ) 与单聊走写扩散，收件箱直接内联消息本体；
大群走读扩散，收件箱只有指针，需二次拉取:

```
→ CONVERSATION_SYNC { "conversationId": "g:500", "cursor": 120, "limit": 0 }

← CONVERSATION_SYNC_ACK { "isSuccess": true, "response": {
      "messages": [
        { "conversationSeq": 121,
          "message": { "conversationId": "g:500", "messageId": "...",
                        "senderId": 10001, "body": "...", "sentAt": "..." } } ],
      "latestSeq": 128,
      "hasMore": false } }
```

- 每个会话独立游标 ( conversationSeq )，跨会话无全序；
- 非群成员请求返回错误 ( isSuccess=false )。

## 9. 重连与补偿

```
断线重连 / 心跳 / 切前台
  → 重新 AUTH ( 必要时先 refresh-token )
  → 立即 SYNC { cursor: 本地游标 }        // 主动兜底，不等 NOTIFY
  → ( 指针条目 ) CONVERSATION_SYNC
```

- NOTIFY 可能丢失或重复 ( 只影响实时性 ): 收到时比较 latestSeq 与本地 cursor，落后才 SYNC；
- token 过期重连前先调 refresh-token；
- 所有应答帧通过 messageId 与请求帧关联。

## 10. 常量与默认值

| 项 | 值 |
|---|---|
| accessToken 有效期 | 2h |
| refreshToken 有效期 | 7 天 |
| WS AUTH 超时 | 15s |
| WS 空闲断开 | ~90s ( 建议 30s 心跳 ) |
| SYNC 默认页大小 | 100 ( 上限 500 ) |
| 搜索页大小 | 默认 20 ( 上限 100 ) |
| 写扩散/读扩散阈值 | 群成员 ≤ 500 内联正本，> 500 走 timeline |

## 11. 尚未支持 ( 前端请勿依赖 )

- 顶号/强制下线通知 ( 契约待定义 ): 同账号新登录会使旧 session 失效，
  旧连接在下次 AUTH 时失败，**不会收到主动断开通知**；
- 撤回/删除/已读回执、群成员管理 API、消息历史向前翻页接口 ( 可用
  CONVERSATION_SYNC 小游标变通 )；
- 业务级失败回执 ( 非好友/被禁言发送无效，无系统消息提示，规划中 )。
