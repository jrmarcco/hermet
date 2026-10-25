#!/usr/bin/env bash
set -euo pipefail

# hermet 本地联调一键启动 ( 经验固化自 gateway × backend 端到端联调 )。
#
# 用法: ./scripts/dev-up.sh [子命令]
#   deps    起 docker 依赖 ( kafka/joint(redis+etcd)/pgsql/es )
#   schema  生成分表 DDL 并建表 ( 幂等, 会重建分表 )
#   config  生成 etc/config-joint.yaml ( 注入 kafka 证书与各组件密码 )
#   build   编译 bin/hermet
#   start   后台启动服务并等待 healthz
#   stop    停止服务
#   seed    建联调用户 a/b@joint.test 并输出 contact 直插 SQL
#   all     以上全流程 ( 默认 )
#
# 环境变量:
#   GATEWAY_DIR   gateway 仓库路径 ( 默认 ../gateway, kafka 部署与证书来源 )
#   SUDO_PASS     sudo 密码 ( 缺省交互输入; 脚本内 docker 均经 sudo )
#
# 经验要点 ( 详见 scripts/DEV.md ):
#   - kafka 是 gateway 强制 mTLS, 证书由 gateway/scripts/docker/kafka 生成,
#     客户端 PEM 从 JKS 导出;
#   - etcd/redis 用无 TLS 联调版 ( scripts/docker/joint );
#   - Debezium 不启动: contact/user_group 读侧物化表需 seed 直插。

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
GATEWAY_DIR="${GATEWAY_DIR:-$(cd "$ROOT_DIR/.." && pwd)/gateway}"
LOG_FILE="${LOG_FILE:-/tmp/hermet-dev.log}"
ADMIN_ADDR="127.0.0.1:10090"

cd "$ROOT_DIR"

# sudo 经 askpass 供密 ( 本机无 tty 时 sudo -v 无法建立可复用 ticket,
# 且管道 stdin 场景 sudo -S 会与 docker exec -i 抢输入——askpass 两者皆避 )。
if [[ -n "${SUDO_PASS:-}" ]]; then
    AP_FILE=$(mktemp); printf '%s' "$SUDO_PASS" > "$AP_FILE"; chmod 600 "$AP_FILE"
    AP_SH=$(mktemp); printf '#!/bin/sh
cat %s
' "$AP_FILE" > "$AP_SH"; chmod +x "$AP_SH"
    export SUDO_ASKPASS="$AP_SH"
    trap 'rm -f "$AP_FILE" "$AP_SH"' EXIT
fi
sudo -n true 2>/dev/null || sudo -A true 2>/dev/null || { echo "sudo 不可用: 传入 SUDO_PASS 或先手动 sudo -v" >&2; exit 1; }
DK() { if [[ -n "${SUDO_ASKPASS:-}" ]]; then sudo -A docker "$@"; else sudo docker "$@"; fi; }

log()  { echo "[$(date +%H:%M:%S)] $*"; }
wait_healthy() { # wait_healthy <容器名> <秒>
    local i=0
    until DK inspect --format '{{.State.Health.Status}}' "$1" 2>/dev/null | grep -q healthy; do
        ((i+=1)); [[ $i -gt $(($2/3)) ]] && { log "✗ $1 未在 ${2}s 内 healthy"; return 1; }
        sleep 3
    done
    log "✓ $1 healthy"
}

cmd_deps() {
    log "== docker 依赖 =="

    # 1. kafka ( gateway 仓库, mTLS 三节点 )。
    local kafka_dir="$GATEWAY_DIR/scripts/docker/kafka"
    if [[ ! -d "$kafka_dir" ]]; then
        log "✗ 未找到 kafka 部署目录: $kafka_dir ( 检查 GATEWAY_DIR )"; exit 1
    fi
    if ! DK ps --format '{{.Names}}' | grep -q '^kafka-1$'; then
        log "启动 kafka ( 首次会生成证书与 .env )..."
        ( cd "$kafka_dir" && [[ -f .env ]] || ./init.sh >/dev/null 2>&1 || true
          [[ -f .env ]] && sed -i 's/^KAFKA_EXTERNAL_IP=.*/KAFKA_EXTERNAL_IP=127.0.0.1/' .env
          $( [[ -n "${SUDO_ASKPASS:-}" ]] && echo "sudo -A" || echo sudo ) chown -R 1000:1000 kafka-1-data kafka-2-data kafka-3-data 2>/dev/null || true
          DK compose up -d >/dev/null )
        wait_healthy kafka-1 120; wait_healthy kafka-2 120; wait_healthy kafka-3 120
    else
        log "✓ kafka 已运行"
    fi

    # 2. joint ( redis + 无 TLS etcd ) / pgsql / es。
    for comp in joint pgsql es; do
        local dir="$SCRIPT_DIR/docker/$comp"
        ( cd "$dir"
          [[ -f .env ]] || ./init.sh >/dev/null 2>&1 || true
          DK compose up -d >/dev/null )
    done
    DK network create jrmarcco_net >/dev/null 2>&1 || true

    # 数据目录属主修正 ( 容器内用户 )。
    if ! DK exec pgsql-hermet-0 true 2>/dev/null; then
        if [[ -n "${SUDO_ASKPASS:-}" ]]; then sudo -A chown -R 999:999 scripts/docker/pgsql/data-0 scripts/docker/pgsql/data-1 2>/dev/null || true; fi
    fi
    if ! DK exec hermet-elasticsearch true 2>/dev/null; then
        if [[ -n "${SUDO_ASKPASS:-}" ]]; then sudo -A chown -R 1000:1000 scripts/docker/es/es-data 2>/dev/null || true; fi
    fi

    wait_healthy pgsql-hermet-0 90
    wait_healthy pgsql-hermet-1 90
    wait_healthy hermet-elasticsearch 120
    DK ps --format '{{.Names}} {{.Status}}' | grep -E 'joint|kafka-1|pgsql|elastic' || true
    log "✓ 依赖就绪 ( joint-redis:16379 / joint-etcd:12379 )"
}

cmd_schema() {
    log "== 分表 DDL ( 19 模板表 × 4 分表 × 2 库, DROP+CREATE 幂等 ) =="
    python3 scripts/sql/pgsql_sharding_gen.py --out /tmp/hermet-sharding >/dev/null
    local pg0 pg1
    pg0=$(grep '^POSTGRES_0_PASSWORD=' scripts/docker/pgsql/.env | cut -d= -f2)
    pg1=$(grep '^POSTGRES_1_PASSWORD=' scripts/docker/pgsql/.env | cut -d= -f2)
    DK exec -i pgsql-hermet-0 sh -c "psql -U hermet_0 -d hermet_0 -v ON_ERROR_STOP=1" \
        < /tmp/hermet-sharding/db_0/init.sql > /tmp/ddl-0.log 2>&1 || { tail -3 /tmp/ddl-0.log; exit 1; }
    DK exec -i pgsql-hermet-1 sh -c "psql -U hermet_1 -d hermet_1 -v ON_ERROR_STOP=1" \
        < /tmp/hermet-sharding/db_1/init.sql > /tmp/ddl-1.log 2>&1 || { tail -3 /tmp/ddl-1.log; exit 1; }
    grep -q ERROR /tmp/ddl-0.log /tmp/ddl-1.log && { grep ERROR /tmp/ddl-*.log | head -3; exit 1; } || true
    log "✓ 建表完成 ( 详见 /tmp/ddl-*.log )"
}

cmd_config() {
    log "== 生成 etc/config-joint.yaml =="
    python3 - "$GATEWAY_DIR" <<'PYEOF'
import sys, yaml, pathlib

gateway = pathlib.Path(sys.argv[1])
certs = gateway / "scripts/docker/kafka/certs"
# 客户端 PEM 若缺失则从 JKS 导出 ( keytool/openssl 需可用 )。
if not (certs / "client.key").exists():
    import subprocess
    subprocess.run(["keytool", "-importkeystore",
                    "-srckeystore", str(certs/"kafka.client.keystore.jks"),
                    "-destkeystore", str(certs/"client.p12"),
                    "-srcstoretype", "JKS", "-deststoretype", "PKCS12",
                    "-srcstorepass", "kafka-secret", "-deststorepass", "kafka-secret"],
                   check=True, capture_output=True)
    subprocess.run(["openssl", "pkcs12", "-in", str(certs/"client.p12"), "-nodes", "-nocerts",
                    "-passin", "pass:kafka-secret", "-out", str(certs/"client.key")],
                   check=True, capture_output=True)

ca, crt, key = ((certs/n).read_text() for n in ("ca-cert", "client.crt", "client.key"))
def envpwd(path, key):
    for line in pathlib.Path(path).read_text().splitlines():
        if line.startswith(key):
            return line.split("=", 1)[1]
    raise SystemExit(f"{key} missing in {path}")

pg0 = envpwd("scripts/docker/pgsql/.env", "POSTGRES_0_PASSWORD")
pg1 = envpwd("scripts/docker/pgsql/.env", "POSTGRES_1_PASSWORD")
es  = envpwd("scripts/docker/es/.env", "ELASTIC_PASSWORD")

cfg = yaml.safe_load(open("etc/config.yaml"))
h = cfg["hermet"]
h["web"]["addr"] = "127.0.0.1:10081"
h["admin"]["addr"] = "127.0.0.1:10090"
h["grpc"]["server"]["addr"] = "127.0.0.1:10091"
h["grpc"]["client"]["notify"]["key_prefix"] = "synp"
h["ignores"] = list(dict.fromkeys(h.get("ignores", []) + ["/api/v1/admin/user"]))
cfg["etcd"] = {"endpoints": ["127.0.0.1:12379"], "dial_timeout": "5s", "tls": {"enabled": False}}
cfg["redis"] = {"addr": "127.0.0.1:16379"}
cfg["es"] = {"addresses": ["http://127.0.0.1:19200"], "password": es, **{k: v for k, v in cfg["es"].items() if k not in ("addresses", "password")}}
cfg["kafka"]["brokers"] = ["127.0.0.1:19092", "127.0.0.1:29092", "127.0.0.1:39092"]
cfg["kafka"]["tls"] = {"ca_cert": ca, "cert_key": key, "cert_pem": crt}
cfg["db"]["sharding"] = [
    {"name": "hermet_0", "dsn": f"postgres://hermet_0:{pg0}@127.0.0.1:15432/hermet_0?sslmode=disable&connect_timeout=3&statement_timeout=5000&idle_in_transaction_session_timeout=5000"},
    {"name": "hermet_1", "dsn": f"postgres://hermet_1:{pg1}@127.0.0.1:25432/hermet_1?sslmode=disable&connect_timeout=3&statement_timeout=5000&idle_in_transaction_session_timeout=5000"},
]
pathlib.Path("etc/config-joint.yaml").write_text(yaml.safe_dump(cfg, allow_unicode=True, sort_keys=False, width=4096))
print("  etc/config-joint.yaml written ( 含证书与密码, 已在 .gitignore 豁免范围外请勿提交 )" if False else "  etc/config-joint.yaml written")
PYEOF
    log "✓ 配置生成"
}

cmd_build() { make build >/dev/null && log "✓ bin/hermet"; }

cmd_start() {
    if curl -sf "http://$ADMIN_ADDR/healthz" >/dev/null 2>&1; then log "✓ 服务已在运行"; return; fi
    log "== 启动 hermet ( 日志: $LOG_FILE ) =="
    nohup ./bin/hermet --config etc/config-joint.yaml > "$LOG_FILE" 2>&1 &
    echo $! > /tmp/hermet-dev.pid
    local i=0
    until curl -sf "http://$ADMIN_ADDR/healthz" >/dev/null 2>&1; do
        ((i+=1)); [[ $i -gt 20 ]] && { log "✗ 启动超时, 最近日志:"; tail -5 "$LOG_FILE"; exit 1; }
        sleep 1
    done
    log "✓ hermet 就绪: web :10081 / grpc :10091 / admin :10090 ( /healthz /readyz /version /metrics )"
}

cmd_stop() {
    [[ -f /tmp/hermet-dev.pid ]] && { kill "$(cat /tmp/hermet-dev.pid)" 2>/dev/null || true; rm -f /tmp/hermet-dev.pid; }
    pkill -f "bin/hermet --config etc/config-joint.yaml" 2>/dev/null || true
    log "✓ 已停止"
}

cmd_seed() {
    log "== 测试数据 ( 用户经 admin API, 好友/群成员关系直插写侧表 ) =="
    curl -sf -X POST http://127.0.0.1:10081/api/v1/admin/user -H 'Content-Type: application/json' \
        -d '{"email":"a@joint.test","nickname":"Alice","gender":"female"}' >/dev/null \
        || log "  a@joint.test 已存在"
    curl -sf -X POST http://127.0.0.1:10081/api/v1/admin/user -H 'Content-Type: application/json' \
        -d '{"email":"b@joint.test","nickname":"Bob","gender":"male"}' >/dev/null \
        || log "  b@joint.test 已存在"
    python3 - <<'PYEOF'
import os, subprocess, time

def q(sql):
    sudo = ["sudo", "-A"] if os.environ.get("SUDO_ASKPASS") else ["sudo"]
    r = subprocess.run(sudo + ["docker", "exec", "pgsql-hermet-0", "psql", "-U", "hermet_0",
                        "-d", "hermet_0", "-tAc", sql], capture_output=True, text=True)
    return r.stdout.strip()

uids = {}
for tb in range(4):
    for line in q(f"SELECT id, email FROM biz_user_{tb} WHERE email LIKE '%joint.test'").splitlines():
        uid, email = line.split("|")
        uids[email.split("@")[0]] = int(uid)
a, b = uids["a"], uids["b"]
print(f"  Alice={a} Bob={b}")

def shard(uid):  # user_id 路由: idx=uid%8 → db=idx%2, tb=idx//2
    idx = uid % 8
    return idx % 2, idx // 2

now = int(time.time() * 1000)
for uid, peer in ((a, b), (b, a)):
    db, tb = shard(uid)
    sql = (f"INSERT INTO contact_{tb} (id,user_id,contact_id,remark_name,source,is_starred,is_blocked,"
           f"is_muted,is_pinned,added_at,deleted_at,created_at,updated_at) VALUES "
           f"(9000000000000{uid%1000},{uid},{peer},'','search',false,false,false,false,{now},0,{now},{now}) "
           f"ON CONFLICT (user_id, contact_id) DO NOTHING;")
    print(f"  [db_{db}] {sql}")
PYEOF
    log "✓ 用户已建; 请在对应分片库执行上面打印的 contact INSERT ( Debezium 未启动, 读侧关系需直插 )"
}

case "${1:-all}" in
    deps)   cmd_deps ;;
    schema) cmd_schema ;;
    config) cmd_config ;;
    build)  cmd_build ;;
    start)  cmd_start ;;
    stop)   cmd_stop ;;
    seed)   cmd_seed ;;
    all)    cmd_deps; cmd_schema; cmd_config; cmd_build; cmd_start; log "🎉 全部就绪" ;;
    *)      grep '^#' "$0" | sed 's/^# \{0,1\}//' | head -22; exit 1 ;;
esac
