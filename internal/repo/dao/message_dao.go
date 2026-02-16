package dao

import (
	"context"

	"github.com/jrmarcco/hermet/internal/pkg/xmongo"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Message struct {
	ID string `bson:"id"`

	SID uint64 `bson:"sid"` // Sender ID ( 发送者 ID )
	MID uint64 `bson:"mid"` // Message ID ( 这里指的是网关消息的 ID )

	DestinationID   string `bson:"destinationId"`   // Destination ID ( 目的地频道 ID )
	DestinationType string `bson:"destinationType"` // Destination Type ( 目的地类型 )

	Content     []byte `bson:"content"`     // Message Content
	ContentType string `bson:"contentType"` // Message Content Type

	SendAt int64 `bson:"sendAt"` // Message Send Time
}

type MessageDao interface {
	Save(ctx context.Context, message *Message) error
}

var _ MessageDao = (*MongoMessageDao)(nil)

type MongoMessageDao struct {
	coll *mongo.Collection
}

func NewMongoMessageDao(collManager *xmongo.CollManager) *MongoMessageDao {
	return &MongoMessageDao{
		coll: collManager.Collection("message"),
	}
}

// Save 保存消息。
// 注意：message 必须是指针类型，因为 MongoDB 的插入操作会修改 message 的值 ( 如 ID 回传) 。
// 同时 Content 字段为 []byte，使用指针类型可以有效避免内存拷贝。
func (d *MongoMessageDao) Save(ctx context.Context, message *Message) error {
	_, err := d.coll.InsertOne(ctx, message)
	return err
}
