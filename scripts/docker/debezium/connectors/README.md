# Debezium Connector 配置说明

> JSON 本身不支持注释，且在 `config` 里塞伪注释键会让 Kafka Connect
> 在日志里逐条输出 "unused configuration" 警告——因此参数说明集中在本文件，
> JSON 保持纯净。

## 文件清单

| 文件 | 监听实例 | 服务发现名 |
|---|---|---|
| `pgsql_hermet_0_connector.json` | hermet_0 分片库 | `pgsql-hermet-0` |
| `pgsql_hermet_1_connector.json` | hermet_1 分片库 | `pgsql-hermet-1` |

两个文件除实例标识 ( `hermet_0` / `hermet_1` ) 外完全一致；
**修改任一参数时两份都要同步改**。

## 参数逐项说明

### 连接与身份

```json
"connector.class": "io.debezium.connector.postgresql.PostgresConnector",
"tasks.max": "1",
```

- `connector.class`: PG 逻辑复制连接器 ( 基于 pgoutput 插件 )。
- `tasks.max=1`: **PG connector 单分区特性，只能为 1** ( 一个 slot 一个任务 )，
  写大也不会并行。

```json
"database.hostname": "pgsql-hermet-0",
"database.port": "5432",
"database.user": "hermet_0",
"database.password": "<passwd>",     // ← 占位符! 部署前必须改为实际密码
"database.dbname": "hermet_0",
```

- `hostname` 是 `jrmarcco_net` 内部服务名 ( compose 容器名 )，不是宿主机地址。
- **密码同步三处** ( 见 `../README.md` ): 这里 / `../pgsql/.env` / `etc/config.yaml` 的
  `db.sharding` DSN。`init.sh` 启动时会检测 `<passwd>` 占位符并提醒。

```json
"topic.prefix": "pgsql_hermet_0",
```

- CDC 事件 topic 前缀。实际 topic 名 = `<topic.prefix>.<schema>.<table>`，
  例如 `pgsql_hermet_0.public.contact_0`。
- backend 的 `internal/cdc/base_processor.go` 按
  `TopicPattern: "pgsql_hermet_%d.public.contact_%d"` 消费——
  **改前缀会直接断开 CDC 消费链路**。

### 表过滤

```json
"schema.include.list": "public",
"table.include.list": "public\\.biz_user_[0-9]+$,public\\.contact_application_[0-9]+$,public\\.contact_[0-9]+$,public\\.chat_group_[0-9]+$,public\\.group_member_[0-9]+$,public\\.user_group_[0-9]+$,public\\.outbox_event_[0-9]+$"
```

- 正则列表 ( 逗号分隔，`\\` 是 JSON 转义的反斜杠 )，匹配**分表后的每张物理表**
  ( 如 `contact_0` ~ `contact_3`，见 `scripts/sql` 的分库分表 )。
- `contact_[0-9]+$` 的 `$` 锚定结尾，避免 `contact_application_*` 被 `contact_*`
  误匹配 ( 这也是两组前缀都带锚点的原因 )。
- **消息表 ( message / inbox_seq / inbox_entry / conversation_seq ) 有意不在监听列表**:
  消息链路走 Kafka 应用层事件 ( 落库后投 `hermet.message.index` )，不走 CDC——
  CDC 只服务低频业务表的读侧宽表同步 ( 见 `scripts/sql/README.md` 的 CQRS 架构 )。
- 新增需要 CDC 的分表时，把正则追加到列表并 `cli.sh update` 重启 connector
  ( 首次会按 snapshot 规则补存量 )。

### 复制槽与出版物 ( PostgreSQL 侧 )

```json
"plugin.name": "pgoutput",
"publication.name": "dbz_publication_hermet_0",
"publication.autocreate.mode": "filtered",
"slot.name": "dbz_slot_hermet_0",
```

- `plugin.name=pgoutput`: PG 官方逻辑解码插件，无需额外安装 ( 对应
  compose 里的 `wal_level=logical` )。
- `publication.name`: PG 出版物 ( 决定哪些表的变更进入 WAL 流 )，
  `filtered` 模式下 Debezium 按 `table.include.list` 自动维护出版物成员。
- `slot.name`: 复制槽。**slot 在 PG 实例内独占 WAL 保留**——
  connector 删除后 slot 不会自动消失，残留的 slot 会让 PG 无限保留 WAL 撑爆磁盘
  ( 受 `max_slot_wal_keep_size=2GB` 上限保护，超限后 connector 报错 )。
  删除 connector 时记得在 PG 侧清理:
  `SELECT pg_drop_replication_slot('dbz_slot_hermet_0');`
- slot/publication 名按实例命名 ( hermet_0 / hermet_1 )，两个 connector
  连不同 PG 实例所以不冲突，命名只为运维排查时一眼对应。

### 快照 ( snapshot )

```json
"snapshot.mode": "initial",
"snapshot.fetch.size": "1000",
```

- `initial`: **仅当** schema history topic / offset 都不存在时 ( 全新部署 ) 才做
  全量快照，之后永远增量。
- 不用 `when_needed`: 该模式在 offset 丢失 / schema history 损坏时会**重新全量快照**，
  大表瞬间灌满 Kafka ( snapshot 风暴 )。若真发生了 offset 丢失，更可控的做法是
  手工评估后临时切换或做定点修复。
- `snapshot.fetch.size=1000`: 快照时每批读取行数，压小以限制对 broker IO 与
  源库读压力的瞬时冲击。
- 重新快照的正规姿势: 删 connector + 删 slot + 删 schema history topic 后重建，
  而不是改 snapshot.mode。

### 数据类型与事件形态

```json
"time.precision.mode": "adaptive_time_microseconds",
"decimal.handling.mode": "precise",
"include.headers": true,
"tombstones.on.delete": true,
```

- `adaptive_time_microseconds`: 按列精度映射时间类型，微秒精度不丢 ( backend 的
  cdc event 解析依赖 )。
- `decimal.handling.mode=precise`: NUMERIC 用精确编码 ( 表内多为 BIGINT，不受影响 )。
- `tombstones.on.delete=true`: 删除事件后追加一条 null 值记录 ( tombstone )，
  供下游 ( Kafka compacted topic 场景 ) 清理；配合 SMT 的 `drop.tombstones=false`
  保留到消费侧。backend 的 cdc consumer 按 `__deleted` 字段识别删除。

### 心跳

```json
"heartbeat.interval.ms": "10000",
"heartbeat.topics.prefix": "__debezium_heartbeat",
```

- 心跳的作用: 监听的表**长期无变更**时，slot 的确认位点不推进会导致
  (1) PG 侧 WAL 持续保留; (2) Kafka offset 过期被删。
  心跳每 10s 产生一条事件 ( topic `__debezium_heartbeat.pgsql_hermet_0` )
  持续推进位点。
- backend 的 `internal/cdc/heartbeat_processor.go` 消费心跳并监控 CDC 链路延迟
  ( `HeartbeatService` 健康报告 )，**不要去掉**。

### 限流 ( 防影响业务链路 )

```json
"max.batch.size": "1024",
"max.queue.size": "4096",
```

- 每批流向 Kafka 的最大记录数 / 内部队列容量。CDC 与消息业务
  ( `synp.message.*` ) 共用 Kafka 集群，压小批量避免 CDC 突发挤占 broker IO
  ( 见 `../../README.md` 的共用集群决策与拆分触发条件 )。
- 队列满时 connector 会阻塞读取 ( 背压到 WAL 位点 )，不会丢数据。

### Schema History ( 必须每 connector 唯一 )

```json
"schema.history.internal.kafka.bootstrap.servers": "kafka-1:9093,kafka-2:9093,kafka-3:9093",
"schema.history.internal.kafka.topic": "schema-changes-pgsql_hermet_0",
```

- 记录监听表的 DDL 变更历史，connector 重启时靠它还原表结构。
- **topic 名必须每个 connector 唯一**——两个库共用会把双方 DDL 混写，
  重启时可能按另一库的 schema 解析 binlog 导致数据错乱
  ( 历史遗留配置曾共用 `schema-changes.dbserver1`，已按实例拆分 )。
- `bootstrap.servers` 指向 Kafka **INTERNAL** listener ( 容器网内 mTLS )，
  与 compose 的 `BOOTSTRAP_SERVERS` 保持一致；此处无法引用环境变量，
  修改 Kafka 拓扑时需同步改这里。

### SMT: ExtractNewRecordState ( unwrap )

```json
"transforms": "unwrap",
"transforms.unwrap.type": "io.debezium.transforms.ExtractNewRecordState",
"transforms.unwrap.drop.tombstones": "false",
"transforms.unwrap.delete.handling.mode": "rewrite",
"transforms.unwrap.add.fields": "op,db,table,ts_ms"
```

- 作用: 把 Debezium 原生的 `before/after/source` 嵌套事件**拍平为单层记录**——
  消息体就是行数据本身，backend 的 cdc event 结构体直接反序列化。
- `delete.handling.mode=rewrite`: 删除事件改写为**保留行字段 + `__deleted: true`**
  的普通记录 ( 而不是只剩主键的删除标记 )，下游统一按 `__deleted` 处理增删改。
- `add.fields`: 附加操作元数据到消息体——`op` ( c/u/d/r )、`db`、`table`、`ts_ms`
  ( 变更时间戳 )，backend 的 processor 依赖这些字段路由事件。
- `drop.tombstones=false`: tombstone ( null 记录 ) 继续下发，配合
  `tombstones.on.delete=true`。

## 修改配置的操作

```bash
# 查看当前生效配置
./cli.sh status pgsql-hermet-0

# 可在线热更的参数 ( 表过滤、限流、心跳等 ): 改 JSON 后
./cli.sh update ./connectors/pgsql_hermet_0_connector.json

# 不可热更的参数 ( topic.prefix / slot.name / schema history topic 等 ):
# 变更等价于新数据流，必须删旧建新
./cli.sh delete pgsql-hermet-0
#   + PG 侧手动清理 slot ( 见上文 )
#   + 删除旧 schema history topic ( kafka 侧 )
#   + 重新 register ( 会触发 initial snapshot，注意时机 )
```

> 删除 connector 前**务必确认 slot 清理**，否则 PG 的 WAL 保留会持续增长
> ( 有 2GB 上限保护，但触顶后 connector 将无法重启，只能清 slot 重快照 )。
