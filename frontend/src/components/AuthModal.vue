<script setup lang="ts">
import { X } from '@lucide/vue'
import axios from 'axios'
import { ref } from 'vue'
import { apiDataEnabled, login, register } from '../lib/api'
import { useForumStore } from '../stores/forum'
import type { AuthSession, Role } from '../types/forum'

const forumStore = useForumStore()
const mode = ref<'login' | 'register'>('login')
const email = ref('demo@student.local')
const password = ref('123456')
const nickname = ref('高一新用户')
const role = ref<Role>('student')
const province = ref('浙江')
const grade = ref('高一')
const error = ref('')
const loading = ref(false)

async function submit() {
  loading.value = true
  error.value = ''
  try {
    if (!apiDataEnabled) {
      forumStore.setSession(createLocalSession())
      return
    }
    const session =
      mode.value === 'login'
        ? await login(email.value, password.value)
        : await register({
            email: email.value,
            password: password.value,
            nickname: nickname.value,
            role: role.value,
            province: province.value,
            grade: grade.value,
          })
    forumStore.setSession(session)
  } catch (err) {
    if (axios.isAxiosError(err) && !err.response) {
      forumStore.setSession(createLocalSession())
    } else {
      error.value = mode.value === 'login' ? '邮箱或密码不正确' : '注册失败，请检查邮箱是否已存在'
    }
  } finally {
    loading.value = false
  }
}

function createLocalSession(): AuthSession {
  const nicknameFromEmail = email.value.split('@')[0]?.replace(/[._-]+/g, '') || '本地用户'
  const isDemo = email.value === 'demo@student.local'
  return {
    token: `local-${Date.now()}`,
    user: {
      id: isDemo ? 1 : Date.now(),
      email: email.value,
      nickname: mode.value === 'register' ? nickname.value : isDemo ? '演示同学' : nicknameFromEmail,
      role: mode.value === 'register' ? role.value : 'student',
      province: mode.value === 'register' ? province.value : '浙江',
      grade: mode.value === 'register' ? grade.value : '高一',
      createdAt: new Date().toISOString(),
    },
  }
}
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
              <input v-model="grade" type="text" />
            </label>
          </div>
          <label>
            省份
            <input v-model="province" type="text" />
          </label>
        </template>

        <p v-if="error" class="form-error">{{ error }}</p>
        <button class="primary-wide" :disabled="loading" type="submit">
          {{ loading ? '处理中...' : mode === 'login' ? '登录' : '创建账号' }}
        </button>
        <p class="helper-text">演示账号：demo@student.local / 123456</p>
      </form>
    </section>
  </div>
</template>
