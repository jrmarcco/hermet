-- ============================================
-- 分库分表 SQL 初始化脚本
-- 分表数量: 4
-- 生成时间: 2026-01-26 16:02:02
-- 原始文件: ./01_write_side.sql
-- ============================================

-- ============================================
-- 表: biz_user (分表数: 4)
-- ============================================

-- 分表 0: biz_user_0
DROP TABLE IF EXISTS biz_user_0;
CREATE TABLE biz_user_0 (
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
    CONSTRAINT chk_user_status_0 CHECK (user_status IN ('active', 'disabled', 'deleted')),
    -- 用户性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_gender_0 CHECK (gender IN ('unknown', 'male', 'female'))
);

COMMENT ON TABLE biz_user_0 IS '【 写入侧 】业务用户信息表';
COMMENT ON COLUMN biz_user_0.id IS '用户 ID';
COMMENT ON COLUMN biz_user_0.email IS '邮箱 ( 唯一 )';
COMMENT ON COLUMN biz_user_0.mobile IS '手机号 ( 唯一 )';
COMMENT ON COLUMN biz_user_0.passwd IS '密码';
COMMENT ON COLUMN biz_user_0.nickname IS '昵称 ( 会被冗余到会话视图 / 联系人视图 )';
COMMENT ON COLUMN biz_user_0.avatar IS '头像 ( 会被冗余到会话视图联系人视图 )';
COMMENT ON COLUMN biz_user_0.gender IS '性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN biz_user_0.tagline IS '用户签名';
COMMENT ON COLUMN biz_user_0.region IS '地区 ( 如：北京市-海淀区 )';
COMMENT ON COLUMN biz_user_0.birthday IS '生日时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_0.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN biz_user_0.user_status IS '用户状态 ( active=正常 / disabled=禁用 / deleted=已注销 )';
COMMENT ON COLUMN biz_user_0.deleted_at IS '软删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_0.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE UNIQUE INDEX uk_biz_user_email_0 ON biz_user_0(email) WHERE deleted_at = 0;

CREATE UNIQUE INDEX uk_biz_user_mobile_0 ON biz_user_0(mobile) WHERE deleted_at = 0;

CREATE INDEX idx_biz_user_nickname_0 ON biz_user_0(nickname) WHERE deleted_at = 0;

CREATE INDEX idx_biz_user_status_0 ON biz_user_0(user_status, deleted_at);

-- 分表 1: biz_user_1
DROP TABLE IF EXISTS biz_user_1;
CREATE TABLE biz_user_1 (
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
    CONSTRAINT chk_user_status_1 CHECK (user_status IN ('active', 'disabled', 'deleted')),
    -- 用户性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_gender_1 CHECK (gender IN ('unknown', 'male', 'female'))
);

COMMENT ON TABLE biz_user_1 IS '【 写入侧 】业务用户信息表';
COMMENT ON COLUMN biz_user_1.id IS '用户 ID';
COMMENT ON COLUMN biz_user_1.email IS '邮箱 ( 唯一 )';
COMMENT ON COLUMN biz_user_1.mobile IS '手机号 ( 唯一 )';
COMMENT ON COLUMN biz_user_1.passwd IS '密码';
COMMENT ON COLUMN biz_user_1.nickname IS '昵称 ( 会被冗余到会话视图 / 联系人视图 )';
COMMENT ON COLUMN biz_user_1.avatar IS '头像 ( 会被冗余到会话视图联系人视图 )';
COMMENT ON COLUMN biz_user_1.gender IS '性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN biz_user_1.tagline IS '用户签名';
COMMENT ON COLUMN biz_user_1.region IS '地区 ( 如：北京市-海淀区 )';
COMMENT ON COLUMN biz_user_1.birthday IS '生日时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_1.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN biz_user_1.user_status IS '用户状态 ( active=正常 / disabled=禁用 / deleted=已注销 )';
COMMENT ON COLUMN biz_user_1.deleted_at IS '软删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_1.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE UNIQUE INDEX uk_biz_user_email_1 ON biz_user_1(email) WHERE deleted_at = 0;

CREATE UNIQUE INDEX uk_biz_user_mobile_1 ON biz_user_1(mobile) WHERE deleted_at = 0;

CREATE INDEX idx_biz_user_nickname_1 ON biz_user_1(nickname) WHERE deleted_at = 0;

CREATE INDEX idx_biz_user_status_1 ON biz_user_1(user_status, deleted_at);

-- 分表 2: biz_user_2
DROP TABLE IF EXISTS biz_user_2;
CREATE TABLE biz_user_2 (
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
    CONSTRAINT chk_user_status_2 CHECK (user_status IN ('active', 'disabled', 'deleted')),
    -- 用户性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_gender_2 CHECK (gender IN ('unknown', 'male', 'female'))
);

COMMENT ON TABLE biz_user_2 IS '【 写入侧 】业务用户信息表';
COMMENT ON COLUMN biz_user_2.id IS '用户 ID';
COMMENT ON COLUMN biz_user_2.email IS '邮箱 ( 唯一 )';
COMMENT ON COLUMN biz_user_2.mobile IS '手机号 ( 唯一 )';
COMMENT ON COLUMN biz_user_2.passwd IS '密码';
COMMENT ON COLUMN biz_user_2.nickname IS '昵称 ( 会被冗余到会话视图 / 联系人视图 )';
COMMENT ON COLUMN biz_user_2.avatar IS '头像 ( 会被冗余到会话视图联系人视图 )';
COMMENT ON COLUMN biz_user_2.gender IS '性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN biz_user_2.tagline IS '用户签名';
COMMENT ON COLUMN biz_user_2.region IS '地区 ( 如：北京市-海淀区 )';
COMMENT ON COLUMN biz_user_2.birthday IS '生日时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_2.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN biz_user_2.user_status IS '用户状态 ( active=正常 / disabled=禁用 / deleted=已注销 )';
COMMENT ON COLUMN biz_user_2.deleted_at IS '软删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_2.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE UNIQUE INDEX uk_biz_user_email_2 ON biz_user_2(email) WHERE deleted_at = 0;

CREATE UNIQUE INDEX uk_biz_user_mobile_2 ON biz_user_2(mobile) WHERE deleted_at = 0;

CREATE INDEX idx_biz_user_nickname_2 ON biz_user_2(nickname) WHERE deleted_at = 0;

CREATE INDEX idx_biz_user_status_2 ON biz_user_2(user_status, deleted_at);

-- 分表 3: biz_user_3
DROP TABLE IF EXISTS biz_user_3;
CREATE TABLE biz_user_3 (
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
    CONSTRAINT chk_user_status_3 CHECK (user_status IN ('active', 'disabled', 'deleted')),
    -- 用户性别 - 枚举 ( unknown=未知 / male=男 / female=女 )
    CONSTRAINT chk_gender_3 CHECK (gender IN ('unknown', 'male', 'female'))
);

COMMENT ON TABLE biz_user_3 IS '【 写入侧 】业务用户信息表';
COMMENT ON COLUMN biz_user_3.id IS '用户 ID';
COMMENT ON COLUMN biz_user_3.email IS '邮箱 ( 唯一 )';
COMMENT ON COLUMN biz_user_3.mobile IS '手机号 ( 唯一 )';
COMMENT ON COLUMN biz_user_3.passwd IS '密码';
COMMENT ON COLUMN biz_user_3.nickname IS '昵称 ( 会被冗余到会话视图 / 联系人视图 )';
COMMENT ON COLUMN biz_user_3.avatar IS '头像 ( 会被冗余到会话视图联系人视图 )';
COMMENT ON COLUMN biz_user_3.gender IS '性别 ( unknown=未知 / male=男 / female=女 )';
COMMENT ON COLUMN biz_user_3.tagline IS '用户签名';
COMMENT ON COLUMN biz_user_3.region IS '地区 ( 如：北京市-海淀区 )';
COMMENT ON COLUMN biz_user_3.birthday IS '生日时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_3.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN biz_user_3.user_status IS '用户状态 ( active=正常 / disabled=禁用 / deleted=已注销 )';
COMMENT ON COLUMN biz_user_3.deleted_at IS '软删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN biz_user_3.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE UNIQUE INDEX uk_biz_user_email_3 ON biz_user_3(email) WHERE deleted_at = 0;

CREATE UNIQUE INDEX uk_biz_user_mobile_3 ON biz_user_3(mobile) WHERE deleted_at = 0;

CREATE INDEX idx_biz_user_nickname_3 ON biz_user_3(nickname) WHERE deleted_at = 0;

CREATE INDEX idx_biz_user_status_3 ON biz_user_3(user_status, deleted_at);

-- ============================================
-- 表: channel (分表数: 4)
-- ============================================

-- 分表 0: channel_0
DROP TABLE IF EXISTS channel_0;
CREATE TABLE channel_0 (
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
    CONSTRAINT chk_channel_status_0 CHECK (channel_status IN ('creating', 'active', 'failed','archived'))
);

COMMENT ON TABLE channel_0 IS '【写入侧】频道表 ( 群聊 )';
COMMENT ON COLUMN channel_0.id IS '频道 ID';
COMMENT ON COLUMN channel_0.channel_name IS '频道名称 ( 单聊为空 )';
COMMENT ON COLUMN channel_0.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel_0.channel_status IS '频道状态 ( creating=创建中 / active=活跃 / failed=失败 / archived=已归档 )';
COMMENT ON COLUMN channel_0.channel_member_count IS '频道成员数量';
COMMENT ON COLUMN channel_0.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN channel_0.last_message_at IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_0.creator_id IS '创建者 ID';
COMMENT ON COLUMN channel_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_0.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_creator_0 ON channel_0(creator_id);

CREATE INDEX idx_channel_status_0 ON channel_0(channel_status);

-- 分表 1: channel_1
DROP TABLE IF EXISTS channel_1;
CREATE TABLE channel_1 (
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
    CONSTRAINT chk_channel_status_1 CHECK (channel_status IN ('creating', 'active', 'failed','archived'))
);

COMMENT ON TABLE channel_1 IS '【写入侧】频道表 ( 群聊 )';
COMMENT ON COLUMN channel_1.id IS '频道 ID';
COMMENT ON COLUMN channel_1.channel_name IS '频道名称 ( 单聊为空 )';
COMMENT ON COLUMN channel_1.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel_1.channel_status IS '频道状态 ( creating=创建中 / active=活跃 / failed=失败 / archived=已归档 )';
COMMENT ON COLUMN channel_1.channel_member_count IS '频道成员数量';
COMMENT ON COLUMN channel_1.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN channel_1.last_message_at IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_1.creator_id IS '创建者 ID';
COMMENT ON COLUMN channel_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_1.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_creator_1 ON channel_1(creator_id);

CREATE INDEX idx_channel_status_1 ON channel_1(channel_status);

-- 分表 2: channel_2
DROP TABLE IF EXISTS channel_2;
CREATE TABLE channel_2 (
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
    CONSTRAINT chk_channel_status_2 CHECK (channel_status IN ('creating', 'active', 'failed','archived'))
);

COMMENT ON TABLE channel_2 IS '【写入侧】频道表 ( 群聊 )';
COMMENT ON COLUMN channel_2.id IS '频道 ID';
COMMENT ON COLUMN channel_2.channel_name IS '频道名称 ( 单聊为空 )';
COMMENT ON COLUMN channel_2.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel_2.channel_status IS '频道状态 ( creating=创建中 / active=活跃 / failed=失败 / archived=已归档 )';
COMMENT ON COLUMN channel_2.channel_member_count IS '频道成员数量';
COMMENT ON COLUMN channel_2.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN channel_2.last_message_at IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_2.creator_id IS '创建者 ID';
COMMENT ON COLUMN channel_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_2.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_creator_2 ON channel_2(creator_id);

CREATE INDEX idx_channel_status_2 ON channel_2(channel_status);

-- 分表 3: channel_3
DROP TABLE IF EXISTS channel_3;
CREATE TABLE channel_3 (
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
    CONSTRAINT chk_channel_status_3 CHECK (channel_status IN ('creating', 'active', 'failed','archived'))
);

COMMENT ON TABLE channel_3 IS '【写入侧】频道表 ( 群聊 )';
COMMENT ON COLUMN channel_3.id IS '频道 ID';
COMMENT ON COLUMN channel_3.channel_name IS '频道名称 ( 单聊为空 )';
COMMENT ON COLUMN channel_3.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel_3.channel_status IS '频道状态 ( creating=创建中 / active=活跃 / failed=失败 / archived=已归档 )';
COMMENT ON COLUMN channel_3.channel_member_count IS '频道成员数量';
COMMENT ON COLUMN channel_3.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN channel_3.last_message_at IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_3.creator_id IS '创建者 ID';
COMMENT ON COLUMN channel_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_3.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_creator_3 ON channel_3(creator_id);

CREATE INDEX idx_channel_status_3 ON channel_3(channel_status);

-- ============================================
-- 表: channel_member (分表数: 4)
-- ============================================

-- 分表 0: channel_member_0
DROP TABLE IF EXISTS channel_member_0;
CREATE TABLE channel_member_0 (
    id BIGINT PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    member_role VARCHAR(16) NOT NULL DEFAULT 'member',      -- 成员角色
    member_nickname VARCHAR(64) NOT NULL DEFAULT '',        -- 成员昵称 ( 群昵称 )

    -- 加入 / 退出时间
    joined_at BIGINT NOT NULL,
    left_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一频道中只能有一条记录
    CONSTRAINT uk_channel_member_channel_user_0 UNIQUE(channel_id, user_id),
    -- 成员角色 - 枚举 ( owner=群主 / admin=管理员 / member=普通成员 )
    CONSTRAINT chk_channel_member_role_0 CHECK (member_role IN ('owner', 'admin', 'member'))
);

COMMENT ON TABLE channel_member_0 IS '【 写入侧 】频道成员表 ( 群聊成员 )';
COMMENT ON COLUMN channel_member_0.id IS '主键 ID';
COMMENT ON COLUMN channel_member_0.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_member_0.user_id IS '用户 ID';
COMMENT ON COLUMN channel_member_0.member_role IS '成员角色 ( owner=群主 / admin=管理员 / member=普通成员 )';
COMMENT ON COLUMN channel_member_0.member_nickname IS '成员昵称 ( 群昵称 )';
COMMENT ON COLUMN channel_member_0.joined_at IS '加入时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_0.left_at IS '退出时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_0.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_member_channel_0 ON channel_member_0(channel_id);

CREATE INDEX idx_channel_member_user_0 ON channel_member_0(user_id);

CREATE INDEX idx_channel_member_active_0 ON channel_member_0(channel_id, left_at) WHERE left_at = 0;

-- 分表 1: channel_member_1
DROP TABLE IF EXISTS channel_member_1;
CREATE TABLE channel_member_1 (
    id BIGINT PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    member_role VARCHAR(16) NOT NULL DEFAULT 'member',      -- 成员角色
    member_nickname VARCHAR(64) NOT NULL DEFAULT '',        -- 成员昵称 ( 群昵称 )

    -- 加入 / 退出时间
    joined_at BIGINT NOT NULL,
    left_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一频道中只能有一条记录
    CONSTRAINT uk_channel_member_channel_user_1 UNIQUE(channel_id, user_id),
    -- 成员角色 - 枚举 ( owner=群主 / admin=管理员 / member=普通成员 )
    CONSTRAINT chk_channel_member_role_1 CHECK (member_role IN ('owner', 'admin', 'member'))
);

COMMENT ON TABLE channel_member_1 IS '【 写入侧 】频道成员表 ( 群聊成员 )';
COMMENT ON COLUMN channel_member_1.id IS '主键 ID';
COMMENT ON COLUMN channel_member_1.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_member_1.user_id IS '用户 ID';
COMMENT ON COLUMN channel_member_1.member_role IS '成员角色 ( owner=群主 / admin=管理员 / member=普通成员 )';
COMMENT ON COLUMN channel_member_1.member_nickname IS '成员昵称 ( 群昵称 )';
COMMENT ON COLUMN channel_member_1.joined_at IS '加入时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_1.left_at IS '退出时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_1.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_member_channel_1 ON channel_member_1(channel_id);

CREATE INDEX idx_channel_member_user_1 ON channel_member_1(user_id);

CREATE INDEX idx_channel_member_active_1 ON channel_member_1(channel_id, left_at) WHERE left_at = 0;

-- 分表 2: channel_member_2
DROP TABLE IF EXISTS channel_member_2;
CREATE TABLE channel_member_2 (
    id BIGINT PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    member_role VARCHAR(16) NOT NULL DEFAULT 'member',      -- 成员角色
    member_nickname VARCHAR(64) NOT NULL DEFAULT '',        -- 成员昵称 ( 群昵称 )

    -- 加入 / 退出时间
    joined_at BIGINT NOT NULL,
    left_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一频道中只能有一条记录
    CONSTRAINT uk_channel_member_channel_user_2 UNIQUE(channel_id, user_id),
    -- 成员角色 - 枚举 ( owner=群主 / admin=管理员 / member=普通成员 )
    CONSTRAINT chk_channel_member_role_2 CHECK (member_role IN ('owner', 'admin', 'member'))
);

COMMENT ON TABLE channel_member_2 IS '【 写入侧 】频道成员表 ( 群聊成员 )';
COMMENT ON COLUMN channel_member_2.id IS '主键 ID';
COMMENT ON COLUMN channel_member_2.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_member_2.user_id IS '用户 ID';
COMMENT ON COLUMN channel_member_2.member_role IS '成员角色 ( owner=群主 / admin=管理员 / member=普通成员 )';
COMMENT ON COLUMN channel_member_2.member_nickname IS '成员昵称 ( 群昵称 )';
COMMENT ON COLUMN channel_member_2.joined_at IS '加入时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_2.left_at IS '退出时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_2.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_member_channel_2 ON channel_member_2(channel_id);

CREATE INDEX idx_channel_member_user_2 ON channel_member_2(user_id);

CREATE INDEX idx_channel_member_active_2 ON channel_member_2(channel_id, left_at) WHERE left_at = 0;

-- 分表 3: channel_member_3
DROP TABLE IF EXISTS channel_member_3;
CREATE TABLE channel_member_3 (
    id BIGINT PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    member_role VARCHAR(16) NOT NULL DEFAULT 'member',      -- 成员角色
    member_nickname VARCHAR(64) NOT NULL DEFAULT '',        -- 成员昵称 ( 群昵称 )

    -- 加入 / 退出时间
    joined_at BIGINT NOT NULL,
    left_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一频道中只能有一条记录
    CONSTRAINT uk_channel_member_channel_user_3 UNIQUE(channel_id, user_id),
    -- 成员角色 - 枚举 ( owner=群主 / admin=管理员 / member=普通成员 )
    CONSTRAINT chk_channel_member_role_3 CHECK (member_role IN ('owner', 'admin', 'member'))
);

COMMENT ON TABLE channel_member_3 IS '【 写入侧 】频道成员表 ( 群聊成员 )';
COMMENT ON COLUMN channel_member_3.id IS '主键 ID';
COMMENT ON COLUMN channel_member_3.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_member_3.user_id IS '用户 ID';
COMMENT ON COLUMN channel_member_3.member_role IS '成员角色 ( owner=群主 / admin=管理员 / member=普通成员 )';
COMMENT ON COLUMN channel_member_3.member_nickname IS '成员昵称 ( 群昵称 )';
COMMENT ON COLUMN channel_member_3.joined_at IS '加入时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_3.left_at IS '退出时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_member_3.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_member_channel_3 ON channel_member_3(channel_id);

CREATE INDEX idx_channel_member_user_3 ON channel_member_3(user_id);

CREATE INDEX idx_channel_member_active_3 ON channel_member_3(channel_id, left_at) WHERE left_at = 0;

-- ============================================
-- 表: channel_read_record (分表数: 4)
-- ============================================

-- 分表 0: channel_read_record_0
DROP TABLE IF EXISTS channel_read_record_0;
CREATE TABLE channel_read_record_0 (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 一个用户在一个频道中只有一条已读记录
    CONSTRAINT uk_read_record_user_channel_0 UNIQUE(user_id, channel_id)
);

COMMENT ON TABLE channel_read_record_0 IS '【 写入侧 】频道消息已读记录表 ( 群聊消息已读记录 )';
COMMENT ON COLUMN channel_read_record_0.id IS '主键 ID';
COMMENT ON COLUMN channel_read_record_0.user_id IS '用户 ID';
COMMENT ON COLUMN channel_read_record_0.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_read_record_0.last_read_message_id IS '最后已读消息 ID';
COMMENT ON COLUMN channel_read_record_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_read_record_0.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_read_user_channel_0 ON channel_read_record_0(user_id, channel_id);

-- 分表 1: channel_read_record_1
DROP TABLE IF EXISTS channel_read_record_1;
CREATE TABLE channel_read_record_1 (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 一个用户在一个频道中只有一条已读记录
    CONSTRAINT uk_read_record_user_channel_1 UNIQUE(user_id, channel_id)
);

COMMENT ON TABLE channel_read_record_1 IS '【 写入侧 】频道消息已读记录表 ( 群聊消息已读记录 )';
COMMENT ON COLUMN channel_read_record_1.id IS '主键 ID';
COMMENT ON COLUMN channel_read_record_1.user_id IS '用户 ID';
COMMENT ON COLUMN channel_read_record_1.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_read_record_1.last_read_message_id IS '最后已读消息 ID';
COMMENT ON COLUMN channel_read_record_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_read_record_1.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_read_user_channel_1 ON channel_read_record_1(user_id, channel_id);

-- 分表 2: channel_read_record_2
DROP TABLE IF EXISTS channel_read_record_2;
CREATE TABLE channel_read_record_2 (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 一个用户在一个频道中只有一条已读记录
    CONSTRAINT uk_read_record_user_channel_2 UNIQUE(user_id, channel_id)
);

COMMENT ON TABLE channel_read_record_2 IS '【 写入侧 】频道消息已读记录表 ( 群聊消息已读记录 )';
COMMENT ON COLUMN channel_read_record_2.id IS '主键 ID';
COMMENT ON COLUMN channel_read_record_2.user_id IS '用户 ID';
COMMENT ON COLUMN channel_read_record_2.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_read_record_2.last_read_message_id IS '最后已读消息 ID';
COMMENT ON COLUMN channel_read_record_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_read_record_2.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_read_user_channel_2 ON channel_read_record_2(user_id, channel_id);

-- 分表 3: channel_read_record_3
DROP TABLE IF EXISTS channel_read_record_3;
CREATE TABLE channel_read_record_3 (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 一个用户在一个频道中只有一条已读记录
    CONSTRAINT uk_read_record_user_channel_3 UNIQUE(user_id, channel_id)
);

COMMENT ON TABLE channel_read_record_3 IS '【 写入侧 】频道消息已读记录表 ( 群聊消息已读记录 )';
COMMENT ON COLUMN channel_read_record_3.id IS '主键 ID';
COMMENT ON COLUMN channel_read_record_3.user_id IS '用户 ID';
COMMENT ON COLUMN channel_read_record_3.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_read_record_3.last_read_message_id IS '最后已读消息 ID';
COMMENT ON COLUMN channel_read_record_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_read_record_3.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_read_user_channel_3 ON channel_read_record_3(user_id, channel_id);

-- ============================================
-- 表: user_contact (分表数: 4)
-- ============================================

-- 分表 0: user_contact_0
DROP TABLE IF EXISTS user_contact_0;
CREATE TABLE user_contact_0 (
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
    CONSTRAINT uk_user_contact_0 UNIQUE(user_id, contact_id),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source_0 CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE user_contact_0 IS '【 写入侧 】用户联系人表';
COMMENT ON COLUMN user_contact_0.id IS '主键 ID';
COMMENT ON COLUMN user_contact_0.user_id IS '联系人 ID';
COMMENT ON COLUMN user_contact_0.contact_id IS '联系人 ID';
COMMENT ON COLUMN user_contact_0.remark_name IS '联系人备注名';
COMMENT ON COLUMN user_contact_0.source IS '添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )';
COMMENT ON COLUMN user_contact_0.tags IS '联系人标签数组';
COMMENT ON COLUMN user_contact_0.is_starred IS '是否星标联系人';
COMMENT ON COLUMN user_contact_0.is_blocked IS '是否黑名单联系人';
COMMENT ON COLUMN user_contact_0.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_0.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_0.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_user_contact_user_0 ON user_contact_0(user_id);

CREATE INDEX idx_user_contact_user_active_0 ON user_contact_0(user_id, deleted_at) WHERE deleted_at = 0;

CREATE INDEX idx_user_contact_user_starred_0 ON user_contact_0(user_id, is_starred) WHERE is_starred = TRUE AND deleted_at = 0;

-- 分表 1: user_contact_1
DROP TABLE IF EXISTS user_contact_1;
CREATE TABLE user_contact_1 (
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
    CONSTRAINT uk_user_contact_1 UNIQUE(user_id, contact_id),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source_1 CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE user_contact_1 IS '【 写入侧 】用户联系人表';
COMMENT ON COLUMN user_contact_1.id IS '主键 ID';
COMMENT ON COLUMN user_contact_1.user_id IS '联系人 ID';
COMMENT ON COLUMN user_contact_1.contact_id IS '联系人 ID';
COMMENT ON COLUMN user_contact_1.remark_name IS '联系人备注名';
COMMENT ON COLUMN user_contact_1.source IS '添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )';
COMMENT ON COLUMN user_contact_1.tags IS '联系人标签数组';
COMMENT ON COLUMN user_contact_1.is_starred IS '是否星标联系人';
COMMENT ON COLUMN user_contact_1.is_blocked IS '是否黑名单联系人';
COMMENT ON COLUMN user_contact_1.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_1.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_1.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_user_contact_user_1 ON user_contact_1(user_id);

CREATE INDEX idx_user_contact_user_active_1 ON user_contact_1(user_id, deleted_at) WHERE deleted_at = 0;

CREATE INDEX idx_user_contact_user_starred_1 ON user_contact_1(user_id, is_starred) WHERE is_starred = TRUE AND deleted_at = 0;

-- 分表 2: user_contact_2
DROP TABLE IF EXISTS user_contact_2;
CREATE TABLE user_contact_2 (
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
    CONSTRAINT uk_user_contact_2 UNIQUE(user_id, contact_id),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source_2 CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE user_contact_2 IS '【 写入侧 】用户联系人表';
COMMENT ON COLUMN user_contact_2.id IS '主键 ID';
COMMENT ON COLUMN user_contact_2.user_id IS '联系人 ID';
COMMENT ON COLUMN user_contact_2.contact_id IS '联系人 ID';
COMMENT ON COLUMN user_contact_2.remark_name IS '联系人备注名';
COMMENT ON COLUMN user_contact_2.source IS '添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )';
COMMENT ON COLUMN user_contact_2.tags IS '联系人标签数组';
COMMENT ON COLUMN user_contact_2.is_starred IS '是否星标联系人';
COMMENT ON COLUMN user_contact_2.is_blocked IS '是否黑名单联系人';
COMMENT ON COLUMN user_contact_2.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_2.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_2.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_user_contact_user_2 ON user_contact_2(user_id);

CREATE INDEX idx_user_contact_user_active_2 ON user_contact_2(user_id, deleted_at) WHERE deleted_at = 0;

CREATE INDEX idx_user_contact_user_starred_2 ON user_contact_2(user_id, is_starred) WHERE is_starred = TRUE AND deleted_at = 0;

-- 分表 3: user_contact_3
DROP TABLE IF EXISTS user_contact_3;
CREATE TABLE user_contact_3 (
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
    CONSTRAINT uk_user_contact_3 UNIQUE(user_id, contact_id),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source_3 CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE user_contact_3 IS '【 写入侧 】用户联系人表';
COMMENT ON COLUMN user_contact_3.id IS '主键 ID';
COMMENT ON COLUMN user_contact_3.user_id IS '联系人 ID';
COMMENT ON COLUMN user_contact_3.contact_id IS '联系人 ID';
COMMENT ON COLUMN user_contact_3.remark_name IS '联系人备注名';
COMMENT ON COLUMN user_contact_3.source IS '添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )';
COMMENT ON COLUMN user_contact_3.tags IS '联系人标签数组';
COMMENT ON COLUMN user_contact_3.is_starred IS '是否星标联系人';
COMMENT ON COLUMN user_contact_3.is_blocked IS '是否黑名单联系人';
COMMENT ON COLUMN user_contact_3.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_3.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_contact_3.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_user_contact_user_3 ON user_contact_3(user_id);

CREATE INDEX idx_user_contact_user_active_3 ON user_contact_3(user_id, deleted_at) WHERE deleted_at = 0;

CREATE INDEX idx_user_contact_user_starred_3 ON user_contact_3(user_id, is_starred) WHERE is_starred = TRUE AND deleted_at = 0;

