package repo

import (
	"github.com/jrmarcco/hermet/internal/pkg/sharding"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"github.com/jrmarcco/jit/xsync"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var RepoFxModule = fx.Module(
	"repo",
	fx.Provide(
		// user repo ( 用户 )
		fx.Annotate(NewDefaultUserRepo, fx.As(new(UserRepo))),

		// contact application repo ( 联系人申请 )
		fx.Annotate(NewDefaultContactApplicationRepo, fx.As(new(ContactApplicationRepo))),
		// contact repo ( 联系人 )
		fx.Annotate(newContactRepo, fx.As(new(ContactRepo))),

		// group application repo ( 聊天群申请 )
		fx.Annotate(NewDefaultGroupApplicationRepo, fx.As(new(GroupApplicationRepo))),
		// group repo ( 聊天群 )
		fx.Annotate(NewDefaultGroupRepo, fx.As(new(GroupRepo))),
		// group member repo ( 聊天群成员 )
		fx.Annotate(newGroupMemberRepo, fx.As(new(GroupMemberRepo))),

		// user group repo ( 用户群组 )
		fx.Annotate(newUserGroupRepo, fx.As(new(UserGroupRepo))),

		// channel repo ( 用户频道 )
		fx.Annotate(newChannelRepo, fx.As(new(ChannelRepo))),

		// message repo ( 消息正本 + 收件箱写扩散 )
		fx.Annotate(newMessageRepo, fx.As(new(MessageRepo))),
	),
)

type contactRepoFxParams struct {
	fx.In

	UserDao        dao.UserDao
	ContactDao     dao.ContactDao
	ContactViewDao dao.ContactViewDao
	ContactRidxDao dao.ContactRidxDao

	DBs            *xsync.Map[string, *gorm.DB] `name:"db_sharding_clients"`
	CShardHelper   *sharding.ShardHelper        `name:"contact_shard_helper"`
	CvShardHelper  *sharding.ShardHelper        `name:"contact_view_shard_helper"`
	CriShardHelper *sharding.ShardHelper        `name:"contact_reverse_index_shard_helper"`
}

func newContactRepo(params contactRepoFxParams) ContactRepo {
	return NewTxContactRepo(
		params.UserDao,
		params.ContactDao,
		params.ContactViewDao,
		params.ContactRidxDao,

		params.DBs,
		params.CShardHelper,
		params.CvShardHelper,
		params.CriShardHelper,
	)
}

type groupMemberRepoFxParams struct {
	fx.In

	UserDao        dao.UserDao
	OutboxEventDao dao.OutboxEventDao

	GroupMemberDao     dao.GroupMemberDao
	GroupMemberViewDao dao.GroupMemberViewDao

	DBs            *xsync.Map[string, *gorm.DB] `name:"db_sharding_clients"`
	OEShardHelper  *sharding.ShardHelper        `name:"outbox_event_shard_helper"`
	GmvShardHelper *sharding.ShardHelper        `name:"group_member_view_shard_helper"`
}

func newGroupMemberRepo(params groupMemberRepoFxParams) GroupMemberRepo {
	return NewTxGroupMemberRepo(
		params.UserDao,
		params.OutboxEventDao,

		params.GroupMemberDao,
		params.GroupMemberViewDao,

		params.DBs,
		params.OEShardHelper,
		params.GmvShardHelper,
	)
}

type userGroupRepoFxParams struct {
	fx.In

	GroupDao         dao.GroupDao
	UserGroupDao     dao.UserGroupDao
	UserGroupViewDao dao.UserGroupViewDao
	UserGroupRidxDao dao.UserGroupRidxDao

	DBs             *xsync.Map[string, *gorm.DB] `name:"db_sharding_clients"`
	UgShardHelper   *sharding.ShardHelper        `name:"user_group_shard_helper"`
	UgvShardHelper  *sharding.ShardHelper        `name:"user_group_view_shard_helper"`
	UgriShardHelper *sharding.ShardHelper        `name:"user_group_reverse_index_shard_helper"`
}

func newUserGroupRepo(params userGroupRepoFxParams) UserGroupRepo {
	return NewTxUserGroupRepo(
		params.GroupDao,
		params.UserGroupDao,
		params.UserGroupViewDao,
		params.UserGroupRidxDao,

		params.DBs,
		params.UgShardHelper,
		params.UgvShardHelper,
		params.UgriShardHelper,
	)
}

type channelRepoFxParams struct {
	fx.In

	ChannelDao     dao.ChannelDao
	ChannelRidxDao dao.ChannelRidxDao

	DBs              *xsync.Map[string, *gorm.DB] `name:"db_sharding_clients"`
	CShardHelper     *sharding.ShardHelper        `name:"channel_shard_helper"`
	CRidxShardHelper *sharding.ShardHelper        `name:"channel_reverse_index_shard_helper"`
}

func newChannelRepo(params channelRepoFxParams) ChannelRepo {
	return NewDefaultChannelRepo(
		params.ChannelDao,
		params.ChannelRidxDao,
		params.DBs,
		params.CShardHelper,
		params.CRidxShardHelper,
	)
}

type messageRepoFxParams struct {
	fx.In

	MessageDao         dao.MessageDao
	ConversationSeqDao dao.ConversationSeqDao
	InboxDao           dao.InboxDao
	ChannelDao         dao.ChannelDao

	DBs *xsync.Map[string, *gorm.DB] `name:"db_sharding_clients"`

	MsgShardHelper     *sharding.ShardHelper `name:"message_shard_helper"`
	ConvSeqShardHelper *sharding.ShardHelper `name:"conversation_seq_shard_helper"`
	InboxShardHelper   *sharding.ShardHelper `name:"inbox_entry_shard_helper"`
	ChannelShardHelper *sharding.ShardHelper `name:"channel_shard_helper"`
}

func newMessageRepo(params messageRepoFxParams) MessageRepo {
	return NewTxMessageRepo(
		params.MessageDao,
		params.ConversationSeqDao,
		params.InboxDao,
		params.ChannelDao,

		params.DBs,
		params.MsgShardHelper,
		params.ConvSeqShardHelper,
		params.InboxShardHelper,
		params.ChannelShardHelper,
	)
}
