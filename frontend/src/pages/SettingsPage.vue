<script setup lang="ts">
import { Check, ChevronLeft, Save, Sparkles } from '@lucide/vue'
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { requestChoiceAdvice } from '../lib/api'
import { subjectLabels, trackLabels } from '../lib/labels'
import { useForumStore } from '../stores/forum'
import type { ChoiceAdvice, ChoiceProfile, Subject, Track } from '../types/forum'

const router = useRouter()
const forumStore = useForumStore()
const saved = ref(false)
const advice = ref<ChoiceAdvice | null>(null)
const adviceLoading = ref(false)
const adviceError = ref('')
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

function save() {
  forumStore.saveChoiceProfile({ ...form })
  saved.value = true
  window.setTimeout(() => {
    saved.value = false
  }, 1800)
}

async function generateAdvice() {
  if (!forumStore.currentUser) {
    forumStore.authOpen = true
    return
  }
  forumStore.saveChoiceProfile({ ...form })
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

function searchSuggestion(keyword: string) {
  forumStore.setKeyword(keyword)
  router.push('/')
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
      <button class="write-button" type="button" @click="save">
        <Save :size="16" /> 保存设置
      </button>
    </section>

    <section class="settings-grid">
      <form class="settings-card" @submit.prevent="save">
        <h2>基础信息</h2>
        <div class="settings-fields">
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

      <aside class="settings-summary">
        <h2>画像摘要</h2>
        <p><strong>{{ form.realName || forumStore.currentUser?.nickname || '未填写称呼' }}</strong></p>
        <p>{{ trackLabels[form.preferredTrack] }} · {{ form.preferredSubjects.map((item) => subjectLabels[item]).join(' + ') }}</p>
        <p>MBTI：{{ form.mbti || '未填写' }}</p>
        <p>目标：{{ form.targetMajors || '未填写目标专业' }}</p>
        <span v-if="saved"><Check :size="16" /> 已保存到本地</span>
        <div class="ai-advice-card">
          <div>
            <strong><Sparkles :size="16" /> AI 个性化建议</strong>
            <small>基于当前画像，输出简短决策提醒</small>
          </div>
          <button type="button" :disabled="adviceLoading" @click="generateAdvice">
            {{ adviceLoading ? '分析中...' : '生成建议' }}
          </button>
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
