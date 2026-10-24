# hermet 中间件部署 ( docker compose )

本目录提供后端依赖中间件的本地/内网部署编排:

- [`pgsql/`](./pgsql) — PostgreSQL 分片集群 ( hermet_0 / hermet_1，逻辑复制配置供 CDC )
- [`debezium/`](./debezium) — Debezium Connect 集群 ( 3 节点，PgSQL CDC → Kafka )
- [`es/`](./es) — Elasticsearch 单节点 ( 消息搜索查询侧，Phase 5 索引目标 )

Kafka / etcd 部署复用 [`gateway/scripts/docker`](../../gateway/scripts/docker) 中的编排 ( backend 与 gateway 共用同一套 Kafka 集群与 etcd 服务发现 )。

所有 compose 共享外部网络 `jrmarcco_net` ( 各目录的 `init.sh` 会自动创建 )。

## 快速开始

```bash
# pgsql ( 分片库 )
cd pgsql && ./init.sh && docker compose up -d

# debezium ( CDC，依赖 pgsql 与 gateway 的 kafka )
cd ../debezium && ./init.sh && docker compose up -d

# ( 可选 ) es 搜索
cd ../es && ./init.sh && docker compose up -d
```

## 变量与密钥约定

- **密码/地址不入版本库**: 所有凭据经 `.env` 注入, 各目录有 `.env.example` 模板
  ( `init.sh` 会生成一份 `.env`, 密码随机生成, 按需修改后再启动 )。
- **镜像版本固定**: compose 中镜像 tag 默认固定 ( 如 `postgres:18.6` /
  `debezium/connect:3.7.0.Final` / `elasticsearch:9.5.5` ), 可经 `.env` 覆盖
  ( `POSTGRES_IMAGE` 等 )。
- **密码同步点** ( 修改 pgsql 密码时需同步三处 ):
  1. `pgsql/.env` ( 数据库实际密码 )
  2. `etc/config.yaml` 的 `db.sharding` DSN
  3. `debezium/connectors/*.json` 的 `database.password`
  ( ES 密码同理: `es/.env` 与 `etc/config.yaml` 的 es 段 )

## 架构定位提醒

- **PG 是唯一事实源**: 消息/收件箱/seq 全部落 PG; ES 只是**异步查询加速层**
  ( 可重建、最终一致 ), 绝不进入 SYNC 同步主链路——详见 `scripts/sql/README.md`。
- **Debezium 与业务共用 Kafka 集群**: CDC 监听的是低频业务表, 消息表不走 CDC
  ( ES 喂养走应用自发索引事件 ), 无需独立集群。拆分触发条件见
  `api/README.md` 的 Kafka 拓扑章节; connector 已配置 `snapshot.mode=initial`
  ( 仅首次快照, 避免 `when_needed` 在 schema history 丢失时触发全量重快照风暴 )。

## 注意事项

- 容器间互访走 `jrmarcco_net` 内部地址 ( 如 `pgsql-hermet-0:5432` );
  宿主机端口映射 ( 15432/25432 ) 供集群外客户端 ( backend、psql ) 使用。
- **pg 镜像升级**: 数据目录绑定持久化 ( `data-0` / `data-1` ), PG **不允许跨主版本
  直接启动旧数据** ( 如 17 的数据目录换 18 镜像会报 `incompatible` ),
  需 `pg_upgrade` 或导出重建。
- **Debezium 客户端证书**: connect 以 mTLS 客户端身份连 Kafka, `init.sh` 自动从
  `gateway/scripts/docker/kafka/certs` 复制 ( 可经 `KAFKA_CERTS_DIR` 指定来源 ),
  keystore 密码需与 kafka 部署的 `KAFKA_TLS_KEY_PASSWORD` 一致。
- **Debezium connector 配置约定**: 每个 connector 的 `slot.name` / `publication.name` /
  `schema.history.internal.kafka.topic` 按实例命名 ( hermet_0 / hermet_1 各自独立,
  schema history topic 共用会导致两库 DDL 历史混写 )。
  **存量环境升级注意**: slot/publication 改名后, 旧 connector 的 slot 不会自动删除
  ( 会一直保留 WAL )——先 `./cli.sh delete pgsql-hermet-N` 再在 PG 侧执行
  `SELECT pg_drop_replication_slot('debezium_slot');` 释放旧 slot。
  消息表 ( message / inbox_* / conversation_seq ) 不在 CDC 监听列表 ( 设计如此,
  ES 喂养走应用自发索引事件, 见 internal/search )。
- **es 数据目录属主**: ES 镜像以 uid 1000 运行, `init.sh` 会将 `es-data` 属主
  修正为 `1000:1000`, 否则启动即失败 ( `AccessDeniedException` )。
- 数据经 bind mount 持久化在各自目录 ( `data-N` / `es-data` ), 销毁集群时手动清理。
