package repo

import (
	"context"
	"errors"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/repo/dao"
	"gorm.io/gorm"
)

type ConversationRepo interface {
	// Create 创建用户会话。
	Create(ctx context.Context, conversation domain.ConversationView) error

	ListViewByUserID(ctx context.Context, userID uint64) ([]domain.ConversationView, error)
}

var _ ConversationRepo = (*DefaultConversationRepo)(nil)

type DefaultConversationRepo struct {
	bizUserDao      dao.BizUserDao
	conversationDao dao.ConversationDao
}

func NewDefaultConversationRepo(bizUserDao dao.BizUserDao, conversationDao dao.ConversationDao) *DefaultConversationRepo {
	return &DefaultConversationRepo{
		bizUserDao:      bizUserDao,
		conversationDao: conversationDao,
	}
}

func (r *DefaultConversationRepo) Create(ctx context.Context, conversation domain.ConversationView) error {
	user, err := r.bizUserDao.FindByID(ctx, conversation.TargetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrRecordNotFound
		}
		return err
	}

	conversation.TargetInfoVer = user.InfoVer
	conversation.ConversationName = user.Nickname
	conversation.ConversationAvatar = user.Avatar

	err = r.conversationDao.CreateView(ctx, r.toEntity(conversation))
	if err != nil {
		return err
	}
	return nil
}

func (r *DefaultConversationRepo) ListViewByUserID(ctx context.Context, userID uint64) ([]domain.ConversationView, error) {
	entities, err := r.conversationDao.ListViewByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	domains := make([]domain.ConversationView, 0, len(entities))
	for i := range entities {
		domains = append(domains, r.toDomain(entities[i]))
	}
	return domains, nil
}

func (r *DefaultConversationRepo) toEntity(conversation domain.ConversationView) dao.ConversationView {
	return dao.ConversationView{
		ID: conversation.ID,

		UserID:        conversation.UserID,
		TargetID:      conversation.TargetID,
		TargetInfoVer: conversation.TargetInfoVer,

		ConversationType:   string(conversation.ConversationType),
		ConversationName:   conversation.ConversationName,
		ConversationRemark: conversation.ConversationRemark,
		ConversationAvatar: conversation.ConversationAvatar,

		IsMuted:   conversation.IsMuted,
		IsPinned:  conversation.IsPinned,
		IsHidden:  conversation.IsHidden,
		IsStarred: conversation.IsStarred,

		LastMessageID:         conversation.LastMessageID,
		LastMessageType:       conversation.LastMessageType,
		LastMessageContent:    conversation.LastMessageContent,
		LastMessageSenderID:   conversation.LastMessageSenderID,
		LastMessageSenderName: conversation.LastMessageSenderName,
		LastMessageTime:       conversation.LastMessageTime,

		UnreadCount:  conversation.UnreadCount,
		MentionCount: conversation.MentionCount,

		OpenedAt: conversation.OpenedAt,
		ClosedAt: conversation.ClosedAt,

		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
	}
}

func (r *DefaultConversationRepo) toDomain(conversation dao.ConversationView) domain.ConversationView {
	return domain.ConversationView{
		ID: conversation.ID,

		UserID:        conversation.UserID,
		TargetID:      conversation.TargetID,
		TargetInfoVer: conversation.TargetInfoVer,

		ConversationType:   domain.ConversationType(conversation.ConversationType),
		ConversationName:   conversation.ConversationName,
		ConversationRemark: conversation.ConversationRemark,
		ConversationAvatar: conversation.ConversationAvatar,

		IsMuted:   conversation.IsMuted,
		IsPinned:  conversation.IsPinned,
		IsHidden:  conversation.IsHidden,
		IsStarred: conversation.IsStarred,

		LastMessageID:         conversation.LastMessageID,
		LastMessageType:       conversation.LastMessageType,
		LastMessageContent:    conversation.LastMessageContent,
		LastMessageSenderID:   conversation.LastMessageSenderID,
		LastMessageSenderName: conversation.LastMessageSenderName,
		LastMessageTime:       conversation.LastMessageTime,

		UnreadCount:  conversation.UnreadCount,
		MentionCount: conversation.MentionCount,

		OpenedAt: conversation.OpenedAt,
		ClosedAt: conversation.ClosedAt,

		CreatedAt: conversation.CreatedAt,
		UpdatedAt: conversation.UpdatedAt,
	}
}
