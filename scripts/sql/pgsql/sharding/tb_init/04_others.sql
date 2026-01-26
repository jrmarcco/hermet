-- ============================================
-- 分库分表 SQL 初始化脚本
-- 分表数量: 4
-- 生成时间: 2026-01-26 16:02:03
-- 原始文件: ./04_others.sql
-- ============================================

-- ============================================
-- 表: channel_application (分表数: 4)
-- ============================================

-- 分表 0: channel_application_0
DROP TABLE IF EXISTS channel_application_0;
CREATE TABLE channel_application_0 (
    id BIGINT PRIMARY KEY,

    applicant_id BIGINT NOT NULL,

    -- 目标渠道信息。
    channel_id BIGINT NOT NULL,
    channel_name VARCHAR(128) NOT NULL DEFAULT '',
    channel_avatar VARCHAR(256) NOT NULL DEFAULT '',

    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    reviewer_id BIGINT NOT NULL DEFAULT 0,
    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_channel_application_status_0 CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE channel_application_0 IS '频道申请表 ( 入群申请 )';
COMMENT ON COLUMN channel_application_0.id IS '主键 ID';
COMMENT ON COLUMN channel_application_0.applicant_id IS '申请人 ID';
COMMENT ON COLUMN channel_application_0.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_application_0.channel_name IS '频道名称';
COMMENT ON COLUMN channel_application_0.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel_application_0.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN channel_application_0.application_message IS '申请验证消息';
COMMENT ON COLUMN channel_application_0.reviewer_id IS '审批人 ID';
COMMENT ON COLUMN channel_application_0.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application_0.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_application_channel_0 ON channel_application_0(channel_id);

CREATE INDEX idx_channel_application_pending_0 ON channel_application_0(channel_id, application_status) WHERE application_status = 'pending';

-- 分表 1: channel_application_1
DROP TABLE IF EXISTS channel_application_1;
CREATE TABLE channel_application_1 (
    id BIGINT PRIMARY KEY,

    applicant_id BIGINT NOT NULL,

    -- 目标渠道信息。
    channel_id BIGINT NOT NULL,
    channel_name VARCHAR(128) NOT NULL DEFAULT '',
    channel_avatar VARCHAR(256) NOT NULL DEFAULT '',

    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    reviewer_id BIGINT NOT NULL DEFAULT 0,
    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_channel_application_status_1 CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE channel_application_1 IS '频道申请表 ( 入群申请 )';
COMMENT ON COLUMN channel_application_1.id IS '主键 ID';
COMMENT ON COLUMN channel_application_1.applicant_id IS '申请人 ID';
COMMENT ON COLUMN channel_application_1.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_application_1.channel_name IS '频道名称';
COMMENT ON COLUMN channel_application_1.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel_application_1.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN channel_application_1.application_message IS '申请验证消息';
COMMENT ON COLUMN channel_application_1.reviewer_id IS '审批人 ID';
COMMENT ON COLUMN channel_application_1.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application_1.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_application_channel_1 ON channel_application_1(channel_id);

CREATE INDEX idx_channel_application_pending_1 ON channel_application_1(channel_id, application_status) WHERE application_status = 'pending';

-- 分表 2: channel_application_2
DROP TABLE IF EXISTS channel_application_2;
CREATE TABLE channel_application_2 (
    id BIGINT PRIMARY KEY,

    applicant_id BIGINT NOT NULL,

    -- 目标渠道信息。
    channel_id BIGINT NOT NULL,
    channel_name VARCHAR(128) NOT NULL DEFAULT '',
    channel_avatar VARCHAR(256) NOT NULL DEFAULT '',

    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    reviewer_id BIGINT NOT NULL DEFAULT 0,
    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_channel_application_status_2 CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE channel_application_2 IS '频道申请表 ( 入群申请 )';
COMMENT ON COLUMN channel_application_2.id IS '主键 ID';
COMMENT ON COLUMN channel_application_2.applicant_id IS '申请人 ID';
COMMENT ON COLUMN channel_application_2.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_application_2.channel_name IS '频道名称';
COMMENT ON COLUMN channel_application_2.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel_application_2.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN channel_application_2.application_message IS '申请验证消息';
COMMENT ON COLUMN channel_application_2.reviewer_id IS '审批人 ID';
COMMENT ON COLUMN channel_application_2.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application_2.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_application_channel_2 ON channel_application_2(channel_id);

CREATE INDEX idx_channel_application_pending_2 ON channel_application_2(channel_id, application_status) WHERE application_status = 'pending';

-- 分表 3: channel_application_3
DROP TABLE IF EXISTS channel_application_3;
CREATE TABLE channel_application_3 (
    id BIGINT PRIMARY KEY,

    applicant_id BIGINT NOT NULL,

    -- 目标渠道信息。
    channel_id BIGINT NOT NULL,
    channel_name VARCHAR(128) NOT NULL DEFAULT '',
    channel_avatar VARCHAR(256) NOT NULL DEFAULT '',

    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    reviewer_id BIGINT NOT NULL DEFAULT 0,
    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_channel_application_status_3 CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE channel_application_3 IS '频道申请表 ( 入群申请 )';
COMMENT ON COLUMN channel_application_3.id IS '主键 ID';
COMMENT ON COLUMN channel_application_3.applicant_id IS '申请人 ID';
COMMENT ON COLUMN channel_application_3.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_application_3.channel_name IS '频道名称';
COMMENT ON COLUMN channel_application_3.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel_application_3.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN channel_application_3.application_message IS '申请验证消息';
COMMENT ON COLUMN channel_application_3.reviewer_id IS '审批人 ID';
COMMENT ON COLUMN channel_application_3.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application_3.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_channel_application_channel_3 ON channel_application_3(channel_id);

CREATE INDEX idx_channel_application_pending_3 ON channel_application_3(channel_id, application_status) WHERE application_status = 'pending';

-- ============================================
-- 表: contact_application (分表数: 4)
-- ============================================

-- 分表 0: contact_application_0
DROP TABLE IF EXISTS contact_application_0;
CREATE TABLE contact_application_0 (
    id BIGINT PRIMARY KEY,

    target_id BIGINT NOT NULL,

    -- 申请用户信息。
    applicant_id BIGINT NOT NULL,
    applicant_name VARCHAR(128) NOT NULL DEFAULT '',
    applicant_avatar VARCHAR(256) NOT NULL DEFAULT '',

    -- 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )
    source VARCHAR(16) NOT NULL DEFAULT 'search',

    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_contact_application_source_0 CHECK (source IN ('search', 'qrcode', 'group')),
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_contact_application_status_0 CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE contact_application_0 IS '联系人申请表 ( 加好友申请 )';
COMMENT ON COLUMN contact_application_0.id IS '主键 ID';
COMMENT ON COLUMN contact_application_0.target_id IS '目标用户 ID';
COMMENT ON COLUMN contact_application_0.applicant_id IS '申请人 ID';
COMMENT ON COLUMN contact_application_0.applicant_name IS '申请人昵称';
COMMENT ON COLUMN contact_application_0.applicant_avatar IS '申请人头像';
COMMENT ON COLUMN contact_application_0.source IS '添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )';
COMMENT ON COLUMN contact_application_0.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN contact_application_0.application_message IS '申请验证消息';
COMMENT ON COLUMN contact_application_0.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application_0.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application_0.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_contact_application_target_0 ON contact_application_0(target_id);

CREATE INDEX idx_contact_application_pending_0 ON contact_application_0(target_id, application_status) WHERE application_status = 'pending';

-- 分表 1: contact_application_1
DROP TABLE IF EXISTS contact_application_1;
CREATE TABLE contact_application_1 (
    id BIGINT PRIMARY KEY,

    target_id BIGINT NOT NULL,

    -- 申请用户信息。
    applicant_id BIGINT NOT NULL,
    applicant_name VARCHAR(128) NOT NULL DEFAULT '',
    applicant_avatar VARCHAR(256) NOT NULL DEFAULT '',

    -- 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )
    source VARCHAR(16) NOT NULL DEFAULT 'search',

    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_contact_application_source_1 CHECK (source IN ('search', 'qrcode', 'group')),
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_contact_application_status_1 CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE contact_application_1 IS '联系人申请表 ( 加好友申请 )';
COMMENT ON COLUMN contact_application_1.id IS '主键 ID';
COMMENT ON COLUMN contact_application_1.target_id IS '目标用户 ID';
COMMENT ON COLUMN contact_application_1.applicant_id IS '申请人 ID';
COMMENT ON COLUMN contact_application_1.applicant_name IS '申请人昵称';
COMMENT ON COLUMN contact_application_1.applicant_avatar IS '申请人头像';
COMMENT ON COLUMN contact_application_1.source IS '添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )';
COMMENT ON COLUMN contact_application_1.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN contact_application_1.application_message IS '申请验证消息';
COMMENT ON COLUMN contact_application_1.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application_1.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application_1.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_contact_application_target_1 ON contact_application_1(target_id);

CREATE INDEX idx_contact_application_pending_1 ON contact_application_1(target_id, application_status) WHERE application_status = 'pending';

-- 分表 2: contact_application_2
DROP TABLE IF EXISTS contact_application_2;
CREATE TABLE contact_application_2 (
    id BIGINT PRIMARY KEY,

    target_id BIGINT NOT NULL,

    -- 申请用户信息。
    applicant_id BIGINT NOT NULL,
    applicant_name VARCHAR(128) NOT NULL DEFAULT '',
    applicant_avatar VARCHAR(256) NOT NULL DEFAULT '',

    -- 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )
    source VARCHAR(16) NOT NULL DEFAULT 'search',

    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_contact_application_source_2 CHECK (source IN ('search', 'qrcode', 'group')),
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_contact_application_status_2 CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE contact_application_2 IS '联系人申请表 ( 加好友申请 )';
COMMENT ON COLUMN contact_application_2.id IS '主键 ID';
COMMENT ON COLUMN contact_application_2.target_id IS '目标用户 ID';
COMMENT ON COLUMN contact_application_2.applicant_id IS '申请人 ID';
COMMENT ON COLUMN contact_application_2.applicant_name IS '申请人昵称';
COMMENT ON COLUMN contact_application_2.applicant_avatar IS '申请人头像';
COMMENT ON COLUMN contact_application_2.source IS '添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )';
COMMENT ON COLUMN contact_application_2.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN contact_application_2.application_message IS '申请验证消息';
COMMENT ON COLUMN contact_application_2.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application_2.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application_2.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_contact_application_target_2 ON contact_application_2(target_id);

CREATE INDEX idx_contact_application_pending_2 ON contact_application_2(target_id, application_status) WHERE application_status = 'pending';

-- 分表 3: contact_application_3
DROP TABLE IF EXISTS contact_application_3;
CREATE TABLE contact_application_3 (
    id BIGINT PRIMARY KEY,

    target_id BIGINT NOT NULL,

    -- 申请用户信息。
    applicant_id BIGINT NOT NULL,
    applicant_name VARCHAR(128) NOT NULL DEFAULT '',
    applicant_avatar VARCHAR(256) NOT NULL DEFAULT '',

    -- 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )
    source VARCHAR(16) NOT NULL DEFAULT 'search',

    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_contact_application_source_3 CHECK (source IN ('search', 'qrcode', 'group')),
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_contact_application_status_3 CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE contact_application_3 IS '联系人申请表 ( 加好友申请 )';
COMMENT ON COLUMN contact_application_3.id IS '主键 ID';
COMMENT ON COLUMN contact_application_3.target_id IS '目标用户 ID';
COMMENT ON COLUMN contact_application_3.applicant_id IS '申请人 ID';
COMMENT ON COLUMN contact_application_3.applicant_name IS '申请人昵称';
COMMENT ON COLUMN contact_application_3.applicant_avatar IS '申请人头像';
COMMENT ON COLUMN contact_application_3.source IS '添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )';
COMMENT ON COLUMN contact_application_3.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN contact_application_3.application_message IS '申请验证消息';
COMMENT ON COLUMN contact_application_3.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application_3.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application_3.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 创建索引
CREATE INDEX idx_contact_application_target_3 ON contact_application_3(target_id);

CREATE INDEX idx_contact_application_pending_3 ON contact_application_3(target_id, application_status) WHERE application_status = 'pending';

