<script setup lang="ts">
import { Check, ChevronRight } from '@lucide/vue'
import { ref } from 'vue'

const emit = defineEmits<{ complete: [{ province: string; grade: string; role: string }] }>()
const step = ref(1)
const agreed = ref(false)
const role = ref('student')
const province = ref('广东')
const grade = ref('高一')

function next() {
  if (step.value === 1 && !agreed.value) return
  if (step.value < 3) {
    step.value += 1
    return
  }
  emit('complete', { role: role.value, province: province.value, grade: grade.value })
}

function skip() {
  emit('complete', { role: role.value, province: province.value, grade: grade.value })
}
</script>

<template>
  <div class="mobile-onboarding-backdrop" role="dialog" aria-modal="true" aria-labelledby="mobile-onboarding-title">
    <section class="mobile-onboarding">
      <div class="mobile-onboarding-mark" aria-hidden="true">π</div>
      <p class="mobile-onboarding-kicker">选科π · 初次使用</p>
      <h1 id="mobile-onboarding-title">
        {{ step === 1 ? '把依据放在一起' : step === 2 ? '先告诉我们一点背景' : '从你的选科开始' }}
      </h1>
      <p v-if="step === 1" class="mobile-onboarding-copy">社区经验、专业要求和官方政策，都可以在这里找到。</p>
      <div v-else-if="step === 2" class="mobile-onboarding-fields">
        <label>身份<select v-model="role"><option value="student">学生</option><option value="parent">家长</option><option value="teacher">老师</option></select></label>
        <label>省份<input v-model="province" maxlength="30" /></label>
        <label>年级<select v-model="grade"><option>高一</option><option>高二</option><option>高三</option><option>其他</option></select></label>
      </div>
      <p v-else class="mobile-onboarding-copy">你可以随时在“我的”里修改画像，也可以现在跳过。</p>
      <label v-if="step === 1" class="mobile-onboarding-consent">
        <input v-model="agreed" type="checkbox" />
        <span>我已阅读并同意<a href="/privacy" target="_blank" rel="noopener">隐私政策</a>和<a href="/terms" target="_blank" rel="noopener">服务条款</a></span>
      </label>
      <div class="mobile-onboarding-progress" aria-label="引导进度">
        <i v-for="item in 3" :key="item" :class="{ active: item <= step }" />
      </div>
      <div class="mobile-onboarding-actions">
        <button class="mobile-onboarding-skip" type="button" @click="skip">稍后设置</button>
        <button class="primary-wide" type="button" :disabled="step === 1 && !agreed" @click="next">
          {{ step === 3 ? '进入社区' : '继续' }} <ChevronRight :size="17" />
        </button>
      </div>
      <p v-if="step === 1 && agreed" class="mobile-onboarding-ready"><Check :size="15" /> 你的选择只用于改善体验</p>
    </section>
  </div>
</template>

<style scoped>
.mobile-onboarding-backdrop { position: fixed; inset: 0; z-index: 1000; display: grid; place-items: center; padding: 24px; background: rgba(17, 21, 18, .56); }
.mobile-onboarding { width: min(100%, 420px); padding: 34px 28px 26px; border-radius: 22px; background: #f7f8f6; color: #111512; box-shadow: 0 24px 80px rgba(0, 0, 0, .22); }
.mobile-onboarding-mark { width: 42px; height: 42px; display: grid; place-items: center; border-radius: 13px; background: #2f6b4f; color: #fff; font: 600 25px Georgia, serif; }
.mobile-onboarding-kicker { margin: 24px 0 10px; color: #667068; font-size: 13px; }
.mobile-onboarding h1 { margin: 0; font-size: clamp(28px, 8vw, 38px); line-height: 1.2; letter-spacing: 0; }
.mobile-onboarding-copy { min-height: 52px; margin: 16px 0 24px; color: #667068; font-size: 16px; line-height: 1.7; }
.mobile-onboarding-fields { display: grid; gap: 13px; margin: 20px 0 24px; }
.mobile-onboarding-fields label { display: grid; gap: 6px; color: #667068; font-size: 13px; }
.mobile-onboarding-fields input, .mobile-onboarding-fields select { min-height: 46px; padding: 0 12px; border: 1px solid #d7ded8; border-radius: 10px; background: #fff; color: #111512; font: inherit; }
.mobile-onboarding-consent { display: flex; gap: 10px; align-items: flex-start; margin: 22px 0 24px; color: #667068; font-size: 13px; line-height: 1.6; }
.mobile-onboarding-consent input { width: 18px; height: 18px; flex: 0 0 auto; accent-color: #2f6b4f; }
.mobile-onboarding-consent a { color: #2f6b4f; text-decoration: underline; }
.mobile-onboarding-progress { display: flex; gap: 6px; margin: 4px 0 22px; }
.mobile-onboarding-progress i { width: 24px; height: 3px; border-radius: 2px; background: #d7ded8; }
.mobile-onboarding-progress i.active { background: #2f6b4f; }
.mobile-onboarding-actions { display: flex; align-items: center; gap: 12px; }
.mobile-onboarding-actions .primary-wide { flex: 1; min-height: 48px; display: inline-flex; align-items: center; justify-content: center; gap: 5px; }
.mobile-onboarding-skip { min-height: 48px; padding: 0 4px; border: 0; background: transparent; color: #667068; font: inherit; }
.mobile-onboarding-ready { display: flex; align-items: center; gap: 5px; margin: 15px 0 0; color: #2f6b4f; font-size: 12px; }
</style>
