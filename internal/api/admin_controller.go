package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/service"
	"go.uber.org/zap"
)

var _ xgin.RouteRegistry = (*AdminHandler)(nil)

// AdminController 是临时管理员 API 的 HTTP Handler。
// 注意：这个 API 只用于临时接口测试使用，后续需要移除。
type AdminHandler struct {
	userService    service.UserService
	messageService service.MessageService
	logger         *zap.Logger
}

func NewAdminHandler(userService service.UserService, messageService service.MessageService, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{
		userService:    userService,
		messageService: messageService,
		logger:         logger,
	}
}

func (h *AdminHandler) Register(engine *gin.Engine) {
	adminV1 := engine.Group("api/v1/admin")

	adminV1.Handle(http.MethodPost, "/user", xgin.B(h.AddUser))
	adminV1.Handle(http.MethodPost, "/message", xgin.B(h.SendMessage))
}

type addUserRequest struct {
	Email  string `json:"email"`
	Mobile string `json:"mobile"`

	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Gender   string `json:"gender"`
	Region   string `json:"region"`
	Birthday int64  `json:"birthday"`
	Tagline  string `json:"tagline"`
}

func (h *AdminHandler) AddUser(ctx *gin.Context, req addUserRequest) (xgin.R, error) {
	if err := h.userService.AddUser(ctx, domain.BizUser{
		Email:    req.Email,
		Mobile:   req.Mobile,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Gender:   domain.UserGender(req.Gender),
		Region:   req.Region,
		Birthday: req.Birthday,
		Tagline:  req.Tagline,
	}); err != nil {
		return xgin.R{}, err
	}

	return xgin.R{
		Code: http.StatusOK,
	}, nil
}

type sendMsgRequest struct {
	DestinationID   uint64 `json:"destinationId"`
	DestinationType string `json:"destinationType"`
	Content         string `json:"content"`
}

func (h *AdminHandler) SendMessage(ctx *gin.Context, req sendMsgRequest) (xgin.R, error) {
	msg := domain.Message{
		ID:  uuid.New().String(),
		SID: 1,
		MID: 1,

		DestinationID:   req.DestinationID,
		DestinationType: domain.ChannelType(req.DestinationType),
		Content:         []byte(req.Content),
		ContentType:     domain.ContentTypeText,
		SendAt:          time.Now().UnixMilli(),
	}

	val, err := json.Marshal(msg)
	if err != nil {
		return xgin.R{}, err
	}

	if err := h.messageService.ProcessUpstreamMsg(ctx, &xmq.Message{Val: val}); err != nil {
		return xgin.R{}, err
	}

	return xgin.R{
		Code: http.StatusOK,
	}, nil
}
