package repo

import (
	"context"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo/dao"
)

type MessageRepo interface {
	Create(ctx context.Context, message domain.Message) error
}

var _ MessageRepo = (*DefaultMessageRepo)(nil)

type DefaultMessageRepo struct {
	messageDao dao.MessageDao
}

func NewDefaultMessageRepo(messageDao dao.MessageDao) *DefaultMessageRepo {
	return &DefaultMessageRepo{messageDao: messageDao}
}

func (r *DefaultMessageRepo) Create(_ context.Context, _ domain.Message) error {
	// TODO: not implemented
	panic("not implemented")
}
