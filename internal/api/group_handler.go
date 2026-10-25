package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/service"
)

var _ xgin.RouteRegistry = (*GroupHandler)(nil)

type GroupHandler struct {
	svc service.GroupService
}

func NewGroupHandler(svc service.GroupService) *GroupHandler {
	return &GroupHandler{
		svc: svc,
	}
}

func (h *GroupHandler) Register(engine *gin.Engine) {
	groupv1 := engine.Group("api/v1/group")

	groupv1.Handle(http.MethodPost, "/create", xgin.BU(h.CreateGroup))
}

type createGroupReq struct {
	Name      string     `json:"name"`
	Avatar    string     `json:"avatar"`
	MemberIDs []xgin.U64 `json:"memberIds"`
}

// createGroup 创建群聊。
func (h *GroupHandler) CreateGroup(ctx *gin.Context, req createGroupReq, au xgin.ContextUser) (xgin.R, error) {
	if req.Name == "" {
		return xgin.R{}, fmt.Errorf("%w: name is required", errs.ErrInvalidParam)
	}

	memberIDs := make([]uint64, 0, len(req.MemberIDs))
	for _, id := range req.MemberIDs {
		memberIDs = append(memberIDs, uint64(id))
	}

	group, err := h.svc.Create(ctx, domain.GroupCreateEvent{
		Name:      req.Name,
		Avatar:    req.Avatar,
		CreatorID: au.UID,

		MemberIDs: memberIDs,
	})
	if err != nil {
		return xgin.R{}, err
	}

	return xgin.R{
		Code: http.StatusOK,
		Data: createGroupResp{
			ID:            xgin.U64(group.ID),
			Name:          group.GroupName,
			Avatar:        group.GroupAvatar,
			Status:        string(group.GroupStatus),
			CreatorID:     xgin.U64(group.CreatorID),
			LastMessageAt: group.LastMessageAt,
			CreatedAt:     group.CreatedAt,
		},
	}, nil
}

type createGroupResp struct {
	ID     xgin.U64 `json:"id"`
	Name   string   `json:"name"`
	Avatar string   `json:"avatar"`
	Status string   `json:"status"`

	CreatorID xgin.U64 `json:"creatorId"`

	LastMessageAt int64 `json:"lastMessageAt"`
	CreatedAt     int64 `json:"createdAt"`
}
