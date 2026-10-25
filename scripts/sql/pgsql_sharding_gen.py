#!/usr/bin/env python3
"""pgsql 分表 DDL 生成器 ( sharding 模板 → <table>_<N> 物理分表 )。

用法:
    python3 scripts/sql/pgsql_sharding_gen.py --db-shards 2 --tb-shards 4 [--out ./sharding]

产物: out/db_<N>/init.sql — 每个分片库一份完整 DDL ( 全部分表 )，
供 psql / exec_sql.sh 执行。表名与 etc/config.yaml 的 sharding.* 前缀一一对应。
"""
import argparse
import pathlib
import re
import sys

SQL_DIR = pathlib.Path(__file__).parent / "pgsql"


def extract_table_blocks() -> list[tuple[str, str]]:
    statements = []
    for t in sorted(SQL_DIR.glob("*.sql")):
        src = t.read_text()
        for m in re.finditer(r"DROP TABLE IF EXISTS (\w+);(.*?)CREATE TABLE \1 \((.*?)\n\);", src, re.S):
            tbl, pre, body = m.group(1), m.group(2), m.group(3)
            block = [f"DROP TABLE IF EXISTS {tbl};{pre}CREATE TABLE {tbl} ({body}\n);"]
            # 表级附属语句随分表复制:
            # - COMMENT ON <tbl>/<tbl>.col 仅当表名匹配 ( 模板含未建表的孤儿注释 );
            # - CREATE INDEX 名含 <tbl> 前缀。
            tail = src[m.end():]
            nxt = re.search(r"DROP TABLE IF EXISTS", tail)
            tail = tail[: nxt.start()] if nxt else tail
            # 语句可能跨行 ( 如多列索引 )，按分号完整收集。
            for stmt in re.finditer(r"((?:COMMENT ON|CREATE (?:UNIQUE )?INDEX).*?;)", tail, re.S):
                full = " ".join(stmt.group(1).split())
                if full.startswith(f"COMMENT ON TABLE {tbl} ") or full.startswith(f"COMMENT ON COLUMN {tbl}."):
                    block.append(full)
                elif re.match(rf"CREATE (?:UNIQUE )?INDEX \S+ ON {tbl} ?\(", full):
                    block.append(full)
            statements.append((tbl, "\n".join(block) + "\n"))
    return statements


def shard_table_refs(block: str, tbl: str, suffix: str) -> str:
    """仅替换表引用位置为分表名，列名/注释中的表名前缀 ( 如 contact_id、
    channel_name ) 不受影响。

    表引用位置: DROP/CREATE TABLE、COMMENT ON TABLE/COLUMN、INDEX ... ON <tbl>(。
    """
    target = tbl + suffix
    out = block.replace(f"DROP TABLE IF EXISTS {tbl};", f"DROP TABLE IF EXISTS {target};")
    out = out.replace(f"CREATE TABLE {tbl} (", f"CREATE TABLE {target} (")
    out = out.replace(f"COMMENT ON TABLE {tbl} ", f"COMMENT ON TABLE {target} ")
    out = out.replace(f"COMMENT ON COLUMN {tbl}.", f"COMMENT ON COLUMN {target}.")
    out = re.sub(rf"(\s)ON {tbl} ?\(", rf"\1ON {target}(", out)
    # 表内约束 ( UNIQUE/CHECK ) 的索引名是 schema 级唯一，分表需加后缀。
    out = re.sub(r"(CONSTRAINT )(\w+)", rf"\g<1>\g<2>{suffix}", out)
    return out


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--db-shards", type=int, default=2)
    parser.add_argument("--tb-shards", type=int, default=4)
    parser.add_argument("--out", default="./sharding")
    args = parser.parse_args()

    tables = extract_table_blocks()
    if not tables:
        print("no template tables found", file=sys.stderr)
        return 1

    out_dir = pathlib.Path(args.out)
    for db in range(args.db_shards):
        db_dir = out_dir / f"db_{db}"
        db_dir.mkdir(parents=True, exist_ok=True)
        parts = [f"-- hermet 分片库 db_{db} 初始化 ( {len(tables)} 张模板表 x {args.tb_shards} 分表 )\n"]
        for tbl, tbl_block in tables:
            for tb in range(args.tb_shards):
                parts.append(f"\n-- ==== {tbl}_{tb} ====\n")
                shard_block = shard_table_refs(tbl_block, tbl, f"_{tb}")
                # 同库内索引名唯一，分表索引名同样追加后缀。
                shard_block = re.sub(
                    r"(CREATE (?:UNIQUE )?INDEX )(\S+)",
                    rf"\1\2_{tb}",
                    shard_block,
                )
                parts.append(shard_block)
        (db_dir / "init.sql").write_text("".join(parts))
        print(f"wrote {db_dir / 'init.sql'}")

    print(f"done: {len(tables)} tables x {args.tb_shards} shards x {args.db_shards} dbs")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
