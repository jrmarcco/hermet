// Package grpc 提供后端对外的 gRPC 服务 ( 网关经服务发现调用 )。
//
// SyncService 契约见 synp-api proto/message/v1/sync.proto:
// 网关收到客户端 SYNC 帧后，以连接鉴权身份盖章调用本服务，
// 应答由网关放入 SYNC_ACK 帧原样返回客户端。
package grpc

import (
	"context"
	"time"

	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo"
	messagev1 "github.com/jrmarcco/synp-api/api/go/message/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// 同步分页默认值与上限。
const (
	defaultSyncLimit = 100
	maxSyncLimit     = 500
)

var _ messagev1.SyncServiceServer = (*SyncServer)(nil)

// SyncServer 是 SyncService 的 gRPC 服务端实现。
type SyncServer struct {
	messagev1.UnimplementedSyncServiceServer

	messageRepo repo.MessageRepo

	// userGroupRepo 用于 ConversationSync 的群成员身份校验 ( 契约要求 )。
	userGroupRepo repo.UserGroupRepo
}

func NewSyncServer(messageRepo repo.MessageRepo, userGroupRepo repo.UserGroupRepo) *SyncServer {
	return &SyncServer{
		messageRepo:   messageRepo,
		userGroupRepo: userGroupRepo,
	}
}

// Sync 返回该用户收件箱中 ( cursor, latest_seq ] 的增量条目 ( 按 seq 升序 )。
func (s *SyncServer) Sync(ctx context.Context, req *messagev1.SyncRequest) (*messagev1.SyncResponse, error) {
	businessID := req.GetBusinessId()
	userID := req.GetUserId()
	if userID == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	limit := normalizeLimit(req.GetLimit())

	entries, latestSeq, hasMore, err := s.messageRepo.ListInboxEntries(ctx, businessID, userID, int64(req.GetCursor()), limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list inbox entries: %v", err)
	}

	resp := &messagev1.SyncResponse{
		Entries: make([]*messagev1.InboxEntry, 0, len(entries)),

		LatestSeq: uint64(latestSeq),
		HasMore:   hasMore,

		// 保留窗口未实现 ( inbox_entry 未按 created_at 清理 )，恒为 false；
		// 引入清理任务后按窗口内最早 seq 计算。
		CursorExpired: false,
	}

	for _, e := range entries {
		resp.Entries = append(resp.Entries, toProtoInboxEntry(e))
	}

	return resp, nil
}

// ConversationSync 返回会话 timeline 的增量消息 ( 读扩散，大群场景 )。
//
// 契约要求服务端校验请求用户是否为该会话 ( 群 ) 的成员。
func (s *SyncServer) ConversationSync(ctx context.Context, req *messagev1.ConversationSyncRequest) (*messagev1.ConversationSyncResponse, error) {
	businessID := req.GetBusinessId()
	userID := req.GetUserId()
	conversationID := req.GetConversationId()

	if userID == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	groupID, ok := domain.ParseGroupConversationID(conversationID)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "conversation_id is not a group conversation")
	}

	// 成员身份校验 ( 身份字段由网关盖章，客户端自报身份不可信 )。
	active, err := s.userGroupRepo.IsActiveMember(ctx, userID, groupID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check group membership: %v", err)
	}
	if !active {
		return nil, status.Error(codes.PermissionDenied, "user is not an active member of the conversation")
	}

	limit := normalizeLimit(req.GetLimit())

	messages, latestSeq, hasMore, err := s.messageRepo.ListConversationTimeline(ctx, businessID, conversationID, int64(req.GetCursor()), limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list conversation timeline: %v", err)
	}

	resp := &messagev1.ConversationSyncResponse{
		Messages: make([]*messagev1.TimelineMessage, 0, len(messages)),

		LatestSeq: uint64(latestSeq),
		HasMore:   hasMore,
	}

	for _, m := range messages {
		resp.Messages = append(resp.Messages, &messagev1.TimelineMessage{
			ConversationSeq: uint64(m.ConversationSeq),

			Message: &messagev1.ChatMessage{
				ConversationId: m.ConversationID,
				MessageId:      m.MessageID,
				SenderId:       m.SenderID,

				Body: m.Body,

				SentAt: unixMilliToTimestamp(m.SentAt),
			},
		})
	}

	return resp, nil
}

func toProtoInboxEntry(e domain.InboxEntry) *messagev1.InboxEntry {
	entry := &messagev1.InboxEntry{
		Seq: uint64(e.Seq),
	}

	if e.Kind == domain.InboxEntryKindPointer {
		entry.Kind = &messagev1.InboxEntry_ConversationUpdate{
			ConversationUpdate: &messagev1.ConversationUpdate{
				ConversationId:        e.ConversationID,
				ConversationLatestSeq: uint64(e.ConversationLatestSeq),
			},
		}
		return entry
	}

	entry.Kind = &messagev1.InboxEntry_Message{
		Message: &messagev1.ChatMessage{
			ConversationId: e.ConversationID,
			MessageId:      e.MessageID,
			SenderId:       e.SenderID,

			Body: e.Body,

			SentAt: unixMilliToTimestamp(e.SentAt),
		},
	}
	return entry
}

func normalizeLimit(limit uint32) int {
	if limit == 0 {
		return defaultSyncLimit
	}
	if limit > maxSyncLimit {
		return maxSyncLimit
	}
	return int(limit)
}

// unixMilliToTimestamp 落库时间 ( Unix 毫秒 ) 转 protobuf Timestamp。
func unixMilliToTimestamp(ms int64) *timestamppb.Timestamp {
	if ms < 0 {
		// 防御: 非法时间落 0 值，避免构造负值时间。
		return timestamppb.New(time.Unix(0, 0))
	}
	return timestamppb.New(time.UnixMilli(ms))
}
