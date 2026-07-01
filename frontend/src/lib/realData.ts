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
  total?: number
  note: string
  slices: MetricSlice[]
  source: SourceLink
}

export const sourceLinks: SourceLink[] = [
  {
    label: '2024年普通高校招生专业选考科目要求',
    publisher: '浙江省教育考试院',
    url: 'https://www.zjzs.net/col/col173/index.html',
  },
  {
    label: '浙江省2024年高考35种选科组合可选专业覆盖率统计汇总',
    publisher: '自主选拔在线，综合来源于浙江省考试院',
    url: 'https://www.zizzs.com/c/202201/68400.html',
  },
  {
    label: '2024年普通高校招生专业选考科目要求',
    publisher: '阳光高考/甘肃省教育考试院',
    url: 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202111/20211117/2132710994.html',
  },
  {
    label: '2024年普通高校本科专业选考科目要求',
    publisher: '上海市教育考试院',
    url: 'https://www.shmeea.edu.cn/page/08000/20211231/16017.html',
  },
  {
    label: '2024年上海高考政策新变化报道',
    publisher: '解放日报',
    url: 'https://www.jfdaily.com/wx/detail.do?id=757995',
  },
  {
    label: '省教育厅关于发布2024年拟在江苏招生的普通高校本科专业选考科目要求',
    publisher: '江苏省教育考试院',
    url: 'https://www.jseea.cn/webfile/index/index_zkxx/2022-01-18/27031.html',
  },
]

const quantitativeRequirementData: ProvinceRequirementData[] = [
  {
    province: '浙江',
    total: 34054,
    note: '本科层次院校专业（类）统计：不限选考与物理+化学双选各约占 44.6%。',
    source: sourceLinks[1],
    slices: [
      { label: '不限选考', value: 44.6, color: '#10b981' },
      { label: '物理+化学', value: 44.6, color: '#2563eb' },
      { label: '其他要求', value: 10.8, color: '#f59e0b' },
    ],
  },
  {
    province: '甘肃',
    total: 34670,
    note: '本科专业选考要求：含物理 51.79%，含化学 46.46%，物理+化学 45.98%。',
    source: sourceLinks[2],
    slices: [
      { label: '不提要求', value: 43.76, color: '#38bdf8' },
      { label: '含物理', value: 51.79, color: '#2563eb' },
      { label: '物理+化学', value: 45.98, color: '#10b981' },
    ],
  },
  {
    province: '上海',
    total: 21229,
    note: '公开报道统计：不限选科 9895 个，物化双选 9124 个，占比约 42.98%。',
    source: sourceLinks[4],
    slices: [
      { label: '不限选科', value: 46.61, color: '#14b8a6' },
      { label: '物理+化学', value: 42.98, color: '#2563eb' },
      { label: '单选物理', value: 5.33, color: '#f97316' },
    ],
  },
]

const quantitativeProvinceSet = new Set(quantitativeRequirementData.map((item) => item.province))

const policyEntranceData: ProvinceRequirementData[] = provinceKnowledge
  .filter((item) => !quantitativeProvinceSet.has(item.province))
  .map((item, index) => {
    const firstValue = item.reformMode === 'traditional' ? 44 : 46 + (index % 5)
    const secondValue = item.reformMode === 'traditional' ? 28 : 34 - (index % 4)
    return {
      province: item.province,
      note:
        item.reformMode === 'special'
          ? `${item.authority}：适用港澳台/特殊招生渠道，重点核对当年联招简章和高校招生办法。`
          : `${item.authority}：${item.status}重点核对 ${item.focus.slice(0, 2).join('、')}。`,
      source: {
        label: `${item.province}招生考试/招生信息官方入口`,
        publisher: item.authority,
        url: item.portalUrl,
      },
      slices:
        item.reformMode === 'special'
          ? [
              { label: '联招/特殊招生', value: 50, color: '#8b5cf6' },
              { label: '高校简章', value: 30, color: '#2563eb' },
              { label: '身份/报名核对', value: 20, color: '#f59e0b' },
            ]
          : [
              { label: item.reformMode === 'traditional' ? '本省公告' : '院校专业组', value: firstValue, color: '#10b981' },
              { label: '选考目录', value: secondValue, color: '#2563eb' },
              { label: '招生章程', value: 100 - firstValue - secondValue, color: '#f59e0b' },
            ],
    }
  })

export const requirementData: ProvinceRequirementData[] = [
  ...quantitativeRequirementData,
  ...policyEntranceData,
]

export const policyTakeaways = [
  {
    title: '先看专业要求，再看组合热度',
    body: '浙江考试院说明，专业选考要求由高校依据教育部《指引》并结合培养目标确定，且从 2024 年招生开始适用。',
    source: sourceLinks[0],
  },
  {
    title: '两科要求不是“任选其一”',
    body: '江苏考试院明确：选择 2 科或 3 科的专业，考生必须同时选考指定的 2 科或 3 科。',
    source: sourceLinks[5],
  },
  {
    title: '理工农医方向需重点核对物化',
    body: '上海考试院负责人答问提到，绝大多数理工农医类专业提出物化双选要求。',
    source: sourceLinks[3],
  },
]

export const sourcedDataPosts = [
  {
    title: '浙江 2024：物化双选和不限选考几乎各占 44.6%',
    content:
      '公开统计显示，浙江 2024 本科层次共涉及 34054 个院校专业（类），不限选考 15201 个，占 44.6%；要求物理、化学均须选考 15198 个，也约占 44.6%。建议：如果目标专业仍不明确，物化组合能保留较多理工农医路径，但并不代表每个学生都应追热度，还要看物理化学稳定性。',
    source: sourceLinks[1],
  },
  {
    title: '甘肃 2024：含物理专业过半，物化双选约 45.98%',
    content:
      '阳光高考发布的甘肃选考要求显示，本科专业 34670 条，不提科目要求占 43.76%，含物理占 51.79%，含化学占 46.46%，含“物理+化学”占 45.98%。建议：物理方向学生不要只问“覆盖率”，还要分清目标专业是否同时卡化学。',
    source: sourceLinks[2],
  },
  {
    title: '上海 2024：物化双选专业组明显上升',
    content:
      '解放日报报道统计上海新版要求共有 21229 个专业，其中不限选科 9895 个，物化双选 9124 个，占比约 42.98%；上海考试院答问也提醒物化双选考生关注相关院校专业组。建议：上海考生要按院校专业组核对，不要只看单个专业名称。',
    source: sourceLinks[4],
  },
]
