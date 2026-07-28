<script setup lang="ts">
import { ArrowRight, AtSign, Eye, EyeOff, GraduationCap, LockKeyhole, Mail, ShieldCheck, Sparkles, UserRound, X } from '@lucide/vue'
import axios from 'axios'
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { login, register, sendEmailVerificationCode } from '../lib/api'
import { currentProfileAuthRedirect, useForumStore } from '../stores/forum'
import type { Role } from '../types/forum'

const forumStore = useForumStore()
const router = useRouter()
const mode = ref<'login' | 'register'>('login')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const verificationCode = ref('')
const nickname = ref('高一新用户')
const role = ref<Role>('student')
const province = ref('浙江')
const grade = ref('高一')
const error = ref('')
const codeMessage = ref('')
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
  loading.value = true
  error.value = ''
  try {
    if (mode.value === 'register' && password.value !== confirmPassword.value) {
      error.value = '两次输入的密码不一致'
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
    if (axios.isAxiosError(err) && !err.response) {
      error.value = '无法连接后端服务，请稍后重试'
    } else {
      if (mode.value === 'login') {
        error.value = '邮箱或密码不正确'
      } else if (axios.isAxiosError(err) && err.response?.data?.error?.code === 'invalid_verification_code') {
        error.value = '验证码错误或已过期，请重新获取'
      } else {
        error.value = '注册失败，请检查邮箱是否已存在'
      }
    }
  } finally {
    loading.value = false
  }
}

async function requestCode() {
  error.value = ''
  codeMessage.value = ''
  if (!email.value) {
    error.value = '请先填写邮箱'
    return
  }
  codeLoading.value = true
  try {
    const result = await sendEmailVerificationCode(email.value)
    startCountdown(result.retryAfterSeconds)
    const quotaMessage = `本邮箱本小时还可发送 ${result.hourlyRemaining} 次`
    codeMessage.value = result.debugCode
      ? `本地调试验证码：${result.debugCode}（${quotaMessage}）`
      : `验证码已发送，请查看邮箱（${quotaMessage}）`
  } catch (err) {
    if (axios.isAxiosError(err) && !err.response) {
      error.value = '无法连接后端服务，验证码未发送'
    } else if (axios.isAxiosError(err) && err.response?.data?.error?.code === 'email_verification_rate_limited') {
      const rateLimit = err.response.data.error as {
        retryAfterSeconds?: number
        hourlyRemaining?: number
      }
      const retryAfterSeconds = Math.max(1, Number(rateLimit.retryAfterSeconds) || 60)
      startCountdown(retryAfterSeconds)
      codeMessage.value =
        rateLimit.hourlyRemaining === 0
          ? `本邮箱本小时发送次数已用完，请在 ${formatRetryAfter(retryAfterSeconds)}后重试`
          : `请求过于频繁，请在 ${formatRetryAfter(retryAfterSeconds)}后重试（本小时剩余 ${rateLimit.hourlyRemaining ?? 0} 次）`
    } else {
      error.value = '验证码发送失败，请稍后重试'
    }
  } finally {
    codeLoading.value = false
  }
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

watch([email, mode], () => {
  codeMessage.value = ''
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
  forumStore.authRedirect = ''
  forumStore.authOpen = false
}

onUnmounted(() => {
  if (countdownTimer) window.clearInterval(countdownTimer)
})
</script>

<template>
  <div class="modal-backdrop auth-experience-backdrop">
    <section class="auth-experience" :class="{ 'is-registering': mode === 'register' }" role="dialog" aria-modal="true" :aria-label="mode === 'login' ? '登录账号' : '注册账号'">
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

      <button class="auth-close-button" type="button" aria-label="关闭" @click="close"><X :size="19" /></button>
      <div ref="authFormPanel" class="auth-form-panel">
        <header class="auth-form-heading">
          <span>{{ mode === 'login' ? '欢迎回来' : '从这里开始' }}</span>
          <h2>{{ mode === 'login' ? '登录你的账号' : '创建选科档案' }}</h2>
          <p>{{ mode === 'login' ? '继续查看你的画像、收藏与消息。' : '注册后即可保存个人信息和选科画像。' }}</p>
        </header>

        <div class="auth-tabs" aria-label="登录方式">
          <button :class="{ active: mode === 'login' }" type="button" @click="mode = 'login'">登录</button>
          <button :class="{ active: mode === 'register' }" type="button" @click="mode = 'register'">注册</button>
        </div>

        <form class="auth-form auth-premium-form" @submit.prevent="submit">
          <label>
            <span>邮箱</span>
            <span class="auth-input-shell"><Mail :size="17" /><input v-model="email" type="email" autocomplete="email" required placeholder="name@example.com" /></span>
          </label>
          <label>
            <span>密码</span>
            <span class="auth-input-shell">
              <LockKeyhole :size="17" />
              <input v-model="password" :type="showPassword ? 'text' : 'password'" :autocomplete="mode === 'login' ? 'current-password' : 'new-password'" required minlength="6" placeholder="至少 6 位字符" />
              <button type="button" :aria-label="showPassword ? '隐藏密码' : '显示密码'" @click="showPassword = !showPassword"><EyeOff v-if="showPassword" :size="17" /><Eye v-else :size="17" /></button>
            </span>
          </label>

          <template v-if="mode === 'register'">
            <label>
              <span>确认密码</span>
              <span class="auth-input-shell"><LockKeyhole :size="17" /><input v-model="confirmPassword" :type="showPassword ? 'text' : 'password'" autocomplete="new-password" required minlength="6" placeholder="再次输入密码" /></span>
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
              <span class="verification-code-row auth-code-row"><span class="auth-input-shell"><AtSign :size="17" /><input v-model="verificationCode" inputmode="numeric" maxlength="6" required placeholder="6 位验证码" /></span><button type="button" :disabled="codeLoading || codeCountdown > 0" @click="requestCode">{{ codeLoading ? '发送中' : codeCountdown > 0 ? codeCountdownLabel : '获取验证码' }}</button></span>
            </label>
            <p v-if="codeMessage" class="helper-text compact">{{ codeMessage }}</p>
          </template>

          <p v-if="error" class="form-error auth-error-banner">{{ error }}</p>
          <button class="auth-submit-button" :disabled="loading" type="submit">
            <span>{{ loading ? '处理中...' : mode === 'login' ? '登录并继续' : '创建账号' }}</span><ArrowRight v-if="!loading" :size="18" />
          </button>
          <small class="auth-privacy-note"><ShieldCheck :size="13" /> 个人画像仅用于站内选科推荐，不会公开成绩信息。</small>
        </form>
      </div>
    </section>
  </div>
</template>
