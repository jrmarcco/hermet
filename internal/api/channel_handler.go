package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/service"
	"go.uber.org/zap"
)

var _ xgin.RouteRegistry = (*ChannelHandler)(nil)

type ChannelHandler struct {
	svc    service.ChannelService
	logger *zap.Logger
}

func NewChannelHandler(svc service.ChannelService, logger *zap.Logger) *ChannelHandler {
	return &ChannelHandler{
		svc:    svc,
		logger: logger,
	}
}

func (h *ChannelHandler) Register(engine *gin.Engine) {
	channelv1 := engine.Group("api/v1/channel")

	channelv1.Handle(http.MethodPost, "/create", xgin.BU(h.CreateChannel))
}

type createChannelReq struct {
	Name      string   `json:"name"`
	Avatar    string   `json:"avatar"`
	MemberIDs []uint64 `json:"memberIds"`
}

// createChannel 创建群聊频道。
func (h *ChannelHandler) CreateChannel(ctx *gin.Context, req createChannelReq, au xgin.ContextUser) (xgin.R, error) {
	if req.Name == "" {
		return xgin.R{}, fmt.Errorf("%w: name is required", errs.ErrInvalidParam)
	}

	event := domain.ChannelCreateEvent{
		Name:    req.Name,
		Avatar:  req.Avatar,
		Creator: au.UID,

		MemberIDs: req.MemberIDs,
	}

	channel, err := h.svc.Create(ctx, event)
	if err != nil {
		return xgin.R{}, err
	}

	return xgin.R{
		Code: http.StatusOK,
		Data: channel,
	}, nil
}
