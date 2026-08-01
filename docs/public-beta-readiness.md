# 选科π开放公测 Readiness 矩阵

> 状态说明：`todo` 未开始，`doing` 正在实现或验证，`blocked` 依赖外部条件，`pass` 已有可复核证据。

## 发布闸门

| 闸门 | 要求 | 状态 | 证据 |
| --- | --- | --- | --- |
| P0-路由可达 | 所有公开路由、登录态路由和管理路由均有页面或 404，不出现空白页 | doing | `frontend/src/router/index.ts` 已有 catch-all 404；`frontend/e2e/not-found.spec.ts` 覆盖未知路由可恢复 UI 与 axe smoke；`frontend/src/lib/appError.ts`、`frontend/src/main.ts` 和 `frontend/src/App.vue` 增加全局错误边界，懒加载 chunk 失败或组件运行时错误会显示刷新/回首页兜底，不直接白屏；`appError.test.ts` 覆盖 chunk 错误识别；Playwright 配置禁用复用旧 preview server，避免本地/E2E 读到过期构建 |
| P0-生产配置 | 生产环境禁止默认密钥、明文管理员密码和任意 CORS | doing | 生产要求强密钥、管理员哈希、严格 CORS、可信代理、TLS SMTP、PostgreSQL 与 S3；`.env.production.example` 与 `deploy/preflight.sh` 拒绝 SQLite、本地存储、非 HTTPS S3/CDN、超过 20 的连接池和未声明 sslmode 的 URL；应用生产启动会执行 S3 `HeadBucket` 与 SMTP TLS/AUTH/NOOP，无真实凭证或依赖不可达时拒绝监听；`deploy/blue-green-deploy.sh` 会加载与 systemd 相同的共享/槽位环境文件，并在 migration 前执行新制品 preflight，随后完成新槽直连 readiness、原子 upstream 切换、失败恢复和旧槽排空；仍需在目标 Linux 主机用真实 secret/TLS 证书执行并保存证据 |
| P0-数据库运行时 | 生产使用托管 PostgreSQL 16，连接池/超时生效且业务读写可用 | doing | pgx 最大连接数 ≤20并配置连接/查询/健康超时；生产 repository 不回退 SQLite；数据库工厂和 `/readyz` 精确要求 goose version 1、已应用状态与核心表齐全，空库、落后/超前、回滚和缺表均拒绝；`deploy/migrate-production.sh` 分离执行 DDL 并复核版本；真实 PostgreSQL 16 HTTP 旅程覆盖用户、互动、消息、管理审核及强类型真实数据接口；仍需托管实例最终切换、生产慢查询观察与 30 分钟负载验收 |
| P0-认证安全 | 登录使用可撤销服务端会话 Cookie，CSRF 防护启用，不向前端返回 JWT | doing | 用户与管理员均使用独立 HttpOnly 会话 Cookie 和 CSRF Cookie；`AuthSession.Token` 明确禁止 JSON 序列化，OpenAPI/生成类型不含 token，注册与登录 HTTP 测试锁定 Cookie-only 响应；退出、重置密码、会话撤销、注销账号及草稿恢复已实现；管理员写请求即使携带任意 Authorization 头也不能绕过 admin CSRF；仍需目标域名真实 Secure Cookie/SMTP 验收 |
| P0-数据来源 | 广东政策、专业要求和来源可追溯；广东以外不展示模拟结构化结论 | doing | PostgreSQL 的四个真实数据接口直接读取强类型表并返回来源 UUID、assetKey、SHA-256 与结构化 `requiredSubjects`；前端政策库只展示 API 已复核来源，已删除静态政策正文生成器，专业要求不再从摘要/标签猜测科目或约束类型，趋势标题/年份/单位来自 API 元数据；广东 SQL 闸门拒绝不完整数据；仍需正式数据导入托管实例并通过闸门 |
| P0-上传安全 | 图片直传对象存储并校验归属、大小、MIME、扩展名、像素尺寸和对象存在性 | doing | 生产 S3 adapter 提供 presigned PUT、Head/Open/Delete、CDN URL，并在启动时用有限超时执行 `HeadBucket`；complete 已实现幂等重试且每次仍复核归属与对象元数据，SQLite/PostgreSQL 均处理并发完成竞争；`cleanup-uploads` 已按生产数据库/对象存储驱动清理过期 pending 对象，systemd timer 每 15 分钟执行且发布包缺少清理二进制时 fail-fast；本地链路、S3 Delete 请求、对象校验和多图部分失败重试已有测试；仍需真实 S3 执行 PUT→complete→CDN GET、过期对象删除，验证 CORS、版本控制与供应商生命周期规则 |
| P0-管理后台 | 管理员独立会话、RBAC、审核/隐藏/封禁/恢复和审计日志可用 | doing | 后端固定 `super_admin/content_editor/moderator` 三角色及逐路由权限，未知角色默认拒绝；会话保存 email/role/permissions，登录契约明确返回权限且不返回 token；审计日志记录真实 email 与角色；前端按权限隐藏模块和高危按钮，旧无权限缓存会话要求重新登录；表驱动 middleware 与 content-editor HTTP 测试覆盖 allow/403；仍需真实后台账号人工回归三角色 |
| P1-契约 | OpenAPI 3.1 覆盖 `/api/v1`，前端类型由契约生成 | doing | `backend/internal/http/handler/response.go` 已为成功/错误响应加入 `requestId`；`docs/openapi/openapi.yaml` 已落当前接口骨架并补充 `/admin/logout`；后台图片上传响应已从泛型对象收紧为 `AdminImageUploadEnvelope`，包含 `url/contentType/size/width/height/name`；`backend/internal/http/openapi_contract_test.go` 覆盖路由契约、受保护写接口安全响应与 requestId；`go -C backend run ./cmd/openapi-types` 生成前端契约类型；`pnpm openapi:check` 检查生成产物未过期 |
| P1-可观测性 | JSON 脱敏日志、requestId、指标、健康面板和告警接入 | doing | 生产 JSON 脱敏日志、requestId、数据库/schema readiness、受 token 保护的 Prometheus HTTP 指标和可选 OTLP/HTTP traces 已实现；trace 仅含路由模板、方法、状态和 requestId，支持 TLS/私有 CA，未配置时禁用；浏览器 `web-vitals` 匿名上报并聚合 histogram；仍需在目标环境接入 Collector/Prometheus/Grafana 并演练告警路由 |
| P1-备份恢复 | PostgreSQL PITR、对象版本控制、恢复演练满足 RPO/RTO | blocked | PostgreSQL 16 goose baseline 已在两个全新 Docker PostgreSQL 16 实例完成连续两次 up → SQLite 实际写入 → 逐表目标哈希/行数/外键验证 → down；当前快照每次迁移 20 张表、373 条记录，`foreignKeysValid=true`、全部 `verified=true`，两次稳定内容完全一致，manifest 保存在 `docs/evidence/postgres-migration-2026-07-31/`；仍未完成托管 PostgreSQL PITR、对象版本恢复和 RPO/RTO 计时演练 |
| P1-自动化验收 | CI 覆盖后端、前端、E2E、安全扫描和 SBOM | doing | CI 覆盖 Go test/vet/race、真实 PostgreSQL repository/HTTP/migration 测试、OpenAPI、前端依赖 audit/lint/30 单测/build、bundle budget、40 条浏览器 E2E、gosec、govulncheck、SPDX SBOM 和 k6 smoke；workflow、发布 shell 与观测资产执行 actionlint/shellcheck/promtool/jq。CI 在 PostgreSQL 16 服务中运行完整 `internal/...` 覆盖率并硬性要求整体 ≥70%，同时归档 coverprofile。全新 PostgreSQL 16 本地证据为整体 80.7%（HTTP 84.9%、handler 80.1%、middleware 88.4%、config 87.0%、observability 90.2%、repository factory 100%、PostgreSQL 80.3%、SQLite 80.5%、service 81.7%、storage 80.3%、logx 85.7%）；整体与关键包目标均达到，仍需远端 CI 绿灯及 artifact/SBOM 归档证据 |
| P1-性能压测 | k6 按 50 并发、15 RPS 压测 30 分钟，读取 p95 <300ms、写入 p95 <500ms、5xx <0.1% | doing | 正式场景使用恒定 50 VU、错峰启动和每 VU 3.33 秒 pacing；本地 PostgreSQL 16 新模型 2 分钟预演达到 `vus_max=50`，1,800 请求、14.60 RPS、0 失败、读取 p95 2.73ms，证据为 `docs/evidence/k6-postgres-2026-07-31/readiness-50vu-2m.txt`；仍需同规格预生产完整 30 分钟、写入负载与 CPU/内存报告 |
| P1-公测主链路体验 | 注册、发布、互动等高频路径有明确 loading/disabled/error 反馈，不因重复点击造成卡死或计数抖动 | doing | 注册默认省份改为广东；登录/注册 submit 增加函数级防重复；验证码、限流、网络失败和注册字段错误改为识别标准 API 错误；发布弹窗上传/发布中禁用关闭、上传和提交按钮，多图部分失败保留成功项并支持失败项重试；会话在发布时失效只显示认证弹窗，重新登录后恢复原草稿，`frontend/e2e/session-expiry-draft.spec.ts` 覆盖；首页帖子状态不再被话题/观察侧栏接口故障覆盖，“换一批”执行真实 refetch；全局搜索为 requirements/provinces 分别显示 loading、局部错误和重试；用户主页、关注、通知、政策、私信和设置均已有真实空态、失败、离线及重试反馈；仍需 Cookie-only 会话恢复、跨标签页同步和长列表翻页体验继续优化 |

## 页面矩阵

| 路由 | 页面 | 角色 | 桌面 | 移动 | loading | empty | error | offline | 权限不足 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/` | 首页 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`home-feed-state.spec.ts` 覆盖帖子列表失败、重试恢复和浏览器离线提示 |
| `/posts/:id` | 帖子详情 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；作者编辑和删除自己帖子已由 `public-beta-journey.spec.ts` 覆盖 |
| `/users/:name` | 用户主页 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`user-profile-state.spec.ts` 覆盖失败态与重试恢复，`responsive-layout.spec.ts` 使用完整画像契约覆盖六档宽度 |
| `/topics` | 话题列表 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`frontend/e2e/topics-policy-document.spec.ts` 覆盖真实话题跳转、加载/无数据/错误态，`advice-observation-state.spec.ts` 覆盖浏览器离线提示 |
| `/topics/:slug` | 话题详情 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`frontend/e2e/topics-policy-document.spec.ts` 覆盖真实讨论、无讨论和请求失败可重试，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/insights` | 观察列表 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`frontend/e2e/insights.spec.ts` 覆盖已复核数据和空态，`insights-offline-state.spec.ts` 覆盖接口失败与断网提示，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/insights/:id` | 观察详情 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`frontend/e2e/insights.spec.ts` 覆盖详情来源和相关帖子，`insights-offline-state.spec.ts` 覆盖失败重试与断网提示，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/advice` | 建议列表 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`advice-observation-state.spec.ts` 覆盖真实帖子、筛选无匹配、taxonomy 失败和浏览器离线提示，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/advice/:id` | 建议详情 | visitor/user | pass | pass | n/a | n/a | pass | n/a | n/a | doing；旧链接已由 `advice-observation-state.spec.ts` 验证重定向到统一帖子详情，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/following` | 关注 | user | pass | pass | pass | pass | pass | pass | pass | doing；`frontend/e2e/following-notifications-state.spec.ts` 覆盖筛选无匹配空态，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/settings` | 设置 | user | pass | pass | pass | n/a | pass | pass | pass | doing；`settings-account-security.spec.ts` 覆盖撤销其它设备会话，真实后端 E2E 覆盖会话展示和账号注销，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/messages` | 私信 | user | pass | pass | pass | pass | pass | pass | pass | doing；会话与消息均消费 cursor/hasMore，会话底部加载更多、线程顶部加载更早消息并保持滚动位置 |
| `/notifications` | 通知 | user | pass | pass | pass | pass | pass | pass | pass | doing；桌面/移动结构共用游标分页，支持加载更多、局部失败重试与去重；组件测试覆盖跨页和失败保留 |
| `/observe` | 观察站 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`advice-observation-state.spec.ts` 覆盖真实观察数据和失败重试，`insights-offline-state.spec.ts` 覆盖断网提示，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/requirements` | 选科要求 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`frontend/e2e/requirements-forum.spec.ts` 覆盖已复核数据和筛选空态，`real-data-offline.spec.ts` 覆盖断网提示与重试 |
| `/requirements/:major` | 专业要求详情 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`frontend/e2e/requirements-forum.spec.ts` 覆盖临床医学讨论论坛跳转，`real-data-offline.spec.ts` 覆盖官方要求离线降级但保留社区讨论，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/knowledge` | 政策资料 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`knowledge-state.spec.ts` 覆盖失败态与重试恢复，`knowledge-verified-state.spec.ts` 覆盖已复核数据，`real-data-offline.spec.ts` 覆盖断网提示 |
| `/knowledge/:province` | 省份资料 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`knowledge-state.spec.ts` 覆盖 API 失败不显示“没有找到该省份”，`knowledge-verified-state.spec.ts` 覆盖广东文件，`real-data-offline.spec.ts` 覆盖断网提示 |
| `/knowledge/:province/docs/:documentId` | 政策文件 | visitor/user | pass | pass | pass | pass | pass | pass | n/a | doing；`topics-policy-document.spec.ts` 覆盖已复核记录、来源、文件哈希和官方入口，`insights-offline-state.spec.ts` 覆盖断网提示，`responsive-layout.spec.ts` 覆盖六档宽度 |
| `/admin` | 管理后台 | admin | pass | pass | pass | doing | pass | todo | doing | doing |
| `/:pathMatch(.*)*` | 404 | visitor/user/admin | pass | pass | n/a | n/a | n/a | n/a | n/a | doing；`frontend/e2e/not-found.spec.ts` 覆盖桌面和移动端恢复状态 |

## API 矩阵

| 接口 | 方法 | 角色 | 成功响应 | 错误响应 | 分页 | 权限 | 状态 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `/api/v1/auth/register` | POST | visitor | `{ data, meta? }` + 服务端会话 Cookie | `{ error }` | n/a | n/a | doing |
| `/api/v1/auth/login` | POST | visitor | `{ data, meta? }` + 服务端会话 Cookie | `{ error }` | n/a | n/a | doing |
| `/api/v1/auth/logout` | POST | user | `{ data, meta? }` + 清除 Cookie | `{ error }` | n/a | user | doing |
| `/api/v1/auth/forgot-password` | POST | visitor | `{ data, meta? }` | `{ error }` | n/a | n/a | doing；`frontend/e2e/public-beta-journey.spec.ts` 覆盖登录弹窗找回密码验证码请求 |
| `/api/v1/auth/reset-password` | POST | visitor | `{ data, meta? }` | `{ error }` | n/a | n/a | doing；`backend/internal/service/forum_session_test.go` 覆盖重置后撤销旧会话，`frontend/e2e/public-beta-journey.spec.ts` 覆盖前端提交新密码 |
| `/api/v1/me` | GET | user | `{ data, meta? }` | `{ error }` | n/a | user | doing |
| `/api/v1/me` | DELETE | user | `{ data, meta? }` + 清除 Cookie | `{ error }` | n/a | user | doing |
| `/api/v1/me/sessions` | GET | user | `{ data, meta? }` | `{ error }` | n/a | user | doing |
| `/api/v1/me/sessions/:id` | DELETE | user | `{ data, meta? }` + 当前会话清除 Cookie | `{ error }` | n/a | user | doing |
| `/api/v1/posts` | GET | visitor/user | `{ data, meta }` | `{ error }` | cursor（latest/recommended/hot） | n/a | pass；三种排序均使用稳定 cursor，HTTP smoke、前端分页测试与 OpenAPI 契约通过；SQLite `LIKE` 仅保留为开发/迁移回退 |
| `/api/v1/posts` | POST | user | `{ data, meta? }` | `{ error }` | n/a | user | doing；HTTP smoke 覆盖真实会话、CSRF 与创建成功 |
| `/api/v1/posts/:id` | GET | visitor/user | `{ data, meta? }` | `{ error }` | n/a | n/a | doing；已有实现及隐藏后 404 smoke，仍缺独立匿名成功契约断言 |
| `/api/v1/posts/:id` | PUT | user/owner | `{ data, meta? }` | `{ error }` | n/a | user + owner | doing；作者可编辑标题、正文、标签、方向、选科和分类，非作者/不存在返回 404 |
| `/api/v1/posts/:id` | DELETE | user/owner | `{ data, meta? }` | `{ error }` | n/a | user + owner | doing；作者软删除自己的帖子，非作者/不存在返回 404；同 E2E 和 repository 测试覆盖 |
| `/api/v1/posts/:id/comments` | POST | user | `{ data, meta? }` | `{ error }` | n/a | user | doing；HTTP smoke 覆盖真实评论成功 |
| `/api/v1/notifications` | GET | user | `{ data, meta }` | `{ error }` | cursor | user | pass；repository 覆盖连续两页无重复，HTTP smoke 覆盖成功 envelope 与匿名 401 标准错误 envelope |
| `/api/v1/messages` | GET | user | `{ data, meta }` | `{ error }` | cursor | user | pass；稳定 `(created_at,id)` 游标、跨页仓储、HTTP envelope、OpenAPI 与真实 PostgreSQL HTTP 旅程通过，前端已消费 meta |
| `/api/v1/messages/:peer` | GET | user | `{ data, meta }` | `{ error }` | cursor | user | pass；repository 覆盖连续两页，HTTP smoke 校验分页 envelope |
| `/api/v1/provinces` | GET | visitor/user | `{ data, meta? }` | `{ error }` | n/a | n/a | pass；`real_data_test.go` 验证广东已复核覆盖状态 |
| `/api/v1/policies` | GET | visitor/user | `{ data, meta? }` | `{ error }` | 未实现；当前返回全部已发布记录 | n/a | doing；真实数据与来源元数据已有 HTTP 测试，仍需游标分页 |
| `/api/v1/requirements` | GET | visitor/user | `{ data, meta? }` | `{ error }` | 未实现；当前返回全部已发布记录 | n/a | doing；方法说明已有 HTTP 测试，仍需游标分页 |
| `/api/v1/sources/:id` | GET | visitor/user | `{ data, meta? }` | `{ error }` | n/a | n/a | pass；`real_data_test.go` 验证来源 URL 与 coverageStatus |
| `/api/v1/uploads/images/presign` | POST | user | `{ data, meta? }` | `{ error }` | n/a | user | doing；本地与生产 S3 adapter 已实现，待真实供应商 CORS/PUT 验收 |
| `/api/v1/uploads/images/:id/complete` | POST | user | `{ data, meta? }` | `{ error }` | n/a | user | doing；已有 MIME、大小、扩展名、尺寸和对象存在性测试，待真实 S3/CDN 验收 |
| `/api/v1/admin/*` | mixed | admin | `{ data, meta? }` | `{ error }` | 部分固定上限，未全面游标化 | admin/RBAC | doing；三角色逐路由 RBAC、审核、隐藏/恢复、封禁与真实 actor 审计已有测试；后台长列表分页仍待补齐 |

## 验收证据规范

## 最终差异审计（2026-08-01）

本节按当前仓库的 `frontend/src/router/index.ts`、`backend/internal/http/server.go` 和 `docs/openapi/openapi.yaml` 逐项比对。未列为 `pass` 的项目不得被默认视为已验收；外部托管服务相关项目仍需在目标环境补证据。

### 页面矩阵缺口

| 路由 | 缺失矩阵项 | 发布前动作 |
| --- | --- | --- |
| `/admin` | `visitor/user` 访问后的登录/401、三种管理员角色的可见模块、offline、403、空数据 | 使用 `super_admin`、`content_editor`、`moderator` 各一真实账号完成浏览器验收；补后台断网后局部错误与重试证据 |
| `/settings` | 注销账号、密码重置、当前会话撤销后的失效态尚未作为独立页面验收项登记 | 在 `settings-account-security.spec.ts` 中保留成功、失败、过期会话三类证据，并登记到发布记录 |
| `/posts/:id` | 匿名成功、隐藏/删除后 404、评论提交失败后的局部保留状态未独立登记 | 补匿名、资源不存在/已隐藏、评论接口失败三项浏览器证据 |
| `/knowledge/:province/docs/:documentId` | 私有原始文件不可直接暴露、来源链接失效时的错误态未登记 | 用真实 `assetKey` 验证 CDN/原始文件访问边界，并记录失败重试结果 |

### API 矩阵缺口

API 矩阵目前将若干不同权限和错误面的接口折叠在一行，以下接口必须拆成独立验收项：

| 接口 | 角色/状态维度 |
| --- | --- |
| `/api/v1/auth/email-verification-code` | visitor；字段校验、限流、邮件服务不可用 |
| `/api/v1/profiles/:name` | visitor/user；不存在、被封禁用户、匿名与已登录视图 |
| `/api/v1/taxonomy`、`/api/v1/content` | visitor/user；空数据、服务端错误、缓存/降级标记 |
| `/api/v1/telemetry/web-vitals` | same-origin visitor/user；跨源拒绝、过大请求、限流且不得泄露敏感字段 |
| `/api/v1/posts/:id/report` | user；重复举报、目标不存在/已删除、CSRF、频率限制 |
| `/api/v1/posts/:id/like`、`/favorite`、`/authors/:name/follow` | user；幂等/反向切换、目标不存在、CSRF、限流 |
| `/api/v1/ai/choice-advice` | user；输入字段限制、超时、上游错误、降级结果、限流和个人信息过滤 |
| `/api/v1/admin/email-config`、`/email-test` | `system.email.read/test`；三角色 403、SMTP 不可用 |
| `/api/v1/admin/content*`、`/content-summary` | `content.*`/`dashboard.read`；三角色 403、空列表、分页上限、写入校验 |
| `/api/v1/admin/uploads/images` | `media.upload`；MIME/大小/对象写入失败、CSRF |
| `/api/v1/admin/audit-logs` | `audit.read`；敏感字段脱敏、空列表、访问拒绝 |
| `/api/v1/admin/reports*` | `moderation.read/act`；空队列、重复处理、目标已恢复/隐藏、审计记录 |
| `/api/v1/admin/users/:id/{ban,restore,password}` | `users.ban/password_reset`；重复操作、目标不存在、管理员自操作保护、审计记录 |

### 架构/契约风险（当前代码仍存在）

- 帖子 latest/recommended/hot 已全部切换为 cursor，HTTP handler、前端调用和 OpenAPI 均不再接受 `offset`；上线仍需在真实 PostgreSQL 数据量下保存 `EXPLAIN ANALYZE` 证据。
- `backend/internal/repository/sqlite/forum_repository.go:127` 仍使用无全文索引的组合 `LIKE` 搜索；SQLite 仅应作为迁移工具/开发回退，生产路径切换前必须确认不会被启用。
- `backend/internal/http/handler/admin.go:522`、`:530` 的政策和专业要求接口当前无 cursor 分页；大数据量上线前需补分页或明确数据量上限并完成 `EXPLAIN ANALYZE` 证据。
- `/admin` 在 Vue router 中没有 `requiresAuth`，权限实际由页面/后台 API 处理；应补未登录直接访问和登录后刷新两条 E2E，避免出现短暂可见或空白状态。

上述项目是“矩阵缺口/验收证据缺口”，不是本次只读审计中擅自修改核心业务的范围。外部 PostgreSQL、S3、SMTP、PITR、30 分钟压测、真实 Linux 蓝绿发布和人工无障碍检查仍以发布闸门中的 `doing/blocked` 为准。

- 每个页面至少保存 375、390、768、1024、1280、1440px 的截图或 Playwright trace。
- 每个 API 至少有契约测试、成功样例、认证失败样例、权限不足样例和字段校验失败样例。
- 每个数据源记录采集时间、来源 URL、文件哈希、适用年份、适用范围、方法说明和 `coverageStatus`。
- 每次发布前保存 `go test ./...`、`go vet ./...`、前端 build、E2E、axe、k6、安全扫描和 SBOM 结果。
- SQLite -> PostgreSQL 迁移演练必须保存 `cmd/sqlite-to-postgres` 生成的表行数与 SHA-256 manifest，并与迁移前 SQLite 快照、迁移后 PostgreSQL 抽样数据一起归档。非 dry-run 现在会在提交事务前重新读取目标表，逐表比较 `targetRows/targetSha256`、设置 `verified`，并拒绝存在未验证外键约束的目标库；任一差异会回滚整个迁移事务。

本地迁移证据：`backend/data/soulcourse.db` 已先完成 dry-run，随后在两个全新 PostgreSQL 16 容器连续完成两次实际写入演练；每次 20 张迁移表、373 条记录，目标行数/哈希全部一致且外键有效。PostgreSQL repository、完整 HTTP 主链路和强类型真实数据接口也已在 PostgreSQL 16 通过；这些证据仍不替代托管实例最终导入、PITR 与对象恢复演练。

## 性能验收命令

CI 仅执行 smoke，正式公测前在预生产或生产同规格环境运行：

只读 smoke：

```bash
K6_SMOKE=1 \
BASE_URL=http://127.0.0.1:1309 \
k6 run scripts/k6/public-beta-readiness.js
```

隔离环境写入 smoke：

```bash
cd backend
APP_ENV=local go run .

K6_SMOKE=1 \
K6_WRITE_SMOKE=1 \
BASE_URL=http://127.0.0.1:1309 \
k6 run ../scripts/k6/public-beta-readiness.js
```

正式 30 分钟读取压测：

```bash
BASE_URL=https://your-public-beta-origin.example \
PUBLIC_BETA_TEST_DURATION=30m \
k6 run scripts/k6/public-beta-readiness.js
```

目标阈值来自脚本内置配置：15 RPS、最多 50 VU、`http_req_failed <0.1%`、读取 p95 `<300ms`。`K6_WRITE_SMOKE=1` 只用于本地/预发隔离环境的一次性写入链路验证，因为它依赖本地/开发环境返回的 `debugCode` 并会产生测试用户和测试内容；正式写入负载仍需准备专用测试账号、数据清理和独立限流策略后再开启。
