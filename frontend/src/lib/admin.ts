import axios from 'axios'
import { defaultApiBasePath } from './runtime'

export const SETTINGS_KEY = 'scf_admin_settings'
export const SESSION_KEY = 'scf_admin_session'

export type ModuleId =
  | 'dashboard'
  | 'posts'
  | 'categories'
  | 'policies'
  | 'requirements'
  | 'insights'
  | 'advice'
  | 'users'
  | 'system'

export type EditableModuleId = Exclude<ModuleId, 'dashboard' | 'system'>
export type PermissionPresetId = 'basic' | 'author' | 'expert' | 'operator' | 'restricted'

export interface AdminModuleInfo {
  id: ModuleId
  label: string
  icon: string
  description: string
}

export interface WorkflowEntry {
  time: string
  action: string
  from: string
  to: string
  note: string
  actor: string
}

export interface AdminRecordPayload {
  imageUrls?: string[]
  workflow?: WorkflowEntry[]
  permissionPreset?: PermissionPresetId
  permissions?: string[]
  [key: string]: unknown
}

export interface AdminRecord {
  id: string
  title: string
  type: string
  status: string
  scope: string
  owner: string
  tags: string[]
  summary: string
  url: string
  priority: string
  sortOrder: number
  payload: AdminRecordPayload
  updatedAt: string
  createdRemote?: boolean
}

export interface AdminState {
  modules: AdminModuleInfo[]
  records: Record<ModuleId, AdminRecord[]>
  audit: string[]
}

export interface AdminSettings {
  apiBase: string
  adminEmail: string
  adminToken: string
}

export interface AdminSession {
  authenticated: boolean
  mode?: 'online'
  email?: string
  signedAt?: string
}

export interface AdminLoginResponse {
  email: string
  token: string
}

export interface AdminEmailConfig {
  enabled: boolean
  host: string
  port: number
  usernameConfigured: boolean
  passwordConfigured: boolean
  fromEmail: string
  fromName: string
  useTLS: boolean
  startTLS: boolean
  emailVerificationTTLMinutes: number
  missing: string[]
}

export interface AdminAuditLog {
  action: string
  recordId: string
  module: string
  detail: string
  actor: string
  createdAt: string
}

export interface AdminApiRecord {
  id: string
  module: EditableModuleId
  title: string
  type: string
  status: string
  scope: string
  owner: string
  tags: string[]
  summary: string
  url: string
  priority: string
  sortOrder: number
  payload: AdminRecordPayload
  updatedAt: string
}

export interface AdminWorkflowAction {
  id: string
  label: string
  nextStatus: string
  description: string
  placeholder: string
  tone: '' | 'positive' | 'warning' | 'danger' | 'primary'
  requiresNote: boolean
}

interface ApiEnvelope<T> {
  data: T
  error?: {
    message?: string
  }
}

const gradeRegions = ['华北', '东北', '华东', '华中', '华南', '西南', '西北', '港澳台'] as const
const provinces = [
  ['北京', '华北', '北京教育考试院'],
  ['天津', '华北', '天津市教育招生考试院'],
  ['河北', '华北', '河北省教育考试院'],
  ['山西', '华北', '山西省招生考试管理中心'],
  ['内蒙古', '华北', '内蒙古自治区教育招生考试中心'],
  ['辽宁', '东北', '辽宁省招生考试办公室'],
  ['吉林', '东北', '吉林省教育考试院'],
  ['黑龙江', '东北', '黑龙江省招生考试院'],
  ['上海', '华东', '上海市教育考试院'],
  ['江苏', '华东', '江苏省教育考试院'],
  ['浙江', '华东', '浙江省教育考试院'],
  ['安徽', '华东', '安徽省教育招生考试院'],
  ['福建', '华东', '福建省教育考试院'],
  ['江西', '华东', '江西省教育考试院'],
  ['山东', '华东', '山东省教育招生考试院'],
  ['河南', '华中', '河南省教育考试院'],
  ['湖北', '华中', '湖北省教育考试院'],
  ['湖南', '华中', '湖南省教育考试院'],
  ['广东', '华南', '广东省教育考试院'],
  ['广西', '华南', '广西壮族自治区招生考试院'],
  ['海南', '华南', '海南省考试局'],
  ['重庆', '西南', '重庆市教育考试院'],
  ['四川', '西南', '四川省教育考试院'],
  ['贵州', '西南', '贵州省招生考试院'],
  ['云南', '西南', '云南省招生考试院'],
  ['西藏', '西南', '西藏自治区教育考试院'],
  ['陕西', '西北', '陕西省教育考试院'],
  ['甘肃', '西北', '甘肃省教育考试院'],
  ['青海', '西北', '青海省教育招生考试院'],
  ['宁夏', '西北', '宁夏教育考试院'],
  ['新疆', '西北', '新疆维吾尔自治区教育考试院'],
  ['香港', '港澳台', '香港考试及评核局 / 港澳台招生信息网'],
  ['澳门', '港澳台', '澳门教育及青年发展局 / 港澳台招生信息网'],
  ['台湾', '港澳台', '港澳台招生信息网 / 普通高校联合招生办公室'],
] as const

const provinceNames = provinces.map(([name]) => name)
const commonScopes = ['全国', '全站', ...provinceNames]

export const moduleDefinitions: AdminModuleInfo[] = [
  { id: 'dashboard', label: '总览', icon: '▦', description: '运营状态、待办和内容风险' },
  { id: 'posts', label: '帖子管理', icon: '✎', description: '经验帖、问答、数据建议的审核与上架' },
  { id: 'categories', label: '分类与标签', icon: '⌘', description: '帖子分类、内容标签和前端入口管理' },
  { id: 'policies', label: '政策库', icon: '▣', description: '全国来源、省份政策、资料包入口' },
  { id: 'requirements', label: '专业要求', icon: '▤', description: '专业选科要求和风险提示' },
  { id: 'insights', label: '趋势数据', icon: '◒', description: '组合热度、匹配度、覆盖率' },
  { id: 'advice', label: '建议库', icon: '✦', description: '选科建议、行动清单和精选笔记' },
  { id: 'users', label: '用户与权限', icon: '◉', description: '用户角色、认证状态、内容权限' },
  { id: 'system', label: '系统配置', icon: '⚙', description: 'SMTP、接口、发布环境' },
]

export const editableModuleIds = moduleDefinitions
  .map((item) => item.id)
  .filter((item): item is EditableModuleId => item !== 'dashboard' && item !== 'system')

export const permissionPresets: Record<PermissionPresetId, { label: string; permissions: string[] }> = {
  basic: { label: '基础用户', permissions: ['read', 'comment', 'favorite'] },
  author: { label: '内容作者', permissions: ['read', 'comment', 'favorite', 'post', 'upload_media'] },
  expert: { label: '认证专家', permissions: ['read', 'comment', 'favorite', 'post', 'upload_media', 'answer_as_expert', 'review_suggestion'] },
  operator: { label: '运营管理员', permissions: ['read', 'comment', 'favorite', 'post', 'upload_media', 'content_review', 'user_review', 'policy_manage'] },
  restricted: { label: '限制用户', permissions: ['read'] },
}

export const permissionLabels: Record<string, string> = {
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

const ownerOptions: Record<EditableModuleId, string[]> = {
  posts: ['内容运营', '社区审核', '选科研究所', '账号系统', '认证作者'],
  categories: ['内容运营', '产品运营', '前端配置', '增长运营'],
  policies: ['政策运营', '阳光高考/学信网', ...provinces.map((item) => item[2])],
  requirements: ['专业要求库', '专家复核组', '数据运营'],
  insights: ['数据运营', '选科研究所', '专家复核组'],
  advice: ['升学规划组', '内容运营', '专家审核', '社区审核'],
  users: ['账号系统', '认证系统', '用户运营', '安全审核'],
}

const typeOptions: Record<EditableModuleId, string[]> = {
  posts: ['经验帖', '家长提问', '数据建议', '经验/问答'],
  categories: ['帖子分类', '主导航入口', '工具入口', '内容入口'],
  policies: ['官方来源', ...gradeRegions],
  requirements: ['物理+化学强约束', '需逐校核对', '人文/交叉'],
  insights: ['组合趋势', '热度数据', '覆盖率数据', '风险提示'],
  advice: ['精选建议', '行动清单', '家长沟通', '专业核对'],
  users: ['学生', '家长', '老师', '规划师', '管理员', '运营人员'],
}

const statusOptions: Record<'users' | 'default', string[]> = {
  users: ['正常', '认证中', '需补充', '认证驳回', '冻结'],
  default: ['已上架', '待审核', '需复核', '退回修改', '草稿', '下架'],
}

const tagOptions: Record<EditableModuleId, string[]> = {
  posts: ['物化生', '史政地', '物化地', '风险核对', '专业覆盖', '数据建议', '学习方法'],
  categories: ['首页可见', '问答', '数据', '官方来源', '省份资料', '专业要求', '工具'],
  policies: ['3+3', '3+1+2', 'traditional', 'special', ...gradeRegions, '招生政策', '照顾政策', '选考要求', '志愿填报'],
  requirements: ['物理', '化学', '医学', '计算机', '人文社科', '不限或按院校'],
  insights: ['热度 96', '热度 88', '覆盖 96.2%', '覆盖 51.2%', '匹配 82.1%'],
  advice: ['专业限制', '家长沟通', '决策流程', '目标探索', '专业覆盖', '行动清单'],
  users: ['初中', '高一', '高二', '高三', '已验证邮箱', '教师认证', '规划师认证', '内容作者', '风险关注'],
}

export function defaultApiBase() {
  return (import.meta.env.VITE_API_BASE_URL ?? defaultApiBasePath()).replace(/\/$/, '')
}

export function loadAdminSettings(): AdminSettings {
  try {
    const raw = JSON.parse(localStorage.getItem(SETTINGS_KEY) || '{}') as Partial<AdminSettings>
    return {
      apiBase: (raw.apiBase || defaultApiBase()).replace(/\/$/, ''),
      adminEmail: raw.adminEmail || '',
      adminToken: raw.adminToken || '',
    }
  } catch {
    localStorage.removeItem(SETTINGS_KEY)
    return {
      apiBase: defaultApiBase(),
      adminEmail: '',
      adminToken: '',
    }
  }
}

export function saveAdminSettings(settings: AdminSettings) {
  localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings))
}

export function loadAdminSession(): AdminSession {
  try {
    return JSON.parse(localStorage.getItem(SESSION_KEY) || '{"authenticated":false}') as AdminSession
  } catch {
    return { authenticated: false }
  }
}

export function saveAdminSession(session: AdminSession) {
  localStorage.setItem(SESSION_KEY, JSON.stringify(session))
}

export function clearAdminSession() {
  localStorage.removeItem(SESSION_KEY)
}

export function createInitialAdminState(): AdminState {
  return {
    modules: structuredClone(moduleDefinitions),
    records: {
      dashboard: [],
      posts: [],
      categories: [],
      policies: [],
      requirements: [],
      insights: [],
      advice: [],
      users: [],
      system: [],
    },
    audit: [],
  }
}

export function createNewRecord(moduleId: EditableModuleId): AdminRecord {
  return {
    id: `${moduleId}-${Date.now()}`,
    title: moduleId === 'users' ? '新建用户条目' : '新建内容条目',
    type: typeOptions[moduleId][0] || '未分类',
    status: moduleId === 'users' ? '认证中' : '草稿',
    scope: moduleId === 'users' ? '全国' : '全国',
    owner: ownerOptions[moduleId][0] || '内容运营',
    tags: [],
    summary: '请补充内容摘要、来源和上架策略。',
    url: '',
    priority: moduleId === 'users' ? '中' : '常规',
    sortOrder: 0,
    payload: {},
    updatedAt: formatDate(new Date().toISOString()),
    createdRemote: false,
  }
}

export function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  return date.toLocaleString('zh-CN', { hour12: false })
}

export function fromApiRecord(record: AdminApiRecord): AdminRecord {
  return {
    id: record.id,
    title: record.title,
    type: record.type,
    status: record.status,
    scope: record.scope,
    owner: record.owner,
    tags: record.tags || [],
    summary: record.summary || '',
    url: record.url || '',
    priority: record.priority || '常规',
    sortOrder: record.sortOrder || 0,
    payload: record.payload || {},
    updatedAt: formatDate(record.updatedAt),
    createdRemote: true,
  }
}

export function toApiRecord(moduleId: EditableModuleId, item: AdminRecord) {
  return {
    id: item.id,
    module: moduleId,
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

export function mediaUrls(item: AdminRecord | null | undefined) {
  const urls = item?.payload?.imageUrls
  return Array.isArray(urls) ? urls.filter((url): url is string => typeof url === 'string' && Boolean(url.trim())) : []
}

export function resolveMediaUrl(apiBase: string, url: string) {
  if (!url || /^(https?:|data:|blob:)/i.test(url)) return url
  const normalized = url.startsWith('/') ? url : `/${url}`
  try {
    const apiOrigin = new URL(apiBase || defaultApiBase(), window.location.href).origin
    return `${apiOrigin}${normalized}`
  } catch {
    return normalized
  }
}

export function statusClass(status: string) {
  if (['已上架', '正常'].includes(status)) return 'ok'
  if (['待审核', '认证中', '需补充'].includes(status)) return 'pending'
  if (['需复核', '退回修改'].includes(status)) return 'warn'
  if (['下架', '认证驳回', '冻结'].includes(status)) return 'danger'
  return 'muted'
}

export function nextPriority(status: string) {
  if (['需复核', '冻结', '下架'].includes(status)) return '高'
  if (['待审核', '认证中', '需补充', '退回修改', '认证驳回'].includes(status)) return '中'
  return '常规'
}

export function workflowStepsFor(moduleId: EditableModuleId, status: string) {
  const userSteps = ['提交认证', '资料核验', '审核结论', '权限生效']
  const contentSteps = ['提交内容', '初审', '复核确认', '上架展示']
  const labels = moduleId === 'users' ? userSteps : contentSteps
  const indexMap =
    moduleId === 'users'
      ? { 认证中: 1, 需补充: 1, 认证驳回: 2, 正常: 3, 冻结: 3 }
      : { 草稿: 0, 退回修改: 0, 待审核: 1, 需复核: 2, 已上架: 3, 下架: 2 }
  const current = indexMap[status as keyof typeof indexMap] ?? 0
  return labels.map((label, index) => ({ label, active: index <= current }))
}

function workflowAction(
  id: string,
  label: string,
  nextStatus: string,
  description: string,
  placeholder: string,
  tone: AdminWorkflowAction['tone'] = '',
  requiresNote = false,
): AdminWorkflowAction {
  return { id, label, nextStatus, description, placeholder, tone, requiresNote }
}

export function workflowActionsFor(item: AdminRecord, moduleId: EditableModuleId): AdminWorkflowAction[] {
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
    return actions[item.status as keyof typeof actions] || []
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
  return actions[item.status as keyof typeof actions] || []
}

export function workflowTrail(item: AdminRecord) {
  return Array.isArray(item.payload?.workflow) ? (item.payload.workflow as WorkflowEntry[]) : []
}

export function permissionsForType(type: string) {
  if (type === '老师' || type === '规划师') return [...permissionPresets.expert.permissions]
  if (type === '管理员' || type === '运营人员') return [...permissionPresets.operator.permissions]
  return [...permissionPresets.basic.permissions]
}

export function presetForPermissions(permissions: string[]): PermissionPresetId {
  const sorted = [...permissions].sort().join(',')
  return (
    (Object.entries(permissionPresets).find(([, preset]) => [...preset.permissions].sort().join(',') === sorted)?.[0] as PermissionPresetId | undefined)
    || 'basic'
  )
}

export function optionsFor(kind: 'type' | 'status' | 'scope' | 'owner' | 'tags', moduleId: EditableModuleId, currentValue: string | string[]) {
  if (kind === 'type') return ensureOptions(typeOptions[moduleId] || ['未分类'], currentValue)
  if (kind === 'status') return ensureOptions(moduleId === 'users' ? statusOptions.users : statusOptions.default, currentValue)
  if (kind === 'scope') {
    const scopes =
      moduleId === 'categories' ? ['全站', '首页', '政策库', '建议库', '选科查询'] : moduleId === 'users' ? ['全国', ...provinceNames] : commonScopes
    return ensureOptions(scopes, currentValue)
  }
  if (kind === 'owner') return ensureOptions(ownerOptions[moduleId] || ['内容运营'], currentValue)
  return ensureOptions(tagOptions[moduleId] || [], currentValue)
}

function ensureOptions(options: string[], selected: string | string[]) {
  const values = Array.isArray(selected) ? selected : [selected]
  return Array.from(new Set([...options, ...values.filter(Boolean)]))
}

function normalizeApiBase(apiBase: string) {
  return (apiBase || defaultApiBase()).replace(/\/$/, '')
}

async function requestAdmin<T>(apiBase: string, path: string, options?: { method?: 'GET' | 'POST' | 'PUT' | 'DELETE'; data?: unknown; token?: string; headers?: Record<string, string> }) {
  try {
    const response = await axios.request<ApiEnvelope<T>>({
      baseURL: normalizeApiBase(apiBase),
      url: path,
      method: options?.method || 'GET',
      data: options?.data,
      headers: {
        ...(options?.token ? { 'X-Admin-Token': options.token } : {}),
        ...(options?.headers || {}),
      },
    })
    return response.data.data
  } catch (error) {
    if (axios.isAxiosError(error)) {
      const message = error.response?.data?.error?.message || error.message
      throw new Error(message)
    }
    throw error
  }
}

export async function adminLogin(apiBase: string, email: string, password: string) {
  return requestAdmin<AdminLoginResponse>(apiBase, '/admin/login', {
    method: 'POST',
    data: { email, password },
  })
}

export async function fetchAdminEmailConfig(apiBase: string, token: string) {
  return requestAdmin<AdminEmailConfig>(apiBase, '/admin/email-config', { token })
}

export async function fetchAdminContent(apiBase: string, token: string) {
  const data = await requestAdmin<{ records: AdminApiRecord[] }>(apiBase, '/admin/content', { token })
  return data.records || []
}

export async function fetchAdminAuditLogs(apiBase: string, token: string) {
  const data = await requestAdmin<{ logs: AdminAuditLog[] }>(apiBase, '/admin/audit-logs', { token })
  return data.logs || []
}

export async function saveAdminContent(apiBase: string, token: string, moduleId: EditableModuleId, item: AdminRecord) {
  const method = item.createdRemote ? 'PUT' : 'POST'
  const path = method === 'POST' ? '/admin/content' : `/admin/content/${encodeURIComponent(item.id)}`
  return requestAdmin<AdminApiRecord>(apiBase, path, {
    method,
    token,
    data: toApiRecord(moduleId, item),
  })
}

export async function saveAdminWorkflow(
  apiBase: string,
  token: string,
  item: AdminRecord,
  action: AdminWorkflowAction,
  note: string,
) {
  return requestAdmin<AdminApiRecord>(apiBase, `/admin/content/${encodeURIComponent(item.id)}/workflow`, {
    method: 'POST',
    token,
    data: {
      action: action.id,
      actionLabel: action.label,
      nextStatus: action.nextStatus,
      note: note || action.description,
      priority: item.priority || nextPriority(action.nextStatus),
      payload: item.payload || {},
    },
  })
}

export async function deleteAdminContent(apiBase: string, token: string, id: string) {
  return requestAdmin<void>(apiBase, `/admin/content/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    token,
  })
}

export async function uploadAdminImage(apiBase: string, token: string, file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return requestAdmin<{ url: string }>(apiBase, '/admin/uploads/images', {
    method: 'POST',
    token,
    data: formData,
  })
}

export async function sendAdminTestEmail(apiBase: string, token: string, email: string) {
  return requestAdmin<{ email: string; debugCode?: string }>(apiBase, '/admin/email-test', {
    method: 'POST',
    token,
    data: { email },
  })
}
