package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/pkg/xgin"
	"github.com/jrmarcco/hermet/internal/search"
)

var _ xgin.RouteRegistry = (*MessageHandler)(nil)

// MessageHandler 消息搜索 HTTP Handler。
type MessageHandler struct {
	svc *search.Service
}

func NewMessageHandler(svc *search.Service) *MessageHandler {
	return &MessageHandler{svc: svc}
}

func (h *MessageHandler) Register(engine *gin.Engine) {
	messageV1 := engine.Group("api/v1/message")

	messageV1.Handle(http.MethodGet, "/search", xgin.Q(h.Search))
}

type searchReq struct {
	ConversationID string `form:"conversationId"`
	Keyword        string `form:"keyword"`
	From           int    `form:"from"`
	Size           int    `form:"size"`
}

type searchResp struct {
	Items []searchResultItem `json:"items"`
	Total int64              `json:"total"`
}

// searchResultItem 搜索结果条目 ( int64 ID 以字符串序列化，见 xgin.U64 )。
type searchResultItem struct {
	MessageID      string   `json:"messageId"`
	ConversationID string   `json:"conversationId"`
	SenderID       xgin.U64 `json:"senderId"`
	ContentType    string   `json:"contentType"`
	SentAt         int64    `json:"sentAt"`
}

// Search 会话内消息搜索 ( 权限: 请求者必须是会话参与者 )。
func (h *MessageHandler) Search(ctx *gin.Context, req searchReq) (xgin.R, error) {
	val, ok := ctx.Get(xgin.ContextKeyAuthUser)
	if !ok {
		return xgin.R{}, errs.ErrUnauthorized
	}

	au, ok := val.(xgin.ContextUser)
	if !ok {
		return xgin.R{}, errs.ErrUnauthorized
	}

	result, err := h.svc.SearchInConversation(
		ctx,
		au.BID, au.UID,
		req.ConversationID, req.Keyword,
		req.From, req.Size,
	)
	if err != nil {
		return xgin.R{}, err
	}

	items := make([]searchResultItem, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, searchResultItem{
			MessageID:      result.Items[i].MessageID,
			ConversationID: result.Items[i].ConversationID,
			SenderID:       xgin.U64(result.Items[i].SenderID),
			ContentType:    result.Items[i].ContentType,
			SentAt:         result.Items[i].SentAt,
		})
	}

	return xgin.R{
		Code: http.StatusOK,
		Data: searchResp{
			Items: items,
			Total: result.Total,
		},
	}, nil
}
