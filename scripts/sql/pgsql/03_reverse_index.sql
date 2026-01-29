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
-- SELECT owner_id FROM contact_reverse_index WHERE user_id = A


-- 频道反向索引表 ( 用于快速查询 - "谁的频道中有某个用户" )
DROP TABLE IF EXISTS channel_reverse_index;
CREATE TABLE channel_reverse_index (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,                       -- 频道用户 ID
    target_id BIGINT NOT NULL,                     -- 频道目标 ID

    channel_type VARCHAR(16) NOT NULL,        -- 频道类型

    created_at BIGINT NOT NULL,

    PRIMARY KEY(user_id, target_id, channel_type),

    -- 约束
    -- 频道类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_channel_type CHECK (channel_type IN ('single', 'group'))
);

COMMENT ON TABLE channel_reverse_index IS '【 反向索引 】频道反向索引表 ( 用于用户资料变更时，快速查询 "谁的频道中有某个用户" )';
COMMENT ON COLUMN channel_reverse_index.id IS '主键 ID';
COMMENT ON COLUMN channel_reverse_index.user_id IS '频道用户 ID';
COMMENT ON COLUMN channel_reverse_index.target_id IS '频道目标 ID';
COMMENT ON COLUMN channel_reverse_index.channel_type IS '频道类型';
COMMENT ON COLUMN channel_reverse_index.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_cri_channel ON channel_reverse_index(user_id);

-- 使用场景示例：
-- 用户 A 修改了昵称/头像，需要更新哪些人的会话视图？
-- SELECT user_id, target_id FROM channel_reverse_index WHERE user_id = A
