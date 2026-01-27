package service

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo"
)

type ConversationService interface {
	GetConversations(ctx context.Context, userID uint64) ([]domain.ConversationView, error)
}

var _ ConversationService = (*DefaultConversationService)(nil)

type DefaultConversationService struct {
	conversationRepo repo.ConversationRepo
}

func NewDefaultConversationService(conversationRepo repo.ConversationRepo) *DefaultConversationService {
	return &DefaultConversationService{
		conversationRepo: conversationRepo,
	}
}

func (s *DefaultConversationService) GetConversations(ctx context.Context, userID uint64) ([]domain.ConversationView, error) {
	return s.conversationRepo.ListViewByUserID(ctx, userID)
}
