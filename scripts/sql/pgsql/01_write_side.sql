-- ============================================================================
-- 写入侧 ( Write Side ) - 标准化设计
-- ============================================================================
-- 业务用户信息表
DROP TABLE IF EXISTS biz_user;
CREATE TABLE biz_user (
    id BIGINT PRIMARY KEY,

    -- 账号信息
    email VARCHAR(128) NOT NULL,
    mobile VARCHAR(16) NOT NULL,
    passwd VARCHAR(128) NOT NULL,

    -- 个人信息 ( 会被冗余到读取侧 )
    nickname VARCHAR(64) NOT NULL,
    avatar VARCHAR(256) NOT NULL DEFAULT '',
    gender VARCHAR(16) NOT NULL DEFAULT 'unknown',
    region VARCHAR(64) NOT NULL DEFAULT '',            -- 地区
    birthday BIGINT NOT NULL DEFAULT 0,                -- 生日 ( Unix 毫秒值 )
    tagline VARCHAR(256) NOT NULL DEFAULT '',          -- 用户签名

    -- 版本控制 ( CQRS 关键字段 )
    info_ver INT NOT NULL DEFAULT 1,                   -- 信息版本号

    -- 状态管理
    user_status VARCHAR(16) NOT NULL DEFAULT 'active',
    deleted_at BIGINT NOT NULL DEFAULT 0,              -- 软删除 ( 0 表示未删除 )

    -- 时间戳
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 用户状态 - 枚举 ( active=正常 / disabled=禁用 / deleted=已注销 )
    CONSTRAINT chk_user_status CHECK (user_status IN ('active', 'disabled', 'deleted')),
    -- 用户性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_gender CHECK (gender IN ('unknown', 'male', 'female'))
);

COMMENT ON TABLE biz_user IS '【 写入侧 】业务用户信息表';
COMMENT ON COLUMN biz_user.id IS '用户 ID';
COMMENT ON COLUMN biz_user.email IS '邮箱 ( 唯一 )';
COMMENT ON COLUMN biz_user.mobile IS '手机号 ( 唯一 )';
COMMENT ON COLUMN biz_user.passwd IS '密码';
COMMENT ON COLUMN biz_user.nickname IS '昵称 ( 会被冗余到会话视图 / 联系人视图 )';
COMMENT ON COLUMN biz_user.avatar IS '头像 ( 会被冗余到会话视图联系人视图 )';
COMMENT ON COLUMN biz_user.gender IS '性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN biz_user.tagline IS '用户签名';
COMMENT ON COLUMN biz_user.region IS '地区 ( 如：北京市-海淀区 )';
COMMENT ON COLUMN biz_user.birthday IS '生日时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN biz_user.user_status IS '用户状态 ( active=正常 / disabled=禁用 / deleted=已注销 )';
COMMENT ON COLUMN biz_user.deleted_at IS '软删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
-- 1. 账号登录相关
CREATE UNIQUE INDEX uk_biz_user_email ON biz_user(email) WHERE deleted_at = 0;
CREATE UNIQUE INDEX uk_biz_user_mobile ON biz_user(mobile) WHERE deleted_at = 0;

-- 2. 搜索用户 ( 昵称模糊搜索 )
CREATE INDEX idx_biz_user_nickname ON biz_user(nickname) WHERE deleted_at = 0;

-- 3. 状态查询
CREATE INDEX idx_biz_user_status ON biz_user(user_status, deleted_at);


-- 频道表 ( 群聊 )
DROP TABLE IF EXISTS channel;
CREATE TABLE channel (
    id BIGINT PRIMARY KEY,

    -- 频道基础信息
    channel_name VARCHAR(128) NOT NULL DEFAULT '',
    channel_avatar VARCHAR(256) NOT NULL DEFAULT '',
    channel_status VARCHAR(16) NOT NULL,
    channel_member_count INT NOT NULL DEFAULT 0,

    info_ver INT NOT NULL DEFAULT 1,                   -- 【 CQRS 关键字段 】信息版本号

    -- 最后消息时间 ( 用于判断活跃度 )
    last_message_at BIGINT NOT NULL DEFAULT 0,

    creator_id BIGINT NOT NULL,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 频道状态 - 枚举 ( creating=创建中 / active=活跃 / failed=失败 / archived=已归档 )
    CONSTRAINT chk_channel_status CHECK (channel_status IN ('creating', 'active', 'failed','archived'))
);

COMMENT ON TABLE channel IS '【写入侧】频道表 ( 群聊 )';
COMMENT ON COLUMN channel.id IS '频道 ID';
COMMENT ON COLUMN channel.channel_name IS '频道名称 ( 单聊为空 )';
COMMENT ON COLUMN channel.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel.channel_status IS '频道状态 ( creating=创建中 / active=活跃 / failed=失败 / archived=已归档 )';
COMMENT ON COLUMN channel.channel_member_count IS '频道成员数量';
COMMENT ON COLUMN channel.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN channel.last_message_at IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel.creator_id IS '创建者 ID';
COMMENT ON COLUMN channel.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_channel_creator ON channel(creator_id);
CREATE INDEX idx_channel_status ON channel(channel_status);


-- 频道成员表 ( 群聊成员 )
DROP TABLE IF EXISTS channel_member;
CREATE TABLE channel_member (
    id BIGINT PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    -- 加入 / 退出时间
    joined_at BIGINT NOT NULL,
    left_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一频道中只能有一条记录
    CONSTRAINT uk_channel_member_channel_user UNIQUE(channel_id, user_id),
    -- 成员角色 - 枚举 ( owner=群主 / admin=管理员 / member=普通成员 )
    CONSTRAINT chk_channel_member_role CHECK (member_role IN ('owner', 'admin', 'member'))
);

COMMENT ON TABLE channel_member IS '【 写入侧 】频道成员表 ( 群聊成员 )';
COMMENT ON COLUMN channel_member.id IS '主键 ID';
COMMENT ON COLUMN channel_member.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_member.user_id IS '用户 ID';
COMMENT ON COLUMN channel_member.joined_at IS '加入时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member.left_at IS '退出时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_channel_member_channel ON channel_member(channel_id);
CREATE INDEX idx_channel_member_user ON channel_member(user_id);
CREATE INDEX idx_channel_member_active ON channel_member(channel_id, left_at) WHERE left_at = 0;


-- 频道消息已读记录表 ( 群聊消息已读记录 )
DROP TABLE IF EXISTS channel_read_record;
CREATE TABLE channel_read_record (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 一个用户在一个频道中只有一条已读记录
    CONSTRAINT uk_read_record_user_channel UNIQUE(user_id, channel_id)
);

COMMENT ON TABLE channel_read_record IS '【 写入侧 】频道消息已读记录表 ( 群聊消息已读记录 )';
COMMENT ON COLUMN channel_read_record.id IS '主键 ID';
COMMENT ON COLUMN channel_read_record.user_id IS '用户 ID';
COMMENT ON COLUMN channel_read_record.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_read_record.last_read_message_id IS '最后已读消息 ID';
COMMENT ON COLUMN channel_read_record.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_read_record.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_channel_read_user_channel ON channel_read_record(user_id, channel_id);


-- 用户联系人表
DROP TABLE IF EXISTS user_contact;
CREATE TABLE user_contact (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,
    contact_id BIGINT NOT NULL,

    remark_name VARCHAR(64) NOT NULL DEFAULT '',                    -- 联系人备注名
    source VARCHAR(16) NOT NULL DEFAULT 'search',                   -- 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )

    tags VARCHAR(128)[] DEFAULT '{}',                               -- 联系人标签

    -- 状态标记
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,  -- 星标
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,  -- 黑名单

    added_at BIGINT NOT NULL,
    deleted_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一联系人中只能有一条记录
    CONSTRAINT uk_user_contact UNIQUE(user_id, contact_id),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE user_contact IS '【 写入侧 】用户联系人表';
COMMENT ON COLUMN user_contact.id IS '主键 ID';
COMMENT ON COLUMN user_contact.user_id IS '联系人 ID';
COMMENT ON COLUMN user_contact.contact_id IS '联系人 ID';
COMMENT ON COLUMN user_contact.remark_name IS '联系人备注名';
COMMENT ON COLUMN user_contact.source IS '添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )';
COMMENT ON COLUMN user_contact.tags IS '联系人标签数组';
COMMENT ON COLUMN user_contact.is_starred IS '是否星标联系人';
COMMENT ON COLUMN user_contact.is_blocked IS '是否黑名单联系人';
COMMENT ON COLUMN user_contact.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_user_contact_user ON user_contact(user_id);
CREATE INDEX idx_user_contact_user_active ON user_contact(user_id, deleted_at) WHERE deleted_at = 0;
CREATE INDEX idx_user_contact_user_starred ON user_contact(user_id, is_starred) WHERE is_starred = TRUE AND deleted_at = 0;
