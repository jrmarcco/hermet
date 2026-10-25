// Package search 提供消息搜索的 ES 查询侧:
//
//   - Indexer: 消费消息索引事件 ( hermet.message.index )，esutil.BulkIndexer 批量写入;
//   - Service: 会话内消息搜索 ( HTTP API 查询入口 );
//   - Reindexer: 从 PG message 表全量重建索引。
//
// 架构定位 ( 见 scripts/sql/README.md 与 scripts/docker/README.md ):
// PG 是唯一事实源，ES 只是异步查询加速层——可重建、最终一致 (~1s NRT)，
// 绝不进入 SYNC 同步主链路。
package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v9"
)

// jsonReader 返回 *bytes.Reader ( esutil.BulkIndexerItem.Body 需要 io.ReadSeeker，
// bulk 失败重试时会重新 Seek 读体 )。
func jsonReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}

// ES 字段类型常量。
const (
	esTypeKeyword = "keyword" // 精确匹配
	esTypeLong    = "long"    // 数值
	esTypeText    = "text"    // 全文检索
	esTypeDate    = "date"    // 时间
)

// esFieldMapping 单个字段的 mapping 定义。
type esFieldMapping struct {
	Type     string `json:"type"`
	Analyzer string `json:"analyzer,omitempty"`
	Format   string `json:"format,omitempty"`
}

// esMappings 文档 mapping。
type esMappings struct {
	// strict: 字段以本文档为准，禁止 ES 自动推断 ( 避免 body 误判等 )。
	Dynamic    string                    `json:"dynamic"`
	Properties map[string]esFieldMapping `json:"properties"`
}

// esIndexBody 建索引请求体 ( settings + mappings )。
type esIndexBody struct {
	Settings struct {
		NumberOfShards   int `json:"number_of_shards"`
		NumberOfReplicas int `json:"number_of_replicas"`
	} `json:"settings"`
	Mappings esMappings `json:"mappings"`
}

// messageIndexMapping 是 hermet-message 索引的 mapping。
//
// 注意: body 是业务自定义 bytes ( 后端不解释 )，统一作为 text 索引；
// 默认 standard analyzer 对中文按字切分 ( 能搜但不精准 )，
// 中文分词需在 ES 侧安装 IK 插件后切换 analyzer ( 生产环境按需 )。
//
// 文档 _id = message_id: 索引链路 at-least-once，重复写入天然幂等。
func messageIndexMapping() esIndexBody {
	return esIndexBody{
		Settings: struct {
			NumberOfShards   int `json:"number_of_shards"`
			NumberOfReplicas int `json:"number_of_replicas"`
		}{
			// 开发环境单节点: 单分片无副本；多节点部署时调大。
			NumberOfShards:   1,
			NumberOfReplicas: 0,
		},
		Mappings: esMappings{
			Dynamic: "strict",
			Properties: map[string]esFieldMapping{
				"messageId":      {Type: esTypeKeyword},
				"businessId":     {Type: esTypeLong},
				"conversationId": {Type: esTypeKeyword},
				"senderId":       {Type: esTypeLong},
				"body":           {Type: esTypeText, Analyzer: "standard"},
				"contentType":    {Type: esTypeKeyword},
				"sentAt":         {Type: esTypeDate, Format: "epoch_millis"},
			},
		},
	}
}

// EnsureMessageIndex 确保消息索引存在 ( 不存在则按 mapping 创建，幂等 )。
// 已存在但 mapping 不一致时不修改 ( mapping 变更需 reindex 迁移 )。
func EnsureMessageIndex(ctx context.Context, client *elasticsearch.Client, index string) error {
	body, err := json.Marshal(messageIndexMapping())
	if err != nil {
		return fmt.Errorf("failed to marshal index mapping: %w", err)
	}

	resp, err := client.Indices.Create(
		index,
		client.Indices.Create.WithContext(ctx),
		client.Indices.Create.WithBody(jsonReader(body)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index [ %s ]: %w", index, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if !resp.IsError() {
		return nil
	}

	// 响应体只读一次: 供类型判定与错误信息共用。
	rawBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("failed to create index [ %s ]: http %d ( unreadable body )", index, resp.StatusCode)
	}

	// resource_already_exists_exception: 索引已存在，幂等成功。
	var errResp struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(rawBody, &errResp); decodeErr == nil && errResp.Error.Type == "resource_already_exists_exception" {
		return nil
	}

	return fmt.Errorf("failed to create index [ %s ]: http %d: %s", index, resp.StatusCode, string(rawBody))
}
