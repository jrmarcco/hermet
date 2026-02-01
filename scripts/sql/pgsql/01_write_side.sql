-- ============================================================================
-- 写入侧 ( Write Side ) - 标准化设计
-- ============================================================================

-- 用户信息表 ( user 是 pgsql 关键字，所以用 biz_user 作为表名 )
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


-- 聊天组表 ( 聊天群 )
DROP TABLE IF EXISTS chat_group;
CREATE TABLE chat_group (
    id BIGINT PRIMARY KEY,

    -- 频道基础信息
    group_name VARCHAR(64) NOT NULL DEFAULT '',
    group_avatar VARCHAR(256) NOT NULL DEFAULT '',
    group_status VARCHAR(16) NOT NULL,
    group_member_count INT NOT NULL DEFAULT 0,

    info_ver INT NOT NULL DEFAULT 1,                   -- 【 CQRS 关键字段 】信息版本号

    -- 最后消息时间 ( 用于判断活跃度 )
    last_message_at BIGINT NOT NULL DEFAULT 0,

    creator_id BIGINT NOT NULL,

    deleted_at BIGINT NOT NULL DEFAULT 0,              -- 软删除 ( 0 表示未删除 )

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 频道状态 - 枚举 ( creating=创建中 / active=活跃 / failed=失败 / archived=已归档 )
    CONSTRAINT chk_group_status CHECK (group_status IN ('creating', 'active', 'failed','archived'))
);

COMMENT ON TABLE chat_group IS '【写入侧】聊天组表 ( 聊天群 )';
COMMENT ON COLUMN chat_group.id IS '聊天组 ID';
COMMENT ON COLUMN chat_group.group_name IS '聊天组名称';
COMMENT ON COLUMN chat_group.group_avatar IS '聊天组头像';
COMMENT ON COLUMN chat_group.group_status IS '聊天组状态 ( creating=创建中 / active=活跃 / failed=失败 / archived=已归档 )';
COMMENT ON COLUMN chat_group.group_member_count IS '聊天组成员数量';
COMMENT ON COLUMN chat_group.info_ver IS '【 CQRS 关键字段 】信息版本号 ( 用于同步检测 )';
COMMENT ON COLUMN chat_group.last_message_at IS '最后消息时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN chat_group.creator_id IS '创建者 ID';
COMMENT ON COLUMN chat_group.deleted_at IS '软删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN chat_group.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN chat_group.updated_at IS '更新时间戳 ( Unix 毫秒值 )';
-- 索引
CREATE INDEX idx_chat_group_creator ON chat_group(creator_id);
CREATE INDEX idx_chat_group_status ON chat_group(group_status);


-- 聊天群成员成员表
DROP TABLE IF EXISTS group_member;
CREATE TABLE group_member (
    id BIGINT PRIMARY KEY,

    group_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    member_role VARCHAR(16) NOT NULL DEFAULT 'member',

    -- 加入 / 退出时间
    joined_at BIGINT NOT NULL,
    left_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一频道中只能有一条记录
    CONSTRAINT uk_group_member_group_user UNIQUE(group_id, user_id),
    -- 成员角色 - 枚举 ( owner=群主 / admin=管理员 / member=普通成员 )
    CONSTRAINT chk_group_member_role CHECK (member_role IN ('owner', 'admin', 'member'))
);

COMMENT ON TABLE group_member IS '【 写入侧 】聊天群成员表';
COMMENT ON COLUMN group_member.id IS '主键 ID';
COMMENT ON COLUMN group_member.group_id IS '聊天群 ID';
COMMENT ON COLUMN group_member.user_id IS '用户 ID';
COMMENT ON COLUMN group_member.member_role IS '成员角色 ( owner=群主 / admin=管理员 / member=普通成员 )';
COMMENT ON COLUMN channel_member.joined_at IS '加入时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN group_member.left_at IS '退出时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN group_member.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN group_member.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_group_member_group ON group_member(group_id);
CREATE INDEX idx_group_member_user ON group_member(user_id);
CREATE INDEX idx_group_member_active ON group_member(group_id, left_at) WHERE left_at = 0;


-- 联系人表
DROP TABLE IF EXISTS contact;
CREATE TABLE contact (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,
    contact_id BIGINT NOT NULL,

    remark_name VARCHAR(64) NOT NULL DEFAULT '',                    -- 联系人备注名
    source VARCHAR(16) NOT NULL DEFAULT 'search',                   -- 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )

    -- 状态标记
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,                      -- 星标
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,                      -- 黑名单
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,                        -- 免打扰
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,                       -- 置顶

    added_at BIGINT NOT NULL,
    deleted_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 唯一约束 - 同一用户在同一联系人中只能有一条记录
    CONSTRAINT uk_contact UNIQUE(user_id, contact_id),
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_user_contact_source CHECK (source IN ('search', 'qrcode', 'group'))
);

COMMENT ON TABLE contact IS '【 写入侧 】联系人表';
COMMENT ON COLUMN contact.id IS '主键 ID';
COMMENT ON COLUMN contact.user_id IS '联系人 ID';
COMMENT ON COLUMN contact.contact_id IS '联系人 ID';
COMMENT ON COLUMN contact.remark_name IS '联系人备注名';
COMMENT ON COLUMN contact.source IS '添加来源 ( search=搜索 / qrcode=扫码 / group=群聊 )';
COMMENT ON COLUMN contact.is_starred IS '是否星标';
COMMENT ON COLUMN contact.is_blocked IS '是否黑名单';
COMMENT ON COLUMN contact.is_muted IS '是否免打扰';
COMMENT ON COLUMN contact.is_pinned IS '是否置顶';
COMMENT ON COLUMN contact.added_at IS '添加时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact.deleted_at IS '删除时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_contact_user ON contact(user_id);


-- 用户聊天群表
DROP TABLE IF EXISTS user_group;
CREATE TABLE user_group (
    id BIGINT PRIMARY KEY,

    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,

    -- 个性化设置
    remark_name VARCHAR(64) NOT NULL DEFAULT '',        -- 群组备注名

    is_muted BOOLEAN NOT NULL DEFAULT FALSE,              -- 免打扰
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,             -- 置顶

    -- 时间信息
    joined_at BIGINT NOT NULL,
    left_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    CONSTRAINT uk_user_group UNIQUE(user_id, group_id)
);


COMMENT ON TABLE user_group IS '【 写入侧 】用户群组表';
COMMENT ON COLUMN user_group.id IS '主键 ID';
COMMENT ON COLUMN user_group.user_id IS '用户 ID';
COMMENT ON COLUMN user_group.group_id IS '聊天群 ID';
COMMENT ON COLUMN user_group.remark_name IS '聊天群备注名';
COMMENT ON COLUMN user_group.is_muted IS '是否免打扰';
COMMENT ON COLUMN user_group.is_pinned IS '是否置顶';
COMMENT ON COLUMN user_group.joined_at IS '加入时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_group.left_at IS '退出时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_group.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN user_group.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_user_group_user ON user_group(user_id);
