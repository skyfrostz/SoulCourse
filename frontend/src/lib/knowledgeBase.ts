export type ReformMode = '3+3' | '3+1+2' | 'traditional' | 'special'

export interface ProvinceKnowledge {
  province: string
  region: '华北' | '东北' | '华东' | '华中' | '华南' | '西南' | '西北' | '港澳台'
  authority: string
  portalUrl: string
  reformMode: ReformMode
  status: string
  focus: string[]
  checklist: string[]
}

export interface KnowledgeSource {
  title: string
  publisher: string
  type: 'official' | 'media' | 'platform'
  url: string
  summary: string
  tags: string[]
}

export const nationalOfficialSources: KnowledgeSource[] = [
  {
    title: '2026年高考各省市招生政策及照顾政策汇总',
    publisher: '阳光高考/学信网',
    type: 'official',
    url: 'https://gaokao.chsi.com.cn/z/gkbmfslq/zszc.jsp',
    summary: '集中查看各省市招生工作规定、实施办法和照顾政策，适合作为省份政策入口。',
    tags: ['招生政策', '照顾政策', '官方汇总'],
  },
  {
    title: '2024年各省市普通高校本科专业选考科目要求汇总',
    publisher: '阳光高考/学信网',
    type: 'official',
    url: 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202201/20220105/2155365943.html',
    summary: '汇总部分省市专业选考科目要求，适合核对“专业是否要求物理/化学”。',
    tags: ['选考要求', '专业目录', '官方汇总'],
  },
  {
    title: '阳光志愿信息服务系统',
    publisher: '阳光高考/学信网',
    type: 'official',
    url: 'https://gaokao.chsi.com.cn/zyck/',
    summary: '整合高校、专业、招生政策和志愿参考信息，适合从选科延展到志愿填报。',
    tags: ['志愿填报', '高校信息', '专业信息'],
  },
  {
    title: '全国省级招考机构入口',
    publisher: '山东省教育招生考试院页面汇总',
    type: 'official',
    url: 'https://www.sdzk.cn/',
    summary: '页面底部列出全国省级招考机构名称，可作为省级考试院入口校验线索。',
    tags: ['考试院入口', '省级招办'],
  },
]

export const provinceKnowledge: ProvinceKnowledge[] = [
  { province: '北京', region: '华北', authority: '北京教育考试院', portalUrl: 'https://www.bjeea.cn/', reformMode: '3+3', status: '已实施新高考，关注等级考与院校专业组选考要求。', focus: ['本科普通批', '等级性考试', '强基/综评'], checklist: ['核对北京教育考试院招生规定', '按院校专业组检查选考科目', '关注本科批志愿设置'] },
  { province: '天津', region: '华北', authority: '天津市教育招生考试院', portalUrl: 'https://www.zhaokao.net/', reformMode: '3+3', status: '已实施新高考，选科需结合专业组选考要求。', focus: ['等级考', '普通本科批', '志愿填报'], checklist: ['查看天津招考资讯网最新文件', '核对专业组选考科目', '关注等级赋分规则'] },
  { province: '河北', region: '华北', authority: '河北省教育考试院', portalUrl: 'https://www.hebeea.edu.cn/', reformMode: '3+1+2', status: '已实施“3+1+2”，首选物理/历史影响专业组范围。', focus: ['首选科目', '再选科目', '志愿批次'], checklist: ['先确认首选科目要求', '再看化学等再选限制', '关注同分排序规则'] },
  { province: '山西', region: '华北', authority: '山西省招生考试管理中心', portalUrl: 'https://www.sxkszx.cn/', reformMode: '3+1+2', status: '第五批改革省份，2025年起新高考落地。', focus: ['适应性演练', '专业组选科要求', '志愿填报变化'], checklist: ['查看新高考实施办法', '参加/参考适应性演练说明', '重点核对物化要求'] },
  { province: '内蒙古', region: '华北', authority: '内蒙古自治区教育招生考试中心', portalUrl: 'https://www.nm.zsks.cn/', reformMode: '3+1+2', status: '第五批改革省份，2025年起新高考落地。', focus: ['民族地区政策', '首选科目', '志愿办法'], checklist: ['核对自治区招生考试中心公告', '关注照顾政策与专项计划', '按专业组查选考要求'] },
  { province: '辽宁', region: '东北', authority: '辽宁省招生考试办公室', portalUrl: 'https://www.lnzsks.com/', reformMode: '3+1+2', status: '已实施“3+1+2”。', focus: ['普通类本科批', '再选科目', '专业组'], checklist: ['确认专业组首选科目', '关注再选科目限制', '核对当年招生计划'] },
  { province: '吉林', region: '东北', authority: '吉林省教育考试院', portalUrl: 'https://www.jleea.com.cn/', reformMode: '3+1+2', status: '第四批改革省份，2024年新高考首考。', focus: ['首届新高考经验', '专业组', '录取规则'], checklist: ['回看首届录取数据', '核对专业选考要求', '关注志愿填报规则变化'] },
  { province: '黑龙江', region: '东北', authority: '黑龙江省招生考试院', portalUrl: 'https://www.lzk.hl.cn/', reformMode: '3+1+2', status: '第四批改革省份，2024年新高考首考。', focus: ['专业组', '首选科目', '院校计划'], checklist: ['核对省招考院文件', '关注物化双选专业', '参考首届录取结果'] },
  { province: '上海', region: '华东', authority: '上海市教育考试院', portalUrl: 'https://www.shmeea.edu.cn/', reformMode: '3+3', status: '已实施“3+3”，院校专业组和选科要求需一起看。', focus: ['院校专业组', '等级考', '物化双选'], checklist: ['查看上海考试院选考目录', '按院校专业组核对', '关注物化双选变化'] },
  { province: '江苏', region: '华东', authority: '江苏省教育考试院', portalUrl: 'https://www.jseea.cn/', reformMode: '3+1+2', status: '已实施“3+1+2”。', focus: ['首选科目', '再选科目', '专业组选考要求'], checklist: ['明确两科/三科要求为均须选考', '核对当年招生计划', '关注院校专业组'] },
  { province: '浙江', region: '华东', authority: '浙江省教育考试院', portalUrl: 'https://www.zjzs.net/', reformMode: '3+3', status: '已实施“3+3”，技术科目与选考目录需特别关注。', focus: ['7选3', '技术科目', '专业覆盖'], checklist: ['查看浙江选考科目要求', '核对专业覆盖率只是参考', '注意当年实际招生计划'] },
  { province: '安徽', region: '华东', authority: '安徽省教育招生考试院', portalUrl: 'https://www.ahzsks.cn/', reformMode: '3+1+2', status: '第四批改革省份，2024年新高考首考。', focus: ['首届数据', '专业组', '物化要求'], checklist: ['查看考试院招生办法', '关注首届投档线变化', '核对专业组选考科目'] },
  { province: '福建', region: '华东', authority: '福建省教育考试院', portalUrl: 'https://www.eeafj.cn/', reformMode: '3+1+2', status: '已实施“3+1+2”。', focus: ['本科批', '院校专业组', '再选科目'], checklist: ['查看福建考试院公告', '核对首选/再选科目', '关注地方专项/综合评价'] },
  { province: '江西', region: '华东', authority: '江西省教育考试院', portalUrl: 'https://www.jxeea.cn/', reformMode: '3+1+2', status: '第四批改革省份，2024年新高考首考。', focus: ['首届新高考', '专业组', '志愿规则'], checklist: ['查看江西考试院招生政策', '参考首届录取数据', '核对物理化学限制'] },
  { province: '山东', region: '华东', authority: '山东省教育招生考试院', portalUrl: 'https://www.sdzk.cn/', reformMode: '3+3', status: '已实施“3+3”，选考科目要求按专业核对。', focus: ['96个志愿', '等级考', '选科目录'], checklist: ['查看山东考试院选考要求公告', '关注本科/专科附件', '同专业不同高校可能不同'] },
  { province: '河南', region: '华中', authority: '河南省教育考试院', portalUrl: 'https://www.haeea.cn/', reformMode: '3+1+2', status: '第五批改革省份，2025年起新高考落地。', focus: ['考生体量', '专业组', '新高考规则'], checklist: ['查看河南考试院选科要求公告', '核对四类选科要求', '关注志愿填报新规则'] },
  { province: '湖北', region: '华中', authority: '湖北省教育考试院', portalUrl: 'https://www.hbea.edu.cn/', reformMode: '3+1+2', status: '已实施“3+1+2”。', focus: ['院校专业组', '再选要求', '投档规则'], checklist: ['查看湖北考试院文件', '核对首选物理/历史', '关注院校专业组差异'] },
  { province: '湖南', region: '华中', authority: '湖南省教育考试院', portalUrl: 'https://jyt.hunan.gov.cn/jyt/sjyt/hnsjyksy/', reformMode: '3+1+2', status: '已实施“3+1+2”。', focus: ['专业组', '招生计划', '志愿填报'], checklist: ['查看湖南考试院招考资讯', '核对专业组选考要求', '关注计划变更说明'] },
  { province: '广东', region: '华南', authority: '广东省教育考试院', portalUrl: 'https://eea.gd.gov.cn/', reformMode: '3+1+2', status: '已实施“3+1+2”。', focus: ['院校专业组', '春季高考', '物化组合'], checklist: ['查看广东考试院通知', '区分春季/夏季高考政策', '核对目标专业组选考'] },
  { province: '广西', region: '华南', authority: '广西壮族自治区招生考试院', portalUrl: 'https://www.gxeea.cn/', reformMode: '3+1+2', status: '第四批改革省份，2024年新高考首考。', focus: ['首届新高考', '志愿规则', '专业组'], checklist: ['查看广西招生考试院文件', '参考首届录取数据', '核对专业组选科要求'] },
  { province: '海南', region: '华南', authority: '海南省考试局', portalUrl: 'https://ea.hainan.gov.cn/', reformMode: '3+3', status: '已实施新高考。', focus: ['标准分', '等级赋分', '选考科目'], checklist: ['查看海南考试局公告', '关注转换分规则', '核对专业选考要求'] },
  { province: '重庆', region: '西南', authority: '重庆市教育考试院', portalUrl: 'https://www.cqksy.cn/', reformMode: '3+1+2', status: '已实施“3+1+2”。', focus: ['专业组', '再选科目', '志愿批次'], checklist: ['查看重庆考试院政策', '核对首选科目要求', '关注录取批次安排'] },
  { province: '四川', region: '西南', authority: '四川省教育考试院', portalUrl: 'https://www.sceea.cn/', reformMode: '3+1+2', status: '第五批改革省份，2025年起新高考落地。', focus: ['适应性演练', '专业组', '志愿变化'], checklist: ['查看四川考试院选考要求说明', '关注适应性演练结果', '核对物化要求'] },
  { province: '贵州', region: '西南', authority: '贵州省招生考试院', portalUrl: 'https://zsksy.guizhou.gov.cn/', reformMode: '3+1+2', status: '第四批改革省份，2024年新高考首考。', focus: ['首届新高考', '志愿填报', '专业组'], checklist: ['查看贵州招生考试院公告', '参考首届录取数据', '核对专业组选科要求'] },
  { province: '云南', region: '西南', authority: '云南省招生考试院', portalUrl: 'https://www.ynzs.cn/', reformMode: '3+1+2', status: '第五批改革省份，2025年起新高考落地。', focus: ['适应性测试', '专业组', '政策过渡'], checklist: ['查看云南招考频道政策', '关注适应性演练', '核对新高考专业组'] },
  { province: '西藏', region: '西南', authority: '西藏自治区教育考试院', portalUrl: 'https://zsks.edu.xizang.gov.cn/', reformMode: 'traditional', status: '截至公开媒体报道的新高考覆盖统计中，暂未纳入已落地省份，需以自治区最新公告为准。', focus: ['民族地区政策', '专项计划', '照顾政策'], checklist: ['优先查看自治区教育考试院公告', '核对照顾政策和专项计划', '不要套用其他省份选科规则'] },
  { province: '陕西', region: '西北', authority: '陕西省教育考试院', portalUrl: 'https://www.sneac.com/', reformMode: '3+1+2', status: '第五批改革省份，2025年起新高考落地。', focus: ['专业组选科要求', '首选科目', '志愿变化'], checklist: ['查看陕西考试院选考科目公告', '核对四类要求', '关注志愿填报新办法'] },
  { province: '甘肃', region: '西北', authority: '甘肃省教育考试院', portalUrl: 'https://www.ganseea.cn/', reformMode: '3+1+2', status: '第四批改革省份，2024年新高考首考。', focus: ['选考要求统计', '物化比例', '专业组'], checklist: ['查看甘肃考试院/阳光高考公告', '重点核对物理+化学', '参考首届录取数据'] },
  { province: '青海', region: '西北', authority: '青海省教育招生考试院', portalUrl: 'https://www.qhjyks.com/', reformMode: '3+1+2', status: '第五批改革省份，2025年起新高考落地。', focus: ['专业组选科要求', '政策过渡', '照顾政策'], checklist: ['查看青海官方选考科目要求', '核对首选/再选科目', '关注民族地区政策'] },
  { province: '宁夏', region: '西北', authority: '宁夏教育考试院', portalUrl: 'https://www.nxjyks.cn/', reformMode: '3+1+2', status: '第五批改革省份，2025年起新高考落地。', focus: ['专业组', '适应性演练', '照顾政策'], checklist: ['查看宁夏教育考试院公告', '核对专业组选考要求', '关注政策过渡安排'] },
  { province: '新疆', region: '西北', authority: '新疆维吾尔自治区教育考试院', portalUrl: 'https://www.xjzk.gov.cn/', reformMode: 'traditional', status: '截至公开媒体报道的新高考覆盖统计中，暂未纳入已落地省份，需以自治区最新公告为准。', focus: ['普通高考政策', '民族地区政策', '专项计划'], checklist: ['优先查看新疆教育考试院公告', '核对照顾政策和专项计划', '不要套用新高考专业组规则'] },
  { province: '香港', region: '港澳台', authority: '香港考试及评核局 / 港澳台招生信息网', portalUrl: 'https://www.hkeaa.edu.hk/tc/ipe/jee/', reformMode: 'special', status: '香港考生主要通过港澳台侨联招、香港中学文凭考试相关渠道和高校面向港澳台招生政策核对。', focus: ['港澳台联招', '香港考区', '内地高校招生'], checklist: ['查看香港考评局联招考试页面', '核对港澳台招生信息网政策', '关注高校单独招生办法'] },
  { province: '澳门', region: '港澳台', authority: '澳门教育及青年发展局 / 港澳台招生信息网', portalUrl: 'https://www.gov.mo/zh-hant/services/ps-1717/ps-1717a/', reformMode: 'special', status: '澳门考生主要通过港澳台侨联招澳门考区及高校面向港澳台招生渠道核对。', focus: ['港澳台联招', '澳门考区', '报名确认'], checklist: ['查看澳门政府服务联招说明', '核对全国联招报名时间', '关注艺术体育类术科要求'] },
  { province: '台湾', region: '港澳台', authority: '港澳台招生信息网 / 普通高校联合招生办公室', portalUrl: 'https://www.gatzs.com.cn/', reformMode: 'special', status: '台湾地区考生适用港澳台招生信息网、全国联招及高校面向台湾学生招生政策，不能套用大陆省份新高考选科规则。', focus: ['港澳台联招', '台湾学生招生', '高校简章'], checklist: ['查看港澳台招生信息网', '核对当年联招简章', '逐校查看台湾学生招生办法'] },
]

export const mediaInsightSources: KnowledgeSource[] = [
  {
    title: '新高考选科走向“3+1+2”反映了什么',
    publisher: '央视网/澎湃新闻',
    type: 'media',
    url: 'https://news.cctv.com/2024/06/06/ARTIKD3oEhncklyGOWwJE3wG240606.shtml',
    summary: '梳理高考综合改革十年变化，指出多批改革省份采用“3+1+2”模式，并关注改革落地中的考试、招录变化。',
    tags: ['改革十年', '3+1+2', '政策观察'],
  },
  {
    title: '新高考模式全面铺开已覆盖29个省份',
    publisher: '中国日报网',
    type: 'media',
    url: 'https://cn.chinadaily.com.cn/a/202506/07/WS6843f67fa310205377036f91.html',
    summary: '报道第五批改革省份开考后，新高考落地范围扩大，并说明“3+1+2”模式和志愿填报演练变化。',
    tags: ['29省', '第五批改革', '志愿演练'],
  },
  {
    title: '新高考改革是一次影响广泛的革新',
    publisher: '人民网教育',
    type: 'media',
    url: 'https://edu.people.com.cn/n1/2024/0625/c1006-40263454.html',
    summary: '从改革实践角度评价“3+1+2”模式，强调合理选科和改革完善。',
    tags: ['改革评价', '合理选科'],
  },
  {
    title: '新高考模式下，选科何以失衡',
    publisher: '界面新闻',
    type: 'media',
    url: 'https://www.jiemian.com/article/14283634.html',
    summary: '关注专业要求对选科的影响，讨论物理、化学要求提高后学生功利化选科与压力问题。',
    tags: ['选科失衡', '物理化学', '学生压力'],
  },
  {
    title: '新高考选科指南',
    publisher: '中国教育在线',
    type: 'platform',
    url: 'https://gaokao.eol.cn/html/g/xuanke/index.shtml',
    summary: '面向学生解释新高考选科、等级赋分和组合覆盖，适合作为通俗解释材料，政策核对仍以考试院为准。',
    tags: ['选科科普', '等级赋分', '组合覆盖'],
  },
  {
    title: '选科要求变化与物理+化学组合讨论',
    publisher: '自主选拔在线',
    type: 'platform',
    url: 'https://www.zizzs.com/gk/gaokao/192156.html',
    summary: '跟踪热门专业选科要求变化，常见观点是理工农医方向更需要关注物理+化学。',
    tags: ['物化组合', '热门专业', '趋势观察'],
  },
]

export const knowledgeTakeaways = [
  '官方政策与招生计划应以省级考试院、阳光高考和高校当年招生章程为准。',
  '媒体和社区内容适合发现焦虑点、经验差异和讨论主题，不应直接替代官方目录。',
  '新高考省份要优先核对院校专业组、首选科目、再选科目和当年实际招生计划。',
  '暂未落地新高考或政策过渡省份，应先看本省最新普通高校招生工作规定，不套用外省规则。',
]
