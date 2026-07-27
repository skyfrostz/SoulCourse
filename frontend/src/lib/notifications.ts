import type { NotificationType } from '../types/forum'

export const notificationTypeLabels: Record<NotificationType, string> = {
  comment: '评论互动',
  like: '赞与收藏',
  favorite: '赞与收藏',
  policy: '政策更新',
  profile: '画像建议',
  system: '系统提醒',
  follow: '关注动态',
}
