package repo

type ChannelRepo interface{}

var _ ChannelRepo = (*DefaultChannelRepo)(nil)

type DefaultChannelRepo struct{}
