# 生产环境部署指南 ( hermet backend + synp gateway )

> 基于当前仓库实际交付能力编写 ( 2026-10 )。部署形态: docker / docker compose。
> 本地联调请看 `scripts/DEV.md`; 本文只讨论生产。

## 1. 拓扑与组件清单

```
                    ┌──────────── gateway ×N ( synp ) ────────────┐
 client ── ws/wss ──▶ ws:10071   grpc:50051   admin:9090           │
                    │   │ AUTH(RS256+JWKS)    │ SyncService 调用   │
                    └───┼─────────────────────┼────────────────────┘
                        │ Kafka(mTLS)         │ etcd 服务发现
                        ▼ upstream/notify     ▼ synp:///synp-backend
                    ┌──────────── backend ×N ( hermet ) ───────────┐
                    │ web:10081  grpc:10091  admin:10090            │
                    │  落库+seq / SyncService / ES 索引 / 搜索       │
                    └──┬─────────┬──────────┬──────────────────────┘
                       ▼         ▼          ▼
                 PG 分片×2   Debezium CDC   ES ( 查询加速 )
                 Redis( session/注册表 )   etcd( 注册发现 )
```

| 组件 | 数量 | 端口 | 部署来源 |
|---|---|---|---|
| gateway | ≥2 | 10071 / 50051 / 9090 | `gateway` 镜像 |
| backend | ≥2 | 10081 / 10091 / 10090 | `hermet` 镜像 |
| kafka ( mTLS ×3 ) | 3 | 19092/29092/39092 | `gateway/scripts/docker/kafka` |
| etcd ( TLS ×3 ) | 3 | 12379 | `gateway/scripts/docker/etcd` |
| pgsql 分片 | 2 | 15432 / 25432 | `backend/scripts/docker/pgsql` |
| redis | 1 ( 可 HA ) | 6379 | 自备 ( 见 §3.5 ) |
| elasticsearch | 1 ( 可扩 ) | 9200 | `backend/scripts/docker/es` |
| debezium ×3 | 3 | 8083 | `backend/scripts/docker/debezium` ( 可选，见 §8 ) |
| prometheus + grafana | 1 | 19090 / 13000 | `gateway/scripts/docker/observability` |

依赖关系决定**启动顺序**: 中间件 → backend → gateway
( gateway 鉴权依赖 backend 的 JWKS 端点，SyncService 依赖 etcd 里的 synp-backend 注册 )。

## 2. 前置要求

- Docker ≥ 24 ( compose v2 )；构建机需 go 1.27 / golangci-lint ( 仅 CI 用 )。
- 机器间内网互通；对外仅暴露 gateway 的 10071 ( 经 LB 终结 wss ) 与 10081 域名侧。
- **admin 端口 ( 9090 / 10090 ) 与数据库/中间件端口不得暴露公网**，仅内网/监控网可达。
- NTP 对时 ( JWT exp 有 60s 容差，时钟漂移过大导致鉴权失败 )。

## 3. 中间件部署

### 3.1 kafka ( mTLS 三节点 )

```bash
cd gateway/scripts/docker/kafka
# 需要 jdk 环境 ( keytool )
./init.sh                       # 生成 .env ( 随机 KAFKA_CLUSTER_ID ) 与自签证书
# 生产必改 .env:
#   KAFKA_EXTERNAL_IP=<对客户端广播的 IP/LB 地址>
#   KAFKA_UI_PASSWORD=<强密码> ( 或注释掉 kafka-ui 服务 )
docker compose up -d
```

要点: advertised 地址必须从 gateway/backend 真实可达；客户端证书 PEM 由
`certs/client.*` 提供 ( JKS 导出方法见 `scripts/DEV.md`，密码与
`KAFKA_TLS_KEY_PASSWORD` 同源 )。生产证书建议替换为内部 CA 签发。

### 3.2 etcd ( TLS 三节点 )

```bash
cd gateway/scripts/docker/etcd && ./init.sh && docker compose up -d
```

自签 Ed25519 证书 ( SAN 经 `SAN_IPS` 覆盖为生产 IP )；key_prefix 统一 `synp`。

### 3.3 pgsql 分片

```bash
cd backend/scripts/docker/pgsql && ./init.sh    # 生成随机密码 .env ( 妥善保管 )
docker compose up -d
```

分库分表参数 ( `db_shard_count=2, tb_shard_count=4` ) **部署期固定**，
扩容需数据迁移，容量规划见 §6。初始化 schema:

```bash
# ./scripts/sql
python3 ./sql/pgsql_sharding_gen.py --out ./sharding
# 两库分别执行 ( 或经 exec_sql.sh ):
docker exec -i pgsql-hermet-0 psql -U hermet_0 -d hermet_0 < sharding/db_0/init.sql
docker exec -i pgsql-hermet-1 psql -U hermet_1 -d hermet_1 < sharding/db_1/init.sql
```

注意: 模板 DDL 为 DROP+CREATE，**仅用于初始化**；线上结构变更走增量 SQL，
不得整份重放。

### 3.4 elasticsearch

```bash
cd backend/scripts/docker/es && ./init.sh && docker compose up -d
```

生产建议: 索引按时间滚动 + ILM ( 当前单索引 `hermet-message`，mapping 见
`internal/search/mapping.go` )；多节点时调大 `number_of_shards` 并
reindex。`ELASTIC_PASSWORD` 妥善保管。

### 3.5 redis

联调用无密码单节点 ( `backend/scripts/docker/joint` )——**生产自备**:
要求 `requirepass` 强密码 ( 两服务配置均支持密码字段 )，会话与连接注册表
依赖它，建议 sentinel/cluster 形态。

## 4. 密钥准备

| 密钥 | 生成方式 | 轮换 |
|---|---|---|
| 网关 JWT RSA 密钥对 | `backend/scripts/gen_jwt_rsa_keys.sh` ( kid=公钥指纹 ) | 新钥入 JWKS 并存 → 切签发 → 观察期后下线旧钥 ( 30s 负缓存 ) |
| kafka mTLS | kafka `certs_gen.sh` | 换证书需重启 kafka 与两端客户端 |
| etcd TLS | etcd `certs_gen.sh` | 同上 |
| 各组件密码 | 各 `init.sh` 随机生成 | 同步点见各 `.env.example` 注释 |

**所有密钥/密码不入版本库** ( `.gitignore` 已覆盖 `.env`/`config-joint.yaml` );
建议经部署密管 ( vault / secrets manager ) 注入环境变量或挂载。

## 5. 服务构建与部署

### 5.1 构建镜像

```bash
# backend
cd backend && make image        # → hermet:<version> ( 含 hermet + reindex 双二进制 )
# gateway
cd gateway && make image        # → synp-gateway:<version>
# 国内网络: make image GOPROXY=https://goproxy.cn,direct
```

版本号取 `git describe`，经 ldflags 注入 `/version` 端点可查。推送到私有
registry 后在生产机 `docker pull`。**配置文件不打包进镜像**，运行时挂载。

### 5.2 生产配置

以 `etc/config.yaml` 为基线，与联调 ( `config-joint.yaml` ) 的差异点:

| 项 | 生产取值 |
|---|---|
| `log.mode` | `prod` ( JSON，面向采集 ) |
| `log.file` | 按需开启 ( 轮转参数 `max_size_mb/max_backups/max_age_days/compress` )；容器推荐 stdout 采集 |
| 所有地址 | 真实内网地址 ( 非 127.0.0.1 ) |
| `grpc.server.addr` | **本机内网 IP:10091** ( 注册进 etcd 供 gateway 解析，必须是 gateway 可达地址 ) |
| etcd / kafka TLS | 全开 ( 联调的 `tls.enabled: false` 禁用 )，证书内联或挂载 |
| 密码字段 | 全部真实值 ( pg DSN / redis / es ) |
| `jwt.gateway` | 生产 RSA 密钥对 ( 弃用仓库内嵌的开发密钥 ) |
| gateway `jwks_endpoint` | `http://<backend 内网地址>:10081/.well-known/jwks.json` |
| `hermet.ignores` | 收紧 ( 移除联调加的 `/api/v1/admin/user`，admin 接口生产禁用或加白名单 IP ) |

### 5.3 启动与验证

```bash
# backend ( 每台 )
docker run -d --name hermet -v $PWD/config.yaml:/app/etc/config.yaml:ro \
  --network <内网> -p 10081 -p 10091 -p 10090 hermet:<version>
# gateway ( 每台 ) 同理, 暴露 10071/50051/9090

# 验证 ( 顺序执行 ):
curl http://<backend>:10090/healthz     # 存活
curl http://<backend>:10090/readyz      # redis 阻断判定 / es 仅展示
curl http://<backend>:10090/version     # 版本核对
curl http://<gateway>:9090/healthz
# 端到端冒烟 ( 在 gateway 机器 ):
go run ./cmd/jointtest   # 需测试账号; 生产可用只读检查替代
```

backend 多实例直接水平复制 ( 无状态；seq 分配依赖 PG 行锁，无需选主 )。
gateway 多节点已按节点隔离 notify 消费组，直接复制。

## 6. 容量与高可用要点

- **Kafka 分区**: `synp.message.upstream` 分区数 = 消费并行度上限 ( 当前 6 )，
  按发送者哈希有序；扩分区不影响有序性但需评估 consumer 再平衡。
- **PG 分片**: 写入吞吐与存储的分片上限; 慢查询监控走 gorm 日志
  ( `db.slow_threshold` )。备份见 §7。
- **ES**: 可整体重建 ( `go run ./cmd/reindex`，从 PG 全量恢复索引 )，
  不作为数据可靠性依赖。
- **滚动升级**: 先 gateway 后 backend ( 或反序均可，两端经 etcd/重连自愈 );
  单实例场景注意 consumer group 会话超时期间的消息堆积 ( at-least-once 兜底 )。

## 7. 运维

- **监控**: `gateway/scripts/docker/observability` 起 prometheus+grafana
  ( 含 synp-alerts 告警规则 ); backend 指标接入: 在
  `prometheus/targets/` 增加 backend admin:10090 抓取目标。
- **日志**: prod JSON 输出，组件标签 ( `component` 字段 ) 按功能域检索
  ( message-service / cdc-* / es-indexer / kafka / conn / server 等 )。
- **备份**: PG 每日 `pg_dump` 各分片 ( 自行接 cron/备份系统 ); ES 免备份
  ( reindex 可重建 ); kafka 依赖 topic 保留策略。
- **排查**: `/healthz` `/readyz` `/version` + pprof ( admin 端口，注意访问控制 )。

## 8. Debezium CDC ( 可选模块 )

读侧物化 ( 联系人/群视图同步 ) 依赖 CDC; 纯消息链路可不部署。
部署: `backend/scripts/docker/debezium` ( 依赖 kafka 与 pg 证书复制，
参数说明见其 `connectors/README.md` )。**connector JSON 内含数据库明文密码**，
配置文件权限 600，不入版本库。

## 9. 已知边界 ( 部署前知悉 )

1. **无 K8s 编排**: 本文为 docker compose 形态; 多机分布需按上表手动拆分，
   或自行容器平台化 ( 镜像已具备无状态水平伸缩条件 )。
2. **backend web/grpc 为明文**: 需部署在内网，对外流量经 LB/网关终结 TLS。
3. **顶号未实现**: 同账号新登录使旧 session 失效，旧连接在下次 AUTH 时才被拒
   ( api 契约缺断开接口，见 `api/README.md` 已知边界 )。
4. **schema 变更无自动迁移工具**: 分表结构演进走人工增量 SQL + 双写切换。
5. **单节点 ES/redis 的 HA** 需按 §3 自行扩展。
