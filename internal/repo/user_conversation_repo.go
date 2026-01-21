package repo

type UserConversationRepo interface{}

var _ UserConversationRepo = (*DefaultUserConversationRepo)(nil)

type DefaultUserConversationRepo struct{}

func NewDefaultUserConversationRepo() *DefaultUserConversationRepo {
	return &DefaultUserConversationRepo{}
}
