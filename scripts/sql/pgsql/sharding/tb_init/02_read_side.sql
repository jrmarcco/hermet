-- ============================================
-- 分库分表 SQL 初始化脚本
-- 分表数量: 4
-- 生成时间: 2026-01-26 16:02:03
-- 原始文件: ./02_read_side.sql
-- ============================================

-- ============================================
-- 表: user_contact_view (分表数: 4)
-- ============================================

-- 分表 0: user_contact_view_0
DROP TABLE IF EXISTS user_contact_view_0;
CREATE TABLE user_contact_view_0 (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,
    contact_id BIGINT NOT NULL,                                 -- 联系人 user_id

    -- 联系人信息 ( 冗余自 user 表 )
    contact_nickname VARCHAR(64) NOT NULL DEFAULT '',           -- 联系人昵称 ( 冗余 )
    contact_avatar VARCHAR(256) NOT NULL DEFAULT '',            -- 联系人头像 ( 冗余 )
    contact_gender VARCHAR(16) NOT NULL DEFAULT 'unknown',      -- 联系人性别 ( unknown=未知 / male=男 / female=女 )
    contact_info_ver INT NOT NULL DEFAULT 1,                    -- 【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )

    -- 好友关系信息 ( 冗余自 user_contact 表 )
    remark_name VARCHAR(64) NOT NULL DEFAULT '',                -- 备注名
    tags VARCHAR(128)[] DEFAULT '{}',                           -- 标签

    -- 状态
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                  -- 星标
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,                  -- 黑名单

    -- 扩展信息 ( 冗余自 user 表 )
    mobile VARCHAR(16) NOT NULL DEFAULT '',                     -- 手机号
    source VARCHAR(16) NOT NULL DEFAULT 'search',               -- 添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )

    added_at BIGINT NOT NULL,
    deleted_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一联系人中只能有一条记录
    CONSTRAINT uk_user_contact_view_0 UNIQUE(user_id, contact_id),
    -- 联系人性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_contact_gender_0 CHECK (contact_gender IN ('unknown', 'male', 'female')),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source_0 CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE user_contact_view_0 IS '【 读取侧 】用户联系人视图表 ( 用于通讯录展示 )';
COMMENT ON COLUMN user_contact_view_0.id IS '主键 ID';
COMMENT ON COLUMN user_contact_view_0.user_id IS '用户 ID';
COMMENT ON COLUMN user_contact_view_0.contact_id IS '联系人 ID ( 好友 ID )';
COMMENT ON COLUMN user_contact_view_0.contact_nickname IS '联系人昵称 ( 冗余自 user 表 )';
COMMENT ON COLUMN user_contact_view_0.contact_avatar IS '联系人头像 ( 冗余自 user 表 )';
COMMENT ON COLUMN user_contact_view_0.contact_gender IS '联系人性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN user_contact_view_0.contact_info_ver IS '【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN user_contact_view_0.remark_name IS '备注名';
COMMENT ON COLUMN user_contact_view_0.tags IS '标签数组';
COMMENT ON COLUMN user_contact_view_0.is_starred IS '是否星标';
COMMENT ON COLUMN user_contact_view_0.is_blocked IS '是否拉黑';
COMMENT ON COLUMN user_contact_view_0.mobile IS '手机号 ( 冗余 )';
COMMENT ON COLUMN user_contact_view_0.source IS '添加来源';
COMMENT ON COLUMN user_contact_view_0.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_0.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_0.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_ucov_list_0 ON user_contact_view_0(
    user_id,
    is_starred DESC,
    contact_nickname
) WHERE deleted_at = 0;

CREATE INDEX idx_ucov_starred_0 ON user_contact_view_0(
    user_id,
    contact_nickname
) WHERE is_starred = TRUE AND deleted_at = 0;

CREATE INDEX idx_ucov_initial_0 ON user_contact_view_0(
    user_id,
    LEFT(contact_nickname, 1),
    contact_nickname
) WHERE deleted_at = 0;

-- 分表 1: user_contact_view_1
DROP TABLE IF EXISTS user_contact_view_1;
CREATE TABLE user_contact_view_1 (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,
    contact_id BIGINT NOT NULL,                                 -- 联系人 user_id

    -- 联系人信息 ( 冗余自 user 表 )
    contact_nickname VARCHAR(64) NOT NULL DEFAULT '',           -- 联系人昵称 ( 冗余 )
    contact_avatar VARCHAR(256) NOT NULL DEFAULT '',            -- 联系人头像 ( 冗余 )
    contact_gender VARCHAR(16) NOT NULL DEFAULT 'unknown',      -- 联系人性别 ( unknown=未知 / male=男 / female=女 )
    contact_info_ver INT NOT NULL DEFAULT 1,                    -- 【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )

    -- 好友关系信息 ( 冗余自 user_contact 表 )
    remark_name VARCHAR(64) NOT NULL DEFAULT '',                -- 备注名
    tags VARCHAR(128)[] DEFAULT '{}',                           -- 标签

    -- 状态
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                  -- 星标
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,                  -- 黑名单

    -- 扩展信息 ( 冗余自 user 表 )
    mobile VARCHAR(16) NOT NULL DEFAULT '',                     -- 手机号
    source VARCHAR(16) NOT NULL DEFAULT 'search',               -- 添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )

    added_at BIGINT NOT NULL,
    deleted_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一联系人中只能有一条记录
    CONSTRAINT uk_user_contact_view_1 UNIQUE(user_id, contact_id),
    -- 联系人性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_contact_gender_1 CHECK (contact_gender IN ('unknown', 'male', 'female')),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source_1 CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE user_contact_view_1 IS '【 读取侧 】用户联系人视图表 ( 用于通讯录展示 )';
COMMENT ON COLUMN user_contact_view_1.id IS '主键 ID';
COMMENT ON COLUMN user_contact_view_1.user_id IS '用户 ID';
COMMENT ON COLUMN user_contact_view_1.contact_id IS '联系人 ID ( 好友 ID )';
COMMENT ON COLUMN user_contact_view_1.contact_nickname IS '联系人昵称 ( 冗余自 user 表 )';
COMMENT ON COLUMN user_contact_view_1.contact_avatar IS '联系人头像 ( 冗余自 user 表 )';
COMMENT ON COLUMN user_contact_view_1.contact_gender IS '联系人性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN user_contact_view_1.contact_info_ver IS '【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN user_contact_view_1.remark_name IS '备注名';
COMMENT ON COLUMN user_contact_view_1.tags IS '标签数组';
COMMENT ON COLUMN user_contact_view_1.is_starred IS '是否星标';
COMMENT ON COLUMN user_contact_view_1.is_blocked IS '是否拉黑';
COMMENT ON COLUMN user_contact_view_1.mobile IS '手机号 ( 冗余 )';
COMMENT ON COLUMN user_contact_view_1.source IS '添加来源';
COMMENT ON COLUMN user_contact_view_1.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_1.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_1.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_ucov_list_1 ON user_contact_view_1(
    user_id,
    is_starred DESC,
    contact_nickname
) WHERE deleted_at = 0;

CREATE INDEX idx_ucov_starred_1 ON user_contact_view_1(
    user_id,
    contact_nickname
) WHERE is_starred = TRUE AND deleted_at = 0;

CREATE INDEX idx_ucov_initial_1 ON user_contact_view_1(
    user_id,
    LEFT(contact_nickname, 1),
    contact_nickname
) WHERE deleted_at = 0;

-- 分表 2: user_contact_view_2
DROP TABLE IF EXISTS user_contact_view_2;
CREATE TABLE user_contact_view_2 (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,
    contact_id BIGINT NOT NULL,                                 -- 联系人 user_id

    -- 联系人信息 ( 冗余自 user 表 )
    contact_nickname VARCHAR(64) NOT NULL DEFAULT '',           -- 联系人昵称 ( 冗余 )
    contact_avatar VARCHAR(256) NOT NULL DEFAULT '',            -- 联系人头像 ( 冗余 )
    contact_gender VARCHAR(16) NOT NULL DEFAULT 'unknown',      -- 联系人性别 ( unknown=未知 / male=男 / female=女 )
    contact_info_ver INT NOT NULL DEFAULT 1,                    -- 【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )

    -- 好友关系信息 ( 冗余自 user_contact 表 )
    remark_name VARCHAR(64) NOT NULL DEFAULT '',                -- 备注名
    tags VARCHAR(128)[] DEFAULT '{}',                           -- 标签

    -- 状态
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                  -- 星标
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,                  -- 黑名单

    -- 扩展信息 ( 冗余自 user 表 )
    mobile VARCHAR(16) NOT NULL DEFAULT '',                     -- 手机号
    source VARCHAR(16) NOT NULL DEFAULT 'search',               -- 添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )

    added_at BIGINT NOT NULL,
    deleted_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一联系人中只能有一条记录
    CONSTRAINT uk_user_contact_view_2 UNIQUE(user_id, contact_id),
    -- 联系人性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_contact_gender_2 CHECK (contact_gender IN ('unknown', 'male', 'female')),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source_2 CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE user_contact_view_2 IS '【 读取侧 】用户联系人视图表 ( 用于通讯录展示 )';
COMMENT ON COLUMN user_contact_view_2.id IS '主键 ID';
COMMENT ON COLUMN user_contact_view_2.user_id IS '用户 ID';
COMMENT ON COLUMN user_contact_view_2.contact_id IS '联系人 ID ( 好友 ID )';
COMMENT ON COLUMN user_contact_view_2.contact_nickname IS '联系人昵称 ( 冗余自 user 表 )';
COMMENT ON COLUMN user_contact_view_2.contact_avatar IS '联系人头像 ( 冗余自 user 表 )';
COMMENT ON COLUMN user_contact_view_2.contact_gender IS '联系人性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN user_contact_view_2.contact_info_ver IS '【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN user_contact_view_2.remark_name IS '备注名';
COMMENT ON COLUMN user_contact_view_2.tags IS '标签数组';
COMMENT ON COLUMN user_contact_view_2.is_starred IS '是否星标';
COMMENT ON COLUMN user_contact_view_2.is_blocked IS '是否拉黑';
COMMENT ON COLUMN user_contact_view_2.mobile IS '手机号 ( 冗余 )';
COMMENT ON COLUMN user_contact_view_2.source IS '添加来源';
COMMENT ON COLUMN user_contact_view_2.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_2.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_2.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_ucov_list_2 ON user_contact_view_2(
    user_id,
    is_starred DESC,
    contact_nickname
) WHERE deleted_at = 0;

CREATE INDEX idx_ucov_starred_2 ON user_contact_view_2(
    user_id,
    contact_nickname
) WHERE is_starred = TRUE AND deleted_at = 0;

CREATE INDEX idx_ucov_initial_2 ON user_contact_view_2(
    user_id,
    LEFT(contact_nickname, 1),
    contact_nickname
) WHERE deleted_at = 0;

-- 分表 3: user_contact_view_3
DROP TABLE IF EXISTS user_contact_view_3;
CREATE TABLE user_contact_view_3 (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,
    contact_id BIGINT NOT NULL,                                 -- 联系人 user_id

    -- 联系人信息 ( 冗余自 user 表 )
    contact_nickname VARCHAR(64) NOT NULL DEFAULT '',           -- 联系人昵称 ( 冗余 )
    contact_avatar VARCHAR(256) NOT NULL DEFAULT '',            -- 联系人头像 ( 冗余 )
    contact_gender VARCHAR(16) NOT NULL DEFAULT 'unknown',      -- 联系人性别 ( unknown=未知 / male=男 / female=女 )
    contact_info_ver INT NOT NULL DEFAULT 1,                    -- 【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )

    -- 好友关系信息 ( 冗余自 user_contact 表 )
    remark_name VARCHAR(64) NOT NULL DEFAULT '',                -- 备注名
    tags VARCHAR(128)[] DEFAULT '{}',                           -- 标签

    -- 状态
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                  -- 星标
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,                  -- 黑名单

    -- 扩展信息 ( 冗余自 user 表 )
    mobile VARCHAR(16) NOT NULL DEFAULT '',                     -- 手机号
    source VARCHAR(16) NOT NULL DEFAULT 'search',               -- 添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )

    added_at BIGINT NOT NULL,
    deleted_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一联系人中只能有一条记录
    CONSTRAINT uk_user_contact_view_3 UNIQUE(user_id, contact_id),
    -- 联系人性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_contact_gender_3 CHECK (contact_gender IN ('unknown', 'male', 'female')),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source_3 CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE user_contact_view_3 IS '【 读取侧 】用户联系人视图表 ( 用于通讯录展示 )';
COMMENT ON COLUMN user_contact_view_3.id IS '主键 ID';
COMMENT ON COLUMN user_contact_view_3.user_id IS '用户 ID';
COMMENT ON COLUMN user_contact_view_3.contact_id IS '联系人 ID ( 好友 ID )';
COMMENT ON COLUMN user_contact_view_3.contact_nickname IS '联系人昵称 ( 冗余自 user 表 )';
COMMENT ON COLUMN user_contact_view_3.contact_avatar IS '联系人头像 ( 冗余自 user 表 )';
COMMENT ON COLUMN user_contact_view_3.contact_gender IS '联系人性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN user_contact_view_3.contact_info_ver IS '【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN user_contact_view_3.remark_name IS '备注名';
COMMENT ON COLUMN user_contact_view_3.tags IS '标签数组';
COMMENT ON COLUMN user_contact_view_3.is_starred IS '是否星标';
COMMENT ON COLUMN user_contact_view_3.is_blocked IS '是否拉黑';
COMMENT ON COLUMN user_contact_view_3.mobile IS '手机号 ( 冗余 )';
COMMENT ON COLUMN user_contact_view_3.source IS '添加来源';
COMMENT ON COLUMN user_contact_view_3.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_3.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_view_3.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_ucov_list_3 ON user_contact_view_3(
    user_id,
    is_starred DESC,
    contact_nickname
) WHERE deleted_at = 0;

CREATE INDEX idx_ucov_starred_3 ON user_contact_view_3(
    user_id,
    contact_nickname
) WHERE is_starred = TRUE AND deleted_at = 0;

CREATE INDEX idx_ucov_initial_3 ON user_contact_view_3(
    user_id,
    LEFT(contact_nickname, 1),
    contact_nickname
) WHERE deleted_at = 0;

-- ============================================
-- 表: user_conversation_view (分表数: 4)
-- ============================================

-- 分表 0: user_conversation_view_0
DROP TABLE IF EXISTS user_conversation_view_0;
CREATE TABLE user_conversation_view_0 (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,

    -- 会话基础信息 ( 冗余 )
    conversation_type VARCHAR(16) NOT NULL,
    conversation_name VARCHAR(128) NOT NULL,                    -- 冗余 ( 群名 或 对方昵称 / 备注名 )
    conversation_remark VARCHAR(128) NOT NULL DEFAULT '',       -- 冗余 ( 备注名 )
    conversation_avatar VARCHAR(256) NOT NULL DEFAULT '',       -- 冗余 ( 群头像 或 对方头像 )
    conversation_target_id BIGINT NOT NULL,                     -- 会话目标 ID ( 用户 ID or 频道 ID )
    conversation_info_ver INT NOT NULL DEFAULT 1,               -- 【 CQRS 关键字段 】版本号 ( 用于检测 channel / user 信息变更 )

    -- 个性化设置
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,                    -- 免打扰
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,                   -- 置顶
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,                   -- 隐藏
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                  -- 星标

    -- 最后消息快照 ( 冗余 - 用于会话列表展示 )
    last_message_id BIGINT NOT NULL DEFAULT 0,
    last_message_type VARCHAR(32) NOT NULL DEFAULT '',          -- 消息类型 ( text/image/voice/video 等 )
    last_message_content VARCHAR(512) NOT NULL DEFAULT '',      -- 消息内容摘要
    last_message_sender_id BIGINT NOT NULL DEFAULT 0,
    last_message_sender_name VARCHAR(64) NOT NULL DEFAULT '',   -- 发送者昵称 ( 冗余 )
    last_message_time BIGINT NOT NULL DEFAULT 0,

    -- 未读数 ( 冗余 - 实时更新 )
    unread_count INT NOT NULL DEFAULT 0,
    mention_count INT NOT NULL DEFAULT 0,                       -- @我的消息数

    -- 会话状态
    opened_at BIGINT NOT NULL,                                  -- 会话创建时间
    closed_at BIGINT NOT NULL DEFAULT 0,                        -- 会话删除时间 ( 0 表示未删除 )

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一会话目标中只能有一条记录（区分单聊和群聊）
    CONSTRAINT uk_user_conversation_0 UNIQUE(user_id, conversation_type, conversation_target_id),
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type_0 CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE user_conversation_view_0 IS '【 读取侧 】用户会话视图表 ( 用于会话列表展示，冗余数据 )';
COMMENT ON COLUMN user_conversation_view_0.id IS '主键 ID';
COMMENT ON COLUMN user_conversation_view_0.user_id IS '用户 ID';
COMMENT ON COLUMN user_conversation_view_0.conversation_type IS '会话类型 ( single=单聊 / group=群聊 )';
COMMENT ON COLUMN user_conversation_view_0.conversation_name IS '会话显示名称 ( 单聊：备注名>昵称 / 群聊：群名 )';
COMMENT ON COLUMN user_conversation_view_0.conversation_remark IS '会话备注名';
COMMENT ON COLUMN user_conversation_view_0.conversation_avatar IS '会话显示头像 ( 单聊：对方头像，群聊：群头像 )';
COMMENT ON COLUMN user_conversation_view_0.conversation_target_id IS '会话目标 ID ( 用户 ID or 频道 ID )';
COMMENT ON COLUMN user_conversation_view_0.conversation_info_ver IS '【 CQRS 关键字段 】会话信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN user_conversation_view_0.is_muted IS '是否免打扰';
COMMENT ON COLUMN user_conversation_view_0.is_pinned IS '是否置顶';
COMMENT ON COLUMN user_conversation_view_0.is_hidden IS '是否隐藏';
COMMENT ON COLUMN user_conversation_view_0.is_starred IS '是否星标';
COMMENT ON COLUMN user_conversation_view_0.last_message_id IS '最后消息 ID';
COMMENT ON COLUMN user_conversation_view_0.last_message_type IS '最后消息类型';
COMMENT ON COLUMN user_conversation_view_0.last_message_content IS '最后消息内容摘要';
COMMENT ON COLUMN user_conversation_view_0.last_message_sender_id IS '最后消息发送者 ID';
COMMENT ON COLUMN user_conversation_view_0.last_message_sender_name IS '最后消息发送者昵称';
COMMENT ON COLUMN user_conversation_view_0.last_message_time IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_0.unread_count IS '未读消息数';
COMMENT ON COLUMN user_conversation_view_0.mention_count IS '@我的消息数';
COMMENT ON COLUMN user_conversation_view_0.opened_at IS '会话创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_0.closed_at IS '会话删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_0.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_ucv_list_0 ON user_conversation_view_0(
    user_id,
    is_pinned DESC,
    last_message_time DESC
) WHERE closed_at = 0 AND is_hidden = FALSE;

CREATE INDEX idx_ucv_pinned_0 ON user_conversation_view_0(
    user_id,
    last_message_time DESC
) WHERE is_pinned = TRUE AND closed_at = 0;

CREATE INDEX idx_ucv_type_0 ON user_conversation_view_0(
    user_id,
    conversation_type,
    last_message_time DESC
) WHERE closed_at = 0;

CREATE INDEX idx_ucv_unread_0 ON user_conversation_view_0(
    user_id,
    unread_count DESC,
    last_message_time DESC
) WHERE unread_count > 0 AND closed_at = 0;

-- 分表 1: user_conversation_view_1
DROP TABLE IF EXISTS user_conversation_view_1;
CREATE TABLE user_conversation_view_1 (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,

    -- 会话基础信息 ( 冗余 )
    conversation_type VARCHAR(16) NOT NULL,
    conversation_name VARCHAR(128) NOT NULL,                    -- 冗余 ( 群名 或 对方昵称 / 备注名 )
    conversation_remark VARCHAR(128) NOT NULL DEFAULT '',       -- 冗余 ( 备注名 )
    conversation_avatar VARCHAR(256) NOT NULL DEFAULT '',       -- 冗余 ( 群头像 或 对方头像 )
    conversation_target_id BIGINT NOT NULL,                     -- 会话目标 ID ( 用户 ID or 频道 ID )
    conversation_info_ver INT NOT NULL DEFAULT 1,               -- 【 CQRS 关键字段 】版本号 ( 用于检测 channel / user 信息变更 )

    -- 个性化设置
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,                    -- 免打扰
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,                   -- 置顶
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,                   -- 隐藏
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                  -- 星标

    -- 最后消息快照 ( 冗余 - 用于会话列表展示 )
    last_message_id BIGINT NOT NULL DEFAULT 0,
    last_message_type VARCHAR(32) NOT NULL DEFAULT '',          -- 消息类型 ( text/image/voice/video 等 )
    last_message_content VARCHAR(512) NOT NULL DEFAULT '',      -- 消息内容摘要
    last_message_sender_id BIGINT NOT NULL DEFAULT 0,
    last_message_sender_name VARCHAR(64) NOT NULL DEFAULT '',   -- 发送者昵称 ( 冗余 )
    last_message_time BIGINT NOT NULL DEFAULT 0,

    -- 未读数 ( 冗余 - 实时更新 )
    unread_count INT NOT NULL DEFAULT 0,
    mention_count INT NOT NULL DEFAULT 0,                       -- @我的消息数

    -- 会话状态
    opened_at BIGINT NOT NULL,                                  -- 会话创建时间
    closed_at BIGINT NOT NULL DEFAULT 0,                        -- 会话删除时间 ( 0 表示未删除 )

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一会话目标中只能有一条记录（区分单聊和群聊）
    CONSTRAINT uk_user_conversation_1 UNIQUE(user_id, conversation_type, conversation_target_id),
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type_1 CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE user_conversation_view_1 IS '【 读取侧 】用户会话视图表 ( 用于会话列表展示，冗余数据 )';
COMMENT ON COLUMN user_conversation_view_1.id IS '主键 ID';
COMMENT ON COLUMN user_conversation_view_1.user_id IS '用户 ID';
COMMENT ON COLUMN user_conversation_view_1.conversation_type IS '会话类型 ( single=单聊 / group=群聊 )';
COMMENT ON COLUMN user_conversation_view_1.conversation_name IS '会话显示名称 ( 单聊：备注名>昵称 / 群聊：群名 )';
COMMENT ON COLUMN user_conversation_view_1.conversation_remark IS '会话备注名';
COMMENT ON COLUMN user_conversation_view_1.conversation_avatar IS '会话显示头像 ( 单聊：对方头像，群聊：群头像 )';
COMMENT ON COLUMN user_conversation_view_1.conversation_target_id IS '会话目标 ID ( 用户 ID or 频道 ID )';
COMMENT ON COLUMN user_conversation_view_1.conversation_info_ver IS '【 CQRS 关键字段 】会话信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN user_conversation_view_1.is_muted IS '是否免打扰';
COMMENT ON COLUMN user_conversation_view_1.is_pinned IS '是否置顶';
COMMENT ON COLUMN user_conversation_view_1.is_hidden IS '是否隐藏';
COMMENT ON COLUMN user_conversation_view_1.is_starred IS '是否星标';
COMMENT ON COLUMN user_conversation_view_1.last_message_id IS '最后消息 ID';
COMMENT ON COLUMN user_conversation_view_1.last_message_type IS '最后消息类型';
COMMENT ON COLUMN user_conversation_view_1.last_message_content IS '最后消息内容摘要';
COMMENT ON COLUMN user_conversation_view_1.last_message_sender_id IS '最后消息发送者 ID';
COMMENT ON COLUMN user_conversation_view_1.last_message_sender_name IS '最后消息发送者昵称';
COMMENT ON COLUMN user_conversation_view_1.last_message_time IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_1.unread_count IS '未读消息数';
COMMENT ON COLUMN user_conversation_view_1.mention_count IS '@我的消息数';
COMMENT ON COLUMN user_conversation_view_1.opened_at IS '会话创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_1.closed_at IS '会话删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_1.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_ucv_list_1 ON user_conversation_view_1(
    user_id,
    is_pinned DESC,
    last_message_time DESC
) WHERE closed_at = 0 AND is_hidden = FALSE;

CREATE INDEX idx_ucv_pinned_1 ON user_conversation_view_1(
    user_id,
    last_message_time DESC
) WHERE is_pinned = TRUE AND closed_at = 0;

CREATE INDEX idx_ucv_type_1 ON user_conversation_view_1(
    user_id,
    conversation_type,
    last_message_time DESC
) WHERE closed_at = 0;

CREATE INDEX idx_ucv_unread_1 ON user_conversation_view_1(
    user_id,
    unread_count DESC,
    last_message_time DESC
) WHERE unread_count > 0 AND closed_at = 0;

-- 分表 2: user_conversation_view_2
DROP TABLE IF EXISTS user_conversation_view_2;
CREATE TABLE user_conversation_view_2 (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,

    -- 会话基础信息 ( 冗余 )
    conversation_type VARCHAR(16) NOT NULL,
    conversation_name VARCHAR(128) NOT NULL,                    -- 冗余 ( 群名 或 对方昵称 / 备注名 )
    conversation_remark VARCHAR(128) NOT NULL DEFAULT '',       -- 冗余 ( 备注名 )
    conversation_avatar VARCHAR(256) NOT NULL DEFAULT '',       -- 冗余 ( 群头像 或 对方头像 )
    conversation_target_id BIGINT NOT NULL,                     -- 会话目标 ID ( 用户 ID or 频道 ID )
    conversation_info_ver INT NOT NULL DEFAULT 1,               -- 【 CQRS 关键字段 】版本号 ( 用于检测 channel / user 信息变更 )

    -- 个性化设置
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,                    -- 免打扰
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,                   -- 置顶
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,                   -- 隐藏
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                  -- 星标

    -- 最后消息快照 ( 冗余 - 用于会话列表展示 )
    last_message_id BIGINT NOT NULL DEFAULT 0,
    last_message_type VARCHAR(32) NOT NULL DEFAULT '',          -- 消息类型 ( text/image/voice/video 等 )
    last_message_content VARCHAR(512) NOT NULL DEFAULT '',      -- 消息内容摘要
    last_message_sender_id BIGINT NOT NULL DEFAULT 0,
    last_message_sender_name VARCHAR(64) NOT NULL DEFAULT '',   -- 发送者昵称 ( 冗余 )
    last_message_time BIGINT NOT NULL DEFAULT 0,

    -- 未读数 ( 冗余 - 实时更新 )
    unread_count INT NOT NULL DEFAULT 0,
    mention_count INT NOT NULL DEFAULT 0,                       -- @我的消息数

    -- 会话状态
    opened_at BIGINT NOT NULL,                                  -- 会话创建时间
    closed_at BIGINT NOT NULL DEFAULT 0,                        -- 会话删除时间 ( 0 表示未删除 )

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一会话目标中只能有一条记录（区分单聊和群聊）
    CONSTRAINT uk_user_conversation_2 UNIQUE(user_id, conversation_type, conversation_target_id),
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type_2 CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE user_conversation_view_2 IS '【 读取侧 】用户会话视图表 ( 用于会话列表展示，冗余数据 )';
COMMENT ON COLUMN user_conversation_view_2.id IS '主键 ID';
COMMENT ON COLUMN user_conversation_view_2.user_id IS '用户 ID';
COMMENT ON COLUMN user_conversation_view_2.conversation_type IS '会话类型 ( single=单聊 / group=群聊 )';
COMMENT ON COLUMN user_conversation_view_2.conversation_name IS '会话显示名称 ( 单聊：备注名>昵称 / 群聊：群名 )';
COMMENT ON COLUMN user_conversation_view_2.conversation_remark IS '会话备注名';
COMMENT ON COLUMN user_conversation_view_2.conversation_avatar IS '会话显示头像 ( 单聊：对方头像，群聊：群头像 )';
COMMENT ON COLUMN user_conversation_view_2.conversation_target_id IS '会话目标 ID ( 用户 ID or 频道 ID )';
COMMENT ON COLUMN user_conversation_view_2.conversation_info_ver IS '【 CQRS 关键字段 】会话信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN user_conversation_view_2.is_muted IS '是否免打扰';
COMMENT ON COLUMN user_conversation_view_2.is_pinned IS '是否置顶';
COMMENT ON COLUMN user_conversation_view_2.is_hidden IS '是否隐藏';
COMMENT ON COLUMN user_conversation_view_2.is_starred IS '是否星标';
COMMENT ON COLUMN user_conversation_view_2.last_message_id IS '最后消息 ID';
COMMENT ON COLUMN user_conversation_view_2.last_message_type IS '最后消息类型';
COMMENT ON COLUMN user_conversation_view_2.last_message_content IS '最后消息内容摘要';
COMMENT ON COLUMN user_conversation_view_2.last_message_sender_id IS '最后消息发送者 ID';
COMMENT ON COLUMN user_conversation_view_2.last_message_sender_name IS '最后消息发送者昵称';
COMMENT ON COLUMN user_conversation_view_2.last_message_time IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_2.unread_count IS '未读消息数';
COMMENT ON COLUMN user_conversation_view_2.mention_count IS '@我的消息数';
COMMENT ON COLUMN user_conversation_view_2.opened_at IS '会话创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_2.closed_at IS '会话删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_2.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_ucv_list_2 ON user_conversation_view_2(
    user_id,
    is_pinned DESC,
    last_message_time DESC
) WHERE closed_at = 0 AND is_hidden = FALSE;

CREATE INDEX idx_ucv_pinned_2 ON user_conversation_view_2(
    user_id,
    last_message_time DESC
) WHERE is_pinned = TRUE AND closed_at = 0;

CREATE INDEX idx_ucv_type_2 ON user_conversation_view_2(
    user_id,
    conversation_type,
    last_message_time DESC
) WHERE closed_at = 0;

CREATE INDEX idx_ucv_unread_2 ON user_conversation_view_2(
    user_id,
    unread_count DESC,
    last_message_time DESC
) WHERE unread_count > 0 AND closed_at = 0;

-- 分表 3: user_conversation_view_3
DROP TABLE IF EXISTS user_conversation_view_3;
CREATE TABLE user_conversation_view_3 (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,

    -- 会话基础信息 ( 冗余 )
    conversation_type VARCHAR(16) NOT NULL,
    conversation_name VARCHAR(128) NOT NULL,                    -- 冗余 ( 群名 或 对方昵称 / 备注名 )
    conversation_remark VARCHAR(128) NOT NULL DEFAULT '',       -- 冗余 ( 备注名 )
    conversation_avatar VARCHAR(256) NOT NULL DEFAULT '',       -- 冗余 ( 群头像 或 对方头像 )
    conversation_target_id BIGINT NOT NULL,                     -- 会话目标 ID ( 用户 ID or 频道 ID )
    conversation_info_ver INT NOT NULL DEFAULT 1,               -- 【 CQRS 关键字段 】版本号 ( 用于检测 channel / user 信息变更 )

    -- 个性化设置
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,                    -- 免打扰
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,                   -- 置顶
    is_hidden BOOLEAN NOT NULL DEFAULT FALSE,                   -- 隐藏
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                  -- 星标

    -- 最后消息快照 ( 冗余 - 用于会话列表展示 )
    last_message_id BIGINT NOT NULL DEFAULT 0,
    last_message_type VARCHAR(32) NOT NULL DEFAULT '',          -- 消息类型 ( text/image/voice/video 等 )
    last_message_content VARCHAR(512) NOT NULL DEFAULT '',      -- 消息内容摘要
    last_message_sender_id BIGINT NOT NULL DEFAULT 0,
    last_message_sender_name VARCHAR(64) NOT NULL DEFAULT '',   -- 发送者昵称 ( 冗余 )
    last_message_time BIGINT NOT NULL DEFAULT 0,

    -- 未读数 ( 冗余 - 实时更新 )
    unread_count INT NOT NULL DEFAULT 0,
    mention_count INT NOT NULL DEFAULT 0,                       -- @我的消息数

    -- 会话状态
    opened_at BIGINT NOT NULL,                                  -- 会话创建时间
    closed_at BIGINT NOT NULL DEFAULT 0,                        -- 会话删除时间 ( 0 表示未删除 )

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一会话目标中只能有一条记录（区分单聊和群聊）
    CONSTRAINT uk_user_conversation_3 UNIQUE(user_id, conversation_type, conversation_target_id),
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type_3 CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE user_conversation_view_3 IS '【 读取侧 】用户会话视图表 ( 用于会话列表展示，冗余数据 )';
COMMENT ON COLUMN user_conversation_view_3.id IS '主键 ID';
COMMENT ON COLUMN user_conversation_view_3.user_id IS '用户 ID';
COMMENT ON COLUMN user_conversation_view_3.conversation_type IS '会话类型 ( single=单聊 / group=群聊 )';
COMMENT ON COLUMN user_conversation_view_3.conversation_name IS '会话显示名称 ( 单聊：备注名>昵称 / 群聊：群名 )';
COMMENT ON COLUMN user_conversation_view_3.conversation_remark IS '会话备注名';
COMMENT ON COLUMN user_conversation_view_3.conversation_avatar IS '会话显示头像 ( 单聊：对方头像，群聊：群头像 )';
COMMENT ON COLUMN user_conversation_view_3.conversation_target_id IS '会话目标 ID ( 用户 ID or 频道 ID )';
COMMENT ON COLUMN user_conversation_view_3.conversation_info_ver IS '【 CQRS 关键字段 】会话信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN user_conversation_view_3.is_muted IS '是否免打扰';
COMMENT ON COLUMN user_conversation_view_3.is_pinned IS '是否置顶';
COMMENT ON COLUMN user_conversation_view_3.is_hidden IS '是否隐藏';
COMMENT ON COLUMN user_conversation_view_3.is_starred IS '是否星标';
COMMENT ON COLUMN user_conversation_view_3.last_message_id IS '最后消息 ID';
COMMENT ON COLUMN user_conversation_view_3.last_message_type IS '最后消息类型';
COMMENT ON COLUMN user_conversation_view_3.last_message_content IS '最后消息内容摘要';
COMMENT ON COLUMN user_conversation_view_3.last_message_sender_id IS '最后消息发送者 ID';
COMMENT ON COLUMN user_conversation_view_3.last_message_sender_name IS '最后消息发送者昵称';
COMMENT ON COLUMN user_conversation_view_3.last_message_time IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_3.unread_count IS '未读消息数';
COMMENT ON COLUMN user_conversation_view_3.mention_count IS '@我的消息数';
COMMENT ON COLUMN user_conversation_view_3.opened_at IS '会话创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_3.closed_at IS '会话删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_conversation_view_3.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_ucv_list_3 ON user_conversation_view_3(
    user_id,
    is_pinned DESC,
    last_message_time DESC
) WHERE closed_at = 0 AND is_hidden = FALSE;

CREATE INDEX idx_ucv_pinned_3 ON user_conversation_view_3(
    user_id,
    last_message_time DESC
) WHERE is_pinned = TRUE AND closed_at = 0;

CREATE INDEX idx_ucv_type_3 ON user_conversation_view_3(
    user_id,
    conversation_type,
    last_message_time DESC
) WHERE closed_at = 0;

CREATE INDEX idx_ucv_unread_3 ON user_conversation_view_3(
    user_id,
    unread_count DESC,
    last_message_time DESC
) WHERE unread_count > 0 AND closed_at = 0;

