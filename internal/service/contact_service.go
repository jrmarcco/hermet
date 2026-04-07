package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/errs"
	"github.com/jrmarcco/hermet/internal/repo"
	"go.uber.org/zap"
)

type ContactService interface {
	Apply(ctx context.Context, event domain.ContactApplyEvent) error

	ReviewApplication(ctx context.Context, userID uint64, event domain.ContactReviewEvent) error

	GetContacts(ctx context.Context, userID uint64) ([]domain.ContactView, error)
	GetApplications(ctx context.Context, userID uint64) ([]domain.ContactApplication, error)
}

var _ ContactService = (*DefaultContactService)(nil)

type DefaultContactService struct {
	bizUserRepo repo.UserRepo

	contactRepo            repo.ContactRepo
	contactApplicationRepo repo.ContactApplicationRepo

	logger *zap.Logger
}

func NewDefaultContactService(
	bizUserRepo repo.UserRepo,
	contactRepo repo.ContactRepo,
	contactApplicationRepo repo.ContactApplicationRepo,
	logger *zap.Logger,
) *DefaultContactService {
	return &DefaultContactService{
		bizUserRepo: bizUserRepo,

		contactRepo:            contactRepo,
		contactApplicationRepo: contactApplicationRepo,

		logger: logger,
	}
}

func (s *DefaultContactService) Apply(ctx context.Context, event domain.ContactApplyEvent) error {
	// 判断是否已经存在联系人。
	contact, err := s.contactRepo.FindByUserIDAndContactID(ctx, event.ApplicantID, event.TargetID)
	if err != nil && !errors.Is(err, errs.ErrRecordNotFound) {
		return err
	}

	// 已经存在联系人。
	if contact.ID != 0 && contact.DeletedAt == 0 {
		return fmt.Errorf("%w: contact already exists", errs.ErrInvalidParam)
	}

	// 查询申请人信息。
	bizUser, err := s.bizUserRepo.FindByID(ctx, event.ApplicantID)
	if err != nil {
		if errors.Is(err, errs.ErrRecordNotFound) {
			return errors.New("target user is not exists")
		}
		return err
	}

	// 提交申请
	ca := domain.ContactApplication{
		ApplicantID:        event.ApplicantID,
		TargetID:           event.TargetID,
		ApplicantName:      bizUser.Nickname,
		ApplicantAvatar:    bizUser.Avatar,
		ApplicationStatus:  domain.ApplicationStatusPending,
		ApplicationMessage: event.Message,
		Source:             event.Source,
	}

	if err := s.contactApplicationRepo.Create(ctx, ca); err != nil {
		return err
	}

	// TODO: 向 target 用户发送申请通知。
	// 向 target 用户发送申请通知。

	return nil
}

func (s *DefaultContactService) ReviewApplication(ctx context.Context, userID uint64, event domain.ContactReviewEvent) error {
	ca, err := s.contactApplicationRepo.FindByID(ctx, event.ApplicationID)
	if err != nil {
		return err
	}

	// 检查用户是否为被申请人。
	if ca.TargetID != userID {
		return fmt.Errorf("%w: you are not the target user", errs.ErrInvalidParam)
	}

	if ca.ApplicationStatus != domain.ApplicationStatusPending {
		return fmt.Errorf("%w: application is already reviewed", errs.ErrInvalidParam)
	}

	err = s.contactApplicationRepo.UpdateStatus(ctx, event.ApplicationID, event.Status)
	if err != nil {
		s.logger.Error("[hermet] failed to review contact application", zap.Error(err))
		return errors.New("failed to review contact application")
	}

	// TODO: 向 applicant 用户发送申请通知。
	// 向 applicant 用户发送申请通知。

	return nil
}

func (s *DefaultContactService) GetContacts(ctx context.Context, userID uint64) ([]domain.ContactView, error) {
	ucvs, err := s.contactRepo.ListViewByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return ucvs, nil
}

func (s *DefaultContactService) GetApplications(ctx context.Context, userID uint64) ([]domain.ContactApplication, error) {
	return s.contactApplicationRepo.ListPendingByTargetID(ctx, userID)
}
