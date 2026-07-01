<script setup lang="ts">
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { ImagePlus, Tag, Trash2, X } from '@lucide/vue'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { createPost } from '../lib/api'
import { subjectLabels } from '../lib/labels'
import { useForumStore } from '../stores/forum'
import type { Category, Subject, Track } from '../types/forum'

const forumStore = useForumStore()
const queryClient = useQueryClient()
const router = useRouter()

const category = ref<Category>(forumStore.publishCategory)
const title = ref('')
const content = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const imagePreviews = ref<string[]>([])
const tagInput = ref('')
const tags = ref<string[]>([])
const track = ref<Track>(forumStore.filter.track)
const electives = ref<Subject[]>([...forumStore.filter.subjects])
const error = ref('')

const subjects: Subject[] = ['chemistry', 'biology', 'politics', 'geography']

const publishMutation = useMutation({
  mutationFn: () =>
    createPost({
      title: title.value,
      content: content.value,
      imageUrls: imagePreviews.value.slice(0, 9),
      tags: tags.value.slice(0, 8),
      track: track.value,
      electives: electives.value,
      category: category.value,
      grade: forumStore.currentUser?.grade ?? '高一',
      province: forumStore.currentUser?.province ?? '全国',
    }),
  onSuccess: (post) => {
    queryClient.invalidateQueries({ queryKey: ['posts'] })
    forumStore.publishOpen = false
    router.push(`/posts/${post.id}`)
  },
  onError: () => {
    error.value = '发布失败，请检查标题、正文和两个再选科目是否完整。'
  },
})

function toggleSubject(subject: Subject) {
  if (electives.value.includes(subject)) {
    electives.value = electives.value.filter((item) => item !== subject)
    return
  }
  electives.value = [...electives.value.slice(-1), subject]
}

function openFilePicker() {
  fileInput.value?.click()
}

async function handleFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  if (!files.length) return

  error.value = ''
  const remaining = Math.max(0, 9 - imagePreviews.value.length)
  const imageFiles = files.filter((file) => file.type.startsWith('image/')).slice(0, remaining)
  const oversized = imageFiles.find((file) => file.size > 6 * 1024 * 1024)
  if (oversized) {
    error.value = '单张图片建议不超过 6MB。'
    input.value = ''
    return
  }

  const previews = await Promise.all(imageFiles.map(readImageAsDataURL))
  imagePreviews.value = [...imagePreviews.value, ...previews]
  input.value = ''
}

function readImageAsDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

function removeImage(index: number) {
  imagePreviews.value = imagePreviews.value.filter((_, itemIndex) => itemIndex !== index)
}

function addTag() {
  const cleaned = tagInput.value.trim().replace(/^#+/, '').replace(/\s+/g, '')
  if (!cleaned) return
  if (!tags.value.includes(cleaned) && tags.value.length < 8) {
    tags.value = [...tags.value, cleaned]
  }
  tagInput.value = ''
}

function handleTagKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== ',' && event.key !== '，') return
  event.preventDefault()
  addTag()
}

function removeTag(tag: string) {
  tags.value = tags.value.filter((item) => item !== tag)
}

function submit() {
  error.value = ''
  if (electives.value.length !== 2) {
    error.value = '请选择两个再选科目。'
    return
  }
  publishMutation.mutate()
}
</script>

<template>
  <div class="modal-backdrop">
    <section class="auth-modal publish-modal">
      <div class="modal-title-row">
        <h2>发布内容</h2>
        <button class="icon-button" type="button" @click="forumStore.publishOpen = false"><X :size="18" /></button>
      </div>

      <div class="auth-tabs">
        <button :class="{ active: category === 'question' }" type="button" @click="category = 'question'">提问</button>
        <button :class="{ active: category === 'experience' }" type="button" @click="category = 'experience'">经验贴</button>
        <button :class="{ active: category === 'data' }" type="button" @click="category = 'data'">数据建议</button>
      </div>

      <form class="auth-form publish-form" @submit.prevent="submit">
        <label>
          标题
          <input v-model="title" required minlength="4" maxlength="80" placeholder="比如：物化生适合目标不明确的人吗？" />
        </label>
        <label>
          正文
          <textarea v-model="content" required minlength="10" maxlength="4000" placeholder="写下你的背景、疑问、经验或数据观察"></textarea>
        </label>
        <div class="upload-panel">
          <div>
            <strong><ImagePlus :size="17" /> 图片</strong>
            <small>支持从手机相册或电脑照片中选择，最多 9 张。</small>
          </div>
          <input ref="fileInput" type="file" accept="image/*" multiple hidden @change="handleFiles" />
          <button type="button" @click="openFilePicker">
            <ImagePlus :size="16" /> 上传/选择照片
          </button>
          <div v-if="imagePreviews.length" class="image-preview-grid">
            <figure v-for="(image, index) in imagePreviews" :key="image">
              <img :src="image" alt="待发布图片预览" />
              <button type="button" aria-label="删除图片" @click="removeImage(index)">
                <Trash2 :size="14" />
              </button>
            </figure>
          </div>
        </div>
        <div class="tag-editor">
          <label>
            标签
            <span>输入关键词后按回车生成标签</span>
            <input
              v-model="tagInput"
              maxlength="20"
              placeholder="例如：物化生"
              @keydown="handleTagKeydown"
            />
          </label>
          <div v-if="tags.length" class="tag-chip-row">
            <button v-for="tag in tags" :key="tag" type="button" @click="removeTag(tag)">
              <Tag :size="13" /> {{ tag }} <X :size="12" />
            </button>
          </div>
        </div>
        <div class="form-row">
          <label>
            方向
            <select v-model="track">
              <option value="physics">物理方向</option>
              <option value="history">历史方向</option>
            </select>
          </label>
          <label>
            类型
            <select v-model="category">
              <option value="question">提问</option>
              <option value="experience">经验贴</option>
              <option value="data">数据建议</option>
            </select>
          </label>
        </div>
        <div class="publish-subjects">
          <button
            v-for="subject in subjects"
            :key="subject"
            type="button"
            :class="{ active: electives.includes(subject) }"
            @click="toggleSubject(subject)"
          >
            {{ subjectLabels[subject] }}
          </button>
        </div>
        <p v-if="error" class="form-error">{{ error }}</p>
        <button class="primary-wide" :disabled="publishMutation.isPending.value" type="submit">发布</button>
      </form>
    </section>
  </div>
</template>
