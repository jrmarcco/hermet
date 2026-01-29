# 消息处理器重构总结

## 问题

之前 `cdc` 包和 `mq` 包存在大量重复代码：
1. 消费逻辑重复
2. 生命周期管理重复
3. 接口命名不一致
4. 职责划分不清

## 解决方案：抽象基础设施层

采用分层架构，将通用的消息处理逻辑抽象到 `internal/pkg/xmq/processor` 包中。

### 架构层次

```
internal/
├── pkg/xmq/processor/         # 基础设施层（通用消息处理）
│   ├── types.go               # 通用接口定义
│   ├── consumer_processor.go  # 通用消费处理器
│   ├── producer_processor.go  # 通用生产处理器
│   └── sharded_consumer_processor.go  # 分片消费处理器
├── cdc/                       # CDC 业务层
│   ├── base_processor.go      # CDC 基础处理器（组合 ShardedConsumerProcessor）
│   ├── contact_processor.go   # 联系人 CDC 处理
│   └── conversation_processor.go  # 会话 CDC 处理
└── mq/                        # 业务消息层
    ├── channel_member_processor.go  # 频道成员处理器
    └── module.go              # 模块初始化
```

## 核心组件

### 1. 基础设施层 (`internal/pkg/xmq/processor`)

#### `ConsumerProcessor`
通用消费处理器，负责：
- 管理多个消费者的生命周期
- 从消息通道读取消息
- 将消息分发给处理函数
- 统一的错误处理和日志记录

```go
type ConsumerProcessor struct {
    cfg             ConsumerProcessorConfig
    consumerFactory xconsumer.ConsumerFactory
    ctx             context.Context
    cancelFunc      context.CancelFunc
    logger          *zap.Logger
}
```

#### `ProducerProcessor`
通用生产处理器，负责：
- 封装消息生产逻辑
- 统一的错误处理和日志记录

```go
type ProducerProcessor struct {
    producer xproduce.Producer
    topic    string
    logger   *zap.Logger
}
```

#### `ShardedConsumerProcessor`
分片消费处理器，适用于需要消费多个分片主题的场景（如 CDC）：
- 根据数据库和表分片配置自动生成主题列表
- 为每个分片主题创建消费者
- 组合使用 `ConsumerProcessor`

```go
type ShardedConsumerProcessor struct {
    *ConsumerProcessor
}
```

### 2. CDC 业务层 (`internal/cdc`)

#### 重构前
```go
type BaseProcessor struct {
    cfg ProcessorConfig
    consumerFactory xconsumer.ConsumerFactory
    ctx        context.Context
    cancelFunc context.CancelFunc
    logger     *zap.Logger
}

func (p *BaseProcessor) Start(processFunc ProcessFunc) error {
    // 大量重复的消费逻辑...
}
```

#### 重构后
```go
type BaseProcessor struct {
    *processor.ShardedConsumerProcessor
}

func NewBaseProcessor(
    cfg ProcessorConfig,
    handler processor.MessageHandler,
    consumerFactory xconsumer.ConsumerFactory,
    logger *zap.Logger,
) *BaseProcessor {
    // 直接使用基础设施层的 ShardedConsumerProcessor
    shardedProcessor := processor.NewShardedConsumerProcessor(...)
    return &BaseProcessor{ShardedConsumerProcessor: shardedProcessor}
}
```

**关键改进：**
- 处理器在构造时接收 `handler`，无需在 `Start` 时传入
- 通过组合复用基础设施层的功能
- 代码量大幅减少，职责更清晰

### 3. 业务消息层 (`internal/mq`)

#### 重构前
```go
type ChannelMemberProcessor struct {
    producer        xproduce.Producer
    consumerFactory xconsumer.ConsumerFactory
    topic      string
    groupID    string
    partitions int32
    ctx        context.Context
    cancelFunc context.CancelFunc
    logger     *zap.Logger
}

func (p *ChannelMemberProcessor) Consume(consumeFunc ConsumeFunc) error {
    // 大量重复的消费逻辑...
}
```

#### 重构后
```go
type ChannelMemberProcessor struct {
    consumerProcessor *processor.ConsumerProcessor
    producerProcessor *processor.ProducerProcessor
    // 配置字段
}

func (p *ChannelMemberProcessor) StartConsumer(handler processor.MessageHandler) error {
    // 创建并启动 ConsumerProcessor
    p.consumerProcessor = processor.NewConsumerProcessor(...)
    return p.consumerProcessor.Start()
}

func (p *ChannelMemberProcessor) Produce(ctx context.Context, key string, val []byte) error {
    return p.producerProcessor.Produce(ctx, key, val)
}
```

**关键改进：**
- 同时支持生产和消费
- 使用 `StartConsumer` 方法延迟设置 handler，解决循环依赖问题
- 通过组合复用基础设施层的功能

## 依赖注入和模块初始化

### CDC 模块 (`internal/cdc/module.go`)

```go
func newContactProcessor(...) (*ContactProcessor, error) {
    processor := NewContactProcessor(
        cfg.DBShardCount,
        cfg.TBShardCount,
        cfg.GroupID,
        cdcService.ProcessContact,  // 直接传入 handler
        consumerFactory,
        zapLogger,
    )
    lifecycle.Append(fx.Hook{
        OnStart: func(_ context.Context) error {
            return processor.Start()  // 无需传入 handler
        },
        ...
    })
}
```

### MQ 模块 (`internal/mq/module.go`)

```go
// 提供 ChannelMemberProcessor
func newChannelMemberProcessor(...) (*ChannelMemberProcessor, error) {
    processor, err := NewChannelMemberProcessor(...)
    return processor, nil
}

// 提供 ChannelMessageProducer 接口
fx.Annotate(
    func(p *ChannelMemberProcessor) service.ChannelMessageProducer {
        return p
    },
    fx.From(new(*ChannelMemberProcessor)),
)

// 启动消费者（在所有依赖准备好后）
func startChannelMemberConsumer(
    processor *ChannelMemberProcessor,
    channelService service.ChannelService,
    lifecycle fx.Lifecycle,
    zapLogger *zap.Logger,
) {
    lifecycle.Append(fx.Hook{
        OnStart: func(_ context.Context) error {
            // 延迟设置 handler，打破循环依赖
            return processor.StartConsumer(channelService.ProcessAddMembersMsg)
        },
        ...
    })
}
```

## 打破循环依赖

### 问题
- `service.ChannelService` 需要 `mq.Processor` 来发送消息
- `mq.ChannelMemberProcessor` 需要 `service.ChannelService` 来处理消息
- 这造成了循环依赖：`service` → `mq` → `service`

### 解决方案
1. **接口抽象**：在 `service` 包中定义 `ChannelMessageProducer` 接口
   ```go
   type ChannelMessageProducer interface {
       Produce(ctx context.Context, key string, val []byte) error
   }
   ```

2. **依赖倒置**：`ChannelService` 依赖接口而非具体实现
   ```go
   type DefaultChannelService struct {
       messageProducer ChannelMessageProducer  // 接口依赖
       ...
   }
   ```

3. **延迟初始化**：在 `mq.ChannelMemberProcessor` 中，消费者的启动延迟到所有依赖都准备好之后
   ```go
   func (p *ChannelMemberProcessor) StartConsumer(handler processor.MessageHandler) error {
       // 在这里创建和启动消费处理器
   }
   ```

4. **fx.Invoke**：使用 `fx.Invoke` 在依赖图构建完成后启动消费者
   ```go
   fx.Invoke(startChannelMemberConsumer)
   ```

## 重构收益

### 1. 消除重复代码
- 消费循环逻辑：从重复实现到共享基础设施
- 生命周期管理：统一的 `ctx`、`cancelFunc`、`Start()`、`Stop()`
- 错误处理和日志记录：集中在基础设施层

### 2. 清晰的职责分层
- **基础设施层**：负责消息的收发，与业务逻辑无关
- **业务层**：专注于业务逻辑，复用基础设施能力

### 3. 易于扩展
- 新增消息类型：只需创建新的 Processor，复用基础设施
- 支持新的消息中间件：只需在基础设施层扩展

### 4. 更好的测试性
- 基础设施层可以独立测试
- 业务层通过 mock handler 进行测试

### 5. 统一的接口
- `processor.MessageHandler`：统一的消息处理函数类型
- `processor.Processor`：统一的处理器接口

## 使用示例

### 添加新的 CDC 处理器

```go
type NewProcessor struct {
    *BaseProcessor
}

func NewNewProcessor(
    dbShardCount, tbShardCount int,
    groupID string,
    handler processor.MessageHandler,
    consumerFactory xconsumer.ConsumerFactory,
    logger *zap.Logger,
) *NewProcessor {
    return &NewProcessor{
        BaseProcessor: NewBaseProcessor(
            ProcessorConfig{
                DBShardCount: dbShardCount,
                TBShardCount: tbShardCount,
                GroupID:      groupID,
                TopicPattern: "your_topic_pattern_%d_%d",
            },
            handler,
            consumerFactory,
            logger,
        ),
    }
}
```

### 添加新的业务消息处理器

```go
type NewBusinessProcessor struct {
    consumerProcessor *processor.ConsumerProcessor
    producerProcessor *processor.ProducerProcessor
}

func NewNewBusinessProcessor(
    producer xproduce.Producer,
    consumerFactory xconsumer.ConsumerFactory,
    topic, groupID string,
    logger *zap.Logger,
) (*NewBusinessProcessor, error) {
    producerProcessor, err := processor.NewProducerProcessor(producer, topic, logger)
    if err != nil {
        return nil, err
    }

    return &NewBusinessProcessor{
        producerProcessor: producerProcessor,
        // 保存配置，在 StartConsumer 时创建 consumerProcessor
    }, nil
}
```

## 注意事项

1. **handler 的传入时机**
   - CDC 处理器：在构造函数中传入（无循环依赖）
   - 业务处理器：在 `StartConsumer` 中传入（打破循环依赖）

2. **Kafka 分区处理**
   - 当前实现为每个分区创建一个消费者
   - 实际上 Kafka 会自动分配分区给消费者组中的消费者
   - 可以考虑优化为创建单个消费者，让 Kafka 自动分配分区

3. **错误处理**
   - 消息处理失败会记录日志，但不会停止消费
   - 根据业务需求，可能需要添加重试机制或死信队列

## 文件清单

### 新增文件
- `internal/pkg/xmq/processor/types.go`
- `internal/pkg/xmq/processor/consumer_processor.go`
- `internal/pkg/xmq/processor/producer_processor.go`
- `internal/pkg/xmq/processor/sharded_consumer_processor.go`
- `internal/mq/module.go`

### 修改文件
- `internal/cdc/types.go`
- `internal/cdc/base_processor.go`
- `internal/cdc/contact_processor.go`
- `internal/cdc/conversation_processor.go`
- `internal/cdc/module.go`
- `internal/mq/types.go`
- `internal/mq/channel_member_processor.go`
- `internal/service/channel_service.go`
- `internal/service/module.go`
- `cmd/main.go`

## 后续优化建议

1. **Kafka 分区优化**：考虑改为单个消费者，让 Kafka 自动分配分区
2. **错误处理增强**：添加重试机制和死信队列
3. **监控指标**：在基础设施层添加 metrics 收集
4. **配置统一**：考虑将 topic、groupID 等配置统一管理
5. **测试覆盖**：为基础设施层添加单元测试
