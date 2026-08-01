# 选科π公测发布与恢复 Runbook

> 适用范围：广东先行公测，单实例 Linux + Nginx 蓝绿槽位，托管 PostgreSQL 16 与 S3 兼容对象存储。
> 本文只允许在发布负责人、数据库负责人和当班运维三方确认后执行。所有时间均使用 `Asia/Shanghai`，证据目录建议为 `docs/evidence/release-YYYY-MM-DD-<release-id>/`。

## 0. 发布记录

| 字段 | 值 |
| --- | --- |
| release id / git SHA | 填写 |
| 发布负责人 / 审批人 / 值班人 | 填写 |
| 目标域名 / 当前活动槽 | 填写 |
| 预生产压测证据链接 | 填写 |
| PostgreSQL 备份/PITR 证据链接 | 填写 |
| S3 版本控制/恢复证据链接 | 填写 |
| SMTP 实投递证据链接 | 填写 |
| 告警、日志、Grafana 链接 | 填写 |
| 发布开始 / 结束时间 | 填写 |

## 1. 发布前阻断检查

发布负责人逐项勾选；任何一项为 `否`、`未知` 或证据为空，都不得进入公测流量。

- [ ] 变更已冻结，OpenAPI 与数据库 migration 无未审批变更；记录 SHA：`________`。
- [ ] 远端 CI 全绿：Go test/vet/race、前端 lint/test/build/E2E、契约、gosec、govulncheck、audit、SBOM；链接：`________`。
- [ ] 生产构建目录包含二进制、前端 `index.html` 和 `deploy/`；SHA-256 manifest：`________`。
- [ ] `/etc/soulcourse/soulcourse.env`、`slot-blue.env`、`slot-green.env` 为 root 拥有且 `0600`；未出现 `ADMIN_PASSWORD` 明文或浏览器 `ADMIN_TOKEN`；检查输出：`________`。
- [ ] `APP_ENV=production`、`DATABASE_DRIVER=postgres`、显式 `sslmode`、连接池 `1..20`、严格 CORS、可信代理、HTTPS S3/CDN、TLS/STARTTLS SMTP 均已核对；蓝绿脚本已加载与 systemd 相同的共享/槽位环境文件并通过 preflight，输出：`________`。
- [ ] 运行时数据库账号无 DDL/migration 权限；迁移账号只在本次发布临时使用，权限回收记录：`________`。
- [ ] PostgreSQL 最近一次备份成功，PITR 可恢复时间点覆盖发布前至少 15 分钟；备份时间/LSN/恢复点：`________`。
- [ ] S3 已启用版本控制和生命周期；最近一次对象清单及版本清单：`________`。
- [ ] 本次发布前已完成恢复演练，实测 `RPO <= 15m`、`RTO <= 2h`；演练开始/恢复可读/恢复写入时间：`________ / ________ / ________`。
- [ ] 受保护的 `/metrics`、`/readyz`、日志、告警路由和 OTLP collector 已有人实时查看；值班确认：`________`。
- [ ] 已建立回滚指挥群，明确数据库负责人和最终回滚决策人；联系方式：`________`。

## 2. 备份与恢复演练

### 2.1 PostgreSQL PITR 验收

1. 记录事故/演练起点 `T0`、目标恢复点和当前数据库 provider 的备份作业 ID。
2. 在隔离恢复实例执行 provider 的 PITR，恢复到 `T0` 前不晚于 15 分钟的时间点；不得直接覆盖生产库。
3. 运行 goose 版本检查、核心表检查和广东数据校验：

```bash
GOOSE_BIN=goose \
DATABASE_URL="$RECOVERY_DATABASE_URL" \
APP_ENV=production \
GUANGDONG_DATA_YEAR="$GUANGDONG_DATA_YEAR" \
deploy/preflight.sh

psql "$RECOVERY_DATABASE_URL" -X -v ON_ERROR_STOP=1 \
  -v target_year="$GUANGDONG_DATA_YEAR" \
  -f scripts/postgres/validate-guangdong-production.sql
```

4. 用专用恢复验证账号执行匿名读取、登录/会话、帖子/评论、消息、政策/要求/来源和管理审计抽样。保存命令输出、行数、抽样 ID 和 SHA-256 manifest。
5. 记录 `Trestore-ready - T0` 为 RPO；记录从恢复开始到应用可读写为 RTO。若任一超过阈值，闸门为 `FAIL`。

| 证据字段 | 值 |
| --- | --- |
| provider / backup job / source snapshot | 填写 |
| T0 / provider recovery point / 恢复实例 | 填写 |
| 恢复开始 / readyz 通过 / 业务读写通过 | 填写 |
| RPO / RTO（分钟） | 填写 |
| 行数、外键、广东数据、抽样与 manifest 链接 | 填写 |
| 失败项、修复人、复测时间 | 填写 |

### 2.2 S3 对象恢复验收

抽取一张公开图片和一份私有原始政策文件，记录 `assetKey`、版本 ID、内容类型、大小和 SHA-256。删除/模拟丢失后，从旧版本恢复到隔离前缀，验证 CDN GET、私有授权下载、哈希一致性和数据库 metadata 关联；记录生命周期清理未完成 multipart upload 的结果。

| 证据字段 | 值 |
| --- | --- |
| bucket / region / versioning / lifecycle 配置截图或输出 | 填写 |
| public assetKey + versionId + SHA-256 + CDN GET | 填写 |
| private assetKey + versionId + SHA-256 + 授权 GET | 填写 |
| 恢复开始 / 首次可读 / RTO（分钟） | 填写 |
| CORS、Head、presigned PUT、complete 验证链接 | 填写 |

## 3. 发布与灰度

### 3.1 发布新槽

在目标 Linux 主机执行，保留完整 stdout/stderr：

```bash
sudo --preserve-env=MIGRATION_DATABASE_URL \
  deploy/blue-green-deploy.sh /opt/soulcourse/releases/<release-id> \
  2>&1 | tee /var/log/soulcourse/release-<release-id>.log
```

运行时配置必须放在 root 所有、权限 `0600` 的 `/etc/soulcourse/soulcourse.env` 和 `/etc/soulcourse/slot-*.env`，其中 `SOULCOURSE_PUBLIC_HEALTH_URL` 必须指向真实域名的 HTTPS `/readyz`。脚本会像 systemd 一样先加载共享文件、再加载目标槽位文件；`MIGRATION_DATABASE_URL` 仅随本次发布命令注入，不写入长期环境文件。切流后会经公网 HTTPS 再验一次 readiness，失败时恢复旧 upstream。

确认新槽直连 `http://127.0.0.1:1309/readyz` 或 `13010` 通过、Nginx `-t` 通过、活动 upstream 注释已更新、旧槽仍保留。记录活动槽、端口、服务状态、Nginx reload 输出和外部 HTTPS smoke：`________`。

> migration 失败时不得手工执行 down 作为常规回滚。先保留旧应用槽；若已发生不可逆数据变化，转入数据库负责人主持的 PITR/修复流程。

### 3.2 10% -> 30% -> 100% 观察门

当前部署脚本是整槽切换，不提供百分比流量路由；因此 10/30/100% 必须由 Nginx/托管 LB/边缘平台实际配置并保存配置版本。若没有可验证的百分比路由能力，只能执行“内部账号 -> 全量切换”，不能声称完成灰度。

每一档至少观察 24 小时。每 15 分钟记录一次仪表盘快照，档位结束时由发布负责人签字。

| 档位 | 流量/账号规则 | 必须观察 | 通过阈值 | 证据 | 决策/时间 |
| --- | --- | --- | --- | --- | --- |
| 内部 | 内部账号 allowlist | 5xx、登录/邮件、上传、核心链路、告警 | 无 P0；Critical/High=0 | dashboard/日志/截图：`___` | `___` |
| 10% | 10% 广东真实用户或明确 LB 权重 | 5xx、p95、注册转化、邮件成功、举报、CPU/内存、DB 连接 | 5xx <0.1%；读取 p95 <300ms；写入 p95 <500ms；CPU/内存 <70% | k6 + metrics + funnel：`___` | `___` |
| 30% | 30% 权重 | 同上，另查慢查询、对象失败、错误聚类 | 同上；无新增 P0/P1 | `___` | `___` |
| 100% | 100% 权重 | 同上，保留旧槽和回滚窗口 | 连续观察满 24h，无回滚条件 | `___` | `___` |

## 4. 回滚决策与执行

### 4.1 立即回滚条件

- P0、安全事件、数据损坏、登录/注册/发帖主链路不可用。
- 5xx `>=0.1%`、读取 p95 `>=300ms`、写入 p95 `>=500ms` 持续 10 分钟，或 CPU/内存 `>=70%` 持续 15 分钟。
- 数据库连接池持续 `>=70%`、对象存储上传/读取失败激增、邮件投递失败影响注册/重置。
- 任意越权、CSRF、会话撤销失效或敏感数据泄漏。

### 4.2 应用回滚

停止放量，记录当前指标和事件时间。使用上一版本目录再次执行蓝绿脚本，或将 Nginx upstream 原子恢复到旧槽后 `nginx -t && systemctl reload nginx`；确认旧槽 `/readyz`、外部 HTTPS smoke、登录和读写主链路，随后 drain 新槽。证据：`________`。

### 4.3 数据库/对象回滚

应用回滚不能撤销已执行 migration 或已写入数据。冻结写入，通知数据库负责人，选择已验证的 PITR 时间点或对象旧版本恢复；在隔离实例完成校验后再切换连接。记录 RPO/RTO、恢复点、manifest、审批和最终数据校验。未经审批不得删除生产库、执行破坏性 migration 或覆盖对象版本。

| 回滚字段 | 值 |
| --- | --- |
| 触发条件 / 决策人 / 事件 ID | 填写 |
| 旧 SHA / 新 SHA / 活动槽 / upstream 文件 | 填写 |
| 开始 / 流量恢复 / 业务验证完成 | 填写 |
| 数据库是否已写入 / PITR 或对象版本恢复点 | 填写 |
| 用户影响、告警、后续修复负责人和截止时间 | 填写 |

## 5. 发布后收尾

- [ ] 10/30/100% 所有档位观察证据已归档，负责人签字。
- [ ] 旧 release、旧槽和 upstream 备份保留至观察窗口结束；之后按保留策略清理。
- [ ] 发布后检查备份、慢查询、对象失败、注册漏斗、邮件成功率、举报和错误聚类。
- [ ] 事件/回滚均已形成时间线、影响范围、根因、修复项和截止日期。
- [ ] 关闭临时迁移凭证、测试账号、allowlist 和压测数据；核对 secret 未进入日志或工件。

## 6. 仍依赖托管平台或凭证的闸门

以下不是本地代码或本地 Docker 证据可以替代的条件；缺少真实配置时状态必须保持 `BLOCKED`：

| 闸门 | 外部依赖 | 必须取得的证据 |
| --- | --- | --- |
| PostgreSQL 生产切换 | 托管 PostgreSQL 16、migration/runtime 两套账号、TLS/网络白名单 | migration 输出、版本/广东数据校验、最低权限证明、PITR 恢复计时 |
| PITR RPO/RTO | provider 7 天 PITR、30 天备份、恢复实例权限与费用配额 | 真实恢复点、RPO/RTO、业务抽样和 manifest |
| S3/CDN | S3 endpoint、bucket、region、access key/secret、CDN、CORS、版本控制、生命周期 | presigned PUT、complete、公开 CDN GET、私有 GET、版本恢复和清理输出 |
| SMTP | SMTP host、TLS、账号/密码、发件域名 DNS/SPF/DKIM/DMARC | 注册和重置邮件真实到达、provider message ID、脱敏日志 |
| 灰度流量 | Nginx/LB/边缘平台的权重或 allowlist 能力 | 10/30/100% 配置版本、实际请求比例、每档 24h 指标 |
| 监控告警 | Prometheus、node_exporter、Grafana、OTLP collector、通知渠道凭证 | 指标接收、告警触发、通知到达、受保护面板访问 |
| 生产发布 | Linux sudo、systemd、Nginx TLS 证书和 DNS | `/readyz`、`nginx -t`、reload、外部 HTTPS smoke、回滚演练 |
| 远端质量闸门 | GitHub Actions secrets、环境审批、SBOM artifact 存储 | workflow green、SBOM 下载链接、审批记录 |

## 7. 阻断结论

在第 6 节证据未齐之前，只能称为“代码与本地演练通过”，不能称为“满足 RPO<=15m/RTO<=2h”或“完成 10%->30%->100% 灰度”。发布负责人应把缺失证据登记为 `blocked`，明确负责人、外部平台、凭证持有人和最晚完成时间。
