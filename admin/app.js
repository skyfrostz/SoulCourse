const SETTINGS_KEY = 'scf_admin_settings'
const SESSION_KEY = 'scf_admin_session'

const gradeRegions = ['华北', '东北', '华东', '华中', '华南', '西南', '西北', '港澳台']
const provinces = [
  ['北京', '华北', '北京教育考试院', '3+3', '已实施新高考，关注等级考与院校专业组选考要求。', 'https://www.bjeea.cn/'],
  ['天津', '华北', '天津市教育招生考试院', '3+3', '已实施新高考，选科需结合专业组选考要求。', 'https://www.zhaokao.net/'],
  ['河北', '华北', '河北省教育考试院', '3+1+2', '已实施“3+1+2”，首选物理/历史影响专业组范围。', 'https://www.hebeea.edu.cn/'],
  ['山西', '华北', '山西省招生考试管理中心', '3+1+2', '第五批改革省份，2025年起新高考落地。', 'https://www.sxkszx.cn/'],
  ['内蒙古', '华北', '内蒙古自治区教育招生考试中心', '3+1+2', '第五批改革省份，2025年起新高考落地。', 'https://www.nm.zsks.cn/'],
  ['辽宁', '东北', '辽宁省招生考试办公室', '3+1+2', '已实施“3+1+2”。', 'https://www.lnzsks.com/'],
  ['吉林', '东北', '吉林省教育考试院', '3+1+2', '第四批改革省份，2024年新高考首考。', 'https://www.jleea.com.cn/'],
  ['黑龙江', '东北', '黑龙江省招生考试院', '3+1+2', '第四批改革省份，2024年新高考首考。', 'https://www.lzk.hl.cn/'],
  ['上海', '华东', '上海市教育考试院', '3+3', '已实施“3+3”，院校专业组和选科要求需一起看。', 'https://www.shmeea.edu.cn/'],
  ['江苏', '华东', '江苏省教育考试院', '3+1+2', '已实施“3+1+2”。', 'https://www.jseea.cn/'],
  ['浙江', '华东', '浙江省教育考试院', '3+3', '已实施“3+3”，技术科目与选考目录需特别关注。', 'https://www.zjzs.net/'],
  ['安徽', '华东', '安徽省教育招生考试院', '3+1+2', '第四批改革省份，2024年新高考首考。', 'https://www.ahzsks.cn/'],
  ['福建', '华东', '福建省教育考试院', '3+1+2', '已实施“3+1+2”。', 'https://www.eeafj.cn/'],
  ['江西', '华东', '江西省教育考试院', '3+1+2', '第四批改革省份，2024年新高考首考。', 'https://www.jxeea.cn/'],
  ['山东', '华东', '山东省教育招生考试院', '3+3', '已实施“3+3”，选考科目要求按专业核对。', 'https://www.sdzk.cn/'],
  ['河南', '华中', '河南省教育考试院', '3+1+2', '第五批改革省份，2025年起新高考落地。', 'https://www.haeea.cn/'],
  ['湖北', '华中', '湖北省教育考试院', '3+1+2', '已实施“3+1+2”。', 'https://www.hbea.edu.cn/'],
  ['湖南', '华中', '湖南省教育考试院', '3+1+2', '已实施“3+1+2”。', 'https://jyt.hunan.gov.cn/jyt/sjyt/hnsjyksy/'],
  ['广东', '华南', '广东省教育考试院', '3+1+2', '已实施“3+1+2”。', 'https://eea.gd.gov.cn/'],
  ['广西', '华南', '广西壮族自治区招生考试院', '3+1+2', '第四批改革省份，2024年新高考首考。', 'https://www.gxeea.cn/'],
  ['海南', '华南', '海南省考试局', '3+3', '已实施新高考。', 'https://ea.hainan.gov.cn/'],
  ['重庆', '西南', '重庆市教育考试院', '3+1+2', '已实施“3+1+2”。', 'https://www.cqksy.cn/'],
  ['四川', '西南', '四川省教育考试院', '3+1+2', '第五批改革省份，2025年起新高考落地。', 'https://www.sceea.cn/'],
  ['贵州', '西南', '贵州省招生考试院', '3+1+2', '第四批改革省份，2024年新高考首考。', 'https://zsksy.guizhou.gov.cn/'],
  ['云南', '西南', '云南省招生考试院', '3+1+2', '第五批改革省份，2025年起新高考落地。', 'https://www.ynzs.cn/'],
  ['西藏', '西南', '西藏自治区教育考试院', 'traditional', '暂未纳入已落地新高考省份，需以自治区最新公告为准。', 'https://zsks.edu.xizang.gov.cn/'],
  ['陕西', '西北', '陕西省教育考试院', '3+1+2', '第五批改革省份，2025年起新高考落地。', 'https://www.sneac.com/'],
  ['甘肃', '西北', '甘肃省教育考试院', '3+1+2', '第四批改革省份，2024年新高考首考。', 'https://www.ganseea.cn/'],
  ['青海', '西北', '青海省教育招生考试院', '3+1+2', '第五批改革省份，2025年起新高考落地。', 'https://www.qhjyks.com/'],
  ['宁夏', '西北', '宁夏教育考试院', '3+1+2', '第五批改革省份，2025年起新高考落地。', 'https://www.nxjyks.cn/'],
  ['新疆', '西北', '新疆维吾尔自治区教育考试院', 'traditional', '暂未纳入已落地新高考省份，需以自治区最新公告为准。', 'https://www.xjzk.gov.cn/'],
  ['香港', '港澳台', '香港考试及评核局 / 港澳台招生信息网', 'special', '适用港澳台侨联招、DSE 及高校面向港澳台招生政策。', 'https://www.hkeaa.edu.hk/tc/ipe/jee/'],
  ['澳门', '港澳台', '澳门教育及青年发展局 / 港澳台招生信息网', 'special', '适用港澳台侨联招澳门考区及高校面向港澳台招生渠道。', 'https://www.gov.mo/zh-hant/services/ps-1717/ps-1717a/'],
  ['台湾', '港澳台', '港澳台招生信息网 / 普通高校联合招生办公室', 'special', '适用港澳台招生信息网、全国联招及高校面向台湾学生招生政策。', 'https://www.gatzs.com.cn/'],
]

const provinceNames = provinces.map(([name]) => name)
const commonScopes = ['全国', '全站', ...provinceNames]
const ownerOptions = {
  posts: ['内容运营', '社区审核', '选科研究所', '账号系统', '认证作者'],
  categories: ['内容运营', '产品运营', '前端配置', '增长运营'],
  policies: ['政策运营', '阳光高考/学信网', ...provinces.map((item) => item[2])],
  requirements: ['专业要求库', '专家复核组', '数据运营'],
  insights: ['数据运营', '选科研究所', '专家复核组'],
  advice: ['升学规划组', '内容运营', '专家审核', '社区审核'],
  users: ['账号系统', '认证系统', '用户运营', '安全审核'],
}
const typeOptions = {
  posts: ['经验帖', '家长提问', '数据建议', '经验/问答'],
  categories: ['帖子分类', '主导航入口', '工具入口', '内容入口'],
  policies: ['官方来源', ...gradeRegions],
  requirements: ['物理+化学强约束', '需逐校核对', '人文/交叉'],
  insights: ['组合趋势', '热度数据', '覆盖率数据', '风险提示'],
  advice: ['精选建议', '行动清单', '家长沟通', '专业核对'],
  users: ['学生', '家长', '老师', '规划师', '管理员', '运营人员'],
}
const statusOptions = {
  users: ['正常', '认证中', '需补充', '认证驳回', '冻结'],
  default: ['已上架', '待审核', '需复核', '退回修改', '草稿', '下架'],
}
const tagOptions = {
  posts: ['物化生', '史政地', '物化地', '风险核对', '专业覆盖', '数据建议', '学习方法'],
  categories: ['首页可见', '问答', '数据', '官方来源', '省份资料', '专业要求', '工具'],
  policies: ['3+3', '3+1+2', 'traditional', 'special', ...gradeRegions, '招生政策', '照顾政策', '选考要求', '志愿填报'],
  requirements: ['物理', '化学', '医学', '计算机', '人文社科', '不限或按院校'],
  insights: ['热度 96', '热度 88', '覆盖 96.2%', '覆盖 51.2%', '匹配 82.1%'],
  advice: ['专业限制', '家长沟通', '决策流程', '目标探索', '专业覆盖', '行动清单'],
  users: ['初中', '高一', '高二', '高三', '已验证邮箱', '教师认证', '规划师认证', '内容作者', '风险关注'],
}
const permissionPresets = {
  basic: { label: '基础用户', permissions: ['read', 'comment', 'favorite'] },
  author: { label: '内容作者', permissions: ['read', 'comment', 'favorite', 'post', 'upload_media'] },
  expert: { label: '认证专家', permissions: ['read', 'comment', 'favorite', 'post', 'upload_media', 'answer_as_expert', 'review_suggestion'] },
  operator: { label: '运营管理员', permissions: ['read', 'comment', 'favorite', 'post', 'upload_media', 'content_review', 'user_review', 'policy_manage'] },
  restricted: { label: '限制用户', permissions: ['read'] },
}
const permissionLabels = {
  read: '浏览内容',
  comment: '评论互动',
  favorite: '收藏关注',
  post: '发布帖子',
  upload_media: '上传媒体',
  answer_as_expert: '专家答疑',
  review_suggestion: '建议库审核',
  content_review: '内容审核',
  user_review: '用户认证审核',
  policy_manage: '政策库维护',
}

const seed = {
  modules: [
    { id: 'dashboard', label: '总览', icon: '▦', description: '运营状态、待办和内容风险' },
    { id: 'posts', label: '帖子管理', icon: '✎', description: '经验帖、问答、数据建议的审核与上架' },
    { id: 'categories', label: '分类与标签', icon: '⌘', description: '帖子分类、内容标签和前端入口管理' },
    { id: 'policies', label: '政策库', icon: '▣', description: '全国来源、省份政策、资料包入口' },
    { id: 'requirements', label: '专业要求', icon: '▤', description: '专业选科要求和风险提示' },
    { id: 'insights', label: '趋势数据', icon: '◒', description: '组合热度、匹配度、覆盖率' },
    { id: 'advice', label: '建议库', icon: '✦', description: '选科建议、行动清单和精选笔记' },
    { id: 'users', label: '用户与权限', icon: '◉', description: '用户角色、认证状态、内容权限' },
    { id: 'system', label: '系统配置', icon: '⚙', description: 'SMTP、接口、发布环境' },
  ],
  records: {
    posts: [
      row('post-1', '物化生适合目标不太明确的人吗？', '经验/问答', '已上架', '浙江', '小周同学', ['物化生', '专业覆盖'], '我现在数学和物理还可以，想听听大家对物化生后续专业覆盖和学习强度的真实感受。'),
      row('post-2', '孩子想选史政地，家长应该怎么判断风险？', '家长提问', '待审核', '山东', '林妈妈', ['史政地', '风险核对'], '孩子文科表达不错，但担心专业选择变窄。'),
      row('post-3', '从最近三届学生看物化地的优劣势', '经验帖', '已上架', '广东', '陈老师', ['物化地', '工科'], '物化地适合物理基础稳、空间理解强的学生。'),
      row('post-4', '2026届各组合专业覆盖率汇总', '数据建议', '需复核', '全国', '选科研究所', ['数据建议', '专业覆盖率'], '基于多省教育考试院公开数据整理。'),
    ],
    categories: [
      row('cat-1', '经验帖', '帖子分类', '已上架', '全站', '内容运营', ['首页可见'], '学生、老师、规划师的真实经验分享。'),
      row('cat-2', '家长提问', '帖子分类', '已上架', '全站', '内容运营', ['问答'], '家长视角提问与回复沉淀。'),
      row('cat-3', '数据建议', '帖子分类', '已上架', '全站', '内容运营', ['数据'], '政策、专业覆盖、趋势分析类内容。'),
      row('cat-4', '政策库', '主导航入口', '已上架', '全站', '产品运营', ['官方来源', '省份资料'], '省级考试院入口和政策核对资料包。'),
      row('cat-5', '选科查询', '工具入口', '已上架', '全站', '产品运营', ['专业要求'], '按专业和组合查询选科约束。'),
    ],
    policies: [
      ...provinces.map(([name, region, authority, mode, status, url]) => row(`province-${name}`, `${name}省份资料包`, region, mode === 'traditional' ? '需复核' : '已上架', name, authority, [mode, region], status, url)),
      row('source-yggk-policy', '2026年高考各省市招生政策及照顾政策汇总', '官方来源', '已上架', '全国', '阳光高考/学信网', ['招生政策', '照顾政策'], '集中查看各省市招生工作规定、实施办法和照顾政策。', 'https://gaokao.chsi.com.cn/z/gkbmfslq/zszc.jsp'),
      row('source-yggk-major', '2024年各省市普通高校本科专业选考科目要求汇总', '官方来源', '已上架', '全国', '阳光高考/学信网', ['选考要求', '专业目录'], '汇总部分省市专业选考科目要求。', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202201/20220105/2155365943.html'),
      row('source-zyck', '阳光志愿信息服务系统', '官方来源', '已上架', '全国', '阳光高考/学信网', ['志愿填报', '高校信息'], '整合高校、专业、招生政策和志愿参考信息。', 'https://gaokao.chsi.com.cn/zyck/'),
    ],
    requirements: ['临床医学', '口腔医学', '计算机科学与技术', '人工智能', '电气工程及其自动化', '法学', '汉语言文学', '金融学', '会计学', '教育学', '数字媒体艺术', '智能医学工程'].map((name, index) => row(`major-${index + 1}`, name, index < 5 ? '物理+化学强约束' : index < 9 ? '需逐校核对' : '人文/交叉', index % 5 === 0 ? '需复核' : '已上架', '全国', '专业要求库', index < 5 ? ['物理', '化学'] : ['不限或按院校'], `专业要求说明：${name} 需结合目标省份目录、院校专业组和当年招生章程核对。`)),
    insights: [
      row('insight-1', '物理 + 化学 + 生物', '组合趋势', '已上架', '全国', '数据运营', ['热度 96', '覆盖 96.2%'], '专业覆盖高，学习强度高。'),
      row('insight-2', '物理 + 化学 + 地理', '组合趋势', '已上架', '全国', '数据运营', ['热度 88', '覆盖 94.8%'], '工科友好，地理赋分需看省份。'),
      row('insight-3', '物理 + 生物 + 地理', '组合趋势', '已上架', '全国', '数据运营', ['热度 74', '匹配 82.1%'], '压力相对均衡，专业覆盖中高。'),
      row('insight-4', '历史 + 政治 + 地理', '组合趋势', '待审核', '全国', '数据运营', ['热度 81', '覆盖 51.2%'], '人文社科清晰，专业边界明确。'),
    ],
    advice: [
      row('advice-1', '先列不能报考清单，再谈兴趣', '精选建议', '已上架', '全国', '升学规划组', ['专业限制', '家长沟通'], '把硬性选科限制先排清，再讨论兴趣、学习强度和长期目标。', '/advice/audit-major-fit'),
      row('advice-2', '一次家庭选科会议的结构', '精选建议', '已上架', '全国', '升学规划组', ['沟通', '决策流程'], '用事实、选择、风险、下一步四段式降低亲子沟通成本。', '/advice/family-meeting'),
      row('advice-3', '目标还不明确时怎么保留弹性', '精选建议', '待审核', '全国', '内容运营', ['目标探索', '专业覆盖'], '用优势科目、可接受排除项和阶段复盘来避免盲目追热门。', '/advice/flexible-choice'),
    ],
    users: [
      row('user-3', '陈老师', '老师', '认证中', '广东', '认证系统', ['教师认证', '内容作者'], 'teacher@example.com'),
      row('user-4', '升学规划师A', '规划师', '正常', '全国', '认证系统', ['规划师认证'], 'counselor@example.com'),
    ],
    system: [],
  },
  audit: [
    '系统初始化后台内容台账',
    '同步省份政策库 34 条',
    '启用邮箱验证码注册策略',
  ],
}

function row(id, title, type, status, scope, owner, tags, summary, url = '') {
  return { id, title, type, status, scope, owner, tags, summary, url, updatedAt: '2026-07-03 20:30', priority: status === '需复核' ? '高' : status === '待审核' ? '中' : '常规' }
}

let state = createInitialState()
let activeModule = 'dashboard'
let selectedId = ''
let query = ''
let statusFilter = '全部'
let smtpConfig = null
let apiConnected = false
let lastSyncMessage = '请登录并同步后端内容库'
let pendingWorkflowAction = null
let adminSession = loadAdminSession()
let loginError = ''

function createInitialState() {
  return {
    modules: structuredClone(seed.modules),
    records: Object.fromEntries(Object.keys(seed.records).map((key) => [key, []])),
    audit: [],
  }
}

function recordsFor(moduleId = activeModule) {
  return state.records[moduleId] || []
}

function filteredRecords() {
  return recordsFor().filter((item) => {
    const matchesQuery = !query || [item.title, item.type, item.status, item.scope, item.owner, item.summary, ...(item.tags || [])].join('').toLowerCase().includes(query.toLowerCase())
    const matchesStatus = statusFilter === '全部' || item.status === statusFilter
    return matchesQuery && matchesStatus
  })
}

function moduleStats() {
  const all = Object.values(state.records).flat()
  return {
    total: all.length,
    published: all.filter((item) => item.status === '已上架' || item.status === '正常').length,
    pending: all.filter((item) => item.status === '待审核' || item.status === '认证中').length,
    review: all.filter((item) => item.status === '需复核').length,
  }
}

function render() {
  if (!isLoggedIn()) {
    document.querySelector('#app').innerHTML = renderLogin()
    bindLoginEvents()
    return
  }
  document.querySelector('#app').innerHTML = `
    <aside class="sidebar">
      <div class="brand"><span>选</span><div><strong>选科知谈</strong><small>Admin Console</small></div></div>
      <nav>${state.modules.map((item) => `<button class="${activeModule === item.id ? 'active' : ''}" data-module="${item.id}"><i>${item.icon}</i><span>${item.label}</span></button>`).join('')}</nav>
      <div class="sidebar-foot"><span>独立后台</span><strong>127.0.0.1:5175</strong></div>
    </aside>
    <main class="main">
      ${renderTopbar()}
      ${activeModule === 'dashboard' ? renderDashboard() : activeModule === 'system' ? renderSystem() : renderModule()}
    </main>
    ${renderDrawer()}
    ${renderWorkflowConfirm()}
  `
  bindEvents()
}

function renderLogin() {
  const cfg = settings()
  return `
    <main class="login-shell">
      <section class="login-panel">
        <div class="login-brand"><span>选</span><div><small>Admin Console</small><h1>选科知谈管理后台</h1><p>登录后可管理内容上架、用户认证、政策库、建议库和系统配置。</p></div></div>
        <label>账号邮箱<input id="login-email" type="email" autocomplete="username" value="${escapeHTML(cfg.adminEmail || '')}" placeholder="admin@example.com" /></label>
        <label>登录密码<input id="login-password" type="password" autocomplete="current-password" placeholder="请输入后台登录密码" /></label>
        <details class="login-advanced"><summary>连接设置</summary><label>API 地址<input id="login-api-base" value="${escapeHTML(cfg.apiBase || defaultApiBase())}" /></label></details>
        ${loginError ? `<p class="login-error">${escapeHTML(loginError)}</p>` : ''}
        <div class="login-actions"><button id="login-admin" class="primary">登录后台</button></div>
      </section>
    </main>
  `
}

function renderTopbar() {
  const module = state.modules.find((item) => item.id === activeModule)
  return `
    <header class="topbar">
      <div><small>当前位置 / ${module.label}</small><h1>${module.label}</h1><p>${module.description}</p></div>
      <div class="top-actions">
        <span class="session-pill">${escapeHTML(adminSession.email || settings().adminEmail || '在线后台')}</span>
        <span class="sync-pill ${apiConnected ? 'ok' : 'local'}">${escapeHTML(lastSyncMessage)}</span>
        <button id="sync-data">同步后台</button>
        <button id="export-data">导出配置</button>
        <button id="logout-admin">退出</button>
      </div>
    </header>
  `
}

function renderDashboard() {
  const stats = moduleStats()
  const health = [
    ['内容总量', stats.total, '覆盖帖子、政策、省份资料、专业要求和趋势数据'],
    ['已上架/正常', stats.published, '当前对前台可见或账号状态正常'],
    ['待处理', stats.pending, '需要审核、认证或运营确认'],
    ['需复核', stats.review, '政策时效或数据来源需要二次核验'],
  ]
  return `
    <section class="metric-grid">${health.map(([label, value, desc]) => `<article><small>${label}</small><strong>${value}</strong><p>${desc}</p></article>`).join('')}</section>
    <section class="dashboard-grid">
      <article class="panel wide"><div class="panel-head"><h2>内容域台账</h2><span>全站信息架构</span></div>${renderInventoryTable()}</article>
      <article class="panel"><div class="panel-head"><h2>运营待办</h2><span>按优先级</span></div>${renderTodoList()}</article>
      <article class="panel"><div class="panel-head"><h2>审计记录</h2><span>最近动作</span></div><ol class="audit-list">${state.audit.slice(-8).reverse().map((item) => `<li>${escapeHTML(item)}</li>`).join('')}</ol></article>
    </section>
  `
}

function renderInventoryTable() {
  return `<table class="dense-table"><thead><tr><th>模块</th><th>记录数</th><th>已上架/正常</th><th>待处理</th><th>需复核</th></tr></thead><tbody>${state.modules.filter((item) => !['dashboard', 'system'].includes(item.id)).map((mod) => {
    const rows = recordsFor(mod.id)
    return `<tr data-open-module="${mod.id}"><td><strong>${mod.label}</strong><small>${mod.description}</small></td><td>${rows.length}</td><td>${rows.filter((r) => r.status === '已上架' || r.status === '正常').length}</td><td>${rows.filter((r) => r.status === '待审核' || r.status === '认证中').length}</td><td>${rows.filter((r) => r.status === '需复核').length}</td></tr>`
  }).join('')}</tbody></table>`
}

function renderTodoList() {
  const todos = Object.entries(state.records).flatMap(([moduleId, rows]) => rows.filter((row) => ['待审核', '需复核', '认证中'].includes(row.status)).map((row) => ({ ...row, moduleId }))).slice(0, 10)
  return `<div class="todo-list">${todos.map((item) => `<button data-module="${item.moduleId}" data-select="${item.id}"><span class="status ${statusClass(item.status)}">${item.status}</span><strong>${escapeHTML(item.title)}</strong><small>${escapeHTML(item.type)} · ${escapeHTML(item.scope)}</small></button>`).join('') || '<p class="empty">暂无待办</p>'}</div>`
}

function renderModule() {
  const rows = filteredRecords()
  const statuses = ['全部', ...Array.from(new Set(recordsFor().map((item) => item.status)))]
  return `
    <section class="workbench">
      <div class="toolbar">
        <label class="search"><span>搜索</span><input id="search-box" value="${escapeHTML(query)}" placeholder="标题、地区、标签、负责人" /></label>
        <label><span>状态</span><select id="status-filter">${statuses.map((status) => `<option ${status === statusFilter ? 'selected' : ''}>${status}</option>`).join('')}</select></label>
        <button id="new-record" class="primary">新建条目</button>
      </div>
      <div class="table-wrap">
        <table class="content-table">
          <thead><tr><th>内容</th><th>类型</th><th>状态</th><th>范围</th><th>负责人/来源</th><th>标签</th><th>更新时间</th><th></th></tr></thead>
          <tbody>${rows.map(renderRow).join('') || '<tr><td colspan="8" class="empty-cell">没有匹配记录</td></tr>'}</tbody>
        </table>
      </div>
    </section>
  `
}

function renderRow(item) {
  return `<tr data-select="${item.id}"><td><strong>${escapeHTML(item.title)}</strong><small>${escapeHTML(item.summary)}</small></td><td>${escapeHTML(item.type)}</td><td><span class="status ${statusClass(item.status)}">${escapeHTML(item.status)}</span></td><td>${escapeHTML(item.scope)}</td><td>${escapeHTML(item.owner)}</td><td><div class="tag-row">${item.tags.map((tag) => `<span>${escapeHTML(tag)}</span>`).join('')}</div></td><td>${escapeHTML(item.updatedAt)}</td><td><button data-select="${item.id}">编辑</button></td></tr>`
}

function renderSystem() {
  return `
    <section class="system-grid">
      <article class="panel"><div class="panel-head"><h2>后端连接</h2><span>Admin API</span></div>
        <div class="status-card ok">当前账号：${escapeHTML(adminSession.email || settings().adminEmail || '未登录')}</div>
        <label>API 地址<input id="api-base" value="${escapeHTML(settings().apiBase || defaultApiBase())}" /></label>
        <button id="save-settings" class="primary">保存并检查连接</button>
      </article>
      <article class="panel"><div class="panel-head"><h2>SMTP 发信状态</h2><span>邮箱验证码</span></div>
        <div id="smtp-status" class="status-card ${smtpConfig?.enabled ? 'ok' : 'warn'}">${smtpConfig ? (smtpConfig.enabled ? 'SMTP 已配置，可发送验证码邮件' : `SMTP 未完整配置：${(smtpConfig.missing || []).join('、') || '未知'}`) : '等待检查'}</div>
        <dl class="field-list">${smtpConfig ? renderSMTPFields(smtpConfig) : ''}</dl>
      </article>
      <article class="panel wide"><div class="panel-head"><h2>发送测试验证码</h2><span>管理端测试</span></div>
        <div class="inline-form"><label>测试邮箱<input id="test-email" type="email" placeholder="name@example.com" /></label><button id="send-test" class="primary">发送测试邮件</button></div>
        <p id="test-result" class="result-text"></p>
      </article>
    </section>
  `
}

function renderSMTPFields(data) {
  const rows = [
    ['SMTP Host', data.host || '-'], ['SMTP Port', data.port], ['发件邮箱', data.fromEmail || '-'], ['发件名称', data.fromName || '-'], ['用户名', data.usernameConfigured ? '已配置' : '缺失'], ['密码', data.passwordConfigured ? '已配置' : '缺失'], ['TLS', data.useTLS ? '开启' : '关闭'], ['STARTTLS', data.startTLS ? '开启' : '关闭'], ['验证码有效期', `${data.emailVerificationTTLMinutes} 分钟`],
  ]
  return rows.map(([key, value]) => `<dt>${escapeHTML(String(key))}</dt><dd>${escapeHTML(String(value))}</dd>`).join('')
}

function renderDrawer() {
  const item = recordsFor().find((row) => row.id === selectedId)
  if (!item || activeModule === 'dashboard' || activeModule === 'system') return ''
  return `
    <aside class="drawer">
      <div class="drawer-head"><div><small>${state.modules.find((m) => m.id === activeModule)?.label}</small><h2>编辑条目</h2></div><button id="close-drawer">×</button></div>
      ${renderWorkflowPanel(item)}
      ${renderEditableFields(item)}
      ${activeModule === 'users' ? renderPermissionPanel(item) : ''}
      <label>摘要<textarea id="edit-summary">${escapeHTML(item.summary)}</textarea></label>
      <label>来源链接<input id="edit-url" value="${escapeHTML(item.url || '')}" /></label>
      <div class="drawer-actions"><button id="save-record" class="primary">保存基础信息</button><button id="delete-record" class="danger">删除</button></div>
    </aside>
  `
}

function renderEditableFields(item) {
  return `
    <label>${activeModule === 'users' ? '用户名称' : '标题'}<input id="edit-title" value="${escapeHTML(item.title)}" /></label>
    <label>${activeModule === 'users' ? '身份类型' : '类型'}${renderSelect('edit-type', optionsFor('type', activeModule, item.type), item.type)}</label>
    <label>状态${renderSelect('edit-status', optionsFor('status', activeModule, item.status), item.status)}</label>
    <label>${activeModule === 'users' ? '所属省份/服务范围' : '范围/地区'}${renderSelect('edit-scope', optionsFor('scope', activeModule, item.scope), item.scope)}</label>
    <label>${activeModule === 'users' ? '账号来源' : '负责人/来源'}${renderSelect('edit-owner', optionsFor('owner', activeModule, item.owner), item.owner)}</label>
    <label>标签${renderMultiSelect('edit-tags', optionsFor('tags', activeModule, item.tags), item.tags || [])}</label>
  `
}

function renderPermissionPanel(item) {
  const payload = item.payload || {}
  const currentPermissions = payload.permissions?.length ? payload.permissions : permissionsForType(item.type)
  const currentPreset = payload.permissionPreset || presetForPermissions(currentPermissions)
  return `
    <section class="permission-card">
      <div class="permission-head"><strong>权限分类</strong><span>${escapeHTML(permissionPresets[currentPreset]?.label || '自定义')}</span></div>
      <div class="permission-presets">
        ${Object.entries(permissionPresets).map(([id, preset]) => `<button class="${id === currentPreset ? 'active' : ''}" data-permission-preset="${id}" type="button">${escapeHTML(preset.label)}</button>`).join('')}
      </div>
      <div class="permission-grid">
        ${Object.entries(permissionLabels).map(([id, label]) => `<label><input type="checkbox" name="edit-permissions" value="${id}" ${currentPermissions.includes(id) ? 'checked' : ''} /><span>${escapeHTML(label)}</span></label>`).join('')}
      </div>
    </section>
  `
}

function renderWorkflowPanel(item) {
  const actions = workflowActionsFor(item, activeModule)
  const trail = workflowTrail(item)
  return `
    <section class="workflow-card">
      <div class="workflow-head"><strong>${activeModule === 'users' ? '身份认证流程' : '内容审核流程'}</strong><span class="status ${statusClass(item.status)}">${escapeHTML(item.status)}</span></div>
      <div class="workflow-steps">${workflowStepsFor(activeModule, item.status).map((step) => `<span class="${step.active ? 'active' : ''}">${escapeHTML(step.label)}</span>`).join('')}</div>
      <div class="workflow-actions">
        ${actions.map((action) => `<button class="${action.tone || ''}" data-workflow-action="${action.id}">${escapeHTML(action.label)}</button>`).join('') || '<p>当前状态暂无待处理动作。</p>'}
      </div>
      <div class="workflow-history">
        <strong>流程记录</strong>
        ${trail.length ? `<ol>${trail.slice(-4).reverse().map((entry) => `<li><span>${escapeHTML(entry.time)}</span><p>${escapeHTML(entry.action)}：${escapeHTML(entry.note || '无补充说明')}</p></li>`).join('')}</ol>` : '<p>暂无审核记录，下一次确认动作会写入这里。</p>'}
      </div>
    </section>
  `
}

function renderWorkflowConfirm() {
  if (!pendingWorkflowAction) return ''
  const item = recordsFor(pendingWorkflowAction.moduleId).find((row) => row.id === pendingWorkflowAction.recordId)
  const action = item ? workflowActionsFor(item, pendingWorkflowAction.moduleId).find((entry) => entry.id === pendingWorkflowAction.actionId) : null
  if (!item || !action) return ''
  return `
    <div class="modal-backdrop">
      <section class="confirm-modal">
        <div class="modal-head"><small>${escapeHTML(state.modules.find((m) => m.id === pendingWorkflowAction.moduleId)?.label || '')}</small><h2>${escapeHTML(action.label)}</h2></div>
        <dl class="confirm-summary">
          <dt>处理对象</dt><dd>${escapeHTML(item.title)}</dd>
          <dt>当前状态</dt><dd>${escapeHTML(item.status)}</dd>
          <dt>确认后状态</dt><dd>${escapeHTML(action.nextStatus)}</dd>
          <dt>流程影响</dt><dd>${escapeHTML(action.description)}</dd>
        </dl>
        <label>处理意见${action.requiresNote ? '<span>必填</span>' : '<span>选填</span>'}<textarea id="workflow-note" placeholder="${escapeHTML(action.placeholder)}"></textarea></label>
        ${pendingWorkflowAction.error ? `<p class="confirm-error">${escapeHTML(pendingWorkflowAction.error)}</p>` : ''}
        <div class="modal-actions"><button id="cancel-workflow">取消</button><button id="confirm-workflow" class="primary">确认执行</button></div>
      </section>
    </div>
  `
}

function bindEvents() {
  document.querySelectorAll('[data-module]').forEach((button) => button.addEventListener('click', () => {
    activeModule = button.dataset.module
    selectedId = button.dataset.select || ''
    query = ''
    statusFilter = '全部'
    render()
  }))
  document.querySelectorAll('[data-open-module]').forEach((row) => row.addEventListener('click', () => {
    activeModule = row.dataset.openModule
    render()
  }))
  document.querySelectorAll('[data-select]').forEach((row) => row.addEventListener('click', (event) => {
    event.stopPropagation()
    selectedId = row.dataset.select
    render()
  }))
  document.querySelector('#search-box')?.addEventListener('input', (event) => {
    query = event.target.value
    render()
  })
  document.querySelector('#status-filter')?.addEventListener('change', (event) => {
    statusFilter = event.target.value
    render()
  })
  document.querySelector('#new-record')?.addEventListener('click', createRecord)
  document.querySelector('#close-drawer')?.addEventListener('click', () => { selectedId = ''; render() })
  document.querySelector('#save-record')?.addEventListener('click', saveRecord)
  document.querySelector('#toggle-publish')?.addEventListener('click', togglePublish)
  document.querySelector('#delete-record')?.addEventListener('click', deleteRecord)
  document.querySelectorAll('[data-workflow-action]').forEach((button) => button.addEventListener('click', () => openWorkflowConfirm(button.dataset.workflowAction)))
  document.querySelectorAll('[data-permission-preset]').forEach((button) => button.addEventListener('click', () => applyPermissionPreset(button.dataset.permissionPreset)))
  document.querySelector('#cancel-workflow')?.addEventListener('click', () => { pendingWorkflowAction = null; render() })
  document.querySelector('#confirm-workflow')?.addEventListener('click', confirmWorkflowAction)
  document.querySelector('#sync-data')?.addEventListener('click', loadRemoteContent)
  document.querySelector('#export-data')?.addEventListener('click', exportData)
  document.querySelector('#logout-admin')?.addEventListener('click', logoutAdmin)
  document.querySelector('#save-settings')?.addEventListener('click', async () => { saveSettings(); await loadConfig(); await loadRemoteContent() })
  document.querySelector('#send-test')?.addEventListener('click', sendTestEmail)
}

function bindLoginEvents() {
  document.querySelector('#login-admin')?.addEventListener('click', loginAdmin)
  document.querySelector('#login-password')?.addEventListener('keydown', (event) => {
    if (event.key === 'Enter') loginAdmin()
  })
}

async function loginAdmin() {
  const apiBase = value('#login-api-base').replace(/\/$/, '')
  const email = value('#login-email').toLowerCase()
  const password = value('#login-password')
  if (!email || !email.includes('@')) {
    loginError = '请输入邮箱形式的后台账号。'
    render()
    return
  }
  if (!password) {
    loginError = '请输入后台登录密码。'
    render()
    return
  }
  localStorage.setItem(SETTINGS_KEY, JSON.stringify({ ...settings(), apiBase, adminEmail: email, adminToken: '' }))
  loginError = ''
  try {
    const loginData = await adminFetch('/admin/login', { method: 'POST', body: JSON.stringify({ email, password }) })
    localStorage.setItem(SETTINGS_KEY, JSON.stringify({ ...settings(), apiBase, adminEmail: email, adminToken: loginData.token }))
    smtpConfig = await adminFetch('/admin/email-config')
    adminSession = { authenticated: true, mode: 'online', email, signedAt: new Date().toISOString() }
    localStorage.setItem(SESSION_KEY, JSON.stringify(adminSession))
    apiConnected = true
    lastSyncMessage = '登录成功，已连接后端'
    await loadRemoteContent()
    render()
  } catch (error) {
    loginError = `登录失败：${error.message}`
    adminSession = { authenticated: false }
    localStorage.removeItem(SESSION_KEY)
    render()
  }
}

function logoutAdmin() {
  localStorage.removeItem(SESSION_KEY)
  localStorage.setItem(SETTINGS_KEY, JSON.stringify({ ...settings(), adminToken: '' }))
  adminSession = { authenticated: false }
  state = createInitialState()
  apiConnected = false
  lastSyncMessage = '已退出后台'
  selectedId = ''
  pendingWorkflowAction = null
  render()
}

function workflowStepsFor(moduleId, status) {
  const userSteps = ['提交认证', '资料核验', '审核结论', '权限生效']
  const contentSteps = ['提交内容', '初审', '复核确认', '上架展示']
  const labels = moduleId === 'users' ? userSteps : contentSteps
  const indexMap = moduleId === 'users'
    ? { 认证中: 1, 需补充: 1, 认证驳回: 2, 正常: 3, 冻结: 3 }
    : { 草稿: 0, 退回修改: 0, 待审核: 1, 需复核: 2, 已上架: 3, 下架: 2 }
  const current = indexMap[status] ?? 0
  return labels.map((label, index) => ({ label, active: index <= current }))
}

function optionsFor(kind, moduleId, currentValue) {
  if (kind === 'type') return ensureOptions(typeOptions[moduleId] || ['未分类'], currentValue)
  if (kind === 'status') return ensureOptions(moduleId === 'users' ? statusOptions.users : statusOptions.default, currentValue)
  if (kind === 'scope') {
    const scopes = moduleId === 'categories' ? ['全站', '首页', '政策库', '建议库', '选科查询'] : moduleId === 'users' ? ['全国', ...provinceNames] : commonScopes
    return ensureOptions(scopes, currentValue)
  }
  if (kind === 'owner') return ensureOptions(ownerOptions[moduleId] || ['内容运营'], currentValue)
  if (kind === 'tags') return ensureOptions(tagOptions[moduleId] || [], currentValue || [])
  return []
}

function ensureOptions(options, selected) {
  const values = Array.isArray(selected) ? selected : [selected]
  return Array.from(new Set([...options, ...values.filter(Boolean)]))
}

function renderSelect(id, options, selected) {
  return `<select id="${id}">${options.map((option) => `<option value="${escapeHTML(option)}" ${option === selected ? 'selected' : ''}>${escapeHTML(option)}</option>`).join('')}</select>`
}

function renderMultiSelect(id, options, selectedValues) {
  const selected = new Set(selectedValues)
  return `<select id="${id}" class="multi-select" multiple size="${Math.min(Math.max(options.length, 4), 7)}">${options.map((option) => `<option value="${escapeHTML(option)}" ${selected.has(option) ? 'selected' : ''}>${escapeHTML(option)}</option>`).join('')}</select>`
}

function applyPermissionPreset(presetId) {
  const preset = permissionPresets[presetId]
  if (!preset) return
  document.querySelectorAll('[data-permission-preset]').forEach((button) => button.classList.toggle('active', button.dataset.permissionPreset === presetId))
  document.querySelectorAll('input[name="edit-permissions"]').forEach((input) => {
    input.checked = preset.permissions.includes(input.value)
  })
}

function permissionsForType(type) {
  if (type === '老师') return permissionPresets.expert.permissions
  if (type === '规划师') return permissionPresets.expert.permissions
  if (type === '管理员' || type === '运营人员') return permissionPresets.operator.permissions
  return permissionPresets.basic.permissions
}

function presetForPermissions(permissions) {
  const sorted = [...permissions].sort().join(',')
  return Object.entries(permissionPresets).find(([, preset]) => [...preset.permissions].sort().join(',') === sorted)?.[0] || 'basic'
}

function workflowActionsFor(item, moduleId) {
  if (moduleId === 'users') {
    const actions = {
      认证中: [
        workflowAction('approve-user', '认证通过', '正常', '用户认证通过后账号恢复正常权限，可继续发布和互动。', '已核验证件/资质来源，认证信息真实有效。', 'positive'),
        workflowAction('request-user-info', '要求补充材料', '需补充', '账号暂不通过认证，等待用户补充证明材料。', '请写明需要补充的材料或核验项。', 'warning', true),
        workflowAction('reject-user', '驳回认证', '认证驳回', '认证申请关闭，用户需重新发起或联系管理员。', '请写明驳回原因，便于后续追溯。', 'danger', true),
      ],
      需补充: [
        workflowAction('resume-user-review', '重新进入审核', '认证中', '材料补齐后重新进入认证核验队列。', '说明收到的新材料或重新核验依据。', 'primary'),
        workflowAction('reject-user', '驳回认证', '认证驳回', '补充材料仍不满足要求，认证申请关闭。', '请写明仍未通过的原因。', 'danger', true),
      ],
      认证驳回: [
        workflowAction('resume-user-review', '重新开启认证', '认证中', '管理员重新打开该账号认证流程。', '说明重新开启认证的依据。', 'primary', true),
      ],
      正常: [
        workflowAction('freeze-user', '冻结账号', '冻结', '账号将暂停认证权益和内容发布权限。', '请写明冻结原因和恢复条件。', 'danger', true),
        workflowAction('recheck-user', '发起身份复核', '认证中', '账号进入复核队列，需重新核对身份或资质。', '说明触发复核的风险点。', 'warning', true),
      ],
      冻结: [
        workflowAction('restore-user', '恢复账号', '正常', '账号恢复正常认证权益和内容权限。', '说明恢复依据。', 'positive', true),
      ],
    }
    return actions[item.status] || []
  }

  const isAdvice = moduleId === 'advice'
  const publishedCopy = isAdvice ? '建议笔记会进入前台建议库展示。' : '内容会进入前台可见状态。'
  const actions = {
    待审核: [
      workflowAction('approve-content', isAdvice ? '审核通过并上架' : '审核通过', '已上架', `审核通过后，${publishedCopy}`, '说明内容来源、风险点和通过理由。', 'positive'),
      workflowAction('send-to-review', '转入复核', '需复核', '内容存在政策时效、数据来源或表达风险，进入二次复核。', '请写明需要复核的具体项。', 'warning', true),
      workflowAction('return-content', '退回修改', '退回修改', '内容不会上架，等待作者或运营修改后重新提交。', '请写明退回修改项。', 'danger', true),
    ],
    需复核: [
      workflowAction('pass-review', '复核通过并上架', '已上架', `复核完成后，${publishedCopy}`, '说明复核依据和最终结论。', 'positive'),
      workflowAction('keep-review', '继续复核', '需复核', '内容仍停留在复核状态，等待补充资料或二次确认。', '写明仍需补充核对的事项。', 'warning', true),
      workflowAction('reject-after-review', '复核不通过下架', '下架', '内容从前台移除或保持不可见。', '请写明复核不通过原因。', 'danger', true),
    ],
    已上架: [
      workflowAction('start-review', '发起复核', '需复核', '已上架内容进入风险复核队列，复核期间保留记录。', '说明触发复核的政策、数据或投诉原因。', 'warning', true),
      workflowAction('unpublish-content', '确认下架', '下架', '内容从前台可见列表移除。', '请写明下架原因。', 'danger', true),
    ],
    退回修改: [
      workflowAction('submit-content-review', '重新提交审核', '待审核', '修改完成后重新进入初审队列。', '说明本次修改了哪些问题。', 'primary'),
    ],
    草稿: [
      workflowAction('submit-content-review', '提交审核', '待审核', '草稿进入初审队列，等待运营审核。', '说明提交审核的范围或重点。', 'primary'),
    ],
    下架: [
      workflowAction('submit-content-review', '整改后提交审核', '待审核', '整改后的内容重新进入初审队列。', '说明整改内容和重新上架依据。', 'primary', true),
    ],
  }
  return actions[item.status] || []
}

function workflowAction(id, label, nextStatus, description, placeholder, tone = '', requiresNote = false) {
  return { id, label, nextStatus, description, placeholder, tone, requiresNote }
}

function workflowTrail(item) {
  return Array.isArray(item.payload?.workflow) ? item.payload.workflow : []
}

function openWorkflowConfirm(actionId) {
  const item = recordsFor().find((row) => row.id === selectedId)
  if (!item) return
  pendingWorkflowAction = { actionId, moduleId: activeModule, recordId: item.id, error: '' }
  render()
}

async function confirmWorkflowAction() {
  if (!pendingWorkflowAction) return
  const item = recordsFor(pendingWorkflowAction.moduleId).find((row) => row.id === pendingWorkflowAction.recordId)
  const action = item ? workflowActionsFor(item, pendingWorkflowAction.moduleId).find((entry) => entry.id === pendingWorkflowAction.actionId) : null
  if (!item || !action) return
  const note = value('#workflow-note')
  if (action.requiresNote && !note) {
    pendingWorkflowAction.error = '该动作必须填写处理意见，方便后续复盘和追责。'
    render()
    return
  }
  const snapshot = structuredClone(item)
  const previousStatus = item.status
  item.status = action.nextStatus
  item.priority = nextPriority(action.nextStatus)
  item.updatedAt = new Date().toLocaleString('zh-CN', { hour12: false })
  item.payload = {
    ...(item.payload || {}),
    workflow: [
      ...workflowTrail(item),
      {
        time: item.updatedAt,
        action: action.label,
        from: previousStatus,
        to: action.nextStatus,
        note: note || action.description,
        actor: 'admin',
      },
    ],
  }
  state.audit.push(`${action.label}「${item.title}」：${note || action.description}`)
  pendingWorkflowAction = null
  const saved = await saveWorkflowRecord(item, action, note)
  if (!saved) Object.assign(item, snapshot)
  render()
}

async function createRecord() {
  const item = row(`${activeModule}-${Date.now()}`, '新建内容条目', '未分类', '草稿', '全国', '内容运营', ['待完善'], '请补充内容摘要、来源和上架策略。')
  state.records[activeModule].unshift(item)
  selectedId = item.id
  state.audit.push(`新建 ${state.modules.find((m) => m.id === activeModule)?.label} 条目：${item.title}`)
  const saved = await saveContentRecord(item)
  if (!saved) {
    state.records[activeModule] = recordsFor().filter((row) => row.id !== item.id)
    selectedId = ''
  }
  render()
}

async function saveRecord() {
  const item = recordsFor().find((row) => row.id === selectedId)
  if (!item) return
  const snapshot = structuredClone(item)
  item.title = value('#edit-title')
  item.type = value('#edit-type')
  item.status = value('#edit-status')
  item.scope = value('#edit-scope')
  item.owner = value('#edit-owner')
  item.tags = selectedValues('#edit-tags')
  item.summary = value('#edit-summary')
  item.url = value('#edit-url')
  if (activeModule === 'users') {
    item.payload = {
      ...(item.payload || {}),
      permissionPreset: document.querySelector('[data-permission-preset].active')?.dataset.permissionPreset || 'basic',
      permissions: checkedValues('input[name="edit-permissions"]'),
    }
  }
  item.updatedAt = new Date().toLocaleString('zh-CN', { hour12: false })
  state.audit.push(`保存 ${item.title}，状态：${item.status}`)
  const saved = await saveContentRecord(item)
  if (saved) {
    selectedId = ''
    pendingWorkflowAction = null
  } else {
    Object.assign(item, snapshot)
  }
  render()
}

async function togglePublish() {
  const item = recordsFor().find((row) => row.id === selectedId)
  if (!item) return
  item.status = item.status === '已上架' ? '下架' : '已上架'
  item.updatedAt = new Date().toLocaleString('zh-CN', { hour12: false })
  state.audit.push(`${item.status === '已上架' ? '上架' : '下架'} ${item.title}`)
  await saveContentRecord(item)
  render()
}

async function deleteRecord() {
  const item = recordsFor().find((row) => row.id === selectedId)
  if (!item || !confirm(`确认删除「${item.title}」？`)) return
  if (!settings().adminToken || !item.createdRemote) {
    apiConnected = false
    lastSyncMessage = '未连接后端，不能删除内容'
    render()
    return
  }
  try {
    await adminFetch(`/admin/content/${encodeURIComponent(item.id)}`, { method: 'DELETE' })
    state.records[activeModule] = recordsFor().filter((row) => row.id !== item.id)
    state.audit.push(`删除 ${item.title}`)
    selectedId = ''
    apiConnected = true
    lastSyncMessage = '已从后端删除'
  } catch (error) {
    apiConnected = false
    lastSyncMessage = `后端删除失败：${error.message}`
  }
  render()
}

async function loadRemoteContent() {
  if (!settings().adminToken) {
    apiConnected = false
    lastSyncMessage = '未登录后台，无法同步内容库'
    render()
    return
  }
  try {
    const data = await adminFetch('/admin/content')
    const grouped = Object.fromEntries(Object.keys(seed.records).map((key) => [key, []]))
    ;(data.records || []).forEach((record) => {
      if (!grouped[record.module]) return
      grouped[record.module].push(fromAPIRecord(record))
    })
    Object.entries(grouped).forEach(([module, rows]) => {
      if (rows.length) state.records[module] = rows
    })
    const auditData = await adminFetch('/admin/audit-logs').catch(() => ({ logs: [] }))
    if (auditData.logs?.length) {
      state.audit = auditData.logs.map((item) => `${formatDate(item.createdAt)} · ${item.detail}`)
    }
    apiConnected = true
    lastSyncMessage = '已连接后端内容库'
  } catch (error) {
    apiConnected = false
    lastSyncMessage = `后端连接失败：${error.message}`
  }
  render()
}

async function saveContentRecord(item) {
  if (!settings().adminToken) {
    apiConnected = false
    lastSyncMessage = '未登录后台，保存失败'
    return false
  }
  try {
    const method = item.createdRemote ? 'PUT' : 'POST'
    const path = method === 'POST' ? '/admin/content' : `/admin/content/${encodeURIComponent(item.id)}`
    const saved = await adminFetch(path, { method, body: JSON.stringify(toAPIRecord(item)) })
    Object.assign(item, fromAPIRecord(saved), { createdRemote: true })
    apiConnected = true
    lastSyncMessage = '已保存到后端'
    return true
  } catch (error) {
    apiConnected = false
    lastSyncMessage = `后端保存失败：${error.message}`
    return false
  }
}

async function saveWorkflowRecord(item, action, note) {
  if (!settings().adminToken || !item.createdRemote) {
    apiConnected = false
    lastSyncMessage = settings().adminToken ? '远端记录尚未创建，流程保存失败' : '未登录后台，流程保存失败'
    return false
  }
  try {
    const saved = await adminFetch(`/admin/content/${encodeURIComponent(item.id)}/workflow`, {
      method: 'POST',
      body: JSON.stringify({
        action: action.id,
        actionLabel: action.label,
        nextStatus: action.nextStatus,
        note: note || action.description,
        priority: item.priority || nextPriority(action.nextStatus),
        payload: item.payload || {},
      }),
    })
    Object.assign(item, fromAPIRecord(saved), { createdRemote: true })
    apiConnected = true
    lastSyncMessage = '审核流程已写入后端'
    return true
  } catch (error) {
    apiConnected = false
    lastSyncMessage = `流程保存失败：${error.message}`
    return false
  }
}

function fromAPIRecord(record) {
  return {
    id: record.id,
    title: record.title,
    type: record.type,
    status: record.status,
    scope: record.scope,
    owner: record.owner,
    tags: record.tags || [],
    summary: record.summary,
    url: record.url || '',
    priority: record.priority || '常规',
    sortOrder: record.sortOrder || 0,
    payload: record.payload || {},
    updatedAt: formatDate(record.updatedAt),
    createdRemote: true,
  }
}

function toAPIRecord(item) {
  return {
    id: item.id,
    module: activeModule,
    title: item.title,
    type: item.type,
    status: item.status,
    scope: item.scope,
    owner: item.owner,
    tags: item.tags || [],
    summary: item.summary,
    url: item.url || '',
    priority: item.priority || '常规',
    sortOrder: item.sortOrder || 0,
    payload: item.payload || {},
  }
}

function formatDate(value) {
  if (!value) return new Date().toLocaleString('zh-CN', { hour12: false })
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return date.toLocaleString('zh-CN', { hour12: false })
}

function exportData() {
  const blob = new Blob([JSON.stringify(state, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `scf-admin-export-${Date.now()}.json`
  a.click()
  URL.revokeObjectURL(url)
}

function loadAdminSession() {
  try {
    return JSON.parse(localStorage.getItem(SESSION_KEY) || '{"authenticated":false}')
  } catch {
    return { authenticated: false }
  }
}

function isLoggedIn() {
  return adminSession?.authenticated === true && Boolean(settings().adminToken)
}

function settings() {
  try {
    return JSON.parse(localStorage.getItem(SETTINGS_KEY) || '{}')
  } catch {
    localStorage.removeItem(SETTINGS_KEY)
    return {}
  }
}

function saveSettings() {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify({ ...settings(), apiBase: value('#api-base').replace(/\/$/, '') }))
}

async function adminFetch(path, options = {}) {
  const cfg = settings()
  const response = await fetch(`${cfg.apiBase || defaultApiBase()}${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', 'X-Admin-Token': cfg.adminToken || '', ...(options.headers || {}) },
  })
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(payload.error?.message || `请求失败：${response.status}`)
  return payload.data
}

function defaultApiBase() {
  const isLocalAdmin =
    ['localhost', '127.0.0.1', '::1'].includes(window.location.hostname) &&
    ['5175', ''].includes(window.location.port)
  return isLocalAdmin ? 'http://localhost:8081/api/v1' : '/api/v1'
}

async function loadConfig() {
  const card = document.querySelector('#smtp-status')
  if (card) card.textContent = '正在检查...'
  try {
    smtpConfig = await adminFetch('/admin/email-config')
  } catch (error) {
    smtpConfig = { enabled: false, missing: [error.message] }
  }
  render()
}

async function sendTestEmail() {
  const result = document.querySelector('#test-result')
  result.textContent = '正在发送...'
  try {
    const data = await adminFetch('/admin/email-test', { method: 'POST', body: JSON.stringify({ email: value('#test-email') }) })
    result.textContent = data.debugCode ? `SMTP 未启用，本地调试验证码：${data.debugCode}` : `测试验证码已发送到 ${data.email}`
  } catch (error) {
    result.textContent = error.message
  }
}

function value(selector) {
  return document.querySelector(selector)?.value?.trim() || ''
}

function selectedValues(selector) {
  const element = document.querySelector(selector)
  if (!element) return []
  return Array.from(element.selectedOptions || []).map((option) => option.value.trim()).filter(Boolean)
}

function checkedValues(selector) {
  return Array.from(document.querySelectorAll(selector)).filter((input) => input.checked).map((input) => input.value)
}

function statusClass(status) {
  if (['已上架', '正常'].includes(status)) return 'ok'
  if (['待审核', '认证中', '需补充'].includes(status)) return 'pending'
  if (['需复核', '退回修改'].includes(status)) return 'warn'
  if (['下架', '认证驳回', '冻结'].includes(status)) return 'danger'
  return 'muted'
}

function nextPriority(status) {
  if (['需复核', '冻结', '下架'].includes(status)) return '高'
  if (['待审核', '认证中', '需补充', '退回修改', '认证驳回'].includes(status)) return '中'
  return '常规'
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, (char) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[char])
}

render()
if (isLoggedIn() && settings().adminToken) {
  loadRemoteContent()
}
