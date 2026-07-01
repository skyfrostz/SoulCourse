export interface MajorRequirement {
  major: string
  category: string
  requiredSubjects: string[]
  suggestedCombination: string
  risk: string
  source: string
}

export const majorRequirements: MajorRequirement[] = [
  {
    major: '临床医学',
    category: '医学',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 生物',
    risk: '多数院校专业组会同时要求物理和化学，生物有助于学习衔接，但仍需逐校核对。',
    source: '教育部选考科目指引、上海/浙江/甘肃公开选考目录',
  },
  {
    major: '计算机科学与技术',
    category: '工学',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 地理 / 生物',
    risk: '理工类专业物化双选要求显著增加，若不选化学，部分院校专业组会受限。',
    source: '上海教育考试院答问、江苏省教育考试院选科要求说明',
  },
  {
    major: '法学',
    category: '法学',
    requiredSubjects: ['不限或政治优先'],
    suggestedCombination: '历史 + 政治 + 地理',
    risk: '多数院校不限选科，但政治对学习兴趣和长期表达训练有帮助。',
    source: '多省普通高校本科专业选考科目要求目录',
  },
  {
    major: '汉语言文学',
    category: '文学',
    requiredSubjects: ['不限'],
    suggestedCombination: '历史 + 政治 + 地理',
    risk: '专业限制通常不强，重点应看语文表达、阅读积累和目标院校层次。',
    source: '多省普通高校本科专业选考科目要求目录',
  },
  {
    major: '机械工程',
    category: '工学',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 地理',
    risk: '工科路径通常强依赖物理，2024 后化学要求需要重点核对。',
    source: '教育部选考科目指引、公开考试院目录',
  },
  {
    major: '师范类',
    category: '教育学',
    requiredSubjects: ['看具体学科方向'],
    suggestedCombination: '按目标任教学科反推',
    risk: '数学/物理/化学师范通常偏物理方向，语文/历史/政治师范可偏历史方向。',
    source: '高校专业组选科目录',
  },
]
