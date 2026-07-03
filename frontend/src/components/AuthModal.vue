<script setup lang="ts">
import { X } from '@lucide/vue'
import axios from 'axios'
import { onUnmounted, ref, watch } from 'vue'
import { login, register, sendEmailVerificationCode } from '../lib/api'
import { useForumStore } from '../stores/forum'
import type { Role } from '../types/forum'

const forumStore = useForumStore()
const mode = ref<'login' | 'register'>('login')
const email = ref('')
const password = ref('')
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
    forumStore.setSession(session)
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
    startCountdown(Math.min(result.expiresInSeconds, 60))
    codeMessage.value = result.debugCode ? `本地调试验证码：${result.debugCode}` : '验证码已发送，请查看邮箱'
  } catch (err) {
    if (axios.isAxiosError(err) && !err.response) {
      error.value = '无法连接后端服务，验证码未发送'
    } else {
      error.value = '验证码发送失败，请稍后重试'
    }
  } finally {
    codeLoading.value = false
  }
}

function startCountdown(seconds: number) {
  codeCountdown.value = seconds
  if (countdownTimer) window.clearInterval(countdownTimer)
  countdownTimer = window.setInterval(() => {
    codeCountdown.value -= 1
    if (codeCountdown.value <= 0 && countdownTimer) {
      window.clearInterval(countdownTimer)
      countdownTimer = undefined
    }
  }, 1000)
}

watch([email, mode], () => {
  codeMessage.value = ''
  verificationCode.value = ''
})

onUnmounted(() => {
  if (countdownTimer) window.clearInterval(countdownTimer)
})
</script>

<template>
  <div class="modal-backdrop">
    <section class="auth-modal">
      <div class="modal-title-row">
        <h2>{{ mode === 'login' ? '登录账号' : '注册账号' }}</h2>
        <button class="icon-button" type="button" @click="forumStore.authOpen = false">
          <X :size="18" />
        </button>
      </div>

      <div class="auth-tabs">
        <button :class="{ active: mode === 'login' }" type="button" @click="mode = 'login'">登录</button>
        <button :class="{ active: mode === 'register' }" type="button" @click="mode = 'register'">注册</button>
      </div>

      <form class="auth-form" @submit.prevent="submit">
        <label>
          邮箱
          <input v-model="email" type="email" required />
        </label>
        <label>
          密码
          <input v-model="password" type="password" required minlength="6" />
        </label>

        <template v-if="mode === 'register'">
          <label>
            昵称
            <input v-model="nickname" type="text" required />
          </label>
          <div class="form-row">
            <label>
              身份
              <select v-model="role">
                <option value="student">学生</option>
                <option value="parent">家长</option>
                <option value="teacher">老师</option>
                <option value="counselor">规划师</option>
              </select>
            </label>
            <label>
              年级
              <select v-model="grade">
                <option v-for="item in gradeOptions" :key="item" :value="item">{{ item }}</option>
              </select>
            </label>
          </div>
          <label>
            省份
            <select v-model="province">
              <option v-for="item in provinceOptions" :key="item" :value="item">{{ item }}</option>
            </select>
          </label>
          <label>
            邮箱验证码
            <div class="verification-code-row">
              <input v-model="verificationCode" inputmode="numeric" maxlength="6" required placeholder="6 位验证码" />
              <button type="button" :disabled="codeLoading || codeCountdown > 0" @click="requestCode">
                {{ codeLoading ? '发送中' : codeCountdown > 0 ? `${codeCountdown}s` : '发送验证码' }}
              </button>
            </div>
          </label>
          <p v-if="codeMessage" class="helper-text compact">{{ codeMessage }}</p>
        </template>

        <p v-if="error" class="form-error">{{ error }}</p>
        <button class="primary-wide" :disabled="loading" type="submit">
          {{ loading ? '处理中...' : mode === 'login' ? '登录' : '创建账号' }}
        </button>
      </form>
    </section>
  </div>
</template>
