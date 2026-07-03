import { computed, ref } from 'vue'
import { adviceNotes } from '../lib/adviceNotes'
import { useForumStore } from '../stores/forum'

export interface AdviceComment {
  id: number
  noteId: string
  author: string
  role: string
  content: string
  createdAt: string
}

interface AdviceEngagementState {
  liked: Record<string, boolean>
  saved: Record<string, boolean>
  likes: Record<string, number>
  saves: Record<string, number>
  comments: Record<string, AdviceComment[]>
}

const storageKey = 'scf_advice_engagement'
const state = ref<AdviceEngagementState>(readState())

const seededComments: Record<string, AdviceComment[]> = Object.fromEntries(
  adviceNotes.map((note, index) => [
    note.id,
    [
      {
        id: index * 100 + 1,
        noteId: note.id,
        author: note.role.includes('教师') ? '高一班主任王老师' : '正在选科的高一生',
        role: note.role.includes('教师') ? '教师认证' : '学生',
        content: note.lens === 'parent'
          ? '这个视角很适合家长会前一起看，重点是把孩子能长期坚持的证据摆出来。'
          : '收藏了，准备按行动清单把目标专业和最近几次考试波动整理出来。',
        createdAt: `2026-07-01T${String(9 + (index % 8)).padStart(2, '0')}:18:00+08:00`,
      },
      {
        id: index * 100 + 2,
        noteId: note.id,
        author: note.author,
        role: note.role,
        content: `补充一句：${note.actions[0]}，这一步最好用本省考试院或阳光高考的最新目录核对。`,
        createdAt: `2026-07-01T${String(10 + (index % 8)).padStart(2, '0')}:42:00+08:00`,
      },
    ],
  ]),
)

export function useAdviceEngagement() {
  const forumStore = useForumStore()

  function noteComments(noteId: string) {
    return computed(() => [...(seededComments[noteId] ?? []), ...(state.value.comments[noteId] ?? [])])
  }

  function noteStats(noteId: string, baseLikes: number, baseSaves: number) {
    return computed(() => getStats(noteId, baseLikes, baseSaves))
  }

  function getStats(noteId: string, baseLikes: number, baseSaves: number) {
    const comments = [...(seededComments[noteId] ?? []), ...(state.value.comments[noteId] ?? [])]
    return {
      liked: state.value.liked[noteId] ?? false,
      saved: state.value.saved[noteId] ?? false,
      likes: state.value.likes[noteId] ?? baseLikes,
      saves: state.value.saves[noteId] ?? baseSaves,
      comments: comments.length,
    }
  }

  function toggleLike(noteId: string, baseLikes: number) {
    const active = !(state.value.liked[noteId] ?? false)
    const current = state.value.likes[noteId] ?? baseLikes
    state.value = {
      ...state.value,
      liked: { ...state.value.liked, [noteId]: active },
      likes: { ...state.value.likes, [noteId]: Math.max(0, current + (active ? 1 : -1)) },
    }
    persist()
  }

  function toggleSave(noteId: string, baseSaves: number) {
    const active = !(state.value.saved[noteId] ?? false)
    const current = state.value.saves[noteId] ?? baseSaves
    state.value = {
      ...state.value,
      saved: { ...state.value.saved, [noteId]: active },
      saves: { ...state.value.saves, [noteId]: Math.max(0, current + (active ? 1 : -1)) },
    }
    persist()
  }

  function addComment(noteId: string, content: string) {
    const comment: AdviceComment = {
      id: Date.now(),
      noteId,
      author: forumStore.currentUser?.nickname ?? '匿名用户',
      role: forumStore.currentUser ? `${forumStore.currentUser.grade} · ${forumStore.currentUser.role}` : '学生',
      content,
      createdAt: new Date().toISOString(),
    }
    state.value = {
      ...state.value,
      comments: {
        ...state.value.comments,
        [noteId]: [...(state.value.comments[noteId] ?? []), comment],
      },
    }
    persist()
  }

  return {
    noteComments,
    noteStats,
    getStats,
    toggleLike,
    toggleSave,
    addComment,
  }
}

function persist() {
  localStorage.setItem(storageKey, JSON.stringify(state.value))
}

function readState(): AdviceEngagementState {
  try {
    const raw = localStorage.getItem(storageKey)
    if (!raw) return { liked: {}, saved: {}, likes: {}, saves: {}, comments: {} }
    const parsed = JSON.parse(raw) as AdviceEngagementState
    return {
      liked: parsed.liked ?? {},
      saved: parsed.saved ?? {},
      likes: parsed.likes ?? {},
      saves: parsed.saves ?? {},
      comments: parsed.comments ?? {},
    }
  } catch {
    localStorage.removeItem(storageKey)
    return { liked: {}, saved: {}, likes: {}, saves: {}, comments: {} }
  }
}
