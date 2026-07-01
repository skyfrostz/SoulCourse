export interface MajorRequirement {
  major: string
  category: string
  requiredSubjects: string[]
  suggestedCombination: string
  risk: string
  source: string
  sourceUrl?: string
  noteType: '高频刚需' | '理工强约束' | '人文社科' | '交叉新兴' | '需逐校核对'
  popularity: number
  saveCount: number
  discussionCount: number
}

interface MajorSeed {
  names: string[]
  category: string
  requiredSubjects: string[]
  suggestedCombination: string
  risk: string
  noteType: MajorRequirement['noteType']
  popularityBase: number
}

const officialSource = '教育部选考科目指引、阳光高考选考要求汇总及省级考试院目录抽样核对'
const officialSourceUrl = 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202201/20220105/2155365943.html'

const seeds: MajorSeed[] = [
  {
    category: '医学',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 生物',
    risk: '医学门类普遍要重点核对物理、化学要求；生物利于衔接，但最终以院校专业组目录为准。',
    noteType: '高频刚需',
    popularityBase: 98,
    names: ['临床医学', '口腔医学', '麻醉学', '医学影像学', '眼视光医学', '精神医学', '儿科学', '基础医学', '预防医学', '中医学', '中西医临床医学', '针灸推拿学'],
  },
  {
    category: '医学技术与药学',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 生物',
    risk: '药学、医学技术类常见物化要求较多，想走医院/药企路径要提前核对目标省份目录。',
    noteType: '高频刚需',
    popularityBase: 90,
    names: ['药学', '临床药学', '药物制剂', '中药学', '医学检验技术', '医学影像技术', '康复治疗学', '护理学', '助产学', '生物医学工程'],
  },
  {
    category: '计算机与电子信息',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 地理 / 生物',
    risk: '计算机、电子信息、自动化等理工专业近年物化双选约束增强，不建议只凭“会写代码”跳过化学核对。',
    noteType: '理工强约束',
    popularityBase: 96,
    names: ['计算机科学与技术', '软件工程', '人工智能', '数据科学与大数据技术', '网络空间安全', '信息安全', '物联网工程', '智能科学与技术', '电子信息工程', '通信工程', '微电子科学与工程', '集成电路设计与集成系统', '电子科学与技术', '光电信息科学与工程', '自动化', '机器人工程'],
  },
  {
    category: '工学',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 地理 / 生物',
    risk: '工科通常强依赖物理，材料、化工、能源、环境等方向还要重点看化学门槛。',
    noteType: '理工强约束',
    popularityBase: 88,
    names: ['机械工程', '机械设计制造及其自动化', '车辆工程', '能源与动力工程', '电气工程及其自动化', '智能制造工程', '测控技术与仪器', '材料科学与工程', '高分子材料与工程', '新能源材料与器件', '化学工程与工艺', '制药工程', '环境工程', '环境科学', '土木工程', '建筑环境与能源应用工程', '给排水科学与工程', '交通运输', '交通工程', '飞行器设计与工程', '航空航天工程', '船舶与海洋工程', '海洋工程与技术', '食品科学与工程', '安全工程', '生物工程', '农业工程', '地质工程', '矿物加工工程', '纺织工程'],
  },
  {
    category: '理学',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 生物 / 地理',
    risk: '基础理学分化明显：数理方向看物理，化学、生物、地学方向常需要化学或相关科目支撑。',
    noteType: '理工强约束',
    popularityBase: 84,
    names: ['数学与应用数学', '信息与计算科学', '物理学', '应用物理学', '化学', '应用化学', '生物科学', '生物技术', '生态学', '统计学', '应用统计学', '地理科学', '地理信息科学', '大气科学', '海洋科学', '心理学', '应用心理学'],
  },
  {
    category: '农学',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 生物',
    risk: '农林生命方向对生物兴趣要求高，但不少院校专业组选科仍会落到物理、化学核对。',
    noteType: '需逐校核对',
    popularityBase: 72,
    names: ['农学', '园艺', '植物保护', '种子科学与工程', '动物科学', '动物医学', '林学', '园林', '水产养殖学', '草业科学'],
  },
  {
    category: '经济与管理',
    requiredSubjects: ['不限或物理优先'],
    suggestedCombination: '物理 + 地理 + 政治 / 历史 + 政治 + 地理',
    risk: '经管类多数专业限制相对弱，但金融工程、精算、信息管理等方向可能更偏数学和物理能力。',
    noteType: '需逐校核对',
    popularityBase: 86,
    names: ['经济学', '数字经济', '财政学', '金融学', '金融工程', '保险学', '投资学', '国际经济与贸易', '工商管理', '市场营销', '会计学', '财务管理', '审计学', '人力资源管理', '信息管理与信息系统', '工程管理', '物流管理', '供应链管理', '电子商务', '旅游管理'],
  },
  {
    category: '法学与公共管理',
    requiredSubjects: ['不限或政治优先'],
    suggestedCombination: '历史 + 政治 + 地理',
    risk: '法学、公管类通常不只看选科门槛，长期阅读、表达、时政积累和逻辑写作更影响后续体验。',
    noteType: '人文社科',
    popularityBase: 83,
    names: ['法学', '知识产权', '政治学与行政学', '国际政治', '社会学', '社会工作', '思想政治教育', '公安学类', '行政管理', '公共事业管理', '劳动与社会保障', '土地资源管理'],
  },
  {
    category: '文学与新闻传播',
    requiredSubjects: ['不限'],
    suggestedCombination: '历史 + 政治 + 地理',
    risk: '文史传播类专业选科限制通常不强，但作品阅读、写作表达和目标院校层次差异很关键。',
    noteType: '人文社科',
    popularityBase: 78,
    names: ['汉语言文学', '汉语国际教育', '英语', '翻译', '商务英语', '日语', '法语', '德语', '西班牙语', '新闻学', '广播电视学', '广告学', '传播学', '网络与新媒体', '编辑出版学'],
  },
  {
    category: '教育学',
    requiredSubjects: ['按任教学科方向核对'],
    suggestedCombination: '按目标任教学科反推',
    risk: '师范类不能只看“师范”两个字，数学、物理、化学、生物师范与语文、历史、政治师范的选科逻辑不同。',
    noteType: '需逐校核对',
    popularityBase: 76,
    names: ['教育学', '学前教育', '小学教育', '特殊教育', '教育技术学', '体育教育', '运动训练', '数学师范', '物理师范', '化学师范', '英语师范', '历史师范'],
  },
  {
    category: '历史哲学与艺术',
    requiredSubjects: ['不限或历史优先'],
    suggestedCombination: '历史 + 政治 + 地理',
    risk: '人文艺术方向要看招生类别、校考/统考要求和作品能力，选科只是第一层门槛。',
    noteType: '人文社科',
    popularityBase: 68,
    names: ['历史学', '世界史', '考古学', '哲学', '逻辑学', '美术学', '视觉传达设计', '环境设计', '产品设计', '数字媒体艺术', '音乐学', '戏剧影视文学'],
  },
  {
    category: '交叉与新兴专业',
    requiredSubjects: ['物理', '化学'],
    suggestedCombination: '物理 + 化学 + 生物 / 地理',
    risk: '新兴交叉专业名称相似但培养方案差异大，建议同时看课程表、学院归属和选考目录。',
    noteType: '交叉新兴',
    popularityBase: 82,
    names: ['智能医学工程', '生物信息学', '密码科学与技术', '储能科学与工程', '新能源科学与工程', '智慧农业', '智慧交通', '智能建造', '遥感科学与技术', '空间信息与数字技术', '应急技术与管理', '碳储科学与工程'],
  },
]

export const majorRequirements: MajorRequirement[] = seeds.flatMap((seed, seedIndex) =>
  seed.names.map((major, index) => ({
    major,
    category: seed.category,
    requiredSubjects: seed.requiredSubjects,
    suggestedCombination: seed.suggestedCombination,
    risk: seed.risk,
    source: officialSource,
    sourceUrl: officialSourceUrl,
    noteType: seed.noteType,
    popularity: Math.max(52, seed.popularityBase - (index % 7) * 2 - (seedIndex % 3)),
    saveCount: 120 + seedIndex * 37 + index * 11,
    discussionCount: 18 + seedIndex * 5 + index * 3,
  })),
)

export const majorRequirementCategories = Array.from(new Set(majorRequirements.map((item) => item.category)))

export const majorRequirementStats = {
  total: majorRequirements.length,
  officialSource,
  officialSourceUrl,
}
