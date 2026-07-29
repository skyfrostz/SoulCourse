import { provinceKnowledge } from './knowledgeBase'

export interface SourceLink {
  label: string
  publisher: string
  url: string
}

export interface MetricSlice {
  label: string
  value: number
  color: string
}

export interface ProvinceRequirementData {
  province: string
  available: boolean
  total?: number
  metricLabel: string
  note: string
  slices: MetricSlice[]
  source: SourceLink
  capturedAt?: string
}

const guangdongPhysicsSource: SourceLink = {
  label: '2025年专科批次第一次征集志愿招生计划表（普通类物理）',
  publisher: '广东省教育考试院',
  url: 'https://eea.gd.gov.cn/attachment/0/587/587942/4755952.pdf',
}

const guangdongHistorySource: SourceLink = {
  label: '2025年专科批次第一次征集志愿招生计划表（普通类历史）',
  publisher: '广东省教育考试院',
  url: 'https://eea.gd.gov.cn/attachment/0/587/587941/4755940.pdf',
}

export const requirementData: ProvinceRequirementData[] = [
  {
    province: '广东',
    available: true,
    total: 16071,
    metricLabel: '招生计划选科要求',
    note: '2025年专科批次第一次征集志愿：物理类1048条、历史类436条官方计划，合计16071个计划数。',
    source: guangdongPhysicsSource,
    capturedAt: '2026-07-29',
    slices: [
      { label: '物理+再选不限', value: 66.42, color: '#2563eb' },
      { label: '历史+再选不限', value: 33.09, color: '#10b981' },
      { label: '有特定再选要求', value: 0.49, color: '#f59e0b' },
    ],
  },
  ...provinceKnowledge
    .filter((item) => item.province !== '广东')
    .map((item): ProvinceRequirementData => ({
      province: item.province,
      available: false,
      metricLabel: '组合级官方统计',
      note: '当前未录入可由省级考试招生机构原始材料复算的组合级人数或招生计划分布。',
      source: {
        label: `${item.province}招生考试官方入口`,
        publisher: item.authority,
        url: item.portalUrl,
      },
      slices: [],
    })),
]

export const policyTakeaways = [
  {
    title: '广东实行“3+1+2”与院校专业组',
    body: '广东改革方案明确，考生从物理、历史中选1门，再从政治、地理、化学、生物中选2门；专业组会提出对应要求。',
    source: {
      label: '广东省深化普通高校考试招生制度综合改革实施方案',
      publisher: '广东省教育考试院',
      url: 'https://eea.gd.gov.cn/tzgg/content/post_2282163.html',
    },
  },
  {
    title: '计划要求不等于考生组合热度',
    body: '当前广东统计只描述指定批次招生计划的首选、再选科目要求，不能推断考生实际选科人数或录取概率。',
    source: guangdongPhysicsSource,
  },
  {
    title: '物理类与历史类分别核对',
    body: '广东官方计划按普通类物理、普通类历史分别发布，本项目合并时仍保留首选科目维度。',
    source: guangdongHistorySource,
  },
]

export interface SourcedDataPost {
  title: string
  content: string
  source: SourceLink
}

export const sourcedDataPosts: SourcedDataPost[] = []
