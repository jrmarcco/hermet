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

var _ xgin.RouteRegistry = (*ContactHandler)(nil)

// ContactHandler 联系人 HTTP Handler。
type ContactHandler struct {
	userSvc    service.UserService
	contactSvc service.ContactService
}

func NewContactHandler(userSvc service.UserService, contactSvc service.ContactService) *ContactHandler {
	return &ContactHandler{
		userSvc:    userSvc,
		contactSvc: contactSvc,
	}
}

func (h *ContactHandler) Register(engine *gin.Engine) {
	contactV1 := engine.Group("api/v1/contact")

	contactV1.Handle(http.MethodPost, "/apply", xgin.BU(h.Apply))
	contactV1.Handle(http.MethodPost, "/review", xgin.BU(h.ReviewApplication))

	contactV1.Handle(http.MethodGet, "/contacts", xgin.U(h.GetContacts))
	contactV1.Handle(http.MethodGet, "/applications", xgin.U(h.GetApplications))
}

type applyContactReq struct {
	ContactID xgin.U64 `json:"contactId"`
	Message   string   `json:"message"`
	Source    string   `json:"source"` // 添加来源 ( search=搜索添加 / qrcode=扫码添加 / group=群聊添加 )
}

// Apply 联系人申请。
// 注意：这里只提交联系人申请，不会立即添加联系人。
func (h *ContactHandler) Apply(ctx *gin.Context, req applyContactReq, au xgin.ContextUser) (xgin.R, error) {
	if req.ContactID == xgin.U64(au.UID) {
		return xgin.R{}, fmt.Errorf("%w: can not add self as contact", errs.ErrInvalidParam)
	}

	source := domain.ContactSource(req.Source)
	if !source.IsValid() {
		return xgin.R{}, fmt.Errorf("%w: invalid source", errs.ErrInvalidParam)
	}

	event := domain.ContactApplyEvent{
		ApplicantID: au.UID,
		TargetID:    uint64(req.ContactID),
		Message:     req.Message,
		Source:      source,
	}
	if err := h.contactSvc.Apply(ctx, event); err != nil {
		return xgin.R{}, err
	}

	return xgin.R{
		Code: http.StatusOK,
	}, nil
}

type reviewApplicationReq struct {
	ApplicationID xgin.U64 `json:"applicationId"`
	Status        string   `json:"status"`
}

// ReviewApplication 审批联系人申请。
// 审批通过会建立联系人关系 ( 双向映射 )，并发送通知。
func (h *ContactHandler) ReviewApplication(ctx *gin.Context, req reviewApplicationReq, au xgin.ContextUser) (xgin.R, error) {
	status := domain.ApplicationStatus(req.Status)
	if !status.IsValid() {
		return xgin.R{}, fmt.Errorf("%w: invalid status", errs.ErrInvalidParam)
	}

	event := domain.ContactReviewEvent{
		ApplicationID: uint64(req.ApplicationID),
		Status:        status,
	}

	if err := h.contactSvc.ReviewApplication(ctx, au.UID, event); err != nil {
		return xgin.R{}, err
	}

	return xgin.R{
		Code: http.StatusOK,
	}, nil
}

type getApplicationsResp struct {
	ID                 xgin.U64 `json:"id"`
	ApplicantID        xgin.U64 `json:"applicantId"`
	ApplicantName      string   `json:"applicantName"`
	ApplicantAvatar    string   `json:"applicantAvatar"`
	ApplicationMessage string   `json:"applicationMessage"`
	Source             string   `json:"source"`
	CreatedAt          int64    `json:"createdAt"`
}

// GetApplications 获取联系人申请。
// 这里是获取向当前登录用户发送的联系人的人申请。
func (h *ContactHandler) GetApplications(ctx *gin.Context, au xgin.ContextUser) (xgin.R, error) {
	applications, err := h.contactSvc.GetApplications(ctx, au.UID)
	if err != nil {
		return xgin.R{}, err
	}

	// 转换为响应对象
	resps := make([]getApplicationsResp, 0, len(applications))
	for i := range applications {
		resps = append(resps, getApplicationsResp{
			ID:                 xgin.U64(applications[i].ID),
			ApplicantID:        xgin.U64(applications[i].ApplicantID),
			ApplicantName:      applications[i].ApplicantName,
			ApplicantAvatar:    applications[i].ApplicantAvatar,
			ApplicationMessage: applications[i].ApplicationMessage,
			Source:             string(applications[i].Source),
			CreatedAt:          applications[i].CreatedAt,
		})
	}

	return xgin.R{
		Code: http.StatusOK,
		Data: resps,
	}, nil
}

type getContactsResp struct {
	ID xgin.U64 `json:"id"`

	UserID    xgin.U64 `json:"userId"`
	ContactID xgin.U64 `json:"contactId"`

	ContactNickname string `json:"contactNickname"`
	ContactAvatar   string `json:"contactAvatar"`
	ContactGender   string `json:"contactGender"`
	ContactMobile   string `json:"contactMobile"`
	ContactInfoVer  int    `json:"contactInfoVer"`

	RemarkName string `json:"remarkName"`
	Source     string `json:"source"`

	Tags []string `json:"tags"`

	AddedAt int64 `json:"addedAt"`
}

func (h *ContactHandler) GetContacts(ctx *gin.Context, au xgin.ContextUser) (xgin.R, error) {
	contacts, err := h.contactSvc.GetContacts(ctx, au.UID)
	if err != nil {
		return xgin.R{}, err
	}

	resps := make([]getContactsResp, 0, len(contacts))
	for i := range contacts {
		resps = append(resps, getContactsResp{
			ID: xgin.U64(contacts[i].ID),

			UserID:    xgin.U64(contacts[i].UserID),
			ContactID: xgin.U64(contacts[i].ContactID),

			ContactNickname: contacts[i].ContactNickname,
			ContactAvatar:   contacts[i].ContactAvatar,
			ContactGender:   contacts[i].ContactGender,
			ContactMobile:   contacts[i].ContactMobile,
			ContactInfoVer:  contacts[i].ContactInfoVer,

			RemarkName: contacts[i].RemarkName,
			Source:     string(contacts[i].Source),

			AddedAt: contacts[i].AddedAt,
		})
	}
	return xgin.R{
		Code: http.StatusOK,
		Data: resps,
	}, nil
}
