package search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/jrmarcco/hermet/internal/domain"
	"github.com/jrmarcco/hermet/internal/repo"
)

// 搜索分页默认值与上限。
const (
	defaultSearchSize = 20
	maxSearchSize     = 100
)

// SearchResultItem 搜索结果条目。
type SearchResultItem struct {
	MessageID      string `json:"messageId"`
	ConversationID string `json:"conversationId"`
	SenderID       uint64 `json:"senderId"`
	ContentType    string `json:"contentType"`
	SentAt         int64  `json:"sentAt"`
}

// SearchResult 搜索结果 ( 按时间倒序 )。
type SearchResult struct {
	Items []SearchResultItem `json:"items"`
	Total int64              `json:"total"`
}

// Service 会话内消息搜索 ( ES 查询侧 )。
type Service struct {
	client *elasticsearch.Client

	userGroupRepo repo.UserGroupRepo

	index string
}

func NewService(
	client *elasticsearch.Client,
	userGroupRepo repo.UserGroupRepo,
	index string,
) *Service {
	return &Service{
		client: client,

		userGroupRepo: userGroupRepo,

		index: index,
	}
}

// SearchInConversation 会话内搜索 ( 按关键词匹配消息正文，时间倒序 )。
//
// 权限: 单聊会话需请求者是会话双方之一；
// 群聊会话需请求者是群内有效成员 ( 与 ConversationSync 一致 )。
func (s *Service) SearchInConversation(
	ctx context.Context,
	businessID, userID uint64,
	conversationID, keyword string,
	from, size int,
) (*SearchResult, error) {
	if keyword == "" {
		return nil, fmt.Errorf("keyword is required")
	}
	if conversationID == "" {
		return nil, fmt.Errorf("conversation_id is required")
	}

	if err := s.validateConversationAccess(ctx, userID, conversationID); err != nil {
		return nil, err
	}

	if size <= 0 {
		size = defaultSearchSize
	}
	if size > maxSearchSize {
		size = maxSearchSize
	}
	if from < 0 {
		from = 0
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []map[string]any{
					{"term": map[string]any{"businessId": businessID}},
					{"term": map[string]any{"conversationId": conversationID}},
				},
				"must": []map[string]any{
					{"match": map[string]any{"body": keyword}},
				},
			},
		},
		"sort": []map[string]any{
			{"sentAt": map[string]any{"order": "desc"}},
		},
		"from": from,
		"size": size,
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	resp, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(s.index),
		s.client.Search.WithBody(jsonReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search elasticsearch: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.IsError() {
		return nil, fmt.Errorf("search failed: %s", resp.String())
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source domain.MessageIndexEvent `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	items := make([]SearchResultItem, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		items = append(items, SearchResultItem{
			MessageID:      hit.Source.MessageID,
			ConversationID: hit.Source.ConversationID,
			SenderID:       hit.Source.SenderID,
			ContentType:    hit.Source.ContentType,
			SentAt:         hit.Source.SentAt,
		})
	}

	return &SearchResult{
		Items: items,
		Total: result.Hits.Total.Value,
	}, nil
}

// validateConversationAccess 校验请求者对会话的访问权限。
func (s *Service) validateConversationAccess(ctx context.Context, userID uint64, conversationID string) error {
	if groupID, ok := domain.ParseGroupConversationID(conversationID); ok {
		active, err := s.userGroupRepo.IsActiveMember(ctx, userID, groupID)
		if err != nil {
			return fmt.Errorf("failed to check group membership: %w", err)
		}
		if !active {
			return fmt.Errorf("user [ %d ] is not a member of conversation [ %s ]", userID, conversationID)
		}
		return nil
	}

	// 单聊会话 s:min:max: 请求者必须是会话双方之一。
	if a, b, ok := domain.ParseSingleConversationID(conversationID); ok {
		if userID != a && userID != b {
			return fmt.Errorf("user [ %d ] is not a participant of conversation [ %s ]", userID, conversationID)
		}
		return nil
	}

	return fmt.Errorf("invalid conversation id [ %s ]", conversationID)
}
