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
		fx.Annotate(newGroupRepo, fx.As(new(GroupRepo))),

		// user group repo ( 用户群组 )
		fx.Annotate(newUserGroupRepo, fx.As(new(UserGroupRepo))),

		// channel repo ( 用户频道 )
		fx.Annotate(NewDefaultChannelRepo, fx.As(new(ChannelRepo))),

		// message repo ( 消息 )
		fx.Annotate(NewDefaultMessageRepo, fx.As(new(MessageRepo))),
	),
)

type contactRepoFxParams struct {
	fx.In

	UserDao        dao.UserDao
	ContactDao     dao.ContactDao
	ContactViewDao dao.ContactViewDao
	ContactRidxDao dao.ContactRidxDao

	DBs            *xsync.Map[string, *gorm.DB] `name:"db_sharding_clients"`
	ShardHelper    *sharding.ShardHelper        `name:"contact_shard_helper"`
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
		params.ShardHelper,
		params.CvShardHelper,
		params.CriShardHelper,
	)
}

type groupRepoFxParams struct {
	fx.In

	UserDao              dao.UserDao
	GroupDao             dao.GroupDao
	GroupMemberDao       dao.GroupMemberDao
	GroupMemberViewDao   dao.GroupMemberViewDao
	GroupMemberOutboxDao dao.GroupMemberOutboxDao

	DBs            *xsync.Map[string, *gorm.DB] `name:"db_sharding_clients"`
	ShardHelper    *sharding.ShardHelper        `name:"group_shard_helper"`
	GMShardHelper  *sharding.ShardHelper        `name:"group_member_shard_helper"`
	GmvShardHelper *sharding.ShardHelper        `name:"group_member_view_shard_helper"`
	GmoShardHelper *sharding.ShardHelper        `name:"group_member_outbox_shard_helper"`
}

func newGroupRepo(params groupRepoFxParams) GroupRepo {
	return NewTxGroupRepo(
		params.UserDao,
		params.GroupDao,
		params.GroupMemberDao,
		params.GroupMemberViewDao,
		params.GroupMemberOutboxDao,

		params.DBs,
		params.ShardHelper,
		params.GMShardHelper,
		params.GmvShardHelper,
		params.GmoShardHelper,
	)
}

type userGroupRepoFxParams struct {
	fx.In

	GroupDao         dao.GroupDao
	UserGroupDao     dao.UserGroupDao
	UserGroupViewDao dao.UserGroupViewDao
	UserGroupRidxDao dao.UserGroupRidxDao

	DBs             *xsync.Map[string, *gorm.DB] `name:"db_sharding_clients"`
	ShardHelper     *sharding.ShardHelper        `name:"user_group_shard_helper"`
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
		params.ShardHelper,
		params.UgvShardHelper,
		params.UgriShardHelper,
	)
}
