import { nationalOfficialSources, type ProvinceKnowledge } from './knowledgeBase'

export interface PolicyDocumentSection {
  heading: string
  paragraphs: string[]
  bullets: string[]
}

export interface PolicyDocument {
  id: string
  title: string
  type: string
  subtitle: string
  sourceName: string
  sourceUrl: string
  downloadUrl: string
  updatedAt: string
  tags: string[]
  abstract: string
  sections: PolicyDocumentSection[]
  checkItems: string[]
  relatedQueries: string[]
}

export function policyDocumentPath(province: string, documentId: string) {
  return `/knowledge/${encodeURIComponent(province)}/docs/${documentId}`
}

export function createProvincePolicyDocuments(province: ProvinceKnowledge): PolicyDocument[] {
  const policySource = nationalOfficialSources[0]
  const subjectSource = nationalOfficialSources[1]
  const volunteerSource = nationalOfficialSources[2]
  const modeLabel = province.reformMode === 'special' ? '特殊招生通道' : province.reformMode
  const subjectModeSummary = subjectModeAdvice(province)

  return [
    {
      id: 'admission-policy',
      title: `${province.province}普通高校招生政策核对入口`,
      type: '招生政策 / 官方文件入口',
      subtitle: `${province.authority}发布文件的网页化阅读版`,
      sourceName: province.authority,
      sourceUrl: province.portalUrl,
      downloadUrl: province.portalUrl,
      updatedAt: '动态更新',
      tags: [modeLabel, ...province.focus],
      abstract: `适合先确认${province.province}当年普通高校招生工作的基本口径，再回到选科、志愿批次、照顾政策和专项计划逐项核对。`,
      sections: [
        {
          heading: '一、文件适用范围',
          paragraphs: [
            `${province.province}当前政策状态：${province.status}`,
            `本页把官方招生政策中与高中选科、志愿填报和录取规则强相关的内容做成网页化阅读版，方便在站内搜索、收藏和对照讨论。`,
          ],
          bullets: [
            `主管入口：${province.authority}`,
            `改革模式：${modeLabel}`,
            `重点关注：${province.focus.join('、')}`,
          ],
        },
        {
          heading: '二、报名、考试与批次核对',
          paragraphs: [
            '阅读招生工作规定时，不要只看标题中的年份，要核对报名条件、考试安排、志愿设置、投档规则、录取批次和照顾政策是否同属同一年份。',
          ],
          bullets: [
            '确认普通类、艺术类、体育类、春季/分类考试是否适用同一套规则。',
            '核对本科批、专科批、提前批、特殊类型批次的志愿结构。',
            '看清“院校专业组”“专业+院校”或传统志愿模式的具体填报单位。',
          ],
        },
        {
          heading: '三、和选科直接相关的条款',
          paragraphs: [
            subjectModeSummary,
            '如果文件只给出报名和录取办法，仍需继续查看本省或阳光高考发布的专业选考科目要求目录。',
          ],
          bullets: [
            '首选科目影响可报专业组范围，尤其是理工农医方向。',
            '再选科目要求中，“化学”“生物”“政治”“地理”的限制需要逐校逐专业看。',
            '同名专业在不同院校、不同专业组中可能有不同选科要求。',
          ],
        },
        {
          heading: '四、家长和学生的使用建议',
          paragraphs: [
            '先把目标专业和目标省份确认，再用官方文件反推可选组合。社区经验可以帮助理解压力和路径，但不能替代官方目录。',
          ],
          bullets: [
            '把目标专业名称复制到官方 PDF 或网页中搜索。',
            '遇到“不限选考”也要继续核对招生计划、专业组和培养方向。',
            '收藏本页后，可在帖子评论区附上目标专业，请认证用户帮忙复核。',
          ],
        },
      ],
      checkItems: [
        '是否是本省考试院或阳光高考官方入口',
        '是否为最新年份文件',
        '是否包含志愿设置和投档规则',
        '是否需要另查专业选考目录',
      ],
      relatedQueries: [`${province.authority} 普通高校招生 工作规定`, `${province.province} 招生政策 照顾政策`, `${province.province} 高考 志愿填报 办法`],
    },
    {
      id: 'subject-requirements-pdf',
      title: `${province.province}本科专业选考科目要求`,
      type: '选考目录 / PDF附件检索',
      subtitle: '把专业选科 PDF 的核对逻辑拆成网页内容',
      sourceName: subjectSource.publisher,
      sourceUrl: subjectSource.url,
      downloadUrl: searchUrl(`${province.authority} 本科专业 选考科目要求 PDF 附件`),
      updatedAt: '按官方附件核验',
      tags: ['专业目录', 'PDF', ...province.focus],
      abstract: `用于判断目标院校专业组是否要求物理、化学或其他再选科目，是选科决策里最需要逐条查的文件。`,
      sections: [
        {
          heading: '一、PDF通常包含什么',
          paragraphs: [
            '专业选考科目要求附件通常按院校、专业或专业组列出首选科目、再选科目、选考限制和备注，是“我这个组合能不能报”的直接依据。',
          ],
          bullets: [
            '院校名称、专业名称、专业代码或专业组名称。',
            '首选科目要求：物理、历史或不限。',
            '再选科目要求：化学、生物、政治、地理或不限。',
            '备注栏可能说明“均须选考”“选考其中一门”等关键口径。',
          ],
        },
        {
          heading: '二、站内核对步骤',
          paragraphs: [
            `针对${province.province}，建议先确认${province.reformMode}模式，再把目标专业逐个放进 PDF 搜索。`,
          ],
          bullets: province.checklist,
        },
        {
          heading: '三、常见误读',
          paragraphs: [
            '“不限”不等于任何学校都能报，“物理+化学”也不代表一定适合每个学生。选科要求只解决资格门槛，不能单独决定组合。',
          ],
          bullets: [
            '不要用一个热门专业覆盖率数字替代逐校核对。',
            '不要把外省目录直接套到本省。',
            '不要忽略当年招生计划和高校招生章程。',
          ],
        },
      ],
      checkItems: [
        '用专业全称和专业简称各搜索一次',
        '记录首选科目要求',
        '记录再选科目是否“均须选考”',
        '把无法确认的条目发到论坛追问',
      ],
      relatedQueries: [`${province.province} 本科专业选考科目要求`, `${province.authority} 选考科目 附件`, `${province.province} 物理 化学 专业要求`],
    },
    {
      id: 'care-policy',
      title: `${province.province}招生政策及照顾政策`,
      type: '政策汇总 / 官方汇总',
      subtitle: '把加分、专项、录取批次合并成可读清单',
      sourceName: policySource.publisher,
      sourceUrl: policySource.url,
      downloadUrl: policySource.url,
      updatedAt: '阳光高考汇总',
      tags: ['照顾政策', '专项计划', '志愿批次'],
      abstract: '适合核对照顾政策、专项计划、报名资格、批次设置和录取办法，尤其适合家长在填报前做最终排查。',
      sections: [
        {
          heading: '一、优先看哪些内容',
          paragraphs: [
            '招生政策及照顾政策往往分散在工作规定、照顾政策说明、专项计划公告和报名通知中，建议按“资格、批次、投档、录取”四类整理。',
          ],
          bullets: [
            '资格：户籍、学籍、报名地、专项计划或照顾项目条件。',
            '批次：提前批、本科批、特殊类型招生、专科批等志愿设置。',
            '投档：平行志愿投档、专业组投档、同分排序规则。',
            '录取：退档、调剂、专业录取规则和高校章程衔接。',
          ],
        },
        {
          heading: '二、和选科的关系',
          paragraphs: [
            '照顾政策通常不直接改变选科要求，但会影响志愿策略、目标院校梯度和风险承受度。',
          ],
          bullets: [
            '专项计划类考生仍要满足专业组或专业选科要求。',
            '特殊类型招生也要看高校章程与本省招生规定。',
            '批次变化可能影响目标专业的填报顺序和兜底策略。',
          ],
        },
      ],
      checkItems: ['确认是否符合照顾政策条件', '确认专项计划报名窗口', '确认批次和投档单位', '与高校招生章程交叉核对'],
      relatedQueries: [`${province.province} 照顾政策`, `${province.province} 专项计划`, `${province.province} 高考 同分排序`],
    },
    {
      id: 'enrollment-rules-pdf',
      title: `${province.province}招生工作规定PDF`,
      type: '招生规定 / PDF附件检索',
      subtitle: '适合完整核对报名、考试、志愿和投档录取',
      sourceName: province.authority,
      sourceUrl: province.portalUrl,
      downloadUrl: searchUrl(`${province.authority} 普通高校招生工作规定 PDF ${new Date().getFullYear()}`),
      updatedAt: '按当年文件核验',
      tags: ['招生规定', 'PDF', '志愿填报'],
      abstract: '招生工作规定是政策库里最像“总纲”的文件，适合在选科之外核对完整高考流程。',
      sections: [
        {
          heading: '一、完整文件阅读顺序',
          paragraphs: [
            '建议不要从头硬读，先找目录，把和自己有关的章节标记出来，再逐条做核对。',
          ],
          bullets: [
            '报名条件与资格审查。',
            '考试安排、成绩发布、赋分或计分规则。',
            '志愿填报时间、批次、投档单位和录取规则。',
            '照顾政策、特殊类型招生和违规处理。',
          ],
        },
        {
          heading: '二、容易遗漏的条款',
          paragraphs: [
            '很多用户只查选科要求，却忽略同分排序、退档规则、专业调剂和高校章程，这些都会影响最终录取风险。',
          ],
          bullets: [
            '同分排序是否涉及单科成绩。',
            '专业调剂范围是在院校内、专业组内还是其他口径。',
            '高校章程是否另有身体条件、语种或单科成绩要求。',
          ],
        },
      ],
      checkItems: ['下载官方 PDF 后保存年份', '标注志愿填报时间线', '核对投档规则', '把重点条款转成家庭讨论清单'],
      relatedQueries: [`${province.province} 招生工作规定 PDF`, `${province.province} 普通高校招生办法`, `${province.province} 志愿填报 时间`],
    },
    {
      id: 'volunteer-major-extension',
      title: `${province.province}高校与专业信息延展`,
      type: '志愿资料 / 专业库入口',
      subtitle: '从选科要求继续延展到高校、专业和培养方向',
      sourceName: volunteerSource.publisher,
      sourceUrl: volunteerSource.url,
      downloadUrl: volunteerSource.url,
      updatedAt: '长期入口',
      tags: ['专业库', '高校库', '志愿参考'],
      abstract: '当选科组合初步确定后，需要继续看专业培养方向、院校层次、招生计划和就业去向，避免只凭组合热度做决定。',
      sections: [
        {
          heading: '一、从选科到志愿的衔接',
          paragraphs: [
            '选科要求回答“能不能报”，专业库和高校库回答“值不值得报、适不适合报”。',
          ],
          bullets: [
            '先看专业培养方向和核心课程。',
            '再看本省招生计划与历年录取情况。',
            '最后结合孩子兴趣、成绩稳定性和家庭资源做取舍。',
          ],
        },
        {
          heading: '二、建议收藏的信息',
          paragraphs: [
            '把每个目标专业的选科要求、培养课程、就业去向和对应院校层次整理到同一张表里，会比刷零散帖子更稳。',
          ],
          bullets: [
            '目标专业：必选科目、可选组合、风险点。',
            '目标院校：专业组、计划数、校区、学费和身体条件。',
            '个人画像：强项学科、压力承受、长期兴趣和地域偏好。',
          ],
        },
      ],
      checkItems: ['用专业库查培养方向', '用政策库查选科门槛', '用社区帖子查真实学习体验', '用个人画像生成建议'],
      relatedQueries: [`${province.province} 高校 专业 信息`, `${province.province} 志愿填报 专业库`, `${province.province} 招生计划 专业组`],
    },
  ]
}

export function findProvincePolicyDocument(province: ProvinceKnowledge | undefined, documentId: string): PolicyDocument | undefined {
  if (!province) return undefined
  return createProvincePolicyDocuments(province).find((document) => document.id === documentId)
}

function subjectModeAdvice(province: ProvinceKnowledge) {
  if (province.reformMode === '3+1+2') {
    return '“3+1+2”省份需要先看首选科目，再看再选科目。理工农医、计算机、电子信息、医学等方向尤其要注意物理和化学是否同时要求。'
  }
  if (province.reformMode === '3+3') {
    return '“3+3”省份通常按专业提出若干选考科目要求，重点是看目标专业是否指定物理、化学或技术等科目。'
  }
  if (province.reformMode === 'special') {
    return '港澳台及特殊招生通道不应直接套用大陆省份新高考专业组选科规则，应优先核对联招简章和高校单独招生办法。'
  }
  return '传统高考或政策过渡地区应先核对本省最新招生工作规定，不要直接套用新高考专业组和选科目录口径。'
}

function searchUrl(query: string) {
  return `https://www.baidu.com/s?wd=${encodeURIComponent(query)}`
}
