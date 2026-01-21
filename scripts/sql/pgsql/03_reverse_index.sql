-- ============================================================================
-- 反向索引 ( Reverse Index ) - 加速数据同步
-- ============================================================================

-- 联系人反向索引表 ( 用于快速查询 - "谁的联系人中有某个用户" )
DROP TABLE IF EXISTS contact_reverse_index;
CREATE TABLE contact_reverse_index (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,                           -- 用户 ID ( 联系人 ID )
    owner_id BIGINT NOT NULL,                          -- 用户 ID ( 谁的联系人 )

    created_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一联系人在同一所有者中只能有一条记录
    CONSTRAINT uk_contact_reverse_index UNIQUE(user_id, owner_id)
);

COMMENT ON TABLE contact_reverse_index IS '【 反向索引 】联系人反向索引表 ( 用于用户资料变更时，快速查询 "谁的联系人中有某个用户" )';
COMMENT ON COLUMN contact_reverse_index.id IS '主键 ID';
COMMENT ON COLUMN contact_reverse_index.user_id IS '联系人用户 ID ( 被添加的人 )';
COMMENT ON COLUMN contact_reverse_index.owner_id IS '联系人所有者 ID ( 谁的联系人 )';
COMMENT ON COLUMN contact_reverse_index.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_cori_contact ON contact_reverse_index(user_id);

-- 使用场景示例
-- 用户 A 修改了昵称/头像，需要更新哪些人的联系人视图？
-- SELECT owner_user_id FROM contact_reverse_index WHERE contact_user_id = A


-- 会话反向索引表 ( 用于快速查询 - "谁的会话中有某个用户" )
DROP TABLE IF EXISTS conversation_reverse_index;
CREATE TABLE conversation_reverse_index (
    conversation_user_id BIGINT NOT NULL,                       -- 会话用户 ID
    conversation_target_id BIGINT NOT NULL,                     -- 会话目标 ID

    conversation_type VARCHAR(16) NOT NULL,                    -- 会话类型

    created_at BIGINT NOT NULL,

    PRIMARY KEY(conversation_user_id, conversation_type, conversation_target_id),

    -- 约束
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE conversation_reverse_index IS '【 反向索引 】会话反向索引表 ( 用于用户资料变更时，快速查询 "谁的会话中有某个用户" )';
COMMENT ON COLUMN conversation_reverse_index.conversation_user_id IS '会话用户 ID';
COMMENT ON COLUMN conversation_reverse_index.conversation_target_id IS '会话目标 ID';
COMMENT ON COLUMN conversation_reverse_index.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_cri_conversation ON conversation_reverse_index(conversation_user_id);

-- 使用场景示例：
-- 用户 A 修改了昵称/头像，需要更新哪些人的会话视图？
-- SELECT owner_user_id, channel_id FROM conversation_reverse_index WHERE peer_user_id = A
