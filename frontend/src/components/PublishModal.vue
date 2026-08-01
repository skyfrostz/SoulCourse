<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { ImagePlus, Tag, Trash2, X } from '@lucide/vue'
import { onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createPost, fetchTaxonomy, uploadImage } from '../lib/api'
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
type UploadedImage = { previewUrl: string; url: string; fileKey: string }

const imagePreviews = ref<UploadedImage[]>([])
const failedImageFiles = ref<File[]>([])
const tagInput = ref('')
const tags = ref<string[]>([])
const imageUploading = ref(false)
const track = ref<Track>(forumStore.filter.track === 'all' ? 'physics' : forumStore.filter.track)
const electives = ref<Subject[]>(forumStore.filter.subjects.length === 2 ? [...forumStore.filter.subjects] : ['chemistry', 'biology'])
const error = ref('')
const MAX_IMAGE_BYTES = 6 * 1024 * 1024
const ALLOWED_IMAGE_TYPES = new Set(['image/jpeg', 'image/png', 'image/gif'])
const taxonomyQuery = useQuery({ queryKey: ['taxonomy'], queryFn: fetchTaxonomy, staleTime: 10 * 60 * 1000 })

const subjects: Subject[] = ['chemistry', 'biology', 'politics', 'geography']

const publishMutation = useMutation({
  mutationFn: async () => {
    const payload = {
      title: title.value,
      content: content.value,
      imageUrls: imagePreviews.value.map((item) => item.url).slice(0, 9),
      tags: tags.value.slice(0, 8),
      track: track.value,
      electives: electives.value,
      category: category.value,
      grade: forumStore.currentUser?.grade ?? '高一',
      province: forumStore.currentUser?.province ?? '全国',
    }
    return createPost(payload)
  },
  onSuccess: (post) => {
    queryClient.invalidateQueries({ queryKey: ['posts'] })
    imagePreviews.value.forEach((image) => URL.revokeObjectURL(image.previewUrl))
    imagePreviews.value = []
    failedImageFiles.value = []
    forumStore.publishOpen = false
    router.push(`/posts/${post.id}`)
  },
  onError: () => {
    error.value = '发布失败，请确认已登录、后端服务可用，且标题、正文和两个再选科目完整。'
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

function imageFileKey(file: File) {
  return [file.name, file.size, file.lastModified, file.type].join(':')
}

function setUploadFailureMessage() {
  const failedCount = failedImageFiles.value.length
  error.value = failedCount
    ? `${failedCount} 张图片上传失败，已保留上传成功的图片。可重新选择图片或重试失败项。`
    : ''
}

function discardFailedImages() {
  failedImageFiles.value = []
  error.value = ''
}

async function uploadFiles(files: File[]) {
  if (!files.length) return

  imageUploading.value = true
  const results = await Promise.allSettled(files.map(uploadPreviewImage))
  const uploaded: UploadedImage[] = []
  const failed: File[] = []

  results.forEach((result, index) => {
    if (result.status === 'fulfilled') {
      uploaded.push(result.value)
    } else {
      failed.push(files[index])
    }
  })

  imagePreviews.value = [...imagePreviews.value, ...uploaded]
  const failedByKey = new Map(failedImageFiles.value.map((file) => [imageFileKey(file), file]))
  failed.forEach((file) => failedByKey.set(imageFileKey(file), file))
  failedImageFiles.value = [...failedByKey.values()]
  setUploadFailureMessage()
  imageUploading.value = false
}

async function handleFiles(event: Event) {
  if (imageUploading.value) return
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  if (!files.length) return

  error.value = ''
  const remaining = Math.max(0, 9 - imagePreviews.value.length - failedImageFiles.value.length)
  if (remaining <= 0) {
    error.value = '最多只能上传 9 张图片。'
    input.value = ''
    return
  }

  const invalidFile = files.find((file) => !ALLOWED_IMAGE_TYPES.has(file.type))
  if (invalidFile) {
    error.value = '图片格式仅支持 JPG、PNG、GIF 或 WebP。'
    input.value = ''
    return
  }

  const imageFiles = files.slice(0, remaining)
  if (files.length > remaining) {
    error.value = `最多只能再上传 ${remaining} 张图片，已先处理前 ${remaining} 张。`
  }
  const oversized = imageFiles.find((file) => file.size > MAX_IMAGE_BYTES)
  if (oversized) {
    error.value = '单张图片建议不超过 6MB。'
    input.value = ''
    return
  }

  const uploadedKeys = new Set(imagePreviews.value.map((image) => image.fileKey))
  const selectedByKey = new Map(imageFiles.map((file) => [imageFileKey(file), file]))
  const filesToUpload = [...selectedByKey.values()].filter((file) => !uploadedKeys.has(imageFileKey(file)))
  const selectedKeys = new Set(filesToUpload.map(imageFileKey))
  failedImageFiles.value = failedImageFiles.value.filter((file) => !selectedKeys.has(imageFileKey(file)))

  await uploadFiles(filesToUpload)
  input.value = ''
}

async function retryFailedImages() {
  if (imageUploading.value || !failedImageFiles.value.length) return
  const files = failedImageFiles.value
  failedImageFiles.value = []
  error.value = ''
  await uploadFiles(files)
}

async function uploadPreviewImage(file: File) {
  const dimensions = await readImageDimensions(file)
  const result = await uploadImage(file, dimensions)
  return { previewUrl: URL.createObjectURL(file), url: result.url, fileKey: imageFileKey(file) }
}

function readImageDimensions(file: File) {
  return new Promise<{ width: number; height: number }>((resolve, reject) => {
    const image = new Image()
    const objectUrl = URL.createObjectURL(file)
    image.onload = () => {
      URL.revokeObjectURL(objectUrl)
      resolve({ width: image.naturalWidth, height: image.naturalHeight })
    }
    image.onerror = () => {
      URL.revokeObjectURL(objectUrl)
      reject(new Error('invalid image'))
    }
    image.src = objectUrl
  })
}

function removeImage(index: number) {
  const removed = imagePreviews.value[index]
  if (removed) URL.revokeObjectURL(removed.previewUrl)
  imagePreviews.value = imagePreviews.value.filter((_, itemIndex) => itemIndex !== index)
}

function closePublishModal() {
  if (publishMutation.isPending.value || imageUploading.value) return
  imagePreviews.value.forEach((image) => URL.revokeObjectURL(image.previewUrl))
  imagePreviews.value = []
  failedImageFiles.value = []
  forumStore.publishOpen = false
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

function toggleControlledTag(tag: string) {
  if (tags.value.includes(tag)) {
    removeTag(tag)
  } else if (tags.value.length < 8) {
    tags.value = [...tags.value, tag]
  }
}

function submit() {
  if (publishMutation.isPending.value) return
  error.value = ''
  if (!forumStore.isAuthed) {
    forumStore.openAuth()
    return
  }
  if (imageUploading.value) {
    error.value = '图片仍在上传，请稍候。'
    return
  }
  if (failedImageFiles.value.length) {
    error.value = '仍有图片上传失败，请先重试，或移除失败图片后再发布。'
    return
  }
  if (typeof navigator !== 'undefined' && !navigator.onLine) {
    error.value = '当前网络不可用，内容已保留，请恢复网络后再发布。'
    return
  }
  if (!title.value.trim() || !content.value.trim()) {
    error.value = '请填写标题和正文后再发布。'
    return
  }
  if (electives.value.length !== 2) {
    error.value = '请选择两个再选科目。'
    return
  }
  addTag()
  publishMutation.mutate()
}

onBeforeUnmount(() => {
  imagePreviews.value.forEach((image) => URL.revokeObjectURL(image.previewUrl))
})
</script>

<template>
  <div class="modal-backdrop">
    <section class="auth-modal publish-modal">
      <div class="modal-title-row">
        <h2>发布内容</h2>
        <button class="icon-button" type="button" :disabled="publishMutation.isPending.value || imageUploading" @click="closePublishModal"><X :size="18" /></button>
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
          <button type="button" :disabled="imageUploading || imagePreviews.length + failedImageFiles.length >= 9" @click="openFilePicker">
            <ImagePlus :size="16" /> {{ imageUploading ? '上传中...' : imagePreviews.length + failedImageFiles.length >= 9 ? '已达 9 张上限' : '上传/选择照片' }}
          </button>
          <div v-if="imagePreviews.length" class="image-preview-grid">
            <figure v-for="(image, index) in imagePreviews" :key="image.url">
              <img :src="image.previewUrl" alt="待发布图片预览" />
              <button type="button" aria-label="删除图片" @click="removeImage(index)">
                <Trash2 :size="14" />
              </button>
            </figure>
          </div>
          <div v-if="failedImageFiles.length" class="upload-failure-actions" role="status">
            <span>{{ failedImageFiles.length }} 张图片待重试</span>
            <button type="button" :disabled="imageUploading" @click="retryFailedImages">
              重试失败图片
            </button>
            <button type="button" :disabled="imageUploading" @click="discardFailedImages">
              移除失败图片
            </button>
          </div>
        </div>
        <div class="tag-editor">
          <label>
            标签
            <span>可自定义名称，输入后按回车或逗号添加</span>
            <input
              v-model="tagInput"
              maxlength="20"
            placeholder="例如：新高考避坑"
              @keydown="handleTagKeydown"
            />
          </label>
          <button v-if="tagInput.trim()" class="tag-add-button" type="button" @click="addTag">
            <Tag :size="14" /> 添加“{{ tagInput.trim() }}”
          </button>
          <div v-if="tags.length" class="tag-chip-row">
            <button v-for="tag in tags" :key="tag" type="button" @click="removeTag(tag)">
              <Tag :size="13" /> {{ tag }} <X :size="12" />
            </button>
          </div>
          <div v-if="taxonomyQuery.data.value" class="tag-section-heading">
            <strong class="tag-section-title">大家正在讨论</strong>
            <span>点击即可选择</span>
          </div>
          <div v-if="taxonomyQuery.data.value" class="tag-chip-row controlled-tag-row">
            <button
              v-for="tag in taxonomyQuery.data.value.topicTags"
              :key="tag.value"
              type="button"
              :class="{ active: tags.includes(tag.value) }"
              :aria-pressed="tags.includes(tag.value)"
              @click="toggleControlledTag(tag.value)"
            >
              # {{ tag.value }}
            </button>
          </div>
          <small>已选标签会直接使用；未选标签时，AI 才会根据正文自动打标。</small>
        </div>
        <label>
          方向
          <select v-model="track">
            <option value="physics">物理方向</option>
            <option value="history">历史方向</option>
          </select>
        </label>
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
        <button class="primary-wide" :disabled="publishMutation.isPending.value || imageUploading" type="submit" :aria-busy="publishMutation.isPending.value || imageUploading">
          {{ imageUploading ? '图片上传中...' : publishMutation.isPending.value ? '发布中...' : '发布' }}
        </button>
      </form>
    </section>
  </div>
</template>
