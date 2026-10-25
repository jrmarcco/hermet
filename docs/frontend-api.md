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
  未登录返回 HTTP 401 ( 无 body )；
  参数校验失败返回 HTTP 400 + 具体校验信息；
  服务端错误返回 HTTP 500 + `message: "internal error"` ( 内部错误细节只进服务端日志 )
- 时间戳: Unix 毫秒值
- **int64 ID 一律以十进制字符串序列化** ( 联系人/会话/群/用户等全部 ID，如
  `"contactId": "134605228232032256"` )——JS `Number` 无法无损承载超过 2^53-1 的整数；
  请求中的 ID 字段同时接受字符串与数字，但**强烈建议统一传字符串**

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
  同时用于 HTTP 鉴权与 WS AUTH；
  payload 中 `user_id` / `business_id` 以**十进制字符串**承载
  ( `"user_id":"134605228232032256"`，JS 可无损读取 )
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
// apply ( ID 建议传字符串，数字形态兼容 )
{ "contactId": "134605228232032256", "message": "hello", "source": "search" }
// source: search=搜索 / qrcode=扫码 / group=群聊添加；不能添加自己

// review
{ "applicationId": "143667005994565632", "status": "approved" }
// status: approved / rejected

// applications → data[] ( ID 均为字符串 )
{ "id", "applicantId", "applicantName", "applicantAvatar",
  "applicationMessage", "source", "createdAt" }

// contacts → data[] ( ID 均为字符串 )
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
// request ( memberIds 建议传字符串数组 )
{ "name": "群名", "avatar": "https://...", "memberIds": ["10001", "10002"] }

// response data: 群信息 ( 含群 ID，ID 为字符串 )
{ "id": "240079435428384768", "name": "群名", "avatar": "...", "status": "active",
  "creatorId": "10001", "lastMessageAt": 0, "createdAt": 1761400000000 }
```

> 建群异步生效 ( 成员分批写入 )，会话列表稍后可见。

### 3.4 会话列表

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/channel/channels` | 当前用户的会话列表 ( 按最后消息时间倒序 ) |

```json
// data[] 关键字段 ( int64 ID 均为字符串 )
{
  "id": "123",
  "destination": "10001:10002",     // 会话目的地 ( 见 §5 会话 ID 规则 )
  "channelType": "single",          // single=单聊 / group=群聊
  "targetId": "10002",
  "channelName": "对方昵称或群名",
  "channelAvatar": "...",
  "isMuted": false, "isPinned": false,
  "lastMessageId": "456",
  "lastMessageType": "text",
  "lastMessageContent": "最后一条消息摘要",
  "lastMessageSenderId": "10001",
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
// senderId 为字符串

> 搜索为异步索引 ( 秒级延迟 )，且只返回定位信息 ( messageId/sentAt )，
> 消息本体经同步链路按 seq 拉取。

## 4. WebSocket 协议

**连接**: `ws://<gateway>:10071`，**text 文本帧** ( protojson 编码；浏览器
`WebSocket.onmessage` 的 `ev.data` 为 string，不是 Blob )

**帧结构** ( `Message` ):

```json
{
  "commandType": "COMMAND_TYPE_UPSTREAM",  // 指令，见下表 ( 枚举名字符串 )
  "messageId": "01JD...",                  // 帧唯一 ID ( 前端建议 ULID )，*_ACK 原样回填
  "body": "..."                            // 按 commandType 解释，JSON 序列化 + base64
}
```

| 指令 | 枚举名 | 方向 | body |
|---|---|---|---|
| HEARTBEAT | `COMMAND_TYPE_HEARTBEAT` | C→G | 空 |
| AUTH | `COMMAND_TYPE_AUTH` | C→G | `{"token": "<accessToken>"}` |
| AUTH_ACK | `COMMAND_TYPE_AUTH_ACK` | G→C | `{"success": true}` 或 `{"success": false, "errorMessage": "..."}` |
| UPSTREAM | `COMMAND_TYPE_UPSTREAM` | C→G | 业务消息 ( §5 ) |
| UPSTREAM_ACK | `COMMAND_TYPE_UPSTREAM_ACK` | G→C | `{"isSuccess": true, "timestampMs": 1761400000000}` |
| SYNC | `COMMAND_TYPE_SYNC` | C→G | `{"cursor": 42, "limit": 0}` |
| SYNC_ACK | `COMMAND_TYPE_SYNC_ACK` | G→C | `{"isSuccess": true, "response": {...}}` |
| CONVERSATION_SYNC | `COMMAND_TYPE_CONVERSATION_SYNC` | C→G | `{"conversationId": "g:100", "cursor": 0, "limit": 0}` |
| CONVERSATION_SYNC_ACK | `COMMAND_TYPE_CONVERSATION_SYNC_ACK` | G→C | `{"isSuccess": true, "response": {...}}` |
| NOTIFY | `COMMAND_TYPE_NOTIFY` | G→C | `{"latestSeq": 43}` |
| REDIRECT / RATE_LIMIT_EXCEEDED | `COMMAND_TYPE_REDIRECT` / `COMMAND_TYPE_RATE_LIMIT_EXCEEDED` | G→C | 预留 |

**规则**

- 连接后 **15s 内**必须完成 AUTH，否则被关闭；鉴权前其他指令一律拒绝；
- 心跳建议 30s 一次，连接空闲约 90s 会被服务端关闭；
- 客户端身份 ( businessId/userId/sessionId ) 由网关从 JWT 盖章，**前端自报身份不可信也不会被采用**；
- 下行帧的零值字段**照常输出** ( `isSuccess: false` / `hasMore: false` /
  `cursorExpired: false` 不会缺省 )，客户端按字段存在编写即可；
- NOTIFY 帧的 `messageId` 由服务端**每次下发独立生成**，与触发消息的
  UPSTREAM 帧 ID 无关——**不要**以 `NOTIFY.messageId` 做帧关联或消息去重
  ( 去重按 `latestSeq` 比对 / 收件箱 `seq` )。

## 5. 业务消息格式 ( UPSTREAM body )

body 是前端自定义 JSON，网关与后端不解释结构，但 backend 按以下字段读取目的地与内容:

```json
{
  "id": "01JD...",                      // 可选，前端本地消息 ID
  "sid": "10001",                       // 可选，仅展示用途 ( 服务端以盖章身份为准 )
  "destinationId": "134605228232032256", // 必填: 单聊=对方 userId；群聊=groupId
                                       // 建议传十进制字符串；数字形态亦兼容
  "destinationType": "single",          // 必填: single=单聊 / group=群聊
  "content": "你好",                     // 必填: 消息内容 ( 文本，UTF-8；搜索按全文索引 )
  "contentType": "text"                 // text / image；缺省按 text 处理
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

- **业务级失败有回执**: 消息被后端拒绝 ( 非好友 / 被屏蔽 / 非群成员 /
  目的地非法 / body 非法 ) 时，后端会向**发送者自己的收件箱**写一条
  `contentType: "system"` 的系统条目 ( 经 NOTIFY → SYNC 链路送达，多端可见 )，
  body 为:

  ```json
  {
    "systemType": "send_failed",
    "reason": "not_allowed",            // not_allowed / invalid_destination / invalid_body
    "upstreamId": "01JD...",            // 触发帧的 messageId，用于关联本地 pending 消息
    "content": "消息发送失败: ..."       // 可直接展示的文案
  }
  ```

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

**messageId 语义 ( 重要 )**: 同步链路条目里的 `messageId` 恒为发送端
UPSTREAM **帧的 messageId** ( 网关帧 ID )，与 body 内业务自定义的 `id` 字段
无关。多端自同步合并本地 pending 消息时按帧 ID 关联，不要按 body.id。

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
  CONVERSATION_SYNC 小游标变通 )。

## 12. 已支持 ( 此前版本未支持 )

- **业务级失败回执** ( §5 ): 非好友/被屏蔽/非群成员/目的地非法/body
  反序列化失败的 UPSTREAM，后端会向发送者收件箱写 `contentType: "system"`
  的系统条目，经同步链路送达，不再静默丢弃。

## 13. 兼容性说明

- 存量数据: 2026-10 之前落库的部分收件箱条目 body 为**纯文本** ( 旧测试
  客户端写入 )，客户端需按纯文本 content 兼容渲染；新数据 body 一律为
  UPSTREAM 原文 JSON；
- 顶号瞬间新连接的 AUTH_ACK 仍属尽力而为送达: 未收到 ACK 时按协议
  重复发送 AUTH 即可 ( 幂等 )。
