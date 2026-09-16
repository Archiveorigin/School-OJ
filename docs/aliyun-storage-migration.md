# 阿里云存储迁移与空间治理

## 1. 结论与适用范围

核查日期：2026-09-07。代码基线：`99c78d5a21fe79e223bcde6f39b40c8d90f78be1`。生产目录：`/opt/School-OJ`。

本项目可以迁移到阿里云，但目前不建议仅为释放磁盘而购买云数据库。推荐顺序为：确认服务器大目录的业务归属 → 建立 OSS 异地备份 → 按题包增长情况迁移对象存储 → 有独立数据库运维或高可用需求时再迁移 RDS PostgreSQL。

本文是方案与操作指南，不代表已实施。此次只读取了项目代码、服务器容量与数据库元数据，没有购买云资源、上传业务数据、修改生产配置或删除服务器文件；未进行真实阿里云实例兼容性测试。

## 2. 实际空间占用

生产根分区约 37.9 GiB，已用约 26.4 GiB，可用约 11.5 GiB，使用率 70%。下表是检查时点的近似值，不代表未来增长量。

| 对象 | 实际占用 | 判断 |
| --- | ---: | --- |
| OJ 的 PostgreSQL 业务库 `oj` | 14.5 MiB | 逻辑数据库大小，包含在下一行数据卷中，不能重复相加 |
| `school-oj_postgres-data` | 69.2 MiB | 包括业务库、系统库和 WAL 等物理数据 |
| `school-oj_minio-data` | 约 38 MiB | 题包、资源、附件、报告及对象存储内部数据 |
| `school-oj_redis-data` | 约 0.23 MiB | 队列等持久化数据 |
| `/var/lib/mysql` | 11.06 GiB | `mysqld` 服务仍在运行，当前 OJ 不使用此数据库，业务归属待查 |
| `/root/.cache/JetBrains` | 4.26 GiB | 远程开发缓存；需要区分正在使用和已过期的后端 |
| `/var/lib/containerd` | 4.69 GiB | 容器镜像内容，包含在用应用、判题环境和可能的回滚镜像 |
| `/var/www` | 2.17 GiB | 其他站点内容，归属待查 |

OJ 三个数据卷合计仅约 107 MiB。即便全部迁出并最终删除对应本地数据，也只能释放约 0.1 GiB；保留迁移回滚副本期间不会立即释放这部分空间。

因此，11 GiB 的 MySQL 与 4.3 GiB 的开发缓存，才是应优先核实的对象。不能把“不是当前 OJ 使用”理解为“可以删除”，也不能把整个 MySQL 目录当作可清理的日志。Docker 显示的可回收镜像同样不等于业务不再需要的镜像。

## 3. 项目数据应该迁到哪里

| 当前组件 | 保存内容 | 合适的阿里云目标 | 功能适配条件 |
| --- | --- | --- | --- |
| PostgreSQL 16 | 用户、权限、课程、考试、团队、题目正文与版本关系、提交源代码、成绩、审计和考试行为日志 | RDS PostgreSQL 16 | 保留数据库引擎、表结构、事务和关联关系；API、Worker 同步改连接 |
| MinIO | 完整题包与测试点、题面图片、工单参考附件、查重报告及源代码归档 | 私有 OSS Bucket | 保留对象键与元数据；适配对象客户端；不绕过 API 鉴权 |
| Redis 7 | 判题任务、租约、通知、限流等 | 当前继续本地运行 | 占用极小；不在本轮顺带更换队列 |
| 数据库导出文件、对象快照 | 恢复材料，不直接提供在线查询 | 独立的私有 OSS 备份 Bucket | 同时备份数据库和被引用对象，并实际演练恢复 |

“把数据库打包上传 OSS”只是备份，不是在线数据库迁移。网站无法直接对 OSS 中的 SQL/ZIP 文件执行当前的查询、登录和判题事务。不要把 OSS 挂载目录作为 PostgreSQL 的数据目录，也不要把运行中的 PostgreSQL/MySQL 数据文件直接打包当作可靠逻辑备份。

### 3.1 三种可行方案

| 方案 | 当前释放空间 | 业务改动 | 适用情况 |
| --- | --- | --- | --- |
| A：本地数据库和 MinIO不变，备份上传 OSS | 在线数据不释放；仅可在验证后轮换旧备份 | 备份、恢复与保留策略，不改变在线功能 | 现在优先做，主要提升数据安全 |
| B：PostgreSQL、Redis 留本地，MinIO 改 OSS | 最终约 38 MiB，未来题包增长不再全部占本地盘 | API 与 Worker 对象存储适配、网络容错和测试 | 题包开始明显增长时做 |
| C：PostgreSQL 改 RDS，MinIO 改 OSS，Redis 留本地 | 最终约 107 MiB 中的绝大部分 | 数据库连接、部署依赖、TLS、对象适配、切换与回滚 | 为托管、高可用和长期扩展做，而不是为当前小库减盘 |

如果 11 GiB 的 MySQL 仍有其他业务使用，应另立该业务的迁移清单，选匹配版本的 RDS MySQL 或其他合适目标；若确认为已停用业务，先逻辑备份、上传、恢复验证，再单独确认停服与清理。不能套用 OJ 的 PostgreSQL 命令操作它。

## 4. 价格和容易遗漏的费用

查价日期为 2026-09-07。以下为人民币预算测算，非实时下单报价，不包含未确认的优惠、资源包、免费额度以及可选服务。地域、冗余类型、存储等级和实际账单都会影响费用。

### 4.1 OSS：容量便宜，下载流量需要计入

官方存储计费说明使用标准本地冗余约 `0.12 元/GB/月` 举例；官方计费案例采用标准同城冗余 `0.15 元/GB/月`、日间公网流出 `0.50 元/GB`、GET `0.01 元/万次`。其中案例明确引用的是 2024-11-20 的价格，本文仅用它们估算量级，下单须重新核价。[存储计费说明](https://help.aliyun.com/zh/oss/storage-fees)、[官方计费案例](https://help.aliyun.com/zh/oss/billing-examples/)、[OSS 报价入口](https://www.aliyun.com/price/detail/oss)。

| 假设使用量 | 标准本地冗余容量费/月 | 按上述单价估算公网流出费/月 | 小计，不含请求等费用 |
| --- | ---: | ---: | ---: |
| 保存 10 GB，不下载 | 1.20 元 | 0 元 | 1.20 元 |
| 保存 10 GB，每月下载 10 GB | 1.20 元 | 5 元 | 6.20 元 |
| 保存 100 GB，不下载 | 12 元 | 0 元 | 12 元 |
| 保存 100 GB，每月下载 100 GB | 12 元 | 50 元 | 62 元 |

同城冗余的 100 GB 容量费按示例约 15 元/月。现有 OJ 数据很小，单做备份可以先预留每月 1～5 元预算观察账单；这是预算余量，不是套餐或费用上限，也不包含那套 11 GiB MySQL。

从服务器上传到 OSS 的流入通常不收费；OSS 下载到当前服务器，若走公网，属于 OSS 公网流出。只有满足同地域等网络条件并实际使用内网 Endpoint，才能采用相应内网免费流量口径。当前服务器的云厂商与地域尚未确认，不能假定享受该条件。[流量计费](https://help.aliyun.com/zh/oss/traffic-fees/)。

项目特有的成本风险：Worker 目前每次判题都会下载完整题包。如果平均题包 10 MiB、一个月 10,000 次提交且没有缓存，下载约 97.7 GiB；按上述流量口径，单下载费就约 49 元，尚未计重判和其他资源。题包只有几十 MiB，不等于每月下载只有几十 MiB。

建议按不可变版本和内容哈希增加有容量上限的 Worker 缓存，并监控命中率。缓存上限须按磁盘预算设置，不应为了省流量又无限占用本机磁盘。

在线题包、考试图片与附件用标准存储；不要自动转为需要解冻的归档类型。较老备份才考虑低频或归档，并核对取回费、最短存储期和恢复等待时间。额外的版本、跨地域副本、KMS、CDN、加速服务均应单独核价。[存储类型计费约束](https://help.aliyun.com/zh/oss/storage-fees)。

### 4.2 RDS：不能按实际 14.5 MiB 推算数据库账单

作为计算资源费用的比较样本，官方 PostgreSQL Serverless 表列出部分中国内地地域每节点 `0.333 元/RCU/小时`、存储 `0.0017 元/GB/小时`；基础系列一个节点，高可用两个节点。假设分配 20 GB、连续运行 720 小时，按表列单价计算如下。20 GB 只是测算配置，不代表所有产品的最低购买容量。[官方 Serverless 费用](https://help.aliyun.com/zh/rds/apsaradb-rds-for-postgresql/pricing-of-serverless-apsaradb-rds-for-sql-server-instances)。

| 计算假设 | 月度测算 |
| --- | ---: |
| 基础系列，持续 0.5 RCU，20 GB | 144.36 元 |
| 基础系列，持续 1 RCU，20 GB | 264.24 元 |
| 高可用，两节点各持续 0.5 RCU、20 GB | 288.72 元 |
| 高可用，两节点各持续 1 RCU、20 GB | 528.48 元 |

计算例：`(0.5 × 0.333 + 20 × 0.0017) × 720 = 144.36`。未额外套用页面优惠，也未包含额外备份、DAS、网络互联等费用；RCU 自动增长会提高账单。这些不是固定规格 RDS 的报价，更不是对考试并发容量的保证。

特别注意：当前官方 SSL 配置文档注明，计费方式为普通 Serverless 的实例暂不支持该 SSL 链路加密能力。因此，这张低规格成本表不能直接作为“公网接生产”的购买建议。若后续迁在线库，优先评估支持 TLS 的固定规格 PostgreSQL 16 实例，正式考试优先考虑高可用系列；按实际地域、规格、备份与网络条件获取控制台报价。不要把不同的 Serverless 产品类型混为一谈。[SSL 支持前提](https://help.aliyun.com/zh/rds/apsaradb-rds-for-postgresql/configure-ssl-encryption-for-an-apsaradb-rds-for-postgresql-instance)。

代码还有持续的 500 ms Outbox 轮询、10 秒健康检查和数据库连接池。因此，不能假设数据库仅在考试时计费、其他时间自动暂停；这是基于代码的风险判断，实际活跃与计费须在云实例验证。

## 5. 需要适配的代码和功能

| 位置 | 当前情况 | 实施迁移时需要做的事 |
| --- | --- | --- |
| `compose.yaml` | API、Worker 的数据库地址固定为 `postgres`，`sslmode=disable` | 两端同步参数化 DSN，挂载 CA；不能仅向 `.env` 添加同名变量就认为生效 |
| `compose.yaml` | 对象地址固定 `minio:9000`，未传入 `MINIO_USE_SSL` | 参数化两端 Endpoint、凭据、Bucket、TLS；生产与本地基础设施用独立配置或 profile |
| `apps/api/internal/config`、`apps/worker/internal/config` | 读取数据库和 MinIO 配置 | 统一校验云端配置，防止缺变量时回落到旧本地库 |
| `apps/api/cmd/api/main.go`、`apps/worker/cmd/worker/main.go` | 直接构造 MinIO Go Client | 验证 OSS S3 兼容，或接入统一对象存储接口及 OSS 原生适配器 |
| `apps/api/internal/services/seed.go` | 启动时检查 Bucket，不存在则创建 | 云端预先创建 Bucket；运行账号只校验，不授予随意创建 Bucket 的权限 |
| `apps/api/internal/handlers/server.go`、`problem_change_tickets.go`、`exam_security.go` | 题包、附件、普通和考试专属图片读写 | 保持原路由、鉴权、版本选择、Content-Type 和下载行为 |
| `apps/api/internal/jplag/service.go` | 查重归档和报告上传 | 与题包一起切换对象实现，验证大文件和失败重试 |
| `apps/worker/internal/streams/consumer.go` | 根据提交绑定版本下载完整 ZIP | 对象键不变，重判不读当前新版本；增加限量缓存、超时和可重试处理 |
| `apps/api/internal/db/db.go`、`apps/worker/internal/db/db.go` | 最大连接分别 30、10 | 核对 RDS 连接限额，并为迁移、管理和扩容保留余量 |
| `apps/api/internal/db/migrations.go`、`apps/api/migrations` | PostgreSQL 专用 SQL、迁移校验和 advisory lock | 同引擎恢复全部结构和迁移记录；不要把迁库顺带改为 MySQL |
| `apps/web` | 通过已有 API 访问数据 | 存储迁移原则上不需改页面，但仍需实际页面回归 |

OSS 提供 S3 兼容接口，但不是对 MinIO SDK 的完整兼容承诺。官方说明要求虚拟主机访问风格，且部分 ETag 语义不同。应先验证签名、地域、Get/Put/Head、大文件分片和元数据；不能只改域名后直接上线。S3 兼容公网地址示例为 `https://s3.oss-cn-hangzhou.aliyuncs.com`；现有 MinIO 构造器的 Endpoint 参数不含协议，TLS 由独立配置控制。[S3 差异](https://help.aliyun.com/zh/oss/developer-reference/compatibility-with-amazon-s3)、[官方 Endpoint 说明](https://help.aliyun.com/zh/oss/developer-reference/use-aws-sdks-to-access-oss)。

若兼容验证不通过，抽取 `Get/Put/Stat` 等实际需要的对象操作，用 OSS 原生 SDK 实现，保留 MinIO 实现用于本地。不要同时改动业务模型和前端契约。

重要业务边界：

- 所有历史 `problem_versions` 对应的对象都保留；不能因题目归档或替换而清掉旧题包，否则历史考试和重判会失效。
- Bucket 必须私有。继续通过 API 的普通或考试专属接口校验后读取，不把题包、测试点、工单材料开放为公共 URL。
- PostgreSQL 中的提交源代码也要完整迁移，仅复制 MinIO 无法恢复提交记录。
- 已认证考生的请求会查询数据库中的活动考试状态。跨云数据库延迟和中断会影响登录后功能、考试限制和排行榜，不能承诺迁移后一定更快。
- 不迁移或清空 Redis 来“顺便释放空间”；必须处理好数据库 Outbox 与现存判题任务的一致性。

## 6. 推荐实施流程

### 阶段一：先建立恢复能力，再治理空间

1. 核实 MySQL 版本、数据库清单与大小、连接来源、关联站点、定时任务和备份责任人。只做元数据调查，避免导出用户数据到公开日志。
2. 核实 JetBrains 后端是否在使用、哪些缓存已过期；记录镜像与当前应用、编译环境、回滚版本的关系。不要执行全局 `docker system prune -a --volumes`。
3. 创建专用私有 OSS 备份 Bucket；地域按服务器位置、延迟和学校的数据管理要求选择。备份可评估同城冗余，开启服务端加密和费用告警。
4. 使用独立 RAM 权限；上传备份账号仅访问指定前缀，删除旧备份由独立受控身份处理。API、Worker 的在线凭据不要兼任备份管理员，不把密钥放入 Git 或前端。
5. 先完整备份 OJ 数据库、逻辑对象和对象清单。配置、必要密钥另行加密托管；仅有数据库不能恢复完整系统。
6. 下载备份校验哈希，并恢复到隔离 PostgreSQL 和对象环境；使用独立 Redis，禁用真实邮件和生产队列消费者，避免恢复演练触发生产副作用。
7. 建议起始保留策略：本地最近 3 份成功快照，云端最近 30 份日备及 12 份月备；轮换前必须确认后续备份已成功。该策略需按数据敏感性与学校要求最终确认。
8. 数据保留周期与确认清理是独立决策。对在用 MySQL 不停服、不删文件；对已停用数据，在恢复成功并获得明确清理确认后才回收原空间。

若只每日备份一次，潜在数据损失窗口可达约 24 小时；需要更低窗口，应另行设计持续日志备份/PITR。恢复时长必须通过演练测出，不能以“上传成功”代替灾备验收。

### 阶段二：数据库备份的操作示例

以下命令用于后续获准实施时在生产服务器的 Bash 中执行，**此次未执行**。执行前安装并安全配置 `ossutil`，使用限定权限凭据；将示例 Bucket 替换为已核实的私有备份 Bucket。禁止开启会回显凭据的命令跟踪。

```bash
set -euo pipefail
umask 077
cd /opt/School-OJ

backup_stamp=$(date -u +%Y%m%dT%H%M%SZ)
backup_dir=$(mktemp -d /opt/School-OJ-backups/oss-export-XXXXXX)
backup_bucket='REPLACE_WITH_PRIVATE_BACKUP_BUCKET'

docker compose exec -T postgres sh -c \
  'exec pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' \
  > "$backup_dir/oj.dump"

test -s "$backup_dir/oj.dump"
docker compose exec -T postgres pg_restore --list \
  < "$backup_dir/oj.dump" > "$backup_dir/oj.contents.txt"

(cd "$backup_dir" && sha256sum oj.dump > SHA256SUMS)
ossutil cp "$backup_dir/oj.dump" \
  "oss://$backup_bucket/postgres/$backup_stamp/oj.dump"
ossutil cp "$backup_dir/SHA256SUMS" \
  "oss://$backup_bucket/postgres/$backup_stamp/SHA256SUMS"
ossutil cp "$backup_dir/oj.contents.txt" \
  "oss://$backup_bucket/postgres/$backup_stamp/oj.contents.txt"
```

该命令生成事务一致的单库备份，但不是与 MinIO、Redis 原子一致的整站快照。首次完整恢复点和正式切换应按下一阶段冻结业务写入。`pg_restore --list` 仅验证备份目录可读，仍要在独立数据库做实际恢复。[阿里云逻辑迁移方法](https://help.aliyun.com/zh/rds/apsaradb-rds-for-postgresql/migrate-data-from-a-self-managed-postgresql-database-to-an-apsaradb-rds-for-postgresql-instance-by-using-pg-dump-and-pg-restore)。

对象备份必须通过 MinIO 的对象接口导出，不能把 Docker 卷中的内部文件直接上传后当作可用 OSS 对象。可用预配置只读别名的 `mc mirror` 导出当前对象到新建空目录，但还必须单独保存对象键、大小、SHA-256、Content-Type 等清单。`mc mirror` 不完整保留对象版本历史及元数据；若 Bucket 开启过对象版本控制，需额外清点和导出历史版本。应用自己的不可变题目版本与 Bucket 的版本控制不是一回事。[MinIO mirror 的限制](https://docs.min.io/aistor/reference/cli/mc-mirror/)。

日常备份可利用不可变对象减少重复上传，但必须确保每次数据库快照引用的全部对象都已备份。对于可覆写对象，需要单独版本化或暂停写入，不把两个不同时间点的简单复制当作一致快照。

### 阶段三：需要在线迁移时，先改代码并演练

1. 先实施第 5 节的配置和客户端适配，在隔离数据集测试。方案 B 与 C 可以分两次发布，不必同时迁移数据库和对象。
2. 若选 RDS，创建 PostgreSQL 16 目标实例、独立业务数据库 `oj` 和必要账号，核对编码、排序规则、时区、插件、连接数与权限。不要恢复到系统数据库 `postgres`。
3. 优先同地域私网访问；无法使用私网时，使用支持 TLS 的实例及 `sslmode=verify-full`，下载 CA 并挂载给 API 和 Worker。公网白名单仅放行服务器固定出口 IP，不使用 `0.0.0.0/0`；同时测试连通、丢包和 SQL 往返延迟。
4. 先创建 OSS 私有在线 Bucket，配置 API 最小读写权限、Worker 只读权限，完成实际 SDK 验证。预复制对象，保留原对象键；需要恢复 Content-Type 等元数据，不能只依赖文件扩展名推断。
5. 对比源与目标对象数量、字节数和独立 SHA-256。当前数据量小，推荐逐对象完整验证，不用 multipart ETag 代替文件哈希。
6. 提前用逻辑备份在目标演练恢复，核对所有业务表、序列、约束、索引、迁移记录及对象引用。在测试环境压测考试请求、榜单、工单和并发判题，并测量实际 RTO。
7. 保持相同 JWT 等必要业务配置，不因迁移无意让所有账号失效；停用默认数据播种。恢复后的迁移过程单独受控执行，不能先启动空库上的自动播种应用。

RDS 恢复命令示例：在装有 PostgreSQL 16 客户端的受控环境执行，凭据预先放入权限为 `0600` 的专用 `PGPASSFILE`，不要把密码放进命令参数或提交到仓库。目标必须为提前核实的空业务库。

```bash
export PGHOST='REPLACE_WITH_RDS_ENDPOINT'
export PGPORT='5432'
export PGUSER='oj_migrator'
export PGDATABASE='oj'
export PGSSLMODE='verify-full'
export PGSSLROOTCERT='/secure/rds-ca.pem'
export PGPASSFILE='/secure/oj-rds.pgpass'

pg_restore --no-owner --no-privileges --exit-on-error \
  --dbname="$PGDATABASE" /path/to/verified/oj.dump
```

这些选项避免套用源数据库的对象所有者和授权，不等于自动配置目标运行账号权限；恢复后仍需明确表、序列及迁移权限。命令不含 `--clean`，不要把失败的半恢复目标当作可反复覆盖的生产数据库。[PostgreSQL 16 pg_restore](https://www.postgresql.org/docs/16/app-pgrestore.html)。

### 阶段四：正式切换、验证和回滚

1. 选择没有进行中考试或比赛的维护窗口，记录原提交、配置和镜像。先开启维护入口，阻止新的登录、提交、工单等业务写入。
2. 让 API Outbox 和 Worker 消化已有任务，检查未投递 Outbox、排队/运行提交、Redis 消费积压和待确认消息；仅看 Stream 长度不能判断队列是否空。不清空 Redis。
3. 任务与后台查重等写入全部收敛后，停止 API、Worker，保留 PostgreSQL、Redis、MinIO。此后导出最终数据库与对象清单，补齐最后对象差量，生成同一恢复点。
4. 恢复目标库并做完整核对；方案 B 不切数据库。将 API 与 Worker 同时改到经过验证的目标，不允许一端用旧库、另一端用新库。
5. 对恢复后的数据库受控执行迁移检查，保持 `SEED_DATA=false`；确认 Compose 最终配置已经使用新连接，但不要把含密码的配置展开结果写入日志。
6. 启动应用、检查健康状态，在维护状态下先完成只读检查；需要真实写入的验收使用事先批准的隔离测试资源。完成后才恢复公众入口，监控数据库连接、错误率、队列积压、对象失败和 OSS 流量。
7. 旧库、旧对象和回滚应用镜像建议保留 7～14 天。到期后再次确认恢复能力及业务引用，获得明确同意后仅删除对应项目资源；不能使用全局卷清理。

回滚边界：如果新目标尚未接收新的业务写入，可以恢复旧连接和旧应用镜像。若新库已经产生提交、工单或成绩等数据，不能直接切回旧库，否则会丢失新数据；必须重新冻结写入，将新增数据及对象同步回原环境或完成经过校验的反向恢复后再切回。

## 7. 功能验收清单

| 范围 | 必须验证 |
| --- | --- |
| 数据完整性 | 表行数、主外键、序列、唯一约束、迁移校验和、全部对象键/哈希及引用一致 |
| 用户与权限 | 登录、原有角色、出题资格、管理员权限、团队成员可见性不变 |
| 题库与工单 | 题面、图片、参考附件、最终题包上传、归档兼容和下载权限 |
| 判题 | 多语言提交、完整测试点读取、结果写回、队列重试与 SSE；历史重判使用原版本 |
| 课程与比赛 | OI/IOI/ACM 计分、封榜、排行榜、教师报表和人工评分 |
| 考试安全 | 入场锁、考试专属图片、跨模块 403、提交归属、行为日志和交卷解锁 |
| 提交详情 | 源代码、资源消耗、绑定版本测试点总数；OI/封榜期间统计不泄露 |
| 网络与成本 | SQL 延迟、连接池、断网恢复、OSS 分片上传、缓存命中、每次判题下载量与账单告警 |
| 恢复能力 | 下载验证、隔离恢复、旧版本题包可用，以及新目标已写入后的回滚演练 |

实施代码后运行前端测试与构建、API/Worker 测试、Compose 校验、应用镜像构建，并使用真实隔离后端完成 Playwright 验收。生产只读核验不等价于完整写入功能验收，不在生产业务数据上做破坏性测试。

## 8. 执行前需要锁定的选择

- 11 GiB MySQL 对应什么业务、是否仍需要在线运行。
- 当前服务器所在厂商、地域和固定出口 IP；与候选阿里云地域的实际网络质量。
- 目标是立刻释放多少空间，还是主要获得异地备份、托管与高可用。
- 月度费用上限、数据保留要求、可接受停机窗口与数据丢失窗口。

就当前 OJ 数据规模，推荐先执行“备份与空间归属核实”，不立即购买 RDS。只有需要托管在线数据库时，再完成准确规格报价和演练；若主要增长来自题包，则优先独立实施 MinIO → OSS。
