# CQRS ( Command Query Responsibility Segregation ) 架构说明

## 架构说明

### CQRS 核心思想

- **【 写入侧 Write Side 】** 标准化设计，保证数据一致性
    - `biz_user` - 用户表
    - `channel` - 频道表
    - `channel_member` - 频道成员表
    - `user_contact` - 用户联系人表

- **【 读取侧 Read Side 】** 宽表设计，冗余数据，优化查询性能
    - `user_conversation_view` - 用户会话视图表 ( 按 user_id 分表 )
    - `user_contact_view` - 用户联系人视图表 ( 按 user_id 分表 )

- **【 反向索引 Reverse Index 】** 加速数据同步
    - `conversation_reverse_index` - 会话反向索引 ( 谁的会话中有某个用户 )
    - `contact_reverse_index` - 联系人反向索引 ( 谁的联系人中有某个用户 )
- **【 版本号机制 】** info_ver 字段 ( 用于同步检测 )
    - 检测数据是否需要同步
        ```
        if (view.conversation_info_ver < user.info_ver) {
            // 需要同步更新
        }
        ```
    - 避免重复更新
        ```
        UPDATE user_conversation_view_{shard}
        SET peer_nickname = ?, conversation_info_ver = ?
        WHERE peer_user_id = ? AND conversation_info_ver < ?;
        ```
    - 客户端缓存失效
        客户端定期检查版本号，如果不一致则刷新数据



### 应用层双写 vs CDC

| 特性 | 应用层双写 | CDC ( Debezium + Kafka ) |
|-----|-----------|----------------------|
| 实时性 | 高 ( 同步写入 ) | 中 (异步延迟 ) |
| 一致性 | 强一致性 ( 事务保证 ) | 最终一致性 |
| 性能影响 | 写入延迟增加 | 写入性能好 |
| 复杂度 | 代码复杂度高 | 基础设施复杂度高 |
| 适用场景 | 小规模、要求强一致性 | 大规模、可容忍延迟 |

## 数据流实现

### 1. 添加好友

**业务场景** 用户 A 添加用户 B 为好友

**写入流程**

```
┌─────────────────────────────────────────────────────────────────┐
│                  Phase 1 - Approved                             │
│                ( B 分片 - 直接写业务表 )                           │
│                                                                 │
│       UPDATE contact_application SET status='approved           │
│                         │                                       │
│                         ▼                                       │
│                   Debezium CDC                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ Kafka: db.contact_application
┌─────────────────────────────────────────────────────────────────┐
│              Phase 2 - Contact Processor                        │
│                 ( Comsume CDC event )                           │    
│                                                                 │
│           status: pending → approved                            │
│                         │                                       │
│         ┌───────────────┴───────────────┐                       │
│         ▼                               ▼                       │
│   写入 A 分片 ( A → B )             写入 B 分片 ( B → A )          │
│   • user_contact                   • user_contact               │
│   • contact_reverse_index          • contact_reverse_index      │
│   • user_contact_view              • user_contact_view          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ Kafka: db.user_contact
┌─────────────────────────────────────────────────────────────────┐
│             Phase 2 - Conversation Processor                    │
│                  ( Comsume CDC event )                          │    
│                                                                 │
│               聚合双向联系人创建事件，创建会话                        │
│         ┌───────────────┴───────────────┐                       │
│         ▼                               ▼                       │
│   写入 A 分片                       写入 B 分片                    │
│   • user_conversation_view         • user_conversation_view     │
│   • conversation_reverse_index     • conversation_reverse_index │
└─────────────────────────────────────────────────────────────────┘
```

**关键点**
- 所有操作在同一个数据库事务中完成
- 保证强一致性，要么全部成功，要么全部回滚
- 需要考虑分表场景 ( A 和 B 可能在不同分片 )

### 2. 发送消息

**业务场景** 用户在频道中发送消息

**写入流程**
```
1. 写入消息 ( MongoDB )

2. 更新会话视图 ( PostgreSQL 事务 )
    - 查询频道所有成员 
        SELECT user_id FROM channel_member WHERE channel_id = ?
    - 批量更新会话视图
        UPDATE user_conversation_view SET
            last_message_id = ?,
            last_message_content = ?,
            last_message_sender_id = ?,
            last_message_sender_name = ?,
            last_message_time = ?,
            unread_count = unread_count + 1
        WHERE user_id IN (成员列表) AND channel_id = ?
```

**关键点**
- 消息存储 ( MongoDB ) 和会话视图更新 ( PostgreSQL ) 分离
- 可以采用异步更新 ( 消息队列 ) 降低延迟
- 需要处理大群组场景 ( 成员数量较多时可以批量更新 )

### 3. 用户修改资料

**业务场景** 用户修改昵称/头像

**写入流程**
```
1. 写入侧
    UPDATE biz_user SET
        nickname = ?,
        avatar = ?,
        info_ver = info_ver + 1
    WHERE id = ?

2. 通过反向索引找到需要更新的记录
    - 更新会话视图
        SELECT owner_user_id, channel_id FROM conversation_reverse_index 
        WHERE peer_user_id = ?
      
    - 对每条记录
        UPDATE user_conversation_view SET
            peer_nickname = ?,
            peer_avatar = ?,
            conversation_name = COALESCE(remark_name, ?),
            conversation_avatar = ?,
            conversation_info_ver = conversation_info_ver + 1
        WHERE user_id = ? AND channel_id = ?
   
    - 更新联系人视图
        SELECT owner_user_id FROM contact_reverse_index 
        WHERE contact_user_id = ?
      
    - 对每条记录
        UPDATE user_contact_view SET
            contact_nickname = ?,
            contact_avatar = ?,
            contact_info_ver = contact_info_ver + 1
        WHERE user_id = ? AND contact_id = ?
```

**关键点**
- 反向索引的作用：快速找到所有需要更新的视图记录
- 版本号机制：避免重复更新，支持客户端缓存失效
- 需要考虑大量好友/会话场景 ( 可以异步批量更新 )

### 4. 群名称/头像修改

**业务场景** 修改群名称或群头像

**写入流程**
```
1. 写入侧
    UPDATE channel SET
        channel_name = ?,
        channel_avatar = ?,
        channel_info_ver = channel_info_ver + 1
    WHERE id = ?

2. 更新所有成员的会话视图
    - 查询群成员
        SELECT user_id FROM channel_member WHERE channel_id = ?
   
    - 批量更新
        UPDATE user_conversation_view SET
            conversation_name = ?,
            conversation_avatar = ?,
            conversation_info_ver = ?
        WHERE user_id IN ( 成员列表 ) AND channel_id = ?
```

**关键点**
- 只需要更新会话视图，不涉及联系人视图
- 大群组场景下，可能需要异步更新 (消息队列)
- 版本号机制保证数据同步

## 查询示例

### 1. 获取会话列表
```
-- 单表查询，性能极好
SELECT
    channel_id,
    conversation_type,
    conversation_name,
    conversation_avatar,
    last_message_content,
    last_message_sender_name,
    last_message_time,
    unread_count,
    is_pinned,
    is_muted
FROM user_conversation_view_{shard}
WHERE user_id = ?
    AND closed_at = 0
    AND is_hidden = FALSE
ORDER BY is_pinned DESC, last_message_time DESC
LIMIT 50;
```

### 2. 获取通讯录列表
```
-- 单表查询，性能极好
SELECT
    contact_id,
    contact_nickname,
    contact_avatar,
    remark_name,
    group_name,
    is_starred
FROM user_contact_view_{shard}
WHERE user_id = ?
    AND deleted_at = 0
ORDER BY is_starred DESC, contact_nickname
LIMIT 1000;

```

### 3. 搜索用户
```
-- 按昵称搜索用户 ( 模糊匹配 )
SELECT 
    id, 
    nickname, 
    avatar, 
    gender, 
    tagline
FROM biz_user
WHERE nickname LIKE '%关键词%'
    AND deleted_at = 0
    AND user_status = 'active'
ORDER BY info_ver DESC  -- 最近活跃的用户优先
LIMIT 20;
```

## Outbox 模式 ⭐

### 为什么需要 Outbox 模式？

Outbox 模式用于保证**数据库写入和消息发送的原子性**

| 问题场景 | 后果 | Outbox 解决方案 |
|---------|------|---------------|
| 数据库写入成功，Kafka 发送失败 | 数据不一致 | ✅ 事务保证 |
| Kafka 发送成功，数据库回滚 | 数据不一致 | ✅ 事务保证 |
| Kafka 暂时不可用 | 消息丢失 | ✅ 持久化到 Outbox 表 |
| 发送失败需要重试 | 没有重试机制 | ✅ Processor 自动重试 |

### 适用场景分析

| 数据流 | 是否推荐 Outbox | 原因 |
|-------|----------------|------|
| 添加好友 | ❌ 不需要 | 同库事务即可 |
| 发送消息 | ✅ 强烈推荐 | 跨库 ( MongoDB + PostgreSQL ) |
| 用户修改资料 | ✅ 推荐 | 异步更新 Kafka |
| 群信息修改 | ✅ 推荐 | 异步更新 Kafka |
