-- ============================================
-- 分库分表 SQL 初始化脚本
-- 分表数量: 4
-- 生成时间: 2026-01-26 16:02:03
-- 原始文件: ./03_reverse_index.sql
-- ============================================

-- ============================================
-- 表: contact_reverse_index (分表数: 4)
-- ============================================

-- 分表 0: contact_reverse_index_0
DROP TABLE IF EXISTS contact_reverse_index_0;
CREATE TABLE contact_reverse_index_0 (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,                           -- 用户 ID ( 联系人 ID )
    owner_id BIGINT NOT NULL,                          -- 用户 ID ( 谁的联系人 )

    created_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一联系人在同一所有者中只能有一条记录
    CONSTRAINT uk_contact_reverse_index_0 UNIQUE(user_id, owner_id)
);

COMMENT ON TABLE contact_reverse_index_0 IS '【 反向索引 】联系人反向索引表 ( 用于用户资料变更时，快速查询 "谁的联系人中有某个用户" )';
COMMENT ON COLUMN contact_reverse_index_0.id IS '主键 ID';
COMMENT ON COLUMN contact_reverse_index_0.user_id IS '联系人用户 ID ( 被添加的人 )';
COMMENT ON COLUMN contact_reverse_index_0.owner_id IS '联系人所有者 ID ( 谁的联系人 )';
COMMENT ON COLUMN contact_reverse_index_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_cori_contact_0 ON contact_reverse_index_0(user_id);

-- 分表 1: contact_reverse_index_1
DROP TABLE IF EXISTS contact_reverse_index_1;
CREATE TABLE contact_reverse_index_1 (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,                           -- 用户 ID ( 联系人 ID )
    owner_id BIGINT NOT NULL,                          -- 用户 ID ( 谁的联系人 )

    created_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一联系人在同一所有者中只能有一条记录
    CONSTRAINT uk_contact_reverse_index_1 UNIQUE(user_id, owner_id)
);

COMMENT ON TABLE contact_reverse_index_1 IS '【 反向索引 】联系人反向索引表 ( 用于用户资料变更时，快速查询 "谁的联系人中有某个用户" )';
COMMENT ON COLUMN contact_reverse_index_1.id IS '主键 ID';
COMMENT ON COLUMN contact_reverse_index_1.user_id IS '联系人用户 ID ( 被添加的人 )';
COMMENT ON COLUMN contact_reverse_index_1.owner_id IS '联系人所有者 ID ( 谁的联系人 )';
COMMENT ON COLUMN contact_reverse_index_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_cori_contact_1 ON contact_reverse_index_1(user_id);

-- 分表 2: contact_reverse_index_2
DROP TABLE IF EXISTS contact_reverse_index_2;
CREATE TABLE contact_reverse_index_2 (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,                           -- 用户 ID ( 联系人 ID )
    owner_id BIGINT NOT NULL,                          -- 用户 ID ( 谁的联系人 )

    created_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一联系人在同一所有者中只能有一条记录
    CONSTRAINT uk_contact_reverse_index_2 UNIQUE(user_id, owner_id)
);

COMMENT ON TABLE contact_reverse_index_2 IS '【 反向索引 】联系人反向索引表 ( 用于用户资料变更时，快速查询 "谁的联系人中有某个用户" )';
COMMENT ON COLUMN contact_reverse_index_2.id IS '主键 ID';
COMMENT ON COLUMN contact_reverse_index_2.user_id IS '联系人用户 ID ( 被添加的人 )';
COMMENT ON COLUMN contact_reverse_index_2.owner_id IS '联系人所有者 ID ( 谁的联系人 )';
COMMENT ON COLUMN contact_reverse_index_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_cori_contact_2 ON contact_reverse_index_2(user_id);

-- 分表 3: contact_reverse_index_3
DROP TABLE IF EXISTS contact_reverse_index_3;
CREATE TABLE contact_reverse_index_3 (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,                           -- 用户 ID ( 联系人 ID )
    owner_id BIGINT NOT NULL,                          -- 用户 ID ( 谁的联系人 )

    created_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一联系人在同一所有者中只能有一条记录
    CONSTRAINT uk_contact_reverse_index_3 UNIQUE(user_id, owner_id)
);

COMMENT ON TABLE contact_reverse_index_3 IS '【 反向索引 】联系人反向索引表 ( 用于用户资料变更时，快速查询 "谁的联系人中有某个用户" )';
COMMENT ON COLUMN contact_reverse_index_3.id IS '主键 ID';
COMMENT ON COLUMN contact_reverse_index_3.user_id IS '联系人用户 ID ( 被添加的人 )';
COMMENT ON COLUMN contact_reverse_index_3.owner_id IS '联系人所有者 ID ( 谁的联系人 )';
COMMENT ON COLUMN contact_reverse_index_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_cori_contact_3 ON contact_reverse_index_3(user_id);

-- ============================================
-- 表: conversation_reverse_index (分表数: 4)
-- ============================================

-- 分表 0: conversation_reverse_index_0
DROP TABLE IF EXISTS conversation_reverse_index_0;
CREATE TABLE conversation_reverse_index_0 (
    conversation_user_id BIGINT NOT NULL,                       -- 会话用户 ID
    conversation_target_id BIGINT NOT NULL,                     -- 会话目标 ID

    conversation_type VARCHAR(16) NOT NULL,                    -- 会话类型

    created_at BIGINT NOT NULL,

    PRIMARY KEY(conversation_user_id, conversation_type, conversation_target_id),

    -- 约束
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type_0 CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE conversation_reverse_index_0 IS '【 反向索引 】会话反向索引表 ( 用于用户资料变更时，快速查询 "谁的会话中有某个用户" )';
COMMENT ON COLUMN conversation_reverse_index_0.conversation_user_id IS '会话用户 ID';
COMMENT ON COLUMN conversation_reverse_index_0.conversation_target_id IS '会话目标 ID';
COMMENT ON COLUMN conversation_reverse_index_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_cri_conversation_0 ON conversation_reverse_index_0(conversation_user_id);

-- 分表 1: conversation_reverse_index_1
DROP TABLE IF EXISTS conversation_reverse_index_1;
CREATE TABLE conversation_reverse_index_1 (
    conversation_user_id BIGINT NOT NULL,                       -- 会话用户 ID
    conversation_target_id BIGINT NOT NULL,                     -- 会话目标 ID

    conversation_type VARCHAR(16) NOT NULL,                    -- 会话类型

    created_at BIGINT NOT NULL,

    PRIMARY KEY(conversation_user_id, conversation_type, conversation_target_id),

    -- 约束
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type_1 CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE conversation_reverse_index_1 IS '【 反向索引 】会话反向索引表 ( 用于用户资料变更时，快速查询 "谁的会话中有某个用户" )';
COMMENT ON COLUMN conversation_reverse_index_1.conversation_user_id IS '会话用户 ID';
COMMENT ON COLUMN conversation_reverse_index_1.conversation_target_id IS '会话目标 ID';
COMMENT ON COLUMN conversation_reverse_index_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_cri_conversation_1 ON conversation_reverse_index_1(conversation_user_id);

-- 分表 2: conversation_reverse_index_2
DROP TABLE IF EXISTS conversation_reverse_index_2;
CREATE TABLE conversation_reverse_index_2 (
    conversation_user_id BIGINT NOT NULL,                       -- 会话用户 ID
    conversation_target_id BIGINT NOT NULL,                     -- 会话目标 ID

    conversation_type VARCHAR(16) NOT NULL,                    -- 会话类型

    created_at BIGINT NOT NULL,

    PRIMARY KEY(conversation_user_id, conversation_type, conversation_target_id),

    -- 约束
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type_2 CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE conversation_reverse_index_2 IS '【 反向索引 】会话反向索引表 ( 用于用户资料变更时，快速查询 "谁的会话中有某个用户" )';
COMMENT ON COLUMN conversation_reverse_index_2.conversation_user_id IS '会话用户 ID';
COMMENT ON COLUMN conversation_reverse_index_2.conversation_target_id IS '会话目标 ID';
COMMENT ON COLUMN conversation_reverse_index_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_cri_conversation_2 ON conversation_reverse_index_2(conversation_user_id);

-- 分表 3: conversation_reverse_index_3
DROP TABLE IF EXISTS conversation_reverse_index_3;
CREATE TABLE conversation_reverse_index_3 (
    conversation_user_id BIGINT NOT NULL,                       -- 会话用户 ID
    conversation_target_id BIGINT NOT NULL,                     -- 会话目标 ID

    conversation_type VARCHAR(16) NOT NULL,                    -- 会话类型

    created_at BIGINT NOT NULL,

    PRIMARY KEY(conversation_user_id, conversation_type, conversation_target_id),

    -- 约束
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type_3 CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE conversation_reverse_index_3 IS '【 反向索引 】会话反向索引表 ( 用于用户资料变更时，快速查询 "谁的会话中有某个用户" )';
COMMENT ON COLUMN conversation_reverse_index_3.conversation_user_id IS '会话用户 ID';
COMMENT ON COLUMN conversation_reverse_index_3.conversation_target_id IS '会话目标 ID';
COMMENT ON COLUMN conversation_reverse_index_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_cri_conversation_3 ON conversation_reverse_index_3(conversation_user_id);

