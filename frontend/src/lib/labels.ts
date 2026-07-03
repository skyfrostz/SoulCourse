import type { Category, Role, Subject, Track } from '../types/forum'

export const trackLabels: Record<Track, string> = {
  physics: '物理方向',
  history: '历史方向',
}

export const subjectLabels: Record<Subject, string> = {
  chemistry: '化学',
  biology: '生物',
  politics: '政治',
  geography: '地理',
}

export const categoryLabels: Record<Category, string> = {
  experience: '经验帖',
  question: '提问',
  data: '数据建议',
}

export const roleLabels: Record<Role, string> = {
  student: '学生',
  parent: '家长',
  teacher: '老师',
  counselor: '规划师',
}

export const subjectAccent: Record<Subject, string> = {
  chemistry: '#0f9f7a',
  biology: '#19a974',
  politics: '#ef4444',
  geography: '#f59e0b',
}
