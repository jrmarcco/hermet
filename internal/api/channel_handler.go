package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/service"
)

var _ xgin.RouteRegistry = (*ChannelHandler)(nil)

type ChannelHandler struct {
	svc service.ChannelService
}

func NewChannelHandler(svc service.ChannelService) *ChannelHandler {
	return &ChannelHandler{
		svc: svc,
	}
}

func (h *ChannelHandler) Register(engine *gin.Engine) {
	channelV1 := engine.Group("api/v1/channel")

	channelV1.Handle(http.MethodGet, "/channels", xgin.U(h.GetChannels))
}

type getChannelResp struct {
	ID xgin.U64 `json:"id"`

	// 会话目的地 ID ( 用于唯一标识一个会话 )。
	// 取值规则:
	//
	//  - ChannelType = single
	// 	  DestinationID = min(UserID, TargetID):max(UserID, TargetID)。
	//  - ChannelType = group
	// 	  DestinationID = TargetID。
	DestinationID string `json:"destination"`

	UserID        xgin.U64 `json:"userId"`
	TargetID      xgin.U64 `json:"targetId"`
	TargetInfoVer int      `json:"targetInfoVer"`

	ChannelType   string `json:"channelType"`
	ChannelName   string `json:"channelName"`
	ChannelRemark string `json:"channelRemark"`
	ChannelAvatar string `json:"channelAvatar"`

	IsMuted   bool `json:"isMuted"`
	IsPinned  bool `json:"isPinned"`
	IsHidden  bool `json:"isHidden"`
	IsStarred bool `json:"isStarred"`

	LastMessageID         xgin.U64 `json:"lastMessageId"`
	LastMessageType       string   `json:"lastMessageType"`
	LastMessageContent    string   `json:"lastMessageContent"`
	LastMessageSenderID   xgin.U64 `json:"lastMessageSenderId"`
	LastMessageSenderName string   `json:"lastMessageSenderName"`
	LastMessageTime       int64    `json:"lastMessageTime"`

	UnreadCount  int `json:"unreadCount"`
	MentionCount int `json:"mentionCount"`
}

func (h *ChannelHandler) GetChannels(ctx *gin.Context, au xgin.ContextUser) (xgin.R, error) {
	channels, err := h.svc.GetChannels(ctx, au.UID)
	if err != nil {
		return xgin.R{}, err
	}

	data := make([]getChannelResp, 0, len(channels))
	for i := range channels {
		data = append(data, getChannelResp{
			ID:            xgin.U64(channels[i].ID),
			DestinationID: relationID(channels[i].UserID, channels[i].TargetID, channels[i].ChannelType),

			UserID:        xgin.U64(channels[i].UserID),
			TargetID:      xgin.U64(channels[i].TargetID),
			TargetInfoVer: channels[i].TargetInfoVer,

			ChannelType:   string(channels[i].ChannelType),
			ChannelName:   channels[i].ChannelName,
			ChannelRemark: channels[i].ChannelRemark,
			ChannelAvatar: channels[i].ChannelAvatar,

			IsMuted:  channels[i].IsMuted,
			IsPinned: channels[i].IsPinned,

			LastMessageID:         xgin.U64(channels[i].LastMessageID),
			LastMessageType:       channels[i].LastMessageType,
			LastMessageContent:    channels[i].LastMessageContent,
			LastMessageSenderID:   xgin.U64(channels[i].LastMessageSenderID),
			LastMessageSenderName: channels[i].LastMessageSenderName,
			LastMessageTime:       channels[i].LastMessageTime,

			UnreadCount:  channels[i].UnreadCount,
			MentionCount: channels[i].MentionCount,
		})
	}

	return xgin.R{
		Code: http.StatusOK,
		Data: data,
	}, nil
}

func relationID(userID, targetID uint64, typ domain.ChannelType) string {
	if typ == domain.ChannelTypeSingle {
		if userID < targetID {
			return fmt.Sprintf("%d:%d", userID, targetID)
		}
		return fmt.Sprintf("%d:%d", targetID, userID)
	}
	return strconv.FormatUint(targetID, 10)
}
