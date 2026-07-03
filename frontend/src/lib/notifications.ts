export type NotificationType = 'comment' | 'policy' | 'profile' | 'system' | 'follow'

export interface AppNotification {
  id: string
  type: NotificationType
  title: string
  summary: string
  body: string
  targetLabel: string
  targetUrl: string
  createdAt: string
  priority: 'high' | 'normal'
}

export const notificationSeeds: AppNotification[] = [
  {
    id: 'comment-physics-topic',
    type: 'comment',
    title: '你关注的话题有新讨论',
    summary: '“物理方向组合怎么选”新增 3 条高质量评论。',
    body: '认证规划师补充了“物理+化学”对医学、工科和农学专业组的影响，建议收藏后和目标专业一起核对。',
    targetLabel: '查看话题',
    targetUrl: '/topics/physics-combo',
    createdAt: '2026-07-03T10:24:00+08:00',
    priority: 'high',
  },
  {
    id: 'profile-mbti',
    type: 'profile',
    title: '完善画像可以提高推荐准确度',
    summary: '你的 MBTI、目标专业和压力承受度还可以继续补充。',
    body: '系统会结合目标省份、意向组合、成绩波动和兴趣画像生成更具体的选科建议。',
    targetLabel: '完善资料',
    targetUrl: '/settings',
    createdAt: '2026-07-03T09:10:00+08:00',
    priority: 'normal',
  },
  {
    id: 'policy-guangdong',
    type: 'policy',
    title: '广东政策库新增网页化阅读',
    summary: '广东招生政策、选考目录和招生工作规定现在可以在站内阅读。',
    body: '进入省份资料包后，点击任一政策卡片即可打开网页化正文，并保留官方 PDF/附件下载入口。',
    targetLabel: '查看广东资料包',
    targetUrl: '/knowledge/%E5%B9%BF%E4%B8%9C',
    createdAt: '2026-07-02T18:45:00+08:00',
    priority: 'high',
  },
  {
    id: 'follow-teacher',
    type: 'follow',
    title: '你关注的王老师更新了评论',
    summary: '王老师在临床医学专业论坛补充了“物化双选”的核对建议。',
    body: '如果目标包含临床医学、口腔医学、药学等方向，建议先看本省专业选考目录，再看高校招生章程中的身体条件和单科要求。',
    targetLabel: '进入临床医学论坛',
    targetUrl: '/requirements/%E4%B8%B4%E5%BA%8A%E5%8C%BB%E5%AD%A6',
    createdAt: '2026-07-02T15:30:00+08:00',
    priority: 'normal',
  },
  {
    id: 'system-publish-tip',
    type: 'system',
    title: '发帖标签已升级',
    summary: '发布窗口支持输入关键词后按回车生成标签。',
    body: '建议用“省份 + 组合 + 目标专业”作为标签，例如：广东、物化生、临床医学，这样更容易被同类用户搜到。',
    targetLabel: '去发帖',
    targetUrl: '/',
    createdAt: '2026-07-01T20:18:00+08:00',
    priority: 'normal',
  },
]

export const notificationTypeLabels: Record<NotificationType, string> = {
  comment: '评论互动',
  policy: '政策更新',
  profile: '画像建议',
  system: '系统提醒',
  follow: '关注动态',
}
