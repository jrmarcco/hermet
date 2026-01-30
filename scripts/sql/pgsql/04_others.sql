-- ============================================================================
-- 其他业务表
-- ============================================================================

-- 聊天群成员 outbox 事件表
-- 注意:
--  这不是标准 outbox 表
--  在系统中仅用于分流 group_member 表的 cdc 事件
--  从而保证 user_group 的创建 ( user_group 和 group_member 分片键不同 )
DROP TABLE IF EXISTS group_member_outbox_event;
CREATE TABLE group_member_outbox_event (
    id BIGINT PRIMARY KEY,

    group_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    op_type CHAR(1) NOT NULL,

    joined_at BIGINT NOT NULL,
    left_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,

    -- 约束
    CONSTRAINT uk_group_member_outbox_event UNIQUE(group_id, user_id, op_type)
    -- 操作类型 - 枚举 ( c=创建 / u=更新 / d=删除 )
    CONSTRAINT chk_group_member_outbox_event_op_type CHECK (op_type IN ('c', 'u', 'd'))
);

COMMENT ON TABLE group_member_outbox_event IS '聊天群成员 outbox 事件表 ( 用于分流 group_member 表的 cdc 事件 )';
COMMENT ON COLUMN group_member_outbox_event.id IS '主键 ID';
COMMENT ON COLUMN group_member_outbox_event.group_id IS '聊天群 ID';
COMMENT ON COLUMN group_member_outbox_event.user_id IS '用户 ID';
COMMENT ON COLUMN group_member_outbox_event.op_type IS '操作类型 ( c=创建 / u=更新 / d=删除 )';
COMMENT ON COLUMN group_member_outbox_event.joined_at IS '加入时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN group_member_outbox_event.left_at IS '退出时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN group_member_outbox_event.created_at IS '创建时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_group_member_outbox_event_group ON group_member_outbox_event(group_id);
CREATE INDEX idx_group_member_outbox_event_user ON group_member_outbox_event(user_id);


-- 联系人申请表 ( 加好友申请 - 以 TargetID 为分片键 )
DROP TABLE IF EXISTS contact_application;
CREATE TABLE contact_application (
    id BIGINT PRIMARY KEY,

    target_id BIGINT NOT NULL,
    applicant_id BIGINT NOT NULL,

    -- 申请用户信息。
    applicant_name VARCHAR(64) NOT NULL DEFAULT '',
    applicant_avatar VARCHAR(256) NOT NULL DEFAULT '',
    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    -- 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )
    source VARCHAR(16) NOT NULL DEFAULT 'search',

    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 添加来源 - 枚举 ( search=搜索 / qrcode=扫码 / group=群聊 )
    CONSTRAINT chk_contact_application_source CHECK (source IN ('search', 'qrcode', 'group')),
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_contact_application_status CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE contact_application IS '联系人申请表 ( 加好友申请 )';
COMMENT ON COLUMN contact_application.id IS '主键 ID';
COMMENT ON COLUMN contact_application.target_id IS '目标用户 ID';
COMMENT ON COLUMN contact_application.applicant_id IS '申请人 ID';
COMMENT ON COLUMN contact_application.applicant_name IS '申请人昵称';
COMMENT ON COLUMN contact_application.applicant_avatar IS '申请人头像';
COMMENT ON COLUMN contact_application.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN contact_application.application_message IS '申请验证消息';
COMMENT ON COLUMN contact_application.source IS '添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )';
COMMENT ON COLUMN contact_application.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN contact_application.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_contact_application_target ON contact_application(target_id);
CREATE INDEX idx_contact_application_pending ON contact_application(target_id, application_status) WHERE application_status = 'pending';


-- 频道申请表 ( 入群申请 - 以 ChannelID 为分片键 )
DROP TABLE IF EXISTS channel_application;
CREATE TABLE channel_application (
    id BIGINT PRIMARY KEY,

    applicant_id BIGINT NOT NULL,

    -- 目标渠道信息。
    channel_id BIGINT NOT NULL,
    channel_name VARCHAR(64) NOT NULL DEFAULT '',
    channel_avatar VARCHAR(256) NOT NULL DEFAULT '',

    application_status VARCHAR(16) NOT NULL DEFAULT 'pending',
    application_message VARCHAR(256) NOT NULL DEFAULT '',

    reviewer_id BIGINT NOT NULL DEFAULT 0,
    reviewed_at BIGINT NOT NULL DEFAULT 0,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    -- 约束
    -- 申请状态 - 枚举 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )
    CONSTRAINT chk_channel_application_status CHECK (application_status IN ('pending', 'approved', 'rejected'))
);

COMMENT ON TABLE channel_application IS '频道申请表 ( 入群申请 )';
COMMENT ON COLUMN channel_application.id IS '主键 ID';
COMMENT ON COLUMN channel_application.applicant_id IS '申请人 ID';
COMMENT ON COLUMN channel_application.channel_id IS '频道 ID';
COMMENT ON COLUMN channel_application.channel_name IS '频道名称';
COMMENT ON COLUMN channel_application.channel_avatar IS '频道头像';
COMMENT ON COLUMN channel_application.application_status IS '申请状态 ( pending=待处理 / approved=已批准 / rejected=已拒绝 )';
COMMENT ON COLUMN channel_application.application_message IS '申请验证消息';
COMMENT ON COLUMN channel_application.reviewer_id IS '审批人 ID';
COMMENT ON COLUMN channel_application.reviewed_at IS '审批时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application.created_at IS '创建时间戳 ( Unix 毫秒值 )';
COMMENT ON COLUMN channel_application.updated_at IS '更新时间戳 ( Unix 毫秒值 )';

-- 索引
CREATE INDEX idx_channel_application_channel ON channel_application(channel_id);
CREATE INDEX idx_channel_application_pending ON channel_application(channel_id, application_status) WHERE application_status = 'pending';
