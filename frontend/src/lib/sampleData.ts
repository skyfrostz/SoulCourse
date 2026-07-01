import type { Comment, Post, SubjectInsight } from '../types/forum'

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
    tags: ['家长提问', '师范', '风险判断'],
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

const categoryBlueprints: Array<{
  category: Post['category']
  authorName: string
  authorRole: Post['authorRole']
  title: (combo: string) => string
  content: (seed: (typeof comboSeeds)[number]) => string
  tags: (combo: string) => string[]
  image: string
}> = [
  {
    category: 'experience',
    authorName: '同组合过来人',
    authorRole: 'student',
    title: (combo) => `${combo} 一个月真实体验：我会重点看这三件事`,
    content: (seed) =>
      `${seed.label} 适合关注 ${seed.target} 的同学。我的感受是：${seed.rhythm}。不要只问“这个组合热门吗”，更要看最近三次大考单科排名、错题修复速度和每天可投入时间。`,
    tags: (combo) => [combo, '真实体验', '学习节奏', '小红书笔记'],
    image: 'https://images.unsplash.com/photo-1513258496099-48168024aec0?auto=format&fit=crop&w=1200&q=80',
  },
  {
    category: 'question',
    authorName: '认真做功课的家长',
    authorRole: 'parent',
    title: (combo) => `家长求助：孩子想选 ${combo}，怎么判断不是一时冲动？`,
    content: (seed) =>
      `孩子目前倾向 ${seed.label}，省份是${seed.province}。我们最担心的是后续专业边界和学习压力。已整理目标方向：${seed.target}。想请老师和同组合同学帮忙看看，哪些证据最值得参考？`,
    tags: (combo) => [combo, '家长提问', '目标专业', '风险核对'],
    image: 'https://images.unsplash.com/photo-1523240795612-9a054b0db644?auto=format&fit=crop&w=1200&q=80',
  },
  {
    category: 'data',
    authorName: '目录核对员',
    authorRole: 'counselor',
    title: (combo) => `${combo} 专业边界速查：先看目录，再看热度`,
    content: (seed) =>
      `${seed.label} 的判断不能只靠社区热度。建议先用本省考试院目录或阳光高考核对 ${seed.target}，再看校内成绩稳定性。特别提醒：同名专业在不同高校专业组里，选考要求可能不同。`,
    tags: (combo) => [combo, '数据建议', '省份目录', '专业覆盖'],
    image: 'https://images.unsplash.com/photo-1460925895917-afdab827c52f?auto=format&fit=crop&w=1200&q=80',
  },
]

const generatedPostSeeds: Array<Omit<Post, 'viewerLiked' | 'viewerFavorited' | 'viewerFollowing'>> = comboSeeds.flatMap((seed, comboIndex) =>
  categoryBlueprints.map((blueprint, categoryIndex) => {
    const id = 100 + comboIndex * categoryBlueprints.length + categoryIndex
    return {
      id,
      authorName: blueprint.authorName,
      authorRole: blueprint.authorRole,
      title: blueprint.title(seed.label),
      content: blueprint.content(seed),
      imageUrls: categoryIndex === 1 ? [] : [blueprint.image],
      tags: blueprint.tags(seed.label),
      track: seed.track,
      electives: seed.electives,
      category: blueprint.category,
      grade: categoryIndex === 1 ? '高一家长' : categoryIndex === 2 ? '高一' : '高二',
      province: seed.province,
      likesCount: 260 + comboIndex * 47 + categoryIndex * 93,
      commentsCount: 38 + comboIndex * 9 + categoryIndex * 17,
      favoritesCount: 104 + comboIndex * 31 + categoryIndex * 58,
      createdAt: `2026-06-${String(25 - (comboIndex % 8)).padStart(2, '0')}T${String(10 + categoryIndex * 3).padStart(2, '0')}:20:00+08:00`,
      updatedAt: `2026-06-${String(25 - (comboIndex % 8)).padStart(2, '0')}T${String(10 + categoryIndex * 3).padStart(2, '0')}:20:00+08:00`,
    }
  }),
)

const allPostSeeds = [...samplePostSeeds, ...generatedPostSeeds]

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
          post.category === 'question'
            ? '建议先把孩子近三次考试单科排名、目标专业清单和本省选考目录放在一起看，不要只凭单次成绩决定。'
            : '这条很有参考价值，我补充一点：最好把同省份招生计划和学校层次一起核对，避免只看组合覆盖率。',
        createdAt: '2026-06-30T12:10:00+08:00',
      },
      {
        id: post.id * 10 + 2,
        postId: post.id,
        author: '正在选科的高一生',
        role: 'student',
        content: '收藏了，准备按这个思路去问班主任和学科老师。',
        createdAt: '2026-06-30T12:36:00+08:00',
      },
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
    summary: '家长提问、亲子沟通、目标专业核对和焦虑管理集中讨论区。',
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
