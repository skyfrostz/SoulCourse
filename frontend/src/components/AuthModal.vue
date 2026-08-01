<script setup lang="ts">
import { ArrowRight, AtSign, CheckCircle2, CircleAlert, Eye, EyeOff, GraduationCap, LoaderCircle, LockKeyhole, Mail, ShieldCheck, Sparkles, UserRound, X } from '@lucide/vue'
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { forgotPassword, login, register, resetPassword, sendEmailVerificationCode } from '../lib/api'
import { ApiClientError } from '../lib/api-contract'
import { currentProfileAuthRedirect, useForumStore } from '../stores/forum'
import type { Role } from '../types/forum'
import { useOnlineState } from '../composables/useOnlineState'

const forumStore = useForumStore()
const router = useRouter()
const { isOffline } = useOnlineState()
const mode = ref<'login' | 'register' | 'reset'>('login')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const verificationCode = ref('')
const nickname = ref('高一新用户')
const role = ref<Role>('student')
const province = ref('广东')
const grade = ref('高一')
const error = ref('')
const success = ref('')
const codeMessage = ref('')
const codeState = ref<'idle' | 'sending' | 'sent' | 'warning' | 'error'>('idle')
const loading = ref(false)
const codeLoading = ref(false)
const codeCountdown = ref(0)
const authFormPanel = ref<HTMLElement | null>(null)
const codeCountdownLabel = computed(() => {
  if (codeCountdown.value < 60) return `${codeCountdown.value}s`
  const minutes = Math.floor(codeCountdown.value / 60)
  const seconds = String(codeCountdown.value % 60).padStart(2, '0')
  return `${minutes}:${seconds}`
})
const codeFeedbackTitle = computed(() => {
  if (codeState.value === 'sending') return '正在安全发送'
  if (codeState.value === 'sent') return '验证码已发送'
  if (codeState.value === 'warning') return '请稍后再试'
  return '暂时无法发送'
})
const codeButtonLabel = computed(() => {
  if (codeLoading.value) return '正在发送'
  if (codeCountdown.value > 0) return codeCountdownLabel.value
  return '获取验证码'
})
let countdownTimer: number | undefined
const gradeOptions = ['初中', '高一', '高二', '高三']
const provinceOptions = [
  '北京',
  '天津',
  '河北',
  '山西',
  '内蒙古',
  '辽宁',
  '吉林',
  '黑龙江',
  '上海',
  '江苏',
  '浙江',
  '安徽',
  '福建',
  '江西',
  '山东',
  '河南',
  '湖北',
  '湖南',
  '广东',
  '广西',
  '海南',
  '重庆',
  '四川',
  '贵州',
  '云南',
  '西藏',
  '陕西',
  '甘肃',
  '青海',
  '宁夏',
  '新疆',
  '香港',
  '澳门',
  '台湾',
]

async function submit() {
  if (loading.value) return
  if (isOffline.value) {
    error.value = '当前网络不可用，请恢复网络后重试。'
    return
  }
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    if ((mode.value === 'register' || mode.value === 'reset') && password.value !== confirmPassword.value) {
      error.value = '两次输入的密码不一致'
      return
    }
    if (mode.value === 'reset') {
      await resetPassword({
        email: email.value,
        verificationCode: verificationCode.value,
        password: password.value,
      })
      mode.value = 'login'
      password.value = ''
      confirmPassword.value = ''
      verificationCode.value = ''
      success.value = '密码已重置，请使用新密码登录。'
      return
    }
    const session =
      mode.value === 'login'
        ? await login(email.value, password.value)
        : await register({
            email: email.value,
            password: password.value,
            verificationCode: verificationCode.value,
            nickname: nickname.value,
            role: role.value,
            province: province.value,
            grade: grade.value,
          })
    const redirect = forumStore.authRedirect
    forumStore.authRedirect = ''
    forumStore.setSession(session)
    if (redirect) {
      const destination = redirect === currentProfileAuthRedirect
        ? `/users/${encodeURIComponent(session.user.nickname)}`
        : redirect
      await router.push(destination)
    }
  } catch (err) {
    const apiError = err instanceof ApiClientError ? err : null
    if (apiError && !apiError.status) {
      error.value = isOffline.value ? '当前网络不可用，请恢复网络后重试。' : '暂时无法连接服务，请稍后重试。'
    } else {
      if (mode.value === 'login') {
        error.value = '邮箱或密码不正确'
      } else if (mode.value === 'reset') {
        if (apiError?.code === 'invalid_verification_code') {
          error.value = '验证码错误或已过期，请重新获取'
        } else if (apiError?.code === 'not_found') {
          error.value = '没有找到这个邮箱对应的账号'
        } else if (apiError?.code === 'invalid_payload') {
          error.value = apiError.message || '请检查邮箱、验证码和新密码'
        } else {
          error.value = '密码重置失败，请稍后重试'
        }
      } else if (apiError?.code === 'invalid_verification_code') {
        error.value = '验证码错误或已过期，请重新获取'
      } else if (apiError?.code === 'email_already_exists') {
        error.value = '这个邮箱已注册，请直接登录'
      } else if (apiError?.code === 'invalid_payload') {
        error.value = apiError.message || '请检查邮箱、密码和昵称是否填写完整'
      } else {
        error.value = '注册失败，请检查邮箱是否已存在'
      }
    }
  } finally {
    loading.value = false
  }
}

async function requestCode() {
  if (codeLoading.value || codeCountdown.value > 0) return
  error.value = ''
  success.value = ''
  codeMessage.value = ''
  if (!email.value) {
    codeState.value = 'error'
    codeMessage.value = '请先填写接收验证码的邮箱地址。'
    return
  }
  if (isOffline.value) {
    codeState.value = 'error'
    codeMessage.value = '当前网络不可用，请恢复网络后再获取验证码。'
    return
  }
  codeLoading.value = true
  codeState.value = 'sending'
  codeMessage.value = `正在发送至 ${maskEmail(email.value)}，请稍候。`
  try {
    const result = mode.value === 'reset'
      ? await forgotPassword(email.value)
      : await sendEmailVerificationCode(email.value)
    startCountdown(result.retryAfterSeconds)
    const quotaMessage = `本邮箱本小时还可发送 ${result.hourlyRemaining} 次`
    codeState.value = 'sent'
    codeMessage.value = result.debugCode
      ? `本地调试验证码：${result.debugCode}（${quotaMessage}）`
      : `${maskEmail(email.value)} · 10 分钟内有效 · ${quotaMessage}`
  } catch (err) {
    const apiError = err instanceof ApiClientError ? err : null
    if (apiError && !apiError.status) {
      codeState.value = 'error'
      codeMessage.value = isOffline.value ? '当前网络不可用，请恢复网络后再获取验证码。' : '暂时无法连接邮件服务，请稍后重试。'
    } else if (apiError?.code === 'email_verification_rate_limited') {
      const retryAfterSeconds = Math.max(1, Number(apiError.details?.retryAfterSeconds) || 60)
      const hourlyRemaining = Number(apiError.details?.hourlyRemaining ?? 0)
      startCountdown(retryAfterSeconds)
      codeState.value = 'warning'
      codeMessage.value =
        hourlyRemaining === 0
          ? `本邮箱本小时发送次数已用完，请在 ${formatRetryAfter(retryAfterSeconds)}后重试`
          : `请求过于频繁，请在 ${formatRetryAfter(retryAfterSeconds)}后重试（本小时剩余 ${hourlyRemaining} 次）`
    } else if (apiError?.code === 'invalid_email') {
      codeState.value = 'error'
      codeMessage.value = '请填写可接收验证码的邮箱地址。'
    } else {
      codeState.value = 'error'
      codeMessage.value = '邮件服务暂时不可用，请稍后重新发送。'
    }
  } finally {
    codeLoading.value = false
  }
}

function maskEmail(value: string) {
  const [localPart, domain] = value.trim().split('@')
  if (!localPart || !domain) return value.trim()
  const visible = localPart.slice(0, Math.min(2, localPart.length))
  return `${visible}${'•'.repeat(Math.max(3, localPart.length - visible.length))}@${domain}`
}

function startCountdown(seconds: number) {
  codeCountdown.value = Math.max(1, Math.ceil(seconds))
  if (countdownTimer) window.clearInterval(countdownTimer)
  countdownTimer = window.setInterval(() => {
    codeCountdown.value -= 1
    if (codeCountdown.value <= 0 && countdownTimer) {
      window.clearInterval(countdownTimer)
      countdownTimer = undefined
    }
  }, 1000)
}

function formatRetryAfter(seconds: number) {
  if (seconds < 60) return `${seconds} 秒`
  return `${Math.ceil(seconds / 60)} 分钟`
}

watch(email, () => {
  error.value = ''
  success.value = ''
  codeMessage.value = ''
  codeState.value = 'idle'
  verificationCode.value = ''
  codeCountdown.value = 0
  if (countdownTimer) {
    window.clearInterval(countdownTimer)
    countdownTimer = undefined
  }
})

watch(mode, () => {
  error.value = ''
  codeMessage.value = ''
  codeState.value = 'idle'
  verificationCode.value = ''
  codeCountdown.value = 0
  if (countdownTimer) {
    window.clearInterval(countdownTimer)
    countdownTimer = undefined
  }
})

watch(mode, async () => {
  await nextTick()
  if (authFormPanel.value) authFormPanel.value.scrollTop = 0
})

function close() {
  if (loading.value || codeLoading.value) return
  forumStore.authRedirect = ''
  forumStore.authMessage = ''
  forumStore.authOpen = false
}

function switchMode(nextMode: 'login' | 'register' | 'reset') {
  if (loading.value || codeLoading.value) return
  mode.value = nextMode
}

onUnmounted(() => {
  if (countdownTimer) window.clearInterval(countdownTimer)
})
</script>

<template>
  <div class="modal-backdrop auth-experience-backdrop">
    <section class="auth-experience" :class="{ 'is-registering': mode === 'register' || mode === 'reset' }" role="dialog" aria-modal="true" :aria-label="mode === 'login' ? '登录账号' : mode === 'register' ? '注册账号' : '重置密码'">
      <aside class="auth-brand-panel">
        <div class="auth-brand-mark"><Sparkles :size="27" /></div>
        <div>
          <span class="auth-eyebrow">SOULCOURSE · 选科π</span>
          <h2>让每一次选择<br />都有真实依据。</h2>
          <p>登录后保存选科画像、收藏关键资料，并及时收到与你有关的讨论动态。</p>
        </div>
        <div class="auth-benefits">
          <span><ShieldCheck :size="17" /> 画像仅与你的账号绑定</span>
          <span><GraduationCap :size="17" /> 持续沉淀选科决策记录</span>
        </div>
      </aside>

      <button class="auth-close-button" type="button" aria-label="关闭" :disabled="loading || codeLoading" @click="close"><X :size="19" /></button>
      <div ref="authFormPanel" class="auth-form-panel">
        <header class="auth-form-heading">
          <span>{{ mode === 'login' ? '欢迎回来' : mode === 'register' ? '从这里开始' : '找回访问权限' }}</span>
          <h2>{{ mode === 'login' ? '登录你的账号' : mode === 'register' ? '创建选科档案' : '重置账号密码' }}</h2>
          <p>{{ mode === 'login' ? '继续查看你的画像、收藏与消息。' : mode === 'register' ? '注册后即可保存个人信息和选科画像。' : '通过邮箱验证码设置一个新密码。' }}</p>
        </header>

        <div class="auth-tabs" aria-label="登录方式">
          <button :class="{ active: mode === 'login' }" type="button" :disabled="loading || codeLoading" @click="switchMode('login')">登录</button>
          <button :class="{ active: mode === 'register' }" type="button" :disabled="loading || codeLoading" @click="switchMode('register')">注册</button>
          <button :class="{ active: mode === 'reset' }" type="button" :disabled="loading || codeLoading" @click="switchMode('reset')">找回</button>
        </div>

        <form class="auth-form auth-premium-form" :aria-busy="loading || codeLoading" @submit.prevent="submit">
          <p v-if="forumStore.authMessage" class="form-error auth-error-banner" role="alert">{{ forumStore.authMessage }}</p>
          <p v-if="isOffline" class="form-error auth-error-banner" role="status">当前网络不可用，恢复网络后可继续。</p>
          <label>
            <span>邮箱</span>
            <span class="auth-input-shell"><Mail :size="17" /><input v-model="email" type="email" autocomplete="email" required placeholder="name@example.com" /></span>
          </label>
          <label>
            <span>密码</span>
            <span class="auth-input-shell">
              <LockKeyhole :size="17" />
              <input v-model="password" :type="showPassword ? 'text' : 'password'" :autocomplete="mode === 'login' ? 'current-password' : 'new-password'" required minlength="6" :placeholder="mode === 'reset' ? '输入新密码' : '至少 6 位字符'" aria-label="密码" />
              <button type="button" :aria-label="showPassword ? '隐藏密码' : '显示密码'" @click="showPassword = !showPassword"><EyeOff v-if="showPassword" :size="17" /><Eye v-else :size="17" /></button>
            </span>
          </label>

          <template v-if="mode === 'register'">
            <label>
              <span>确认密码</span>
              <span class="auth-input-shell"><LockKeyhole :size="17" /><input v-model="confirmPassword" :type="showPassword ? 'text' : 'password'" autocomplete="new-password" required minlength="6" placeholder="再次输入密码" aria-label="确认密码" /></span>
            </label>
            <label>
              <span>昵称</span>
              <span class="auth-input-shell"><UserRound :size="17" /><input v-model="nickname" type="text" required maxlength="40" placeholder="社区中显示的名字" /></span>
            </label>
            <div class="form-row auth-select-row">
              <label><span>身份</span><select v-model="role"><option value="student">学生</option><option value="parent">家长</option><option value="teacher">老师</option><option value="counselor">规划师</option></select></label>
              <label><span>年级</span><select v-model="grade"><option v-for="item in gradeOptions" :key="item" :value="item">{{ item }}</option></select></label>
            </div>
            <label><span>省份</span><select v-model="province"><option v-for="item in provinceOptions" :key="item" :value="item">{{ item }}</option></select></label>
            <label>
              <span>邮箱验证码</span>
              <span class="verification-code-row auth-code-row">
                <span class="auth-input-shell"><AtSign :size="17" /><input v-model="verificationCode" inputmode="numeric" maxlength="6" required placeholder="6 位验证码" aria-describedby="auth-code-feedback" /></span>
                <button class="auth-code-send" :class="`is-${codeState}`" type="button" :disabled="codeLoading || codeCountdown > 0" :aria-busy="codeLoading" @click="requestCode">
                  <LoaderCircle v-if="codeLoading" class="auth-code-spinner" :size="15" />
                  <CheckCircle2 v-else-if="codeCountdown > 0 && codeState === 'sent'" :size="15" />
                  <span>{{ codeButtonLabel }}</span>
                </button>
              </span>
            </label>
            <Transition name="auth-code-status">
              <div v-if="codeMessage" id="auth-code-feedback" class="auth-code-feedback" :class="`is-${codeState}`" role="status" aria-live="polite">
                <span class="auth-code-feedback-icon">
                  <LoaderCircle v-if="codeState === 'sending'" class="auth-code-spinner" :size="16" />
                  <CheckCircle2 v-else-if="codeState === 'sent'" :size="16" />
                  <CircleAlert v-else :size="16" />
                </span>
                <span><strong>{{ codeFeedbackTitle }}</strong><small>{{ codeMessage }}</small></span>
              </div>
            </Transition>
          </template>

          <template v-if="mode === 'reset'">
            <label>
              <span>确认新密码</span>
              <span class="auth-input-shell"><LockKeyhole :size="17" /><input v-model="confirmPassword" :type="showPassword ? 'text' : 'password'" autocomplete="new-password" required minlength="6" placeholder="再次输入新密码" aria-label="确认新密码" /></span>
            </label>
            <label>
              <span>邮箱验证码</span>
              <span class="verification-code-row auth-code-row">
                <span class="auth-input-shell"><AtSign :size="17" /><input v-model="verificationCode" inputmode="numeric" maxlength="6" required placeholder="6 位验证码" aria-describedby="auth-code-feedback" /></span>
                <button class="auth-code-send" :class="`is-${codeState}`" type="button" :disabled="codeLoading || codeCountdown > 0" :aria-busy="codeLoading" @click="requestCode">
                  <LoaderCircle v-if="codeLoading" class="auth-code-spinner" :size="15" />
                  <CheckCircle2 v-else-if="codeCountdown > 0 && codeState === 'sent'" :size="15" />
                  <span>{{ codeButtonLabel }}</span>
                </button>
              </span>
            </label>
            <Transition name="auth-code-status">
              <div v-if="codeMessage" id="auth-code-feedback" class="auth-code-feedback" :class="`is-${codeState}`" role="status" aria-live="polite">
                <span class="auth-code-feedback-icon">
                  <LoaderCircle v-if="codeState === 'sending'" class="auth-code-spinner" :size="16" />
                  <CheckCircle2 v-else-if="codeState === 'sent'" :size="16" />
                  <CircleAlert v-else :size="16" />
                </span>
                <span><strong>{{ codeFeedbackTitle }}</strong><small>{{ codeMessage }}</small></span>
              </div>
            </Transition>
          </template>

          <p v-if="error" class="form-error auth-error-banner">{{ error }}</p>
          <p v-if="success" class="form-success auth-success-banner">{{ success }}</p>
          <button class="auth-submit-button" :disabled="loading || codeLoading || isOffline" :aria-busy="loading" type="submit">
            <LoaderCircle v-if="loading" class="auth-code-spinner" :size="18" />
            <span>{{ loading ? '处理中...' : mode === 'login' ? '登录并继续' : mode === 'register' ? '创建账号' : '重置密码' }}</span><ArrowRight v-if="!loading" :size="18" />
          </button>
          <button v-if="mode === 'login'" class="auth-link-button" type="button" @click="switchMode('reset')">忘记密码？通过邮箱验证码找回</button>
          <small class="auth-privacy-note"><ShieldCheck :size="13" /> 个人画像仅用于站内选科推荐，不会公开成绩信息。</small>
        </form>
      </div>
    </section>
  </div>
</template>
