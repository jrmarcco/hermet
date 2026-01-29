package service

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo"
)

type ChannelService interface {
	GetChannels(ctx context.Context, userID uint64) ([]domain.Channel, error)
}

var _ ChannelService = (*DefaultChannelService)(nil)

type DefaultChannelService struct {
	channelRepo repo.ChannelRepo
}

func NewDefaultChannelService(channelRepo repo.ChannelRepo) *DefaultChannelService {
	return &DefaultChannelService{
		channelRepo: channelRepo,
	}
}

func (s *DefaultChannelService) GetChannels(ctx context.Context, userID uint64) ([]domain.Channel, error) {
	return s.channelRepo.ListByUserID(ctx, userID)
}
