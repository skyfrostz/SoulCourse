<script setup lang="ts">
import { Check, ChevronLeft, LoaderCircle, RefreshCcw, Save, ShieldCheck, Sparkles, Trash2, WifiOff } from '@lucide/vue'
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useGlobalSearch } from '../composables/useGlobalSearch'
import { deleteMyAccount, fetchMyProfile, fetchMySessions, requestChoiceAdvice, revokeMySession, updateMyProfile } from '../lib/api'
import { subjectLabels, trackLabels } from '../lib/labels'
import { useForumStore } from '../stores/forum'
import { isNativeApp } from '../lib/mobile'
import { loadMobileTelemetryConsent, setMobileTelemetryConsent } from '../lib/mobileTelemetry'
import type { AccountSession, ChoiceAdvice, ChoiceProfile, Subject, Track } from '../types/forum'

const router = useRouter()
const forumStore = useForumStore()
const { runSearch } = useGlobalSearch()
const saved = ref(false)
const saving = ref(false)
const bio = ref('')
const advice = ref<ChoiceAdvice | null>(null)
const adviceLoading = ref(false)
const adviceError = ref('')
const profileLoading = ref(false)
const profileError = ref('')
const saveError = ref('')
const sessions = ref<AccountSession[]>([])
const sessionsLoading = ref(false)
const sessionsError = ref('')
const sessionActionId = ref<number | null>(null)
const sessionMessage = ref('')
const deletePassword = ref('')
const deleteConfirmation = ref('')
const deleteLoading = ref(false)
const deleteError = ref('')
const isOffline = ref(typeof navigator !== 'undefined' ? !navigator.onLine : false)
const diagnosticsConsent = ref(false)
const form = reactive<ChoiceProfile>({ ...forumStore.choiceProfile })
const subjects: Subject[] = ['chemistry', 'biology', 'politics', 'geography']
const tracks: Track[] = ['physics', 'history']

function toggleSubject(subject: Subject) {
  if (form.preferredSubjects.includes(subject)) {
    if (form.preferredSubjects.length > 1) {
      form.preferredSubjects = form.preferredSubjects.filter((item) => item !== subject)
    }
    return
  }
  form.preferredSubjects = [...form.preferredSubjects.slice(-1), subject]
}

async function save() {
  if (!forumStore.currentUser) {
    forumStore.openAuth('/settings')
    return
  }
  if (isOffline.value) {
    saveError.value = '当前网络不可用，恢复连接后再保存。'
    return
  }
  saving.value = true
  saveError.value = ''
  try {
    const profile = await updateMyProfile({ bio: bio.value, choiceProfile: { ...form } })
    forumStore.saveChoiceProfile(profile.choiceProfile)
    saved.value = true
    window.setTimeout(() => {
      saved.value = false
    }, 1800)
  } catch {
    saveError.value = '保存失败，请检查网络后重试。'
  } finally {
    saving.value = false
  }
}

async function generateAdvice() {
  if (!forumStore.currentUser) {
    forumStore.authOpen = true
    return
  }
  await save()
  adviceLoading.value = true
  adviceError.value = ''
  try {
    advice.value = await requestChoiceAdvice({ ...form }, '请基于我的个人画像给出选科下一步建议')
  } catch {
    adviceError.value = 'AI 建议暂时不可用，请稍后再试。'
  } finally {
    adviceLoading.value = false
  }
}

async function loadProfile() {
  if (!forumStore.currentUser) return
  profileLoading.value = true
  profileError.value = ''
  try {
    const profile = await fetchMyProfile()
    bio.value = profile.bio
    Object.assign(form, profile.choiceProfile)
  } catch {
    profileError.value = '个人资料加载失败，请稍后重试。'
  } finally {
    profileLoading.value = false
  }
}

async function loadSessions() {
  if (!forumStore.currentUser) return
  sessionsLoading.value = true
  sessionsError.value = ''
  try {
    sessions.value = await fetchMySessions()
  } catch {
    sessionsError.value = '会话列表加载失败，请稍后重试。'
  } finally {
    sessionsLoading.value = false
  }
}

async function revokeSession(session: AccountSession) {
  if (session.current || sessionActionId.value) return
  sessionActionId.value = session.id
  sessionsError.value = ''
  sessionMessage.value = ''
  try {
    await revokeMySession(session.id)
    sessions.value = sessions.value.filter((item) => item.id !== session.id)
    sessionMessage.value = '已撤销该设备会话。'
  } catch {
    sessionsError.value = '撤销失败，请刷新后重试。'
  } finally {
    sessionActionId.value = null
  }
}

async function deleteAccount() {
  if (deleteLoading.value) return
  deleteError.value = ''
  if (deleteConfirmation.value.trim() !== '注销账号') {
    deleteError.value = '请先输入“注销账号”确认操作。'
    return
  }
  if (!deletePassword.value) {
    deleteError.value = '请输入当前密码。'
    return
  }
  deleteLoading.value = true
  try {
    await deleteMyAccount(deletePassword.value)
    await forumStore.logout()
    await router.push('/')
  } catch {
    deleteError.value = '账号注销失败，请确认密码是否正确。'
  } finally {
    deleteLoading.value = false
  }
}

function formatSessionTime(value: string) {
  if (!value) return '未知时间'
  return new Date(value).toLocaleString('zh-CN')
}

function updateOnlineState() {
  isOffline.value = typeof navigator !== 'undefined' ? !navigator.onLine : false
}

onMounted(() => {
  window.addEventListener('online', updateOnlineState)
  window.addEventListener('offline', updateOnlineState)
  void loadProfile()
  void loadSessions()
  if (isNativeApp) {
    void loadMobileTelemetryConsent().then(async () => {
      const { Preferences } = await import('@capacitor/preferences')
      const result = await Preferences.get({ key: 'mobile_diagnostics_consent' })
      diagnosticsConsent.value = result.value === 'true'
    })
  }
})

async function updateDiagnosticsConsent(value: boolean) {
  diagnosticsConsent.value = value
  await setMobileTelemetryConsent(value)
}

onBeforeUnmount(() => {
  window.removeEventListener('online', updateOnlineState)
  window.removeEventListener('offline', updateOnlineState)
})

function searchSuggestion(keyword: string) {
  void runSearch(keyword)
}
</script>

<template>
  <main class="detail-page">
    <button class="back-link" @click="router.push('/')"><ChevronLeft :size="17" /> 返回论坛</button>
    <section class="settings-hero">
      <div>
        <div class="breadcrumb">个人中心 / 设置</div>
        <h1>个人信息与选科画像</h1>
        <p>完善这些信息后，后续可以据此推荐更贴近你的组合、帖子和数据建议。</p>
      </div>
      <button class="write-button" type="button" :disabled="saving || profileLoading || isOffline" @click="save">
        <Save :size="16" /> {{ saving ? '保存中' : '保存设置' }}
      </button>
    </section>

    <section v-if="!forumStore.currentUser" class="empty-state detail-empty-state public-page-state">
      <Sparkles :size="34" />
      <h1>登录后完善画像</h1>
      <p>画像、AI 建议和保存结果会跟随账号同步。登录后可以继续编辑。</p>
      <button class="primary-wide compact" type="button" @click="forumStore.openAuth('/settings')">登录 / 注册</button>
    </section>

    <section v-else-if="profileLoading" class="empty-state detail-empty-state public-page-state">
      <RefreshCcw class="state-spin" :size="34" />
      <h1>正在加载个人设置</h1>
      <p>请稍等，正在同步你的选科画像。</p>
    </section>

    <section v-else-if="profileError || isOffline" class="empty-state detail-empty-state public-page-state">
      <WifiOff v-if="isOffline" :size="34" />
      <Sparkles v-else :size="34" />
      <h1>{{ isOffline ? '当前网络不可用' : '个人设置加载失败' }}</h1>
      <p>{{ isOffline ? '恢复网络后再继续编辑，避免保存失败。' : profileError }}</p>
      <button v-if="!isOffline" class="primary-wide compact" type="button" @click="loadProfile">重试</button>
    </section>

    <section v-else class="settings-grid">
      <form class="settings-card" @submit.prevent="save">
        <h2>基础信息</h2>
        <div class="settings-fields">
          <label class="full-field">个人简介<input v-model="bio" maxlength="300" placeholder="简单介绍你的年级、目标或正在纠结的问题" /></label>
          <label>姓名/称呼<input v-model="form.realName" placeholder="例如：小周" /></label>
          <label>所在城市<input v-model="form.city" placeholder="例如：杭州" /></label>
          <label>学校类型<input v-model="form.schoolType" placeholder="重点高中 / 普通高中 / 国际部" /></label>
          <label>年级排名<input v-model="form.gradeRank" placeholder="例如：前 20% / 年级 80 名" /></label>
          <label>MBTI<input v-model="form.mbti" placeholder="例如：INTJ / ENFP" /></label>
          <label>目标城市<input v-model="form.targetCities" placeholder="例如：上海、杭州、南京" /></label>
        </div>
      </form>

      <form class="settings-card" @submit.prevent="save">
        <h2>成绩与学科稳定性</h2>
        <div class="score-grid">
          <label>物理<input v-model="form.physicsScore" placeholder="分数/等级" /></label>
          <label>历史<input v-model="form.historyScore" placeholder="分数/等级" /></label>
          <label>化学<input v-model="form.chemistryScore" placeholder="分数/等级" /></label>
          <label>生物<input v-model="form.biologyScore" placeholder="分数/等级" /></label>
          <label>政治<input v-model="form.politicsScore" placeholder="分数/等级" /></label>
          <label>地理<input v-model="form.geographyScore" placeholder="分数/等级" /></label>
        </div>
        <label class="full-field">学科稳定性
          <select v-model="form.subjectStability">
            <option>很稳定</option>
            <option>中等</option>
            <option>波动较大</option>
          </select>
        </label>
      </form>

      <form class="settings-card settings-card-wide" @submit.prevent="save">
        <h2>选科倾向与推荐偏好</h2>
        <div class="preference-row">
          <button
            v-for="track in tracks"
            :key="track"
            type="button"
            :class="{ active: form.preferredTrack === track }"
            @click="form.preferredTrack = track"
          >
            {{ trackLabels[track] }}
          </button>
        </div>
        <div class="publish-subjects">
          <button
            v-for="subject in subjects"
            :key="subject"
            type="button"
            :class="{ active: form.preferredSubjects.includes(subject) }"
            @click="toggleSubject(subject)"
          >
            {{ subjectLabels[subject] }}
          </button>
        </div>
        <div class="settings-fields">
          <label>目标专业方向<input v-model="form.targetMajors" placeholder="例如：临床医学、计算机、法学、师范" /></label>
          <label>学习风格
            <select v-model="form.learningStyle">
              <option>理解推导型</option>
              <option>记忆积累型</option>
              <option>刷题反馈型</option>
              <option>项目探索型</option>
            </select>
          </label>
          <label>压力承受
            <select v-model="form.pressureTolerance">
              <option>较低</option>
              <option>中等</option>
              <option>较高</option>
            </select>
          </label>
          <label>推荐重点
            <select v-model="form.recommendationFocus">
              <option>专业覆盖率优先</option>
              <option>赋分风险更低</option>
              <option>学习强度更均衡</option>
              <option>就业方向更清晰</option>
            </select>
          </label>
        </div>
      </form>

      <section class="settings-card settings-card-wide account-security-card">
        <div class="settings-card-heading">
          <div>
            <h2><ShieldCheck :size="18" /> 账号安全</h2>
            <p>查看当前账号登录设备。发现异常时，可以撤销其它设备的会话。</p>
          </div>
          <button type="button" :disabled="sessionsLoading" @click="loadSessions">
            <RefreshCcw v-if="!sessionsLoading" :size="15" />
            <LoaderCircle v-else class="state-spin" :size="15" />
            刷新
          </button>
        </div>
        <p v-if="sessionsError" class="form-error">{{ sessionsError }}</p>
        <p v-if="sessionMessage" class="form-success">{{ sessionMessage }}</p>
        <div v-if="sessionsLoading && !sessions.length" class="session-state">
          <LoaderCircle class="state-spin" :size="20" /> 正在加载会话
        </div>
        <div v-else-if="!sessions.length" class="session-state">暂无可显示的登录会话。</div>
        <div v-else class="session-list">
          <article v-for="session in sessions" :key="session.id" class="session-item">
            <div>
              <strong>{{ session.current ? '当前设备' : `设备会话 #${session.id}` }}</strong>
              <small>创建：{{ formatSessionTime(session.createdAt) }} · 过期：{{ formatSessionTime(session.expiresAt) }}</small>
            </div>
            <span v-if="session.current" class="session-badge">当前</span>
            <button v-else type="button" :disabled="sessionActionId === session.id" @click="revokeSession(session)">
              {{ sessionActionId === session.id ? '撤销中' : '撤销' }}
            </button>
          </article>
        </div>
      </section>

      <section v-if="isNativeApp" class="settings-card settings-card-wide diagnostics-card">
        <div>
          <h2><ShieldCheck :size="18" /> 匿名诊断</h2>
          <p>只帮助我们定位崩溃和网络问题，不上传帖子、私信、搜索词或设备标识。</p>
        </div>
        <label class="settings-toggle">
          <input :checked="diagnosticsConsent" type="checkbox" @change="updateDiagnosticsConsent(($event.target as HTMLInputElement).checked)" />
          <span>{{ diagnosticsConsent ? '已开启' : '未开启' }}</span>
        </label>
      </section>

      <section class="settings-card settings-card-wide danger-zone-card">
        <div>
          <h2><Trash2 :size="18" /> 注销账号</h2>
          <p>注销后会清除邮箱、密码和当前所有登录会话；已发布内容会保留为社区上下文，但账号无法再登录。</p>
        </div>
        <div class="settings-fields">
          <label>输入“注销账号”确认<input v-model="deleteConfirmation" autocomplete="off" placeholder="注销账号" /></label>
          <label>当前密码<input v-model="deletePassword" type="password" autocomplete="current-password" placeholder="请输入当前密码" /></label>
        </div>
        <p v-if="deleteError" class="form-error">{{ deleteError }}</p>
        <button class="danger-action-button" type="button" :disabled="deleteLoading || deleteConfirmation.trim() !== '注销账号' || !deletePassword" @click="deleteAccount">
          <Trash2 :size="15" /> {{ deleteLoading ? '注销中' : '确认注销账号' }}
        </button>
      </section>

      <aside class="settings-summary">
        <h2>画像摘要</h2>
        <p><strong>{{ form.realName || forumStore.currentUser?.nickname || '未填写称呼' }}</strong></p>
        <p>{{ trackLabels[form.preferredTrack] }} · {{ form.preferredSubjects.map((item) => subjectLabels[item]).join(' + ') }}</p>
        <p>MBTI：{{ form.mbti || '未填写' }}</p>
        <p>目标：{{ form.targetMajors || '未填写目标专业' }}</p>
        <span v-if="saved"><Check :size="16" /> 已保存到账号</span>
        <div class="ai-advice-card">
          <div>
            <strong><Sparkles :size="16" /> AI 个性化建议</strong>
            <small>基于当前画像，输出简短决策提醒</small>
          </div>
          <button type="button" :disabled="adviceLoading" @click="generateAdvice">
            {{ adviceLoading ? '分析中...' : '生成建议' }}
          </button>
          <p v-if="saveError" class="ai-advice-error">{{ saveError }}</p>
          <p v-if="adviceError" class="ai-advice-error">{{ adviceError }}</p>
          <template v-if="advice">
            <blockquote>{{ advice.summary }}</blockquote>
            <h3>重点提醒</h3>
            <ul>
              <li v-for="item in advice.risks" :key="item">{{ item }}</li>
            </ul>
            <h3>下一步</h3>
            <ol>
              <li v-for="item in advice.actions" :key="item">{{ item }}</li>
            </ol>
            <div class="ai-query-row">
              <button
                v-for="item in advice.querySuggestions"
                :key="item"
                type="button"
                @click="searchSuggestion(item)"
              >
                {{ item }}
              </button>
            </div>
            <small class="ai-source">{{ advice.source === 'ai' ? 'AI 生成，请结合官方政策核对' : '本地兜底建议' }}</small>
          </template>
        </div>
      </aside>
    </section>
  </main>
</template>
