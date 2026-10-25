# hermet 本地开发/联调启动指南

> 联调经验固化 ( gateway × backend 端到端验证, 2026-10 )。
> 一键脚本: `./scripts/dev-up.sh` ( 子命令见脚本头部注释 )。

## 拓扑与端口

| 组件 | 端口 | 来源 |
|---|---|---|
| kafka ×3 ( mTLS ) + ui | 19092/29092/39092 | `../gateway/scripts/docker/kafka` |
| joint-redis / joint-etcd ( 无 TLS ) | 16379 / 12379 | `scripts/docker/joint` ( 本仓库 ) |
| pgsql 分片 ×2 | 15432 / 25432 | `scripts/docker/pgsql` |
| elasticsearch | 19200 | `scripts/docker/es` |
| hermet: web / grpc / admin | 10081 / 10091 / 10090 | `etc/config-joint.yaml` |

## 快速开始

```bash
SUDO_PASS=<密码> ./scripts/dev-up.sh          # 全流程: 依赖→建表→配置→编译→启动
./scripts/dev-up.sh seed                      # 测试用户 + contact 直插 SQL
cd ../gateway && ./scripts/dev-up.sh          # 再起 gateway
cd ../gateway && go run ./cmd/jointtest       # 端到端验证 ( 单聊 )
go run ./cmd/jointtest --group-id <id>        # 群聊模式
./scripts/dev-up.sh stop                      # 停服务 ( docker 依赖保留 )
```

## 为什么 kafka 证书来自 gateway

gateway 对 kafka **强制 mTLS** ( 无证书不启动 )，证书由
`../gateway/scripts/docker/kafka/certs_gen.sh` 生成 ( JKS 格式 )。
Go 客户端需要 PEM——`dev-up.sh config` 会自动从 JKS 导出
`client.key`/`client.crt` 并连同 `ca-cert` 内联进 `etc/config-joint.yaml`。

etcd 与 redis 联调用无 TLS 单节点 ( `scripts/docker/joint` )；
生产/集群形态用 gateway 仓库的 etcd TLS 三节点部署。

## 测试数据 ( seed )

Debezium **不参与**消息主链路联调 ( 读侧物化表靠 CDC，另行验证 )，
因此好友关系与群成员关系需直插写侧表：

1. `dev-up.sh seed` 经 admin API 建用户 `a/b@joint.test` ( 密码 `hermet@2026` )，
   并打印按 user_id 路由的 `contact` INSERT ( 路由: `idx=uid%8 → db=idx%2, tb=idx//2` )；
2. 建群走 HTTP `/api/v1/group/create` ( Kafka 异步写 group_member，可验证 )；
3. 发群消息前需直插 `user_group` 两行 ( 同一路由规则 )。

## 已知坑 ( 联调实录 )

- **docker 需 sudo**: 脚本经 `SUDO_PASS` 预认证；手动操作先 `sudo -v`，
  且 `sudo docker exec -i ... < file` 时**不要**用 `echo pass | sudo -S` ( 密码会进 stdin )。
- **`go build ./...` 报 permission denied**: docker 数据目录 ( root 属主 ) 被工具链扫描，
  `sudo chmod -R o+rX scripts/docker/*/data-* scripts/docker/es/es-data`。
- **frontend 镜像拉取超时** ( DeadlineExceeded ): Docker Hub 瞬时抖动，重试即可。
- **WS 帧 message_id 去重**: gateway 按帧 ID 幂等，测试工具重复运行需唯一 ID
  ( jointtest 已内置时间戳 )。
- **protojson 特性**: 枚举序列化为名字 ( 如 `COMMAND_TYPE_AUTH_ACK` )、
  uint64/bytes 序列化为字符串/base64——客户端解析时注意类型。
- **ES 搜索为异步** ( 秒级延迟 )，且 `conversationId` 含冒号需 URL 编码。

## 排查入口

```
curl 127.0.0.1:10090/healthz   # 存活
curl 127.0.0.1:10090/readyz    # redis ( 阻断 ) / es ( 仅展示 )
curl 127.0.0.1:10090/version   # 构建版本
tail -f /tmp/hermet-dev.log    # 服务日志
```
