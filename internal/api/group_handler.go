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

var _ xgin.RouteRegistry = (*GroupHandler)(nil)

type GroupHandler struct {
	svc    service.GroupService
	logger *zap.Logger
}

func NewGroupHandler(svc service.GroupService, logger *zap.Logger) *GroupHandler {
	return &GroupHandler{
		svc:    svc,
		logger: logger,
	}
}

func (h *GroupHandler) Register(engine *gin.Engine) {
	groupv1 := engine.Group("api/v1/group")

	groupv1.Handle(http.MethodPost, "/create", xgin.BU(h.CreateGroup))
}

type createGroupReq struct {
	Name      string   `json:"name"`
	Avatar    string   `json:"avatar"`
	MemberIDs []uint64 `json:"memberIds"`
}

// createGroup 创建群聊。
func (h *GroupHandler) CreateGroup(ctx *gin.Context, req createGroupReq, au xgin.ContextUser) (xgin.R, error) {
	if req.Name == "" {
		return xgin.R{}, fmt.Errorf("%w: name is required", errs.ErrInvalidParam)
	}

	event := domain.GroupCreateEvent{
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
