-- ============================================================================
-- 消息链路表 ( 落库即事实，seq 游标同步 )
-- 契约见 synp-api proto/message/v1 与 api/README.md 的同步模型。
-- ============================================================================

-- 收件箱 seq 分配表 ( 以 user_id 为分片键 )
-- seq 在 ( business_id, user_id ) 维度严格单调递增，行锁原子自增。
-- seq 允许有空洞 ( 消费重投时幂等跳过会浪费序号 )，只需保证单调。
DROP TABLE IF EXISTS inbox_seq;
CREATE TABLE inbox_seq (
    business_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    seq BIGINT NOT NULL DEFAULT 0,

    updated_at BIGINT NOT NULL,

    PRIMARY KEY (business_id, user_id)
);

COMMENT ON TABLE inbox_seq IS '收件箱 seq 分配表';
COMMENT ON COLUMN inbox_seq.business_id IS '业务 ID';
COMMENT ON COLUMN inbox_seq.user_id IS '用户 ID';
COMMENT ON COLUMN inbox_seq.seq IS '该用户收件箱当前最大 seq';
COMMENT ON COLUMN inbox_seq.updated_at IS '更新时间戳 ( Unix 毫秒值 )';


-- 收件箱条目表 ( 以 user_id 为分片键 )
-- 写扩散条目: 单聊/小群消息本体内联，一次 SYNC 即可渲染；
-- 读扩散指针: 大群会话更新，需再发起 CONVERSATION_SYNC 拉取正文。
DROP TABLE IF EXISTS inbox_entry;
CREATE TABLE inbox_entry (
    business_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    seq BIGINT NOT NULL,

    -- 条目类型 ( 1=写扩散条目 ( 消息内联 ) / 2=读扩散指针 ( 会话更新 ) )
    kind SMALLINT NOT NULL,

    -- 关联消息 ID ( 链路 at-least-once，重投按 ( user_id, message_id ) 幂等消化 )。
    message_id VARCHAR(64) NOT NULL,

    conversation_id VARCHAR(64) NOT NULL,
    -- 读扩散指针: 会话 timeline 当前最大 seq。
    conversation_latest_seq BIGINT NOT NULL DEFAULT 0,

    -- 写扩散条目: 消息本体内联。
    sender_id BIGINT NOT NULL DEFAULT 0,
    body BYTEA,
    content_type VARCHAR(32) NOT NULL DEFAULT '',
    sent_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,

    PRIMARY KEY (business_id, user_id, seq),

    CONSTRAINT uk_inbox_entry_message UNIQUE(business_id, user_id, message_id),
    CONSTRAINT chk_inbox_entry_kind CHECK (kind IN (1, 2))
);

COMMENT ON TABLE inbox_entry IS '收件箱条目表 ( 写扩散条目 / 读扩散指针 )';
COMMENT ON COLUMN inbox_entry.business_id IS '业务 ID';
COMMENT ON COLUMN inbox_entry.user_id IS '收件人用户 ID';
COMMENT ON COLUMN inbox_entry.seq IS '收件箱序号 ( 客户端以此去重并推进游标 )';
COMMENT ON COLUMN inbox_entry.kind IS '条目类型 ( 1=消息内联 / 2=会话更新指针 )';
COMMENT ON COLUMN inbox_entry.message_id IS '关联消息 ID ( 幂等去重键之一 )';
COMMENT ON COLUMN inbox_entry.conversation_id IS '会话 ID ( 单聊 s:min:max / 群聊 g:group_id )';
COMMENT ON COLUMN inbox_entry.conversation_latest_seq IS '读扩散指针: 会话 timeline 当前最大 seq';
COMMENT ON COLUMN inbox_entry.sender_id IS '写扩散条目: 发送者 ID';
COMMENT ON COLUMN inbox_entry.body IS '写扩散条目: 消息本体 ( 与 UPSTREAM 帧 body 一致 )';
COMMENT ON COLUMN inbox_entry.content_type IS '写扩散条目: 消息类型';
COMMENT ON COLUMN inbox_entry.sent_at IS '写扩散条目: 落库时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN inbox_entry.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 保留窗口: 按 created_at 定期清理过期条目 ( cursor 超出窗口时 SYNC 返回 cursor_expired )。
CREATE INDEX idx_inbox_entry_created ON inbox_entry(created_at);


-- 消息正本表 ( 以 conversation_id 为分片键 )
-- 会话维度全量保留 ( 不随收件箱窗口清理 )，供读扩散 timeline、ES 索引与审计使用。
DROP TABLE IF EXISTS message;
CREATE TABLE message (
    id BIGINT PRIMARY KEY,

    message_id VARCHAR(64) NOT NULL,
    business_id BIGINT NOT NULL,

    conversation_id VARCHAR(64) NOT NULL,
    -- 会话内序号 ( 群聊读扩散 timeline 用，单聊恒为 0 )。
    conversation_seq BIGINT NOT NULL DEFAULT 0,

    sender_id BIGINT NOT NULL,

    body BYTEA NOT NULL,
    content_type VARCHAR(32) NOT NULL DEFAULT '',

    sent_at BIGINT NOT NULL,

    created_at BIGINT NOT NULL,

    CONSTRAINT uk_message_message_id UNIQUE(business_id, message_id)
);

COMMENT ON TABLE message IS '消息正本表';
COMMENT ON COLUMN message.id IS '主键 ID ( 雪花 ID )';
COMMENT ON COLUMN message.message_id IS '消息 ID ( 网关 UPSTREAM 帧的 message_id，幂等去重键 )';
COMMENT ON COLUMN message.business_id IS '业务 ID';
COMMENT ON COLUMN message.conversation_id IS '会话 ID ( 单聊 s:min:max / 群聊 g:group_id )';
COMMENT ON COLUMN message.conversation_seq IS '会话内序号 ( 群聊读扩散 timeline，由后端单调递增分配 )';
COMMENT ON COLUMN message.sender_id IS '发送者 ID ( 网关盖章，前端自报身份不可信 )';
COMMENT ON COLUMN message.body IS '消息本体 ( 与 UPSTREAM 帧 body 一致，后端不解释 )';
COMMENT ON COLUMN message.content_type IS '消息类型';
COMMENT ON COLUMN message.sent_at IS '落库时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN message.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 会话 timeline 查询 ( 读扩散 CONVERSATION_SYNC: ( conversation_id, conversation_seq ) 范围扫描 )。
CREATE INDEX idx_message_conversation ON message(conversation_id, conversation_seq);
