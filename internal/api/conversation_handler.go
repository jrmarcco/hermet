package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/service"
	"go.uber.org/zap"
)

var _ xgin.RouteRegistry = (*ConversationHandler)(nil)

type ConversationHandler struct {
	conversationSvc service.ConversationService
	logger          *zap.Logger
}

func NewConversationHandler(conversationSvc service.ConversationService, logger *zap.Logger) *ConversationHandler {
	return &ConversationHandler{
		conversationSvc: conversationSvc,
		logger:          logger,
	}
}

func (h *ConversationHandler) Register(engine *gin.Engine) {
	conversationV1 := engine.Group("api/v1/conversation")

	conversationV1.Handle(http.MethodGet, "/conversations", xgin.U(h.GetConversations))
}

type getConversationResp struct {
	ID uint64 `json:"id"`

	// 关系 ID ( 用于唯一标识一个会话 )。
	// 取值规则:
	//
	//  - ConversationType = single
	// 	  RelationID = min(UserID, TargetID):max(UserID, TargetID)。
	//  - ConversationType = group
	// 	  RelationID = TargetID。
	RelationID string `json:"relationId"`

	UserID        uint64 `json:"userId"`
	TargetID      uint64 `json:"targetId"`
	TargetInfoVer int    `json:"targetInfoVer"`

	ConversationType   string `json:"conversationType"`
	ConversationName   string `json:"conversationName"`
	ConversationRemark string `json:"conversationRemark"`
	ConversationAvatar string `json:"conversationAvatar"`

	IsMuted   bool `json:"isMuted"`
	IsPinned  bool `json:"isPinned"`
	IsHidden  bool `json:"isHidden"`
	IsStarred bool `json:"isStarred"`

	LastMessageID         uint64 `json:"lastMessageId"`
	LastMessageType       string `json:"lastMessageType"`
	LastMessageContent    string `json:"lastMessageContent"`
	LastMessageSenderID   uint64 `json:"lastMessageSenderId"`
	LastMessageSenderName string `json:"lastMessageSenderName"`
	LastMessageTime       int64  `json:"lastMessageTime"`

	UnreadCount  int `json:"unreadCount"`
	MentionCount int `json:"mentionCount"`
}

func (h *ConversationHandler) GetConversations(ctx *gin.Context, au xgin.ContextUser) (xgin.R, error) {
	conversations, err := h.conversationSvc.GetConversations(ctx, au.UID)
	if err != nil {
		return xgin.R{}, err
	}

	data := make([]getConversationResp, 0, len(conversations))
	for i := range conversations {
		data = append(data, getConversationResp{
			ID:         conversations[i].ID,
			RelationID: relationID(conversations[i].UserID, conversations[i].TargetID, conversations[i].ConversationType),

			UserID:        conversations[i].UserID,
			TargetID:      conversations[i].TargetID,
			TargetInfoVer: conversations[i].TargetInfoVer,

			ConversationType:   string(conversations[i].ConversationType),
			ConversationName:   conversations[i].ConversationName,
			ConversationRemark: conversations[i].ConversationRemark,
			ConversationAvatar: conversations[i].ConversationAvatar,

			IsMuted:   conversations[i].IsMuted,
			IsPinned:  conversations[i].IsPinned,
			IsHidden:  conversations[i].IsHidden,
			IsStarred: conversations[i].IsStarred,

			LastMessageID:         conversations[i].LastMessageID,
			LastMessageType:       conversations[i].LastMessageType,
			LastMessageContent:    conversations[i].LastMessageContent,
			LastMessageSenderID:   conversations[i].LastMessageSenderID,
			LastMessageSenderName: conversations[i].LastMessageSenderName,
			LastMessageTime:       conversations[i].LastMessageTime,

			UnreadCount:  conversations[i].UnreadCount,
			MentionCount: conversations[i].MentionCount,
		})
	}

	return xgin.R{
		Code: http.StatusOK,
		Data: data,
	}, nil
}

func relationID(userID, targetID uint64, conversationType domain.ConversationType) string {
	if conversationType == domain.ConversationTypeSingle {
		if userID < targetID {
			return fmt.Sprintf("%d:%d", userID, targetID)
		}
		return fmt.Sprintf("%d:%d", targetID, userID)
	}
	return strconv.FormatUint(targetID, 10)
}
