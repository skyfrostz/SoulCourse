import { majorRequirements } from './majorRequirements'
import type { Comment, Post, SubjectInsight, Subject, Track } from '../types/forum'

const samplePostSeeds: Array<Omit<Post, 'viewerLiked' | 'viewerFavorited' | 'viewerFollowing'>> = [
  {
    id: 1,
    authorName: '清北学长',
    authorRole: 'student',
    title: '物化生真的是有优势吗？过来人的真实感受',
    content:
      '新高考第一届，很多人都在摸索。我当时选物化生，压力大但机会也多。现在回头看，关键不是组合热门，而是能不能长期稳定投入。',
    imageUrls: ['https://images.unsplash.com/photo-1522202176988-66273c2fd55f?auto=format&fit=crop&w=1200&q=80'],
    tags: ['物化生', '学长经验', '学习节奏'],
    track: 'physics',
    electives: ['chemistry', 'biology'],
    category: 'experience',
    grade: '高三毕业生',
    province: '浙江',
    likesCount: 1200,
    commentsCount: 212,
    favoritesCount: 316,
    createdAt: '2026-06-30T08:30:00+08:00',
    updatedAt: '2026-06-30T08:30:00+08:00',
  },
  {
    id: 2,
    authorName: '云中月',
    authorRole: 'student',
    title: '史政地组合的学习节奏和提分思路',
    content:
      '文科组合也有天花板，分享如何兼顾背诵与理解，稳步提分和安排复盘时间。材料题不是只靠背，方法比时间更重要。',
    imageUrls: ['https://images.unsplash.com/photo-1456513080510-7bf3a84b82f8?auto=format&fit=crop&w=1200&q=80'],
    tags: ['史政地', '提分方法'],
    track: 'history',
    electives: ['politics', 'geography'],
    category: 'experience',
    grade: '高二',
    province: '山东',
    likesCount: 642,
    commentsCount: 86,
    favoritesCount: 144,
    createdAt: '2026-06-29T19:10:00+08:00',
    updatedAt: '2026-06-29T19:10:00+08:00',
  },
  {
    id: 3,
    authorName: '焦虑的家长',
    authorRole: 'parent',
    title: '孩子高一，选科还在纠结，希望听听大家的建议',
    content:
      '孩子理科还可以，但不喜欢物理，未来想当老师，选史政地还是史政生更合适？家里想尊重孩子，也想把风险看清楚。',
    imageUrls: [],
    tags: ['提问', '师范', '风险判断'],
    track: 'history',
    electives: ['politics', 'biology'],
    category: 'question',
    grade: '高一',
    province: '江苏',
    likesCount: 102,
    commentsCount: 37,
    favoritesCount: 58,
    createdAt: '2026-06-29T16:45:00+08:00',
    updatedAt: '2026-06-29T16:45:00+08:00',
  },
  {
    id: 4,
    authorName: '地理小能手',
    authorRole: 'student',
    title: '物化地组合：如何平衡理科思维与文科记忆？',
    content:
      '地理被低估了。学好地理对理科生来说是提分利器，但它需要图表训练和知识网络，不是考前突击就能稳定的科目。',
    imageUrls: ['https://images.unsplash.com/photo-1500530855697-b586d89ba3ee?auto=format&fit=crop&w=1200&q=80'],
    tags: ['物化地', '地理', '图表题'],
    track: 'physics',
    electives: ['chemistry', 'geography'],
    category: 'experience',
    grade: '高三',
    province: '广东',
    likesCount: 389,
    commentsCount: 54,
    favoritesCount: 121,
    createdAt: '2026-06-28T15:20:00+08:00',
    updatedAt: '2026-06-28T15:20:00+08:00',
  },
  {
    id: 5,
    authorName: '选科研究所',
    authorRole: 'counselor',
    title: '2026届各组合专业覆盖率汇总',
    content:
      '基于多省教育考试院公开数据整理。选科前一定要把目标专业组、学校层次、赋分规则一起看，不要只看一个覆盖率数字。',
    imageUrls: [],
    tags: ['数据建议', '专业覆盖率', '选科要求'],
    track: 'physics',
    electives: ['chemistry', 'biology'],
    category: 'data',
    grade: '高一',
    province: '全国',
    likesCount: 1100,
    commentsCount: 128,
    favoritesCount: 480,
    createdAt: '2026-06-27T20:00:00+08:00',
    updatedAt: '2026-06-27T20:00:00+08:00',
  },
  {
    id: 6,
    authorName: '奋斗的高三党',
    authorRole: 'student',
    title: '从学渣到班级前十：我的逆袭经验',
    content:
      '选科后成绩波动很大，找到适合自己的节奏最重要。附各科提分方法：物理重建模型，化学整理反应链，生物用错题回扣课本。',
    imageUrls: ['https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80'],
    tags: ['逆袭经验', '学习方法'],
    track: 'physics',
    electives: ['chemistry', 'biology'],
    category: 'experience',
    grade: '高三',
    province: '湖北',
    likesCount: 611,
    commentsCount: 73,
    favoritesCount: 199,
    createdAt: '2026-06-26T11:30:00+08:00',
    updatedAt: '2026-06-26T11:30:00+08:00',
  },
  {
    id: 7,
    authorName: '数据观察员',
    authorRole: 'counselor',
    title: '浙江2024选考目录：物化双选和不限选考几乎持平',
    content:
      '基于浙江省教育考试院目录的公开统计，本科层次 34054 个院校专业（类）中，不限选考和物理+化学双选各约 44.6%。中肯建议：物化确实保留大量理工农医路径，但对物理、化学连续投入要求高，不适合只为了“覆盖率”盲选。',
    imageUrls: ['https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?auto=format&fit=crop&w=1200&q=80'],
    tags: ['真实数据', '浙江选科', '物理化学'],
    track: 'physics',
    electives: ['chemistry', 'biology'],
    category: 'data',
    grade: '高一',
    province: '浙江',
    likesCount: 433,
    commentsCount: 41,
    favoritesCount: 260,
    createdAt: '2026-06-30T21:20:00+08:00',
    updatedAt: '2026-06-30T21:20:00+08:00',
  },
  {
    id: 8,
    authorName: '升学规划师B',
    authorRole: 'counselor',
    title: '上海2024：为什么物化双选专业组明显变多？',
    content:
      '上海公开报道显示，2024 选考目录中物化双选约占 42.98%。这意味着理工农医方向需要更早核对院校专业组。中肯建议：上海考生不要只问“这个专业要不要物理”，还要看同一院校专业组里是否同时要求化学。',
    imageUrls: [],
    tags: ['真实数据', '上海高考', '专业组'],
    track: 'physics',
    electives: ['chemistry', 'geography'],
    category: 'data',
    grade: '高一',
    province: '上海',
    likesCount: 366,
    commentsCount: 29,
    favoritesCount: 188,
    createdAt: '2026-06-30T20:10:00+08:00',
    updatedAt: '2026-06-30T20:10:00+08:00',
  },
]

const comboSeeds: Array<{
  label: string
  track: Post['track']
  electives: Post['electives']
  province: string
  target: string
  rhythm: string
}> = [
  { label: '物化生', track: 'physics', electives: ['chemistry', 'biology'], province: '浙江', target: '医学、药学、生命科学和传统工科', rhythm: '三科都要持续投入，适合校内排名稳定且能复盘错题的人' },
  { label: '物化地', track: 'physics', electives: ['chemistry', 'geography'], province: '广东', target: '计算机、电子信息、智能制造和地理信息', rhythm: '物化保专业边界，地理靠图表和材料题拉开差距' },
  { label: '物生地', track: 'physics', electives: ['biology', 'geography'], province: '湖北', target: '部分理科、管理、心理和地学方向', rhythm: '压力相对均衡，但医学药学等方向要重点核对化学门槛' },
  { label: '物化政', track: 'physics', electives: ['chemistry', 'politics'], province: '山东', target: '理工农医和公安、法学交叉兴趣', rhythm: '学科跨度大，适合既能刷体系题也愿意长期积累时政的人' },
  { label: '物生政', track: 'physics', electives: ['biology', 'politics'], province: '重庆', target: '心理、公共卫生、管理和部分理科方向', rhythm: '要接受化学受限，同时用政治表达能力补充长期竞争力' },
  { label: '物政地', track: 'physics', electives: ['politics', 'geography'], province: '四川', target: '地理信息、公共管理、经济管理和部分工科', rhythm: '适合物理基础还行但化学生物压力较大的学生，专业目录必须逐条查' },
  { label: '史政地', track: 'history', electives: ['politics', 'geography'], province: '江苏', target: '法学、新闻传播、公共管理、师范和人文社科', rhythm: '路径清晰但不是轻松组合，材料分析和时政积累要长期做' },
  { label: '史政生', track: 'history', electives: ['politics', 'biology'], province: '福建', target: '师范、法学、心理、护理和公共事业管理', rhythm: '适合人文表达强又保留一点生命科学兴趣的人，医学方向需谨慎核对' },
  { label: '史地生', track: 'history', electives: ['geography', 'biology'], province: '湖南', target: '教育、旅游管理、地理科学、心理和部分农林方向', rhythm: '体验相对温和，但要提前接受理工农医专业边界' },
  { label: '史化生', track: 'history', electives: ['chemistry', 'biology'], province: '上海', target: '护理、中药、食品、心理和部分交叉专业', rhythm: '历史方向叠加化学生物，适合兴趣明确但要逐校确认首选科目要求' },
  { label: '史化政', track: 'history', electives: ['chemistry', 'politics'], province: '安徽', target: '法学、公安、药事管理、公共政策和教育方向', rhythm: '政治利于表达，化学保留少量理科可能，但学习负担不低' },
  { label: '史化地', track: 'history', electives: ['chemistry', 'geography'], province: '广西', target: '地理科学、资源环境、管理和部分师范方向', rhythm: '兼具材料、图表和化学体系，适合不排斥理科思维的历史方向学生' },
]

const experienceVariants = [
  {
    authorName: '同组合过来人',
    title: (seed: (typeof comboSeeds)[number]) => `${seed.label} 一个月真实体验：我会重点看这三件事`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `${seed.label} 适合关注 ${seed.target} 的同学。我的感受是：${seed.rhythm}。不要只问“这个组合热门吗”，更要看最近三次大考单科排名、错题修复速度和每天可投入时间。`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '真实体验', '学习节奏', '选科笔记'],
    image: 'https://images.unsplash.com/photo-1513258496099-48168024aec0?auto=format&fit=crop&w=1200&q=80',
  },
  {
    authorName: '晚自习复盘员',
    title: (seed: (typeof comboSeeds)[number]) => `选了${seed.label}后，我把晚自习改成了这套顺序`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `刚分班时最慌的是任务堆在一起。后来我把“当天新课、错题归档、周末大复盘”拆开做，${seed.rhythm}。如果你也看重${seed.target}，别只看覆盖率，先看自己每天能不能稳定完成闭环。`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '晚自习', '错题复盘', '真实记录'],
    image: 'https://images.unsplash.com/photo-1503676260728-1c00da094a0b?auto=format&fit=crop&w=1200&q=80',
  },
  {
    authorName: '县中住宿生',
    title: (seed: (typeof comboSeeds)[number]) => `${seed.province}普通高中选${seed.label}，资源不够时怎么补`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `我们学校不是每个组合都有最强师资，所以我更关注课后答疑和同桌学习氛围。${seed.label}对应${seed.target}，但真实体验要看学校开班质量、老师稳定性和同组合人数。`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '普通高中', '学校资源', '同伴氛围'],
    image: 'https://images.unsplash.com/photo-1523050854058-8df90110c9f1?auto=format&fit=crop&w=1200&q=80',
  },
  {
    authorName: '高三回看高一',
    title: (seed: (typeof comboSeeds)[number]) => `如果重选${seed.label}，我会提前问老师这 4 个问题`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `高一选科时我只问“好不好提分”，后来发现更该问：目标专业卡不卡科、校内排名能否稳定、老师排课是否成熟、自己是否接受${seed.rhythm}。这几个答案比热度更管用。`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '高三回看', '避坑清单', '老师建议'],
    image: 'https://images.unsplash.com/photo-1523580846011-d3a5bc25702b?auto=format&fit=crop&w=1200&q=80',
  },
]

const questionVariants = [
  {
    authorName: '高一纠结星人',
    authorRole: 'student' as const,
    grade: '高一',
    title: (seed: (typeof comboSeeds)[number]) => `${seed.label}要不要硬撑？我最怕选完后每天都在补短板`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `我在${seed.province}，现在倾向${seed.label}，目标大概是${seed.target}。但最近两次小考有一科波动很大，老师说别只看兴趣。想问同组合的同学：你们是怎么判断“能长期学下去”的？`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '学生提问', '学习压力', '长期投入'],
  },
  {
    authorName: '一起查目录的妈妈',
    authorRole: 'parent' as const,
    grade: '高一家长',
    title: (seed: (typeof comboSeeds)[number]) => `孩子想选${seed.label}，我该支持兴趣还是先卡专业边界？`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `孩子说喜欢${seed.label}的学习节奏，但家里担心${seed.target}后续选择受限。我们不想替孩子做决定，只想把证据摆清楚：成绩排名、专业目录、学校开班和心理承压，该先看哪个？`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '亲子沟通', '目标专业', '风险核对'],
  },
  {
    authorName: '转科边缘人',
    authorRole: 'student' as const,
    grade: '高二',
    title: (seed: (typeof comboSeeds)[number]) => `高二发现${seed.label}不适合，还来得及调整吗？`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `分班后才发现${seed.rhythm}这句话太真实了。现在想保留${seed.target}，但每天补弱科很累。有没有转过组合的同学分享一下成本：课程进度、赋分排名、老师态度会不会很难受？`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '转科成本', '高二焦虑', '真实求助'],
  },
  {
    authorName: '班主任让我再想想',
    authorRole: 'student' as const,
    grade: '高一',
    title: (seed: (typeof comboSeeds)[number]) => `班主任说别为“热门”选${seed.label}，我该怎么复盘？`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `我原本觉得${seed.label}覆盖面不错，但老师提醒我先看校内排名和错题类型。想请大家帮我列一个复盘表：最近三次成绩、目标专业、兴趣、家庭资源、学校师资，权重怎么排？`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '班主任建议', '复盘表', '决策证据'],
  },
  {
    authorName: '目标专业摇摆中',
    authorRole: 'student' as const,
    grade: '高一',
    title: (seed: (typeof comboSeeds)[number]) => `还没定专业，先选${seed.label}会不会太冒险？`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `现在只知道自己可能会看${seed.target}，但没有特别坚定的专业。身边同学都在聊覆盖率，我反而更乱。想问：不确定专业时，是先保选择面，还是先保自己擅长科目的排名？`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '未定专业', '覆盖率', '赋分排名'],
  },
  {
    authorName: '教务处路过',
    authorRole: 'teacher' as const,
    grade: '教师',
    title: (seed: (typeof comboSeeds)[number]) => `学校${seed.label}开班人数不稳，会影响学习体验吗？`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `我们学校今年${seed.label}咨询量不少，但最终开班还要看人数。想收集大家的真实情况：小班是不是一定更好？如果师资临时调配，学生该怎么判断这个组合值不值得坚持？`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '开班人数', '学校资源', '教师视角'],
  },
]

const dataVariants = [
  {
    authorName: '目录核对员',
    title: (seed: (typeof comboSeeds)[number]) => `${seed.label}专业边界速查：先看目录，再看热度`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `${seed.label} 的判断不能只靠社区热度。建议先用本省考试院目录或阳光高考核对 ${seed.target}，再看校内成绩稳定性。特别提醒：同名专业在不同高校专业组里，选考要求可能不同。`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '数据建议', '省份目录', '专业覆盖'],
    image: 'https://images.unsplash.com/photo-1460925895917-afdab827c52f?auto=format&fit=crop&w=1200&q=80',
  },
  {
    authorName: '赋分样本观察',
    title: (seed: (typeof comboSeeds)[number]) => `${seed.label}别只比原始分，排名波动更值得看`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `公开讨论里最容易被忽略的是赋分排名。${seed.label}如果竞争人群集中，原始分好看也不等于最终优势。建议把近三次校内排名、同组合人数和目标专业目录放在同一张表里。`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '赋分排名', '数据对比', '校内样本'],
    image: 'https://images.unsplash.com/photo-1551288049-bebda4e38f71?auto=format&fit=crop&w=1200&q=80',
  },
  {
    authorName: '专业组拆解员',
    title: (seed: (typeof comboSeeds)[number]) => `同样是${seed.label}，不同院校专业组差别很大`,
    content: (seed: (typeof comboSeeds)[number]) =>
      `不要把“能报某专业”理解成“所有学校都能报”。${seed.target}里很多方向要逐校看专业组，尤其是强基、医学、工科实验班和师范类方向，入口名称相似但限制可能不同。`,
    tags: (seed: (typeof comboSeeds)[number]) => [seed.label, '专业组', '逐校核对', '院校差异'],
    image: 'https://images.unsplash.com/photo-1506784983877-45594efa4cbe?auto=format&fit=crop&w=1200&q=80',
  },
]

const categoryBlueprints: Array<{
  category: Post['category']
  authorName: (seed: (typeof comboSeeds)[number], comboIndex: number) => string
  authorRole: (seed: (typeof comboSeeds)[number], comboIndex: number) => Post['authorRole']
  grade: (seed: (typeof comboSeeds)[number], comboIndex: number) => string
  title: (seed: (typeof comboSeeds)[number], comboIndex: number) => string
  content: (seed: (typeof comboSeeds)[number], comboIndex: number) => string
  tags: (seed: (typeof comboSeeds)[number], comboIndex: number) => string[]
  image: (seed: (typeof comboSeeds)[number], comboIndex: number) => string
}> = [
  {
    category: 'experience',
    authorName: (_seed, index) => experienceVariants[index % experienceVariants.length].authorName,
    authorRole: () => 'student',
    grade: (_seed, index) => (index % 3 === 0 ? '高三毕业生' : index % 3 === 1 ? '高二' : '大一在读'),
    title: (seed, index) => experienceVariants[index % experienceVariants.length].title(seed),
    content: (seed, index) => experienceVariants[index % experienceVariants.length].content(seed),
    tags: (seed, index) => experienceVariants[index % experienceVariants.length].tags(seed),
    image: (_seed, index) => experienceVariants[index % experienceVariants.length].image,
  },
  {
    category: 'question',
    authorName: (_seed, index) => questionVariants[index % questionVariants.length].authorName,
    authorRole: (_seed, index) => questionVariants[index % questionVariants.length].authorRole,
    grade: (_seed, index) => questionVariants[index % questionVariants.length].grade,
    title: (seed, index) => questionVariants[index % questionVariants.length].title(seed),
    content: (seed, index) => questionVariants[index % questionVariants.length].content(seed),
    tags: (seed, index) => questionVariants[index % questionVariants.length].tags(seed),
    image: () => '',
  },
  {
    category: 'data',
    authorName: (_seed, index) => dataVariants[index % dataVariants.length].authorName,
    authorRole: () => 'counselor',
    grade: () => '高一',
    title: (seed, index) => dataVariants[index % dataVariants.length].title(seed),
    content: (seed, index) => dataVariants[index % dataVariants.length].content(seed),
    tags: (seed, index) => dataVariants[index % dataVariants.length].tags(seed),
    image: (_seed, index) => dataVariants[index % dataVariants.length].image,
  },
]

const generatedPostSeeds: Array<Omit<Post, 'viewerLiked' | 'viewerFavorited' | 'viewerFollowing'>> = comboSeeds.flatMap((seed, comboIndex) =>
  categoryBlueprints.map((blueprint, categoryIndex) => {
    const id = 100 + comboIndex * categoryBlueprints.length + categoryIndex
    const image = blueprint.image(seed, comboIndex)
    return {
      id,
      authorName: blueprint.authorName(seed, comboIndex),
      authorRole: blueprint.authorRole(seed, comboIndex),
      title: blueprint.title(seed, comboIndex),
      content: blueprint.content(seed, comboIndex),
      imageUrls: image ? [image] : [],
      tags: blueprint.tags(seed, comboIndex),
      track: seed.track,
      electives: seed.electives,
      category: blueprint.category,
      grade: blueprint.grade(seed, comboIndex),
      province: seed.province,
      likesCount: 260 + comboIndex * 47 + categoryIndex * 93,
      commentsCount: 38 + comboIndex * 9 + categoryIndex * 17,
      favoritesCount: 104 + comboIndex * 31 + categoryIndex * 58,
      createdAt: `2026-06-${String(25 - (comboIndex % 8)).padStart(2, '0')}T${String(10 + categoryIndex * 3).padStart(2, '0')}:20:00+08:00`,
      updatedAt: `2026-06-${String(25 - (comboIndex % 8)).padStart(2, '0')}T${String(10 + categoryIndex * 3).padStart(2, '0')}:20:00+08:00`,
    }
  }),
)

const majorForumImages = [
  'https://images.unsplash.com/photo-1434030216411-0b793f4b4173?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1509062522246-3755977927d7?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1522202176988-66273c2fd55f?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1513258496099-48168024aec0?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?auto=format&fit=crop&w=1200&q=80',
]

function inferMajorTrack(item: (typeof majorRequirements)[number]): Track {
  if (item.suggestedCombination.includes('历史') && !item.suggestedCombination.startsWith('物理')) return 'history'
  if (['法学与公共管理', '文学与新闻传播', '历史哲学与艺术'].includes(item.category)) return 'history'
  return 'physics'
}

function inferMajorElectives(item: (typeof majorRequirements)[number], track: Track): Subject[] {
  const text = `${item.requiredSubjects.join(' ')} ${item.suggestedCombination} ${item.category}`
  const picked: Subject[] = []
  if (text.includes('化学')) picked.push('chemistry')
  if (text.includes('生物') && picked.length < 2) picked.push('biology')
  if (text.includes('政治') && picked.length < 2) picked.push('politics')
  if (text.includes('地理') && picked.length < 2) picked.push('geography')
  const fallback: Subject[] = track === 'physics' ? ['chemistry', 'biology', 'geography', 'politics'] : ['politics', 'geography', 'biology', 'chemistry']
  fallback.forEach((subject) => {
    if (picked.length < 2 && !picked.includes(subject)) picked.push(subject)
  })
  return picked.slice(0, 2)
}

const majorQuestionVariants = [
  {
    authorName: '专业摇摆的高一生',
    authorRole: 'student' as const,
    grade: '高一',
    title: (major: string, combo: string) => `想冲${major}，现在选${combo}是不是太早锁死？`,
    content: (item: (typeof majorRequirements)[number], _combo: string) =>
      `我对${item.major}有兴趣，但还没到“非它不可”的程度。查到建议组合是${item.suggestedCombination}，也有人说先保优势科排名。想问：如果目标还会变，是先按${item.major}卡门槛，还是先选自己更稳的组合？`,
    tags: (item: (typeof majorRequirements)[number]) => [item.major, item.category, '学生提问', '目标摇摆', '专业论坛'],
  },
  {
    authorName: '课程表研究员',
    authorRole: 'student' as const,
    grade: '大一在读',
    title: (major: string) => `${major}大学课程到底学什么？高中要提前补哪科？`,
    content: (item: (typeof majorRequirements)[number]) =>
      `身边很多人只看${item.major}名字，不看课程表。想收集同专业同学的真实反馈：高中阶段最该补的是数学、物理、化学、生物，还是阅读表达？哪些课跟想象完全不一样？`,
    tags: (item: (typeof majorRequirements)[number]) => [item.major, item.category, '课程表', '学长答疑', '专业论坛'],
  },
  {
    authorName: '一起查目录的家长',
    authorRole: 'parent' as const,
    grade: '高一家长',
    title: (major: string) => `孩子说喜欢${major}，我们该怎么判断不是三分钟热度？`,
    content: (item: (typeof majorRequirements)[number]) =>
      `孩子最近突然对${item.major}很感兴趣，我们查到${item.noteType}和“${item.suggestedCombination}”这些信息，但不知道该如何验证兴趣。想问老师和同学：看纪录片、读专业介绍、做职业访谈，哪个更能判断是否适合？`,
    tags: (item: (typeof majorRequirements)[number]) => [item.major, item.category, '亲子沟通', '兴趣验证', '专业论坛'],
  },
  {
    authorName: '老师让我逐校查',
    authorRole: 'student' as const,
    grade: '高一',
    title: (major: string) => `${major}是不是每个学校要求都一样？我越查越乱`,
    content: (item: (typeof majorRequirements)[number]) =>
      `我以为${item.major}只要记住一个选科要求就行，结果老师说要看省份目录和院校专业组。有没有人能讲清楚：同名专业为什么会有不同要求？普通院校、强基、实验班是不是要分开查？`,
    tags: (item: (typeof majorRequirements)[number]) => [item.major, item.category, '逐校核对', '专业组', '专业论坛'],
  },
  {
    authorName: '分数线收藏夹',
    authorRole: 'student' as const,
    grade: '高二',
    title: (major: string) => `只够普通本科，选${major}还值得坚持吗？`,
    content: (item: (typeof majorRequirements)[number]) =>
      `我不一定能冲到很高层次学校，但确实对${item.major}有兴趣。想问大家：专业热度、学校层次、城市和转专业机会该怎么排序？如果只能在普通院校读，哪些现实问题要提前想清楚？`,
    tags: (item: (typeof majorRequirements)[number]) => [item.major, item.category, '学校层次', '现实选择', '专业论坛'],
  },
]

const majorForumPostSeeds: Array<Omit<Post, 'viewerLiked' | 'viewerFavorited' | 'viewerFollowing'>> = majorRequirements.flatMap((item, majorIndex) => {
  const track = inferMajorTrack(item)
  const electives = inferMajorElectives(item, track)
  const requirementText = item.requiredSubjects.join(' / ')
  const comboText = `${track === 'physics' ? '物理' : '历史'} + ${electives.map((subject) => ({
    chemistry: '化学',
    biology: '生物',
    politics: '政治',
    geography: '地理',
  })[subject]).join(' + ')}`
  const baseDate = String(28 - (majorIndex % 9)).padStart(2, '0')
  const baseLikes = 88 + (majorIndex % 18) * 7
  const baseFavorites = 64 + (majorIndex % 15) * 9
  const questionVariant = majorQuestionVariants[majorIndex % majorQuestionVariants.length]

  return [
    {
      id: 10000 + majorIndex * 3,
      authorName: '专业目录研究员',
      authorRole: 'counselor',
      title: `${item.major}选科要求速查：先看门槛，再看院校专业组`,
      content: `想报${item.major}，第一步不是看热度，而是核对本省招生目录。常见口径是${requirementText}，更稳妥的备选组合是${item.suggestedCombination}。${item.risk}`,
      imageUrls: [majorForumImages[majorIndex % majorForumImages.length]],
      tags: [item.major, item.category, item.noteType, '选科要求', '专业论坛'],
      track,
      electives,
      category: 'data',
      grade: '高一',
      province: '全国',
      likesCount: baseLikes + 120,
      commentsCount: 3,
      favoritesCount: baseFavorites + 160,
      createdAt: `2026-06-${baseDate}T09:20:00+08:00`,
      updatedAt: `2026-06-${baseDate}T09:20:00+08:00`,
    },
    {
      id: 10000 + majorIndex * 3 + 1,
      authorName: '同专业学长姐',
      authorRole: 'student',
      title: `想学${item.major}，高中阶段最该提前确认的三件事`,
      content: `${item.major}不是只看名字就能判断适不适合。我的建议是：先确认${requirementText}是否硬性要求，再看大学课程表，最后用最近三次考试判断自己能不能长期维持${comboText}的投入。`,
      imageUrls: majorIndex % 3 === 0 ? [majorForumImages[(majorIndex + 2) % majorForumImages.length]] : [],
      tags: [item.major, item.category, '学长经验', '避坑指南', '专业论坛'],
      track,
      electives,
      category: 'experience',
      grade: '大一在读',
      province: ['浙江', '江苏', '广东', '山东', '上海'][majorIndex % 5],
      likesCount: baseLikes + 76,
      commentsCount: 3,
      favoritesCount: baseFavorites + 104,
      createdAt: `2026-06-${baseDate}T15:40:00+08:00`,
      updatedAt: `2026-06-${baseDate}T15:40:00+08:00`,
    },
    {
      id: 10000 + majorIndex * 3 + 2,
      authorName: questionVariant.authorName,
      authorRole: questionVariant.authorRole,
      title: questionVariant.title(item.major, comboText),
      content: questionVariant.content(item, comboText),
      imageUrls: [],
      tags: questionVariant.tags(item),
      track,
      electives,
      category: 'question',
      grade: questionVariant.grade,
      province: ['北京', '湖北', '四川', '福建', '重庆'][majorIndex % 5],
      likesCount: baseLikes + 32,
      commentsCount: 3,
      favoritesCount: baseFavorites + 58,
      createdAt: `2026-06-${baseDate}T20:10:00+08:00`,
      updatedAt: `2026-06-${baseDate}T20:10:00+08:00`,
    },
  ]
})

const allPostSeeds = [...samplePostSeeds, ...generatedPostSeeds, ...majorForumPostSeeds]

export const samplePosts: Post[] = allPostSeeds.map((post) => ({
  ...post,
  viewerLiked: false,
  viewerFavorited: false,
  viewerFollowing: false,
}))

const featuredComments: Record<number, Comment[]> = {
  1: [
    {
      id: 1,
      postId: 1,
      author: '学海无涯',
      role: 'student',
      content: '同是物化生，完全同意！物理前期难，但后期提分空间大。',
      createdAt: '2026-06-30T09:15:00+08:00',
    },
    {
      id: 2,
      postId: 1,
      author: '星辰大海',
      role: 'student',
      content: '请问你是怎么平衡化学和生物的学习的？感觉时间总是不够用。',
      createdAt: '2026-06-30T09:45:00+08:00',
    },
    {
      id: 3,
      postId: 1,
      author: '清北学长',
      role: 'student',
      content: '我把化学知识点系统整理成框架，生物重在理解和记忆，晚上固定两科轮流复盘。',
      createdAt: '2026-06-30T10:05:00+08:00',
    },
  ],
}

export const sampleComments: Record<number, Comment[]> = Object.fromEntries(
  samplePosts.map((post) => [
    post.id,
    featuredComments[post.id] ?? [
      {
        id: post.id * 10 + 1,
        postId: post.id,
        author: post.category === 'question' ? '认证规划师林老师' : '同省同组合同学',
        role: post.category === 'question' ? 'counselor' : 'student',
        content:
          post.tags.includes('专业论坛')
            ? `如果目标是${post.tags[0]}，建议把目标院校专业组、课程表和本省选考目录放在一起看，别只看专业名。`
            : post.category === 'question'
              ? '建议先把孩子近三次考试单科排名、目标专业清单和本省选考目录放在一起看，不要只凭单次成绩决定。'
              : '这条很有参考价值，我补充一点：最好把同省份招生计划和学校层次一起核对，避免只看组合覆盖率。',
        createdAt: '2026-06-30T12:10:00+08:00',
      },
      {
        id: post.id * 10 + 2,
        postId: post.id,
        author: '正在选科的高一生',
        role: 'student',
        content: post.tags.includes('专业论坛')
          ? `收藏了，准备用“${post.tags[0]} + 选科要求”去查我们省的官方目录。`
          : '收藏了，准备按这个思路去问班主任和学科老师。',
        createdAt: '2026-06-30T12:36:00+08:00',
      },
      ...(post.tags.includes('专业论坛')
        ? [
            {
              id: post.id * 10 + 3,
              postId: post.id,
              author: '高校专业课助教',
              role: 'teacher' as const,
              content: `${post.tags[0]}后续差异主要在学校培养方案和学院资源，选科只是准入门槛，别忘了看课程体系。`,
              createdAt: '2026-06-30T13:02:00+08:00',
            },
          ]
        : []),
    ],
  ]),
)

export const sampleInsights: SubjectInsight[] = [
  {
    id: 1,
    combination: '物化生',
    trend: '专业覆盖高',
    heat: 96,
    matchRate: 91.5,
    advice: '适合数理基础稳定、能承受连续刷题和记忆任务的学生。',
    details: '物化生通常拥有较高专业覆盖度，但对持续投入和三科平衡要求很高。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 2,
    combination: '物化地',
    trend: '工科友好',
    heat: 88,
    matchRate: 84.2,
    advice: '适合物理化学较稳且喜欢图表、空间分析的学生。',
    details: '物化地适合空间分析和图表能力较强的学生，需结合本省赋分规则判断。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 3,
    combination: '物生地',
    trend: '压力均衡',
    heat: 74,
    matchRate: 78.4,
    advice: '适合想保留理工方向但化学压力较大的学生。',
    details: '物生地压力相对均衡，但需要核对目标专业是否限选化学。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 4,
    combination: '史政地',
    trend: '人文社科清晰',
    heat: 81,
    matchRate: 73.1,
    advice: '适合表达、记忆、材料分析能力强且接受专业范围收窄的学生。',
    details: '史政地路径清晰，适合人文社科方向，但政治和地理都需要持续积累。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 5,
    combination: '物化政',
    trend: '交叉选择',
    heat: 79,
    matchRate: 80.6,
    advice: '适合想保留理工方向，同时对法学、公安、公共管理有兴趣的学生。',
    details: '物化政跨度较大，物理化学保专业边界，政治提供表达和时政材料分析能力，但学习节奏会比较紧。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 6,
    combination: '物政地',
    trend: '风险核对',
    heat: 63,
    matchRate: 69.8,
    advice: '适合物理可保底但化学生物压力较大的学生。',
    details: '物政地要重点核对目标专业是否要求化学。适合经济管理、地理信息、公共管理等方向，但传统理工农医会有明显限制。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 7,
    combination: '史政生',
    trend: '师范/心理友好',
    heat: 66,
    matchRate: 67.4,
    advice: '适合人文表达强，同时对心理、教育、护理、公共卫生有兴趣的学生。',
    details: '史政生不是医学捷径，医学和药学仍要逐校核对物理化学要求。它更适合人文社科基础上保留生命科学兴趣。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 8,
    combination: '史地生',
    trend: '压力温和',
    heat: 58,
    matchRate: 62.8,
    advice: '适合兴趣较广、愿意接受专业边界的历史方向学生。',
    details: '史地生体验相对温和，但理工农医限制明显。适合教育、旅游管理、地理科学、心理和部分农林方向。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 9,
    combination: '史化生',
    trend: '小众组合',
    heat: 44,
    matchRate: 59.2,
    advice: '适合历史方向明确，同时化学和生物兴趣稳定的学生。',
    details: '史化生需要重点核对首选科目要求。它能保留少量生命科学相关兴趣，但不等同于医学方向通行证。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 10,
    combination: '史化地',
    trend: '图表/材料并重',
    heat: 41,
    matchRate: 57.9,
    advice: '适合不排斥化学体系、又喜欢地图图表和材料分析的学生。',
    details: '史化地专业边界较窄，但如果目标在资源环境、地理科学、部分师范和管理方向，可以作为个性化组合核对。',
    updatedAt: '2026-06-30T08:00:00+08:00',
  },
]

export const sampleTopics = [
  {
    id: 1,
    slug: 'physics-combo',
    title: '物理方向组合怎么选',
    summary: '集中讨论物化生、物化地、物生地、物化政等物理方向组合的专业覆盖、学习压力和风险核对。',
    viewsCount: 7600,
    postsCount: samplePosts.filter((post) => post.track === 'physics').length,
    createdAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 2,
    slug: 'history-careers',
    title: '历史方向就业前景',
    summary: '讨论史政地、史政生、史地生等历史方向组合如何连接法学、师范、新闻、公管和管理类专业。',
    viewsCount: 6200,
    postsCount: samplePosts.filter((post) => post.track === 'history').length,
    createdAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 3,
    slug: 'chemistry-importance',
    title: '化学到底有多重要',
    summary: '用专业目录和真实经验讨论物理+化学要求变多后，哪些学生值得坚持化学，哪些学生应及时避坑。',
    viewsCount: 5100,
    postsCount: samplePosts.filter((post) => post.electives.includes('chemistry')).length,
    createdAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 4,
    slug: 'grade-one-timeline',
    title: '高二选科时间线',
    summary: '从高一试探、高二分班到高三志愿填报，沉淀不同学校的选科节奏、资料清单和老师建议。',
    viewsCount: 4300,
    postsCount: samplePosts.filter((post) => post.tags.some((tag) => tag.includes('学习') || tag.includes('风险'))).length,
    createdAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 5,
    slug: 'score-improvement',
    title: '选科后如何提分',
    summary: '讨论选科后成绩波动、错题修复、时间分配和赋分策略，让组合选择回到可执行的学习计划。',
    viewsCount: 3800,
    postsCount: samplePosts.filter((post) => post.category === 'experience').length,
    createdAt: '2026-06-30T08:00:00+08:00',
  },
  {
    id: 6,
    slug: 'parents-guide',
    title: '家长如何帮孩子选科',
    summary: '学生提问、亲子沟通、目标专业核对和焦虑管理集中讨论区。',
    viewsCount: 3600,
    postsCount: samplePosts.filter((post) => post.category === 'question').length,
    createdAt: '2026-06-30T08:00:00+08:00',
  },
]

export const sampleTopicDetails = Object.fromEntries(
  sampleTopics.map((topic) => {
    const posts =
      topic.slug === 'physics-combo'
        ? samplePosts.filter((post) => post.track === 'physics').slice(0, 18)
        : topic.slug === 'history-careers'
          ? samplePosts.filter((post) => post.track === 'history').slice(0, 18)
          : topic.slug === 'chemistry-importance'
            ? samplePosts.filter((post) => post.electives.includes('chemistry')).slice(0, 18)
            : topic.slug === 'parents-guide'
              ? samplePosts.filter((post) => post.category === 'question').slice(0, 18)
              : samplePosts.filter((post) => post.category !== 'data').slice(0, 18)
    return [topic.slug, { topic, posts }]
  }),
)
