<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import type {
  AdminEmailConfig,
  AdminRecord,
  AdminSession,
  AdminSettings,
  EditableModuleId,
  ModuleId,
} from '../lib/admin'
import { appAssetUrl } from '../lib/runtime'
import {
  adminLogin,
  clearAdminSession,
  createInitialAdminState,
  createNewRecord,
  defaultApiBase,
  deleteAdminContent,
  editableModuleIds,
  fetchAdminAuditLogs,
  fetchAdminContent,
  fetchAdminEmailConfig,
  formatDate,
  fromApiRecord,
  loadAdminSession,
  loadAdminSettings,
  mediaUrls,
  moduleDefinitions,
  nextPriority,
  optionsFor,
  permissionLabels,
  permissionPresets,
  permissionsForType,
  presetForPermissions,
  resolveMediaUrl,
  saveAdminContent,
  saveAdminSession,
  saveAdminSettings,
  saveAdminWorkflow,
  sendAdminTestEmail,
  statusClass,
  uploadAdminImage,
  workflowActionsFor,
  workflowStepsFor,
  workflowTrail,
} from '../lib/admin'

const state = createInitialAdminState()

const activeModule = ref<ModuleId>('dashboard')
const selectedId = ref('')
const query = ref('')
const statusFilter = ref('全部')
const smtpConfig = ref<AdminEmailConfig | null>(null)
const apiConnected = ref(false)
const lastSyncMessage = ref('请登录并同步后端内容库')
const loginError = ref('')
const testEmail = ref('')
const testEmailResult = ref('')
const draftRecord = ref<AdminRecord | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)

const settings = ref<AdminSettings>(loadAdminSettings())
const adminSession = ref<AdminSession>(loadAdminSession())
const loginForm = ref({
  email: settings.value.adminEmail || '',
  password: '',
  apiBase: settings.value.apiBase || defaultApiBase(),
})
const pendingWorkflowActionId = ref('')
const workflowNote = ref('')
const workflowError = ref('')

const isLoggedIn = computed(() => adminSession.value.authenticated && Boolean(settings.value.adminToken))
const currentModule = computed(() => moduleDefinitions.find((item) => item.id === activeModule.value) || moduleDefinitions[0])
const activeRecords = computed(() => state.records[activeModule.value])
const currentRecord = computed(() => {
  if (!isEditableModule(activeModule.value)) return null
  return state.records[activeModule.value].find((item) => item.id === selectedId.value) || null
})
const allRecords = computed(() => editableModuleIds.flatMap((moduleId) => state.records[moduleId]))
const dashboardStats = computed(() => ({
  total: allRecords.value.length,
  published: allRecords.value.filter((item) => item.status === '已上架' || item.status === '正常').length,
  pending: allRecords.value.filter((item) => item.status === '待审核' || item.status === '认证中').length,
  review: allRecords.value.filter((item) => item.status === '需复核').length,
}))
const inventoryRows = computed(() =>
  editableModuleIds.map((moduleId) => {
    const module = moduleDefinitions.find((item) => item.id === moduleId)!
    const rows = state.records[moduleId]
    return {
      ...module,
      total: rows.length,
      published: rows.filter((row) => row.status === '已上架' || row.status === '正常').length,
      pending: rows.filter((row) => row.status === '待审核' || row.status === '认证中').length,
      review: rows.filter((row) => row.status === '需复核').length,
    }
  }),
)
const todoRows = computed(() =>
  editableModuleIds
    .flatMap((moduleId) =>
      state.records[moduleId]
        .filter((row) => ['待审核', '需复核', '认证中'].includes(row.status))
        .map((row) => ({ ...row, moduleId })),
    )
    .slice(0, 10),
)
const filteredRows = computed(() =>
  activeRecords.value.filter((item) => {
    const matchesQuery =
      !query.value ||
      [item.title, item.type, item.status, item.scope, item.owner, item.summary, ...(item.tags || [])]
        .join('')
        .toLowerCase()
        .includes(query.value.toLowerCase())
    const matchesStatus = statusFilter.value === '全部' || item.status === statusFilter.value
    return matchesQuery && matchesStatus
  }),
)
const statusOptionsForActiveModule = computed(() => ['全部', ...new Set(activeRecords.value.map((item) => item.status))])
const selectedWorkflowActions = computed(() =>
  currentRecord.value && isEditableModule(activeModule.value) ? workflowActionsFor(currentRecord.value, activeModule.value) : [],
)
const selectedWorkflowAction = computed(() => selectedWorkflowActions.value.find((item) => item.id === pendingWorkflowActionId.value) || null)
const selectedWorkflowSteps = computed(() =>
  currentRecord.value && isEditableModule(activeModule.value) ? workflowStepsFor(activeModule.value, currentRecord.value.status) : [],
)
const recentWorkflowTrail = computed(() => (currentRecord.value ? workflowTrail(currentRecord.value).slice(-4).reverse() : []))
const typeOptionsForDraft = computed(() =>
  draftRecord.value && isEditableModule(activeModule.value) ? optionsFor('type', activeModule.value, draftRecord.value.type) : [],
)
const scopeOptionsForDraft = computed(() =>
  draftRecord.value && isEditableModule(activeModule.value) ? optionsFor('scope', activeModule.value, draftRecord.value.scope) : [],
)
const ownerOptionsForDraft = computed(() =>
  draftRecord.value && isEditableModule(activeModule.value) ? optionsFor('owner', activeModule.value, draftRecord.value.owner) : [],
)
const tagOptionsForDraft = computed(() =>
  draftRecord.value && isEditableModule(activeModule.value) ? optionsFor('tags', activeModule.value, draftRecord.value.tags) : [],
)
const statusOptionsForDraft = computed(() =>
  draftRecord.value && isEditableModule(activeModule.value) ? optionsFor('status', activeModule.value, draftRecord.value.status) : [],
)
const currentImages = computed(() => mediaUrls(draftRecord.value))
const permissionEntries = Object.entries(permissionLabels)
const permissionPresetEntries = Object.entries(permissionPresets)
const currentPermissions = computed(() => {
  if (!draftRecord.value) return []
  const payloadPermissions = draftRecord.value.payload.permissions
  if (Array.isArray(payloadPermissions) && payloadPermissions.length) return payloadPermissions
  return permissionsForType(draftRecord.value.type)
})
const currentPermissionPreset = computed(() => {
  if (!draftRecord.value) return 'basic'
  const preset = draftRecord.value.payload.permissionPreset
  return preset || presetForPermissions(currentPermissions.value)
})

watch([activeModule, selectedId], syncDraftFromSelected, { immediate: true })

onMounted(async () => {
  if (!isLoggedIn.value) return
  try {
    smtpConfig.value = await fetchAdminEmailConfig(settings.value.apiBase, settings.value.adminToken)
  } catch {
    smtpConfig.value = null
  }
  await loadRemoteContent()
})

function isEditableModule(moduleId: ModuleId): moduleId is EditableModuleId {
  return editableModuleIds.includes(moduleId as EditableModuleId)
}

function persistSettings() {
  saveAdminSettings(settings.value)
}

function updateSettings(partial: Partial<AdminSettings>) {
  settings.value = {
    ...settings.value,
    ...partial,
  }
  persistSettings()
}

function syncDraftFromSelected() {
  draftRecord.value = currentRecord.value ? structuredClone(currentRecord.value) : null
}

function openModule(moduleId: ModuleId, recordId = '') {
  activeModule.value = moduleId
  selectedId.value = recordId
  query.value = ''
  statusFilter.value = '全部'
}

function openDrawer(recordId: string) {
  selectedId.value = recordId
}

function closeDrawer() {
  selectedId.value = ''
  draftRecord.value = null
}

async function handleLogin() {
  const apiBase = loginForm.value.apiBase.replace(/\/$/, '')
  const email = loginForm.value.email.trim().toLowerCase()
  const password = loginForm.value.password
  if (!email || !email.includes('@')) {
    loginError.value = '请输入邮箱形式的后台账号。'
    return
  }
  if (!password) {
    loginError.value = '请输入后台登录密码。'
    return
  }
  updateSettings({ apiBase, adminEmail: email, adminToken: '' })
  loginError.value = ''
  try {
    const loginData = await adminLogin(apiBase, email, password)
    updateSettings({ apiBase, adminEmail: email, adminToken: loginData.token })
    smtpConfig.value = await fetchAdminEmailConfig(apiBase, loginData.token)
    adminSession.value = { authenticated: true, mode: 'online', email, signedAt: new Date().toISOString() }
    saveAdminSession(adminSession.value)
    apiConnected.value = true
    lastSyncMessage.value = '登录成功，已连接后端'
    loginForm.value.password = ''
    await loadRemoteContent()
  } catch (error) {
    loginError.value = `登录失败：${toErrorMessage(error)}`
    adminSession.value = { authenticated: false }
    clearAdminSession()
  }
}

function logoutAdmin() {
  clearAdminSession()
  updateSettings({ adminToken: '' })
  adminSession.value = { authenticated: false }
  Object.assign(state, createInitialAdminState())
  apiConnected.value = false
  lastSyncMessage.value = '已退出后台'
  pendingWorkflowActionId.value = ''
  workflowNote.value = ''
  workflowError.value = ''
  closeDrawer()
}

async function loadRemoteContent() {
  if (!settings.value.adminToken) {
    apiConnected.value = false
    lastSyncMessage.value = '未登录后台，无法同步内容库'
    return
  }
  try {
    const [records, logs] = await Promise.all([
      fetchAdminContent(settings.value.apiBase, settings.value.adminToken),
      fetchAdminAuditLogs(settings.value.apiBase, settings.value.adminToken),
    ])
    for (const moduleId of editableModuleIds) {
      state.records[moduleId] = []
    }
    for (const record of records) {
      if (!isEditableModule(record.module)) continue
      state.records[record.module].push(fromApiRecord(record))
    }
    state.audit = logs.length ? logs.map((item) => `${formatDate(item.createdAt)} · ${item.detail}`) : []
    apiConnected.value = true
    lastSyncMessage.value = '已连接后端内容库'
    syncDraftFromSelected()
  } catch (error) {
    apiConnected.value = false
    lastSyncMessage.value = `后端连接失败：${toErrorMessage(error)}`
  }
}

async function createRecordNow() {
  if (!isEditableModule(activeModule.value)) return
  const item = createNewRecord(activeModule.value)
  state.records[activeModule.value].unshift(item)
  selectedId.value = item.id
  state.audit.push(`新建 ${currentModule.value.label} 条目：${item.title}`)
  try {
    const saved = await saveAdminContent(settings.value.apiBase, settings.value.adminToken, activeModule.value, item)
    Object.assign(item, fromApiRecord(saved))
    lastSyncMessage.value = '已保存到后端'
    apiConnected.value = true
    syncDraftFromSelected()
  } catch (error) {
    state.records[activeModule.value] = state.records[activeModule.value].filter((row) => row.id !== item.id)
    selectedId.value = ''
    apiConnected.value = false
    lastSyncMessage.value = `后端保存失败：${toErrorMessage(error)}`
  }
}

async function saveCurrentRecord() {
  if (!draftRecord.value || !currentRecord.value || !isEditableModule(activeModule.value)) return
  const snapshot = structuredClone(currentRecord.value)
  const nextRecord = structuredClone(draftRecord.value)
  nextRecord.updatedAt = formatDate(new Date().toISOString())
  if (activeModule.value === 'users') {
    const permissions = currentPermissions.value
    nextRecord.payload = {
      ...nextRecord.payload,
      permissionPreset: currentPermissionPreset.value,
      permissions,
    }
  }
  Object.assign(currentRecord.value, nextRecord)
  state.audit.push(`保存 ${nextRecord.title}，状态：${nextRecord.status}`)
  try {
    const saved = await saveAdminContent(settings.value.apiBase, settings.value.adminToken, activeModule.value, currentRecord.value)
    Object.assign(currentRecord.value, fromApiRecord(saved))
    lastSyncMessage.value = '已保存到后端'
    apiConnected.value = true
    closeDrawer()
  } catch (error) {
    Object.assign(currentRecord.value, snapshot)
    apiConnected.value = false
    lastSyncMessage.value = `后端保存失败：${toErrorMessage(error)}`
    syncDraftFromSelected()
  }
}

async function deleteCurrentRecord() {
  if (!currentRecord.value || !isEditableModule(activeModule.value)) return
  if (!window.confirm(`确认删除「${currentRecord.value.title}」？`)) return
  if (!settings.value.adminToken || !currentRecord.value.createdRemote) {
    apiConnected.value = false
    lastSyncMessage.value = '未连接后端，不能删除内容'
    return
  }
  try {
    await deleteAdminContent(settings.value.apiBase, settings.value.adminToken, currentRecord.value.id)
    state.records[activeModule.value] = state.records[activeModule.value].filter((row) => row.id !== currentRecord.value?.id)
    state.audit.push(`删除 ${currentRecord.value.title}`)
    lastSyncMessage.value = '已从后端删除'
    apiConnected.value = true
    closeDrawer()
  } catch (error) {
    apiConnected.value = false
    lastSyncMessage.value = `后端删除失败：${toErrorMessage(error)}`
  }
}

function openWorkflowConfirm(actionId: string) {
  pendingWorkflowActionId.value = actionId
  workflowNote.value = ''
  workflowError.value = ''
}

function closeWorkflowConfirm() {
  pendingWorkflowActionId.value = ''
  workflowNote.value = ''
  workflowError.value = ''
}

async function confirmWorkflowAction() {
  const item = currentRecord.value
  const action = selectedWorkflowAction.value
  if (!item || !action) return
  if (action.requiresNote && !workflowNote.value.trim()) {
    workflowError.value = '该动作必须填写处理意见，方便后续复盘和追责。'
    return
  }
  const snapshot = structuredClone(item)
  const previousStatus = item.status
  item.status = action.nextStatus
  item.priority = nextPriority(action.nextStatus)
  item.updatedAt = formatDate(new Date().toISOString())
  item.payload = {
    ...item.payload,
    workflow: [
      ...workflowTrail(item),
      {
        time: item.updatedAt,
        action: action.label,
        from: previousStatus,
        to: action.nextStatus,
        note: workflowNote.value.trim() || action.description,
        actor: 'admin',
      },
    ],
  }
  state.audit.push(`${action.label}「${item.title}」：${workflowNote.value.trim() || action.description}`)
  try {
    const saved = await saveAdminWorkflow(settings.value.apiBase, settings.value.adminToken, item, action, workflowNote.value.trim())
    Object.assign(item, fromApiRecord(saved))
    lastSyncMessage.value = '审核流程已写入后端'
    apiConnected.value = true
    closeWorkflowConfirm()
    syncDraftFromSelected()
  } catch (error) {
    Object.assign(item, snapshot)
    apiConnected.value = false
    lastSyncMessage.value = `流程保存失败：${toErrorMessage(error)}`
    syncDraftFromSelected()
  }
}

async function handleMediaUpload(event: Event) {
  const item = currentRecord.value
  if (!item || !isEditableModule(activeModule.value)) return
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  if (!files.length) return
  if (!settings.value.adminToken) {
    apiConnected.value = false
    lastSyncMessage.value = '未登录后台，图片上传失败'
    input.value = ''
    return
  }
  const snapshot = structuredClone(item)
  try {
    const uploads = await Promise.all(files.slice(0, Math.max(0, 9 - mediaUrls(item).length)).map((file) => uploadAdminImage(settings.value.apiBase, settings.value.adminToken, file)))
    const uploadedUrls = uploads.map((entry) => entry.url)
    item.payload = {
      ...item.payload,
      imageUrls: [...mediaUrls(item), ...uploadedUrls].slice(0, 9),
    }
    item.updatedAt = formatDate(new Date().toISOString())
    const saved = await saveAdminContent(settings.value.apiBase, settings.value.adminToken, activeModule.value, item)
    Object.assign(item, fromApiRecord(saved))
    state.audit.push(`上传 ${uploadedUrls.length} 张图片到「${item.title}」`)
    lastSyncMessage.value = `已上传并保存 ${uploadedUrls.length} 张图片`
    apiConnected.value = true
    syncDraftFromSelected()
  } catch (error) {
    Object.assign(item, snapshot)
    apiConnected.value = false
    lastSyncMessage.value = `图片上传失败：${toErrorMessage(error)}`
    syncDraftFromSelected()
  } finally {
    input.value = ''
  }
}

async function removeMediaImageAt(index: number) {
  const item = currentRecord.value
  if (!item || !isEditableModule(activeModule.value)) return
  const snapshot = structuredClone(item)
  item.payload = {
    ...item.payload,
    imageUrls: mediaUrls(item).filter((_, itemIndex) => itemIndex !== index),
  }
  item.updatedAt = formatDate(new Date().toISOString())
  try {
    const saved = await saveAdminContent(settings.value.apiBase, settings.value.adminToken, activeModule.value, item)
    Object.assign(item, fromApiRecord(saved))
    state.audit.push(`移除「${item.title}」的一张图片`)
    lastSyncMessage.value = '已移除图片并保存'
    apiConnected.value = true
    syncDraftFromSelected()
  } catch (error) {
    Object.assign(item, snapshot)
    apiConnected.value = false
    lastSyncMessage.value = `后端保存失败：${toErrorMessage(error)}`
    syncDraftFromSelected()
  }
}

function applyPermissionPreset(presetId: string) {
  if (!draftRecord.value) return
  const preset = permissionPresets[presetId as keyof typeof permissionPresets]
  if (!preset) return
  draftRecord.value.payload = {
    ...draftRecord.value.payload,
    permissionPreset: presetId as keyof typeof permissionPresets,
    permissions: [...preset.permissions],
  }
}

function togglePermission(permission: string, checked: boolean) {
  if (!draftRecord.value) return
  const next = new Set(currentPermissions.value)
  if (checked) next.add(permission)
  else next.delete(permission)
  const permissions = [...next]
  draftRecord.value.payload = {
    ...draftRecord.value.payload,
    permissionPreset: presetForPermissions(permissions),
    permissions,
  }
}

async function checkConnection() {
  updateSettings({ apiBase: settings.value.apiBase.replace(/\/$/, '') })
  try {
    smtpConfig.value = await fetchAdminEmailConfig(settings.value.apiBase, settings.value.adminToken)
  } catch (error) {
    smtpConfig.value = {
      enabled: false,
      host: '',
      port: 0,
      usernameConfigured: false,
      passwordConfigured: false,
      fromEmail: '',
      fromName: '',
      useTLS: false,
      startTLS: false,
      emailVerificationTTLMinutes: 0,
      missing: [toErrorMessage(error)],
    }
  }
  await loadRemoteContent()
}

async function sendTestMailNow() {
  testEmailResult.value = '正在发送...'
  try {
    const data = await sendAdminTestEmail(settings.value.apiBase, settings.value.adminToken, testEmail.value.trim())
    testEmailResult.value = data.debugCode ? `SMTP 未启用，本地调试验证码：${data.debugCode}` : `测试验证码已发送到 ${data.email}`
  } catch (error) {
    testEmailResult.value = toErrorMessage(error)
  }
}

function exportData() {
  const blob = new Blob([JSON.stringify(state, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `scf-admin-export-${Date.now()}.json`
  anchor.click()
  URL.revokeObjectURL(url)
}

function formatSmtpRows(data: AdminEmailConfig) {
  return [
    ['SMTP Host', data.host || '-'],
    ['SMTP Port', String(data.port || '-')],
    ['发件邮箱', data.fromEmail || '-'],
    ['发件名称', data.fromName || '-'],
    ['用户名', data.usernameConfigured ? '已配置' : '缺失'],
    ['密码', data.passwordConfigured ? '已配置' : '缺失'],
    ['TLS', data.useTLS ? '开启' : '关闭'],
    ['STARTTLS', data.startTLS ? '开启' : '关闭'],
    ['验证码有效期', `${data.emailVerificationTTLMinutes} 分钟`],
  ]
}

function toErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : '请求失败'
}
</script>

<template>
  <main v-if="!isLoggedIn" class="login-shell">
    <section class="login-panel">
      <div class="login-brand">
        <span class="admin-logo-frame"><img :src="appAssetUrl('/brand/logo-mark.png')" alt="选科π logo" /></span>
        <div>
          <small>Admin Console</small>
          <h1>选科π管理后台</h1>
          <p>登录后可管理内容上架、用户认证、政策库、建议库和系统配置。</p>
        </div>
      </div>

      <label>
        账号邮箱
        <input v-model="loginForm.email" type="email" autocomplete="username" placeholder="admin@example.com" />
      </label>
      <label>
        登录密码
        <input v-model="loginForm.password" type="password" autocomplete="current-password" placeholder="请输入后台登录密码" @keydown.enter="handleLogin" />
      </label>
      <details class="login-advanced">
        <summary>连接设置</summary>
        <label>
          API 地址
          <input v-model="loginForm.apiBase" />
        </label>
      </details>
      <p v-if="loginError" class="login-error">{{ loginError }}</p>
      <div class="login-actions">
        <button class="primary" type="button" @click="handleLogin">登录后台</button>
      </div>
    </section>
  </main>

  <div v-else class="admin-console-page">
    <aside class="sidebar">
      <div class="brand">
        <span class="admin-logo-frame"><img :src="appAssetUrl('/brand/logo-mark.png')" alt="选科π logo" /></span>
        <div>
          <strong>选科π</strong>
          <small>Admin Console</small>
        </div>
      </div>

      <nav>
        <button
          v-for="item in state.modules"
          :key="item.id"
          :class="{ active: activeModule === item.id }"
          type="button"
          @click="openModule(item.id)"
        >
          <i>{{ item.icon }}</i>
          <span>{{ item.label }}</span>
        </button>
      </nav>

      <div class="sidebar-foot">
        <span>统一后台</span>
        <strong>Vue + Go</strong>
      </div>
    </aside>

    <main class="main">
      <header class="topbar">
        <div>
          <small>当前位置 / {{ currentModule.label }}</small>
          <h1>{{ currentModule.label }}</h1>
          <p>{{ currentModule.description }}</p>
        </div>
        <div class="top-actions">
          <span class="session-pill">{{ adminSession.email || settings.adminEmail || '在线后台' }}</span>
          <span class="sync-pill" :class="apiConnected ? 'ok' : 'local'">{{ lastSyncMessage }}</span>
          <button type="button" @click="loadRemoteContent">同步后台</button>
          <button type="button" @click="exportData">导出配置</button>
          <button type="button" @click="logoutAdmin">退出</button>
        </div>
      </header>

      <template v-if="activeModule === 'dashboard'">
        <section class="metric-grid">
          <article>
            <small>内容总量</small>
            <strong>{{ dashboardStats.total }}</strong>
            <p>覆盖帖子、政策、省份资料、专业要求和趋势数据。</p>
          </article>
          <article>
            <small>已上架/正常</small>
            <strong>{{ dashboardStats.published }}</strong>
            <p>当前对前台可见或账号状态正常。</p>
          </article>
          <article>
            <small>待处理</small>
            <strong>{{ dashboardStats.pending }}</strong>
            <p>需要审核、认证或运营确认。</p>
          </article>
          <article>
            <small>需复核</small>
            <strong>{{ dashboardStats.review }}</strong>
            <p>政策时效或数据来源需要二次核验。</p>
          </article>
        </section>

        <section class="dashboard-grid">
          <article class="panel wide">
            <div class="panel-head">
              <h2>内容域台账</h2>
              <span>全站信息架构</span>
            </div>
            <table class="dense-table">
              <thead>
                <tr>
                  <th>模块</th>
                  <th>记录数</th>
                  <th>已上架/正常</th>
                  <th>待处理</th>
                  <th>需复核</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in inventoryRows" :key="item.id" data-open-module @click="openModule(item.id)">
                  <td>
                    <strong>{{ item.label }}</strong>
                    <small>{{ item.description }}</small>
                  </td>
                  <td>{{ item.total }}</td>
                  <td>{{ item.published }}</td>
                  <td>{{ item.pending }}</td>
                  <td>{{ item.review }}</td>
                </tr>
              </tbody>
            </table>
          </article>

          <article class="panel">
            <div class="panel-head">
              <h2>运营待办</h2>
              <span>按优先级</span>
            </div>
            <div class="todo-list">
              <button
                v-for="item in todoRows"
                :key="`${item.moduleId}-${item.id}`"
                type="button"
                @click="openModule(item.moduleId, item.id)"
              >
                <span class="status" :class="statusClass(item.status)">{{ item.status }}</span>
                <strong>{{ item.title }}</strong>
                <small>{{ item.type }} · {{ item.scope }}</small>
              </button>
              <p v-if="!todoRows.length" class="empty">暂无待办</p>
            </div>
          </article>

          <article class="panel">
            <div class="panel-head">
              <h2>审计记录</h2>
              <span>最近动作</span>
            </div>
            <ol class="audit-list">
              <li v-for="item in state.audit.slice(-8).reverse()" :key="item">{{ item }}</li>
              <li v-if="!state.audit.length">暂无审计记录</li>
            </ol>
          </article>
        </section>
      </template>

      <section v-else-if="activeModule === 'system'" class="system-grid">
        <article class="panel">
          <div class="panel-head">
            <h2>后端连接</h2>
            <span>Admin API</span>
          </div>
          <div class="status-card ok">当前账号：{{ adminSession.email || settings.adminEmail || '未登录' }}</div>
          <label>
            API 地址
            <input v-model="settings.apiBase" />
          </label>
          <button class="primary" type="button" @click="checkConnection">保存并检查连接</button>
        </article>

        <article class="panel">
          <div class="panel-head">
            <h2>SMTP 发信状态</h2>
            <span>邮箱验证码</span>
          </div>
          <div id="smtp-status" class="status-card" :class="smtpConfig?.enabled ? 'ok' : 'warn'">
            {{
              smtpConfig
                ? smtpConfig.enabled
                  ? 'SMTP 已配置，可发送验证码邮件'
                  : `SMTP 未完整配置：${(smtpConfig.missing || []).join('、') || '未知'}`
                : '等待检查'
            }}
          </div>
          <dl v-if="smtpConfig" class="field-list">
            <template v-for="[label, value] in formatSmtpRows(smtpConfig)" :key="label">
              <dt>{{ label }}</dt>
              <dd>{{ value }}</dd>
            </template>
          </dl>
        </article>

        <article class="panel wide">
          <div class="panel-head">
            <h2>发送测试验证码</h2>
            <span>管理端测试</span>
          </div>
          <div class="inline-form">
            <label>
              测试邮箱
              <input v-model="testEmail" type="email" placeholder="name@example.com" />
            </label>
            <button class="primary" type="button" @click="sendTestMailNow">发送测试邮件</button>
          </div>
          <p class="result-text">{{ testEmailResult }}</p>
        </article>
      </section>

      <section v-else class="workbench">
        <div class="toolbar">
          <label class="search">
            <span>搜索</span>
            <input v-model="query" placeholder="标题、地区、标签、负责人" />
          </label>
          <label>
            <span>状态</span>
            <select v-model="statusFilter">
              <option v-for="status in statusOptionsForActiveModule" :key="status" :value="status">{{ status }}</option>
            </select>
          </label>
          <button class="primary" type="button" @click="createRecordNow">新建条目</button>
        </div>

        <div class="table-wrap">
          <table class="content-table">
            <thead>
              <tr>
                <th>内容</th>
                <th>类型</th>
                <th>状态</th>
                <th>范围</th>
                <th>负责人/来源</th>
                <th>标签</th>
                <th>更新时间</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in filteredRows" :key="item.id" @click="openDrawer(item.id)">
                <td>
                  <strong>{{ item.title }}</strong>
                  <small>{{ item.summary }}</small>
                </td>
                <td>{{ item.type }}</td>
                <td><span class="status" :class="statusClass(item.status)">{{ item.status }}</span></td>
                <td>{{ item.scope }}</td>
                <td>{{ item.owner }}</td>
                <td>
                  <div class="tag-row">
                    <span v-for="tag in item.tags" :key="tag">{{ tag }}</span>
                  </div>
                </td>
                <td>{{ item.updatedAt }}</td>
                <td>
                  <button type="button" @click.stop="openDrawer(item.id)">编辑</button>
                </td>
              </tr>
              <tr v-if="!filteredRows.length">
                <td colspan="8" class="empty-cell">没有匹配记录</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </main>

    <aside v-if="draftRecord && isEditableModule(activeModule)" class="drawer">
      <div class="drawer-head">
        <div>
          <small>{{ currentModule.label }}</small>
          <h2>编辑条目</h2>
        </div>
        <button type="button" @click="closeDrawer">×</button>
      </div>

      <section class="workflow-card">
        <div class="workflow-head">
          <strong>{{ activeModule === 'users' ? '身份认证流程' : '内容审核流程' }}</strong>
          <span class="status" :class="statusClass(currentRecord?.status || '')">{{ currentRecord?.status }}</span>
        </div>
        <div class="workflow-steps">
          <span v-for="step in selectedWorkflowSteps" :key="step.label" :class="{ active: step.active }">{{ step.label }}</span>
        </div>
        <div class="workflow-actions">
          <button
            v-for="action in selectedWorkflowActions"
            :key="action.id"
            :class="action.tone"
            type="button"
            @click="openWorkflowConfirm(action.id)"
          >
            {{ action.label }}
          </button>
          <p v-if="!selectedWorkflowActions.length">当前状态暂无待处理动作。</p>
        </div>
        <div class="workflow-history">
          <strong>流程记录</strong>
          <ol v-if="recentWorkflowTrail.length">
            <li v-for="entry in recentWorkflowTrail" :key="`${entry.time}-${entry.action}`">
              <span>{{ entry.time }}</span>
              <p>{{ entry.action }}：{{ entry.note || '无补充说明' }}</p>
            </li>
          </ol>
          <p v-else>暂无审核记录，下一次确认动作会写入这里。</p>
        </div>
      </section>

      <label>
        {{ activeModule === 'users' ? '用户名称' : '标题' }}
        <input v-model="draftRecord.title" />
      </label>
      <label>
        {{ activeModule === 'users' ? '身份类型' : '类型' }}
        <select v-model="draftRecord.type">
          <option v-for="option in typeOptionsForDraft" :key="option" :value="option">{{ option }}</option>
        </select>
      </label>
      <label>
        状态
        <select v-model="draftRecord.status">
          <option v-for="option in statusOptionsForDraft" :key="option" :value="option">{{ option }}</option>
        </select>
      </label>
      <label>
        {{ activeModule === 'users' ? '所属省份/服务范围' : '范围/地区' }}
        <select v-model="draftRecord.scope">
          <option v-for="option in scopeOptionsForDraft" :key="option" :value="option">{{ option }}</option>
        </select>
      </label>
      <label>
        {{ activeModule === 'users' ? '账号来源' : '负责人/来源' }}
        <select v-model="draftRecord.owner">
          <option v-for="option in ownerOptionsForDraft" :key="option" :value="option">{{ option }}</option>
        </select>
      </label>
      <label>
        标签
        <select v-model="draftRecord.tags" class="multi-select" multiple :size="Math.min(Math.max(tagOptionsForDraft.length, 4), 7)">
          <option v-for="option in tagOptionsForDraft" :key="option" :value="option">{{ option }}</option>
        </select>
      </label>

      <section v-if="activeModule === 'users'" class="permission-card">
        <div class="permission-head">
          <strong>权限分类</strong>
          <span>{{ permissionPresets[currentPermissionPreset].label }}</span>
        </div>
        <div class="permission-presets">
          <button
            v-for="[presetId, preset] in permissionPresetEntries"
            :key="presetId"
            :class="{ active: presetId === currentPermissionPreset }"
            type="button"
            @click="applyPermissionPreset(presetId)"
          >
            {{ preset.label }}
          </button>
        </div>
        <div class="permission-grid">
          <label v-for="[permissionId, label] in permissionEntries" :key="permissionId">
            <input
              type="checkbox"
              :checked="currentPermissions.includes(permissionId)"
              @change="togglePermission(permissionId, ($event.target as HTMLInputElement).checked)"
            />
            <span>{{ label }}</span>
          </label>
        </div>
      </section>

      <section v-else class="media-card">
        <div class="media-head">
          <strong>本地图片</strong>
          <span>{{ currentImages.length }}/9</span>
        </div>
        <input ref="fileInput" type="file" accept="image/*" multiple hidden @change="handleMediaUpload" />
        <div class="media-actions">
          <button type="button" @click="fileInput?.click()">选择本地图片</button>
          <small>上传后会保存到当前条目的图片列表。</small>
        </div>
        <div v-if="currentImages.length" class="media-preview-grid">
          <figure v-for="(url, index) in currentImages" :key="`${url}-${index}`">
            <img :src="resolveMediaUrl(settings.apiBase, url)" :alt="`内容图片 ${index + 1}`" />
            <button type="button" aria-label="移除图片" @click="removeMediaImageAt(index)">×</button>
          </figure>
        </div>
        <p v-else class="media-empty">还没有本地图片。</p>
      </section>

      <label>
        摘要
        <textarea v-model="draftRecord.summary" />
      </label>
      <label>
        来源链接
        <input v-model="draftRecord.url" />
      </label>

      <div class="drawer-actions">
        <button class="primary" type="button" @click="saveCurrentRecord">保存基础信息</button>
        <button class="danger" type="button" @click="deleteCurrentRecord">删除</button>
      </div>
    </aside>

    <div v-if="selectedWorkflowAction && currentRecord" class="modal-backdrop">
      <section class="confirm-modal">
        <div class="modal-head">
          <small>{{ currentModule.label }}</small>
          <h2>{{ selectedWorkflowAction.label }}</h2>
        </div>
        <dl class="confirm-summary">
          <dt>处理对象</dt>
          <dd>{{ currentRecord.title }}</dd>
          <dt>当前状态</dt>
          <dd>{{ currentRecord.status }}</dd>
          <dt>确认后状态</dt>
          <dd>{{ selectedWorkflowAction.nextStatus }}</dd>
          <dt>流程影响</dt>
          <dd>{{ selectedWorkflowAction.description }}</dd>
        </dl>
        <label>
          处理意见
          <span>{{ selectedWorkflowAction.requiresNote ? '必填' : '选填' }}</span>
          <textarea v-model="workflowNote" :placeholder="selectedWorkflowAction.placeholder" />
        </label>
        <p v-if="workflowError" class="confirm-error">{{ workflowError }}</p>
        <div class="modal-actions">
          <button type="button" @click="closeWorkflowConfirm">取消</button>
          <button class="primary" type="button" @click="confirmWorkflowAction">确认执行</button>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.admin-console-page,
.login-shell {
  color: #0f172a;
  font-family:
    Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC",
    "Microsoft YaHei", sans-serif;
  font-synthesis: none;
  text-rendering: optimizeLegibility;
  -webkit-font-smoothing: antialiased;
}

.admin-console-page *,
.login-shell * {
  box-sizing: border-box;
}

button,
input,
select,
textarea {
  font: inherit;
}

button {
  cursor: pointer;
}

.admin-console-page {
  display: grid;
  grid-template-columns: 248px minmax(0, 1fr);
  min-height: 100vh;
  background: #f6f8fb;
}

.sidebar {
  position: sticky;
  top: 0;
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: 18px;
  height: 100vh;
  padding: 18px;
  border-right: 1px solid #dbe3ee;
  background: #0f172a;
  color: #e2e8f0;
}

.brand {
  display: grid;
  grid-template-columns: 42px 1fr;
  align-items: center;
  gap: 10px;
}

.brand strong {
  display: block;
  color: #fff;
}

.brand small {
  color: #94a3b8;
}

.sidebar nav {
  display: grid;
  align-content: start;
  gap: 6px;
}

.sidebar nav button {
  display: grid;
  grid-template-columns: 28px 1fr;
  align-items: center;
  height: 40px;
  padding: 0 10px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: #cbd5e1;
  text-align: left;
  font-weight: 850;
}

.sidebar nav button.active,
.sidebar nav button:hover {
  background: #1e293b;
  color: #fff;
}

.sidebar nav i {
  font-style: normal;
  color: #5eead4;
}

.sidebar-foot {
  display: grid;
  gap: 4px;
  padding: 12px;
  border: 1px solid #334155;
  border-radius: 8px;
  background: #111827;
}

.sidebar-foot span {
  color: #94a3b8;
  font-size: 12px;
}

.main {
  min-width: 0;
  padding: 18px 22px 32px;
}

.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 18px;
  margin-bottom: 14px;
  padding: 18px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.04);
}

.topbar small {
  color: #0f766e;
  font-weight: 900;
}

.topbar h1 {
  margin: 4px 0;
  color: #0f172a;
  font-size: 24px;
}

.topbar p {
  margin: 0;
  color: #64748b;
}

.top-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}

.session-pill,
.sync-pill {
  display: inline-flex;
  align-items: center;
  min-height: 30px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 900;
}

.session-pill {
  background: #eef2ff;
  color: #3730a3;
}

.sync-pill {
  max-width: 280px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sync-pill.ok {
  background: #ecfdf5;
  color: #0f766e;
}

.sync-pill.local {
  background: #f1f5f9;
  color: #64748b;
}

.top-actions button,
.toolbar button,
.drawer-actions button,
.inline-form button,
.panel button {
  height: 36px;
  padding: 0 12px;
  border: 1px solid #dbe3ee;
  border-radius: 7px;
  background: #fff;
  color: #0f766e;
  font-weight: 900;
  white-space: nowrap;
}

.primary {
  border-color: #0f9f7a;
  background: #0f9f7a;
  color: #fff;
}

.drawer-actions .danger {
  border-color: #fecaca;
  background: #fff1f2;
  color: #be123c;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.metric-grid article,
.panel,
.workbench {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.04);
}

.metric-grid article {
  display: grid;
  gap: 8px;
  padding: 16px;
}

.metric-grid small {
  color: #64748b;
  font-weight: 900;
}

.metric-grid strong {
  font-size: 28px;
}

.metric-grid p {
  margin: 0;
  color: #64748b;
  font-size: 13px;
  line-height: 1.5;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.7fr);
  gap: 12px;
}

.panel {
  display: grid;
  gap: 14px;
  padding: 16px;
}

.panel.wide {
  grid-row: span 2;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.panel-head h2 {
  margin: 0;
  font-size: 17px;
}

.panel-head span {
  color: #64748b;
  font-size: 12px;
  font-weight: 900;
}

.dense-table,
.content-table {
  width: 100%;
  border-collapse: collapse;
}

.dense-table th,
.dense-table td,
.content-table th,
.content-table td {
  padding: 10px;
  border-bottom: 1px solid #edf2f7;
  text-align: left;
  vertical-align: top;
}

.dense-table th,
.content-table th {
  color: #64748b;
  font-size: 12px;
  font-weight: 900;
}

.dense-table tr[data-open-module],
.content-table tbody tr {
  cursor: pointer;
}

.dense-table tr:hover,
.content-table tbody tr:hover {
  background: #f8fafc;
}

td strong {
  display: block;
}

td small {
  display: -webkit-box;
  overflow: hidden;
  color: #64748b;
  line-height: 1.45;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.todo-list {
  display: grid;
  align-content: start;
  gap: 8px;
  max-height: 430px;
  overflow-y: auto;
  padding-right: 4px;
}

.todo-list button {
  display: grid;
  align-content: start;
  justify-items: start;
  gap: 6px;
  min-height: 76px;
  padding: 11px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #fff;
  text-align: left;
}

.todo-list button:hover {
  border-color: #99f6e4;
}

.todo-list button strong {
  display: -webkit-box;
  overflow: hidden;
  color: #0f172a;
  font-size: 14px;
  line-height: 1.35;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.todo-list button small {
  display: block;
  color: #64748b;
  font-size: 12px;
  line-height: 1.4;
}

.status {
  display: inline-flex;
  align-items: center;
  width: max-content;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 900;
}

.status.ok {
  background: #ecfdf5;
  color: #0f766e;
}

.status.pending {
  background: #eff6ff;
  color: #1d4ed8;
}

.status.warn {
  background: #fffbeb;
  color: #92400e;
}

.status.muted {
  background: #f1f5f9;
  color: #64748b;
}

.status.danger {
  background: #fff1f2;
  color: #be123c;
}

.audit-list {
  display: grid;
  gap: 8px;
  margin: 0;
  padding-left: 18px;
  color: #475569;
  line-height: 1.5;
}

.workbench {
  overflow: hidden;
}

.toolbar {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) 180px auto;
  gap: 12px;
  align-items: end;
  padding: 14px;
  border-bottom: 1px solid #e2e8f0;
}

.toolbar label,
.drawer label,
.system-grid label,
.confirm-modal label {
  display: grid;
  gap: 6px;
  color: #334155;
  font-size: 12px;
  font-weight: 900;
}

.toolbar span {
  color: #64748b;
}

.toolbar input,
.toolbar select,
.drawer input,
.drawer select,
.drawer textarea,
.system-grid input,
.confirm-modal textarea,
.login-panel input {
  width: 100%;
  padding: 0 10px;
  border: 1px solid #dbe3ee;
  border-radius: 7px;
  outline: 0;
}

.toolbar input,
.toolbar select,
.drawer input,
.drawer select,
.system-grid input,
.login-panel input {
  height: 38px;
}

.drawer textarea,
.confirm-modal textarea {
  min-height: 96px;
  padding: 10px;
  resize: vertical;
}

.table-wrap {
  overflow: auto;
}

.content-table td:nth-child(1) {
  min-width: 300px;
}

.tag-row {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.tag-row span {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 7px;
  border-radius: 999px;
  background: #f0fdfa;
  color: #0f766e;
  font-size: 12px;
  font-weight: 850;
}

.empty,
.empty-cell {
  color: #94a3b8;
  text-align: center;
}

.drawer {
  position: fixed;
  top: 0;
  right: 0;
  z-index: 20;
  display: grid;
  align-content: start;
  gap: 12px;
  width: min(430px, 100vw);
  height: 100vh;
  padding: 18px;
  overflow: auto;
  border-left: 1px solid #dbe3ee;
  background: #fff;
  box-shadow: -24px 0 60px rgba(15, 23, 42, 0.14);
}

.drawer-head {
  display: flex;
  justify-content: space-between;
  align-items: start;
  gap: 12px;
}

.drawer-head small {
  color: #0f766e;
  font-weight: 900;
}

.drawer-head h2 {
  margin: 4px 0 0;
}

.drawer-head button {
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 7px;
  background: #f1f5f9;
  font-size: 22px;
}

.drawer-actions {
  display: flex;
  gap: 8px;
}

.drawer .multi-select {
  height: auto;
  min-height: 116px;
  padding: 7px 10px;
}

.system-grid {
  display: grid;
  grid-template-columns: 360px minmax(0, 1fr);
  gap: 12px;
}

.system-grid .wide {
  grid-column: 1 / -1;
}

.status-card {
  padding: 14px;
  border: 1px solid #dbe3ee;
  border-radius: 8px;
  background: #f8fafc;
  color: #475569;
  font-weight: 900;
}

.status-card.ok {
  border-color: rgba(15, 159, 122, 0.32);
  background: #ecfdf5;
  color: #0f766e;
}

.status-card.warn {
  border-color: rgba(245, 158, 11, 0.36);
  background: #fffbeb;
  color: #92400e;
}

.field-list {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 8px 12px;
  margin: 0;
  color: #475569;
}

.field-list dt {
  color: #64748b;
  font-weight: 850;
}

.field-list dd {
  margin: 0;
  font-weight: 800;
}

.inline-form {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 12px;
}

.result-text {
  min-height: 24px;
  margin: 0;
  color: #475569;
  font-weight: 850;
}

.workflow-card,
.media-card,
.permission-card {
  display: grid;
  gap: 12px;
  padding: 14px;
  border: 1px solid #dbe3ee;
  border-radius: 8px;
}

.workflow-card,
.media-card {
  background: #f8fafc;
}

.permission-card {
  background: #fff;
}

.workflow-head,
.media-head,
.permission-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.workflow-head strong,
.media-head strong,
.permission-head strong {
  font-size: 15px;
}

.permission-head span,
.media-head span {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 999px;
  background: #ecfdf5;
  color: #0f766e;
  font-size: 12px;
  font-weight: 900;
}

.workflow-steps {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}

.workflow-steps span {
  display: grid;
  place-items: center;
  min-height: 34px;
  padding: 0 8px;
  border: 1px solid #dbe3ee;
  border-radius: 7px;
  background: #fff;
  color: #64748b;
  font-size: 12px;
  font-weight: 900;
  text-align: center;
}

.workflow-steps span.active {
  border-color: rgba(15, 159, 122, 0.34);
  background: #ecfdf5;
  color: #0f766e;
}

.workflow-actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.workflow-actions button {
  min-height: 38px;
  padding: 8px 10px;
  border: 1px solid #dbe3ee;
  border-radius: 7px;
  background: #fff;
  color: #0f766e;
  font-weight: 900;
  text-align: center;
}

.workflow-actions .positive {
  border-color: rgba(15, 159, 122, 0.36);
  background: #0f9f7a;
  color: #fff;
}

.workflow-actions .warning {
  border-color: #fde68a;
  background: #fffbeb;
  color: #92400e;
}

.workflow-actions .danger {
  border-color: #fecaca;
  background: #fff1f2;
  color: #be123c;
}

.workflow-actions .primary {
  border-color: #bfdbfe;
  background: #eff6ff;
  color: #1d4ed8;
}

.workflow-actions p,
.workflow-history p {
  margin: 0;
  color: #64748b;
  font-size: 12px;
  line-height: 1.5;
}

.workflow-history {
  display: grid;
  gap: 8px;
  padding-top: 10px;
  border-top: 1px solid #e2e8f0;
}

.workflow-history > strong {
  font-size: 13px;
}

.workflow-history ol {
  display: grid;
  gap: 8px;
  margin: 0;
  padding-left: 18px;
}

.workflow-history li span {
  display: block;
  color: #64748b;
  font-size: 11px;
  font-weight: 850;
}

.workflow-history li p {
  margin: 2px 0 0;
  color: #334155;
}

.permission-presets,
.permission-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.permission-presets button {
  min-height: 36px;
  padding: 7px 10px;
  border: 1px solid #dbe3ee;
  border-radius: 7px;
  background: #f8fafc;
  color: #475569;
  font-weight: 900;
}

.permission-presets button.active {
  border-color: #0f9f7a;
  background: #0f9f7a;
  color: #fff;
}

.permission-grid label {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: center;
  min-height: 40px;
  gap: 8px;
  padding: 7px 9px;
  overflow: hidden;
  border: 1px solid #e2e8f0;
  border-radius: 7px;
  background: #f8fafc;
  color: #334155;
  font-size: 12px;
  font-weight: 850;
}

.permission-grid input[type='checkbox'] {
  width: 18px;
  height: 18px;
  margin: 0;
  accent-color: #0f9f7a;
}

.media-actions {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 10px;
}

.media-actions button {
  height: 36px;
  padding: 0 12px;
  border: 1px solid #0f9f7a;
  border-radius: 7px;
  background: #0f9f7a;
  color: #fff;
  font-weight: 900;
}

.media-actions small,
.media-empty {
  color: #64748b;
  line-height: 1.45;
}

.media-preview-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.media-preview-grid figure {
  position: relative;
  min-width: 0;
  aspect-ratio: 1;
  overflow: hidden;
  border: 1px solid #dbe3ee;
  border-radius: 8px;
  background: #fff;
}

.media-preview-grid img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.media-preview-grid button {
  position: absolute;
  top: 6px;
  right: 6px;
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 1px solid rgba(15, 23, 42, 0.12);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.92);
  color: #be123c;
  font-size: 16px;
  font-weight: 950;
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 40;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgba(15, 23, 42, 0.38);
  backdrop-filter: blur(2px);
}

.confirm-modal {
  display: grid;
  gap: 14px;
  width: min(560px, calc(100vw - 32px));
  padding: 18px;
  border: 1px solid #dbe3ee;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 26px 80px rgba(15, 23, 42, 0.24);
}

.modal-head small {
  color: #0f766e;
  font-weight: 900;
}

.modal-head h2 {
  margin: 4px 0 0;
  font-size: 20px;
}

.confirm-summary {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr);
  gap: 8px 12px;
  margin: 0;
  padding: 12px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
}

.confirm-summary dt {
  color: #64748b;
  font-size: 12px;
  font-weight: 900;
}

.confirm-summary dd {
  margin: 0;
  color: #0f172a;
  font-weight: 850;
  line-height: 1.45;
}

.confirm-modal label span {
  margin-left: 6px;
  color: #64748b;
  font-size: 12px;
}

.confirm-error {
  margin: 0;
  padding: 9px 10px;
  border: 1px solid #fecaca;
  border-radius: 7px;
  background: #fff1f2;
  color: #be123c;
  font-size: 13px;
  font-weight: 850;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.modal-actions button {
  height: 38px;
  padding: 0 14px;
  border: 1px solid #dbe3ee;
  border-radius: 7px;
  background: #fff;
  color: #0f766e;
  font-weight: 900;
}

.login-shell {
  display: grid;
  place-items: center;
  min-height: 100vh;
  padding: 28px;
  background: linear-gradient(135deg, #0f172a 0%, #111827 44%, #134e4a 100%);
}

.login-panel {
  display: grid;
  gap: 16px;
  width: min(460px, calc(100vw - 32px));
  padding: 24px;
  border: 1px solid rgba(226, 232, 240, 0.18);
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 30px 90px rgba(0, 0, 0, 0.28);
}

.login-brand {
  display: grid;
  grid-template-columns: 48px 1fr;
  align-items: start;
  gap: 12px;
}

.login-brand small {
  color: #0f766e;
  font-weight: 900;
}

.login-brand h1 {
  margin: 2px 0;
  color: #0f172a;
  font-size: 24px;
}

.login-brand p {
  margin: 0;
  color: #64748b;
  line-height: 1.55;
}

.login-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.login-actions button {
  height: 40px;
  padding: 0 14px;
  border: 1px solid #dbe3ee;
  border-radius: 7px;
  background: #fff;
  color: #0f766e;
  font-weight: 900;
}

.login-error {
  margin: 0;
  padding: 10px 12px;
  border: 1px solid #fecaca;
  border-radius: 7px;
  background: #fff1f2;
  color: #be123c;
  font-size: 13px;
  font-weight: 850;
}

.brand .admin-logo-frame,
.login-brand .admin-logo-frame {
  display: grid;
  place-items: center;
  overflow: hidden;
  border: 1px solid rgba(99, 102, 241, 0.18);
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 10px 24px rgba(67, 97, 238, 0.16);
}

.brand .admin-logo-frame {
  width: 44px;
  height: 44px;
}

.login-brand .admin-logo-frame {
  width: 58px;
  height: 58px;
}

.admin-logo-frame img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

@media (max-width: 1080px) {
  .admin-console-page {
    grid-template-columns: 1fr;
  }

  .sidebar {
    position: static;
    height: auto;
  }

  .sidebar nav {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .metric-grid,
  .dashboard-grid,
  .system-grid {
    grid-template-columns: 1fr;
  }

  .panel.wide {
    grid-row: auto;
  }

  .toolbar,
  .topbar {
    grid-template-columns: 1fr;
  }

  .topbar {
    display: grid;
  }
}

@media (max-width: 640px) {
  .main {
    padding: 12px;
  }

  .sidebar nav,
  .metric-grid,
  .workflow-steps,
  .workflow-actions,
  .permission-presets,
  .permission-grid,
  .modal-actions {
    grid-template-columns: 1fr;
  }

  .toolbar,
  .inline-form,
  .media-actions,
  .confirm-summary {
    grid-template-columns: 1fr;
  }

  .top-actions {
    display: grid;
    justify-content: stretch;
  }

  .session-pill,
  .sync-pill {
    justify-content: center;
    max-width: none;
  }

  .media-preview-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
