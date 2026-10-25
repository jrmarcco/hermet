package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/pkg/xmq"
	"github.com/jrmarcco/hermet/internal/service"
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ xgin.RouteRegistry = (*AdminHandler)(nil)

// AdminController 是临时管理员 API 的 HTTP Handler。
// 注意：这个 API 只用于临时接口测试使用，后续需要移除。
type AdminHandler struct {
	userService    service.UserService
	messageService service.MessageService
}

func NewAdminHandler(userService service.UserService, messageService service.MessageService) *AdminHandler {
	return &AdminHandler{
		userService:    userService,
		messageService: messageService,
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
	if req.Email == "" && req.Mobile == "" {
		return xgin.R{}, fmt.Errorf("%w: email or mobile is required", errs.ErrInvalidParam)
	}
	if req.Nickname == "" {
		return xgin.R{}, fmt.Errorf("%w: nickname is required", errs.ErrInvalidParam)
	}

	// gender 缺省按 unknown 处理，非法取值直接拒绝——
	// 否则会穿透到数据库触发 check 约束，以 500 泄漏内部 SQL 错误。
	gender := domain.UserGender(req.Gender)
	if gender == "" {
		gender = domain.UserGenderUnknown
	}
	if !gender.IsValid() {
		return xgin.R{}, fmt.Errorf("%w: invalid gender", errs.ErrInvalidParam)
	}

	if err := h.userService.AddUser(ctx, domain.BizUser{
		Email:    req.Email,
		Mobile:   req.Mobile,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Gender:   gender,
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
	SenderID        xgin.U64 `json:"senderId"`
	DestinationID   xgin.U64 `json:"destinationId"`
	DestinationType string   `json:"destinationType"`
	Content         string   `json:"content"`
}

func (h *AdminHandler) SendMessage(ctx *gin.Context, req sendMsgRequest) (xgin.R, error) {
	body := domain.Message{
		ID:  uuid.NewString(),
		SID: uint64(req.SenderID),

		DestinationID:   uint64(req.DestinationID),
		DestinationType: domain.ChannelType(req.DestinationType),

		Content:     []byte(req.Content),
		ContentType: domain.ContentTypeText,

		SendAt: time.Now().UnixMilli(),
	}

	bodyVal, err := json.Marshal(body)
	if err != nil {
		return xgin.R{}, err
	}

	// 模拟网关盖章后的 UpstreamMessage ( protojson 编码 )。
	senderID := uint64(req.SenderID)
	if senderID == 0 {
		senderID = 1
	}
	upstream := &messagev1.UpstreamMessage{
		MessageId:  body.ID,
		BusinessId: service.DefaultBusinessID,
		UserId:     senderID,

		Body: bodyVal,

		OccurredAt: timestamppb.Now(),
	}
	upstreamVal, err := protojson.Marshal(upstream)
	if err != nil {
		return xgin.R{}, err
	}

	if err := h.messageService.ProcessUpstreamMsg(ctx, &xmq.Message{Val: upstreamVal}); err != nil {
		return xgin.R{}, err
	}

	return xgin.R{
		Code: http.StatusOK,
	}, nil
}
