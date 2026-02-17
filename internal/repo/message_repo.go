package repo

import "github.com/jrmarcco/hermet/internal/repo/dao"

type MessageRepo interface{}

var _ MessageRepo = (*DefaultMessageRepo)(nil)

type DefaultMessageRepo struct {
	messageDao dao.MessageDao
}

func NewDefaultMessageRepo(messageDao dao.MessageDao) *DefaultMessageRepo {
	return &DefaultMessageRepo{messageDao: messageDao}
}
