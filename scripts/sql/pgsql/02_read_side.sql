-- ============================================================================
-- 读取侧 ( Read Side ) - 冗余宽表设计
-- ============================================================================

-- 联系人视图表 ( 通讯录专用 - 冗余好友信息 )
DROP TABLE IF EXISTS contact_view;
CREATE TABLE contact_view (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,
    contact_id BIGINT NOT NULL,                                 -- 联系人 user_id

    -- 联系人信息 ( 冗余自 biz_user 表 )
    contact_nickname VARCHAR(64) NOT NULL DEFAULT '',           -- 联系人昵称 ( 冗余 )
    contact_avatar VARCHAR(256) NOT NULL DEFAULT '',            -- 联系人头像 ( 冗余 )
    contact_gender VARCHAR(16) NOT NULL DEFAULT 'unknown',      -- 联系人性别 ( unknown=未知 / male=男 / female=女 )
    contact_info_ver INT NOT NULL DEFAULT 1,                    -- 【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )

    -- 好友关系信息 ( 冗余自 user_contact 表 )
    remark_name VARCHAR(64) NOT NULL DEFAULT '',                -- 备注名
    tags VARCHAR(128)[] NOT NULL DEFAULT '{}',                           -- 标签

    -- 状态
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                  -- 星标
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,                  -- 黑名单

    -- 扩展信息 ( 冗余自 biz_user 表 )
    mobile VARCHAR(16) NOT NULL DEFAULT '',                     -- 手机号
    source VARCHAR(16) NOT NULL DEFAULT 'search',               -- 添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )

    added_at BIGINT NOT NULL,
    deleted_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一联系人中只能有一条记录
    CONSTRAINT uk_contact_view UNIQUE(user_id, contact_id),
    -- 联系人性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_contact_gender CHECK (contact_gender IN ('unknown', 'male', 'female')),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE contact_view IS '【 读取侧 】联系人视图表 ( 用于通讯录展示 )';
COMMENT ON COLUMN contact_view.id IS '主键 ID';
COMMENT ON COLUMN contact_view.user_id IS '用户 ID';
COMMENT ON COLUMN contact_view.contact_id IS '联系人 ID ( 好友 ID )';
COMMENT ON COLUMN contact_view.contact_nickname IS '联系人昵称 ( 冗余自 biz_user 表 )';
COMMENT ON COLUMN contact_view.contact_avatar IS '联系人头像 ( 冗余自 biz_user 表 )';
COMMENT ON COLUMN contact_view.contact_gender IS '联系人性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN contact_view.contact_info_ver IS '【 CQRS 关键字段 】联系人信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN contact_view.remark_name IS '备注名';
COMMENT ON COLUMN contact_view.tags IS '标签数组';
COMMENT ON COLUMN contact_view.is_starred IS '是否星标';
COMMENT ON COLUMN contact_view.is_blocked IS '是否拉黑';
COMMENT ON COLUMN contact_view.mobile IS '手机号 ( 冗余 )';
COMMENT ON COLUMN contact_view.source IS '添加来源';
COMMENT ON COLUMN contact_view.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_view.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_view.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_view.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引 ( 针对通讯录查询优化 )
-- 1. 通讯录列表查询 ( 按昵称排序 )
CREATE INDEX idx_cov_list ON contact_view(
    user_id,
    is_starred DESC,
    contact_nickname
) WHERE deleted_at = 0;

-- 2. 星标好友查询
CREATE INDEX idx_cov_starred ON contact_view(
    user_id,
    contact_nickname
) WHERE is_starred = TRUE AND deleted_at = 0;

-- 3. 按首字母查询 ( 如果需要 )
CREATE INDEX idx_cov_initial ON contact_view(
    user_id,
    LEFT(contact_nickname, 1),
    contact_nickname
) WHERE deleted_at = 0;


-- 频道成员视图表 ( 群聊成员展示专用 - 冗余成员信息 )
DROP TABLE IF EXISTS channel_member_view;
CREATE TABLE channel_member_view (
    id BIGINT PRIMARY KEY,

    channel_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    -- 成员关系信息 ( 冗余自 channel_member 表 )
    member_role VARCHAR(16) NOT NULL DEFAULT 'member',      -- 成员角色
    member_nickname VARCHAR(64) NOT NULL DEFAULT '',        -- 群昵称

    -- 用户信息 ( 冗余自 biz_user 表 )
    user_nickname VARCHAR(128) NOT NULL DEFAULT '',         -- 昵称 ( 冗余 )
    user_avatar VARCHAR(512) NOT NULL DEFAULT '',           -- 头像 ( 冗余 )
    user_gender VARCHAR(16) NOT NULL DEFAULT 'unknown',     -- 性别
    user_remark VARCHAR(128) NOT NULL DEFAULT '',           -- 备注名
    user_info_ver INT NOT NULL DEFAULT 1,                   -- 【 CQRS 关键字段 】用户信息版本号

    -- 时间信息
    joined_at BIGINT NOT NULL,
    left_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    CONSTRAINT uk_channel_member_view UNIQUE(channel_id, user_id),
    CONSTRAINT chk_member_role CHECK (member_role IN ('owner', 'admin', 'member')),
    CONSTRAINT chk_user_gender CHECK (user_gender IN ('unknown', 'male', 'female'))
);

COMMENT ON TABLE channel_member_view IS '【 读取侧 】频道成员视图表 ( 用于群成员列表展示 )';
COMMENT ON COLUMN channel_member_view.user_info_ver IS '【 CQRS 关键字段 】用户信息版本号 ( 用于同步检测 )';

-- 索引 ( 针对群成员查询优化 )
-- 1. 群成员列表查询
CREATE INDEX idx_cmv_channel_active ON channel_member_view(
    channel_id,
    member_role,
    user_nickname
) WHERE left_at = 0;

-- 2. 用户的所有群聊
CREATE INDEX idx_cmv_user_channels ON channel_member_view(
    user_id,
    joined_at DESC
) WHERE left_at = 0;


-- 用户会话视图表 ( 会话列表专用 - 冗余所有展示字段 )
DROP TABLE IF EXISTS conversation_view;
CREATE TABLE conversation_view (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,
    target_id BIGINT NOT NULL,                                  -- 会话目标 ID ( 用户 ID or 频道 ID )
    target_info_ver INT NOT NULL DEFAULT 1,                     -- 【 CQRS 关键字段 】目标信息版本号 ( 用于检测 channel / user 信息变更 )

    -- 会话基础信息 ( 冗余 )
    conversation_type VARCHAR(16) NOT NULL,                     -- 会话类型 ( single=单聊 / group=群聊 )
    conversation_name VARCHAR(128) NOT NULL,                    -- 冗余 ( 群名 或 对方昵称 / 备注名 )
    conversation_remark VARCHAR(128) NOT NULL DEFAULT '',       -- 冗余 ( 备注名 )
    conversation_avatar VARCHAR(256) NOT NULL DEFAULT '',       -- 冗余 ( 群头像 或 对方头像 )

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
    CONSTRAINT uk_conversation_view UNIQUE(user_id, target_id, conversation_type),
    -- 会话类型 - 枚举 ( single=单聊 / group=群聊 )
    CONSTRAINT chk_conversation_type CHECK (conversation_type IN ('single', 'group'))
);

COMMENT ON TABLE conversation_view IS '【 读取侧 】用户会话视图表 ( 用于会话列表展示，冗余数据 )';
COMMENT ON COLUMN conversation_view.id IS '主键 ID';
COMMENT ON COLUMN conversation_view.user_id IS '用户 ID';
COMMENT ON COLUMN conversation_view.target_id IS '会话目标 ID ( 用户 ID or 频道 ID )';
COMMENT ON COLUMN conversation_view.target_info_ver IS '【 CQRS 关键字段 】目标信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN conversation_view.conversation_type IS '会话类型 ( single=单聊 / group=群聊 )';
COMMENT ON COLUMN conversation_view.conversation_name IS '会话显示名称 ( 单聊：备注名>昵称 / 群聊：群名 )';
COMMENT ON COLUMN conversation_view.conversation_remark IS '会话备注名';
COMMENT ON COLUMN conversation_view.conversation_avatar IS '会话显示头像 ( 单聊：对方头像，群聊：群头像 )';
COMMENT ON COLUMN conversation_view.is_muted IS '是否免打扰';
COMMENT ON COLUMN conversation_view.is_pinned IS '是否置顶';
COMMENT ON COLUMN conversation_view.is_hidden IS '是否隐藏';
COMMENT ON COLUMN conversation_view.is_starred IS '是否星标';
COMMENT ON COLUMN conversation_view.last_message_id IS '最后消息 ID';
COMMENT ON COLUMN conversation_view.last_message_type IS '最后消息类型';
COMMENT ON COLUMN conversation_view.last_message_content IS '最后消息内容摘要';
COMMENT ON COLUMN conversation_view.last_message_sender_id IS '最后消息发送者 ID';
COMMENT ON COLUMN conversation_view.last_message_sender_name IS '最后消息发送者昵称';
COMMENT ON COLUMN conversation_view.last_message_time IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN conversation_view.unread_count IS '未读消息数';
COMMENT ON COLUMN conversation_view.mention_count IS '@我的消息数';
COMMENT ON COLUMN conversation_view.opened_at IS '会话创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN conversation_view.closed_at IS '会话删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN conversation_view.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN conversation_view.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引 ( 针对会话列表查询优化 )
-- 1. 会话列表主查询 ( 按时间倒序 )
CREATE INDEX idx_cv_list ON conversation_view(
    user_id,
    is_pinned DESC,
    last_message_time DESC
) WHERE closed_at = 0 AND is_hidden = FALSE;

-- 2. 置顶会话查询
CREATE INDEX idx_cv_pinned ON conversation_view(
    user_id,
    last_message_time DESC
) WHERE is_pinned = TRUE AND closed_at = 0;

-- 3. 按会话类型过滤
CREATE INDEX idx_cv_type ON conversation_view(
    user_id,
    conversation_type,
    last_message_time DESC
) WHERE closed_at = 0;

-- 4. 有未读消息的会话
CREATE INDEX idx_cv_unread ON conversation_view(
    user_id,
    unread_count DESC,
    last_message_time DESC
) WHERE unread_count > 0 AND closed_at = 0;
