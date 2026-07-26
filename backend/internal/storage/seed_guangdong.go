package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const guangdongPhaseOneMigration = "20260726-guangdong-content-v1"

type seededUser struct {
	email    string
	nickname string
	role     string
	grade    string
}

type seededPost struct {
	email           string
	title           string
	content         string
	tags            []string
	track           string
	electives       []string
	category        string
	grade           string
	sourceNoteID    string
	sourceTitle     string
	sourceAuthor    string
	sourceLikes     int
	sourceComments  int
	sourceFavorites int
}

var guangdongUsers = []seededUser{
	{"yuexuan01@soulcourse.cn", "粤选小队", "student", "高一"},
	{"shenzhen-note@soulcourse.cn", "深圳高一日记", "student", "高一"},
	{"gz-physics@soulcourse.cn", "广州物化生", "student", "高二"},
	{"foshan-study@soulcourse.cn", "佛山学习搭子", "student", "高三"},
	{"dongguan-parent@soulcourse.cn", "东莞家长笔记", "parent", "高一家长"},
	{"huizhou-growth@soulcourse.cn", "惠州成长记录", "student", "高二"},
	{"zhuhai-review@soulcourse.cn", "珠海复盘簿", "student", "高三"},
	{"lingnan-counselor@soulcourse.cn", "岭南生涯老师", "counselor", "升学规划"},
	{"gd-data@soulcourse.cn", "粤考数据站", "teacher", "教研"},
	{"gaosan-review@soulcourse.cn", "高三复盘君", "student", "毕业生"},
	{"chaoshan-parent@soulcourse.cn", "潮汕陪读妈妈", "parent", "高二家长"},
	{"bayarea-observer@soulcourse.cn", "湾区升学观察", "teacher", "教研"},
}

var guangdongPostImageCounts = map[int]int{
	3: 3, 7: 2, 10: 3, 11: 3, 12: 3, 15: 3, 16: 3, 17: 3,
	18: 3, 20: 2, 21: 3, 22: 3, 28: 3, 29: 3,
}

var guangdongPosts = []seededPost{
	{"yuexuan01@soulcourse.cn", "物化绑定后，广东高一应该怎么选？", "先把目标专业分成三栏：明确想读、可能会读、确定不读。理工农医方向要逐校核对物理和化学是否同时必选，不能只看一张全国覆盖率表。\n\n再看校内长期排名，而不是一次月考分数。最后比较第三门科目的学习成本和赋分环境。没有明确目标时，物化通常保留的路径更多，但前提是两科都能稳定跟上。", []string{"广东选科", "物化绑定", "专业限制"}, "physics", []string{"chemistry", "geography"}, "experience", "高一", "66a8084b0000000005020a81", "纯主观！给广东新高一的选科建议", "情绪病Zzz", 1480, 1068, 612},
	{"lingnan-counselor@soulcourse.cn", "别只问哪个组合最热门，先检查这四项", "选科时最容易被“热门组合”带着走。更有效的顺序是：专业硬性要求、校内可开班组合、本人连续三次大考排名、每周可投入时间。\n\n赋分竞争只能作为辅助信息，因为同一科目在不同学校的教学资源和学生层次差异很大。先排除明确不适合的组合，再在剩下的方案里比较。", []string{"选科避坑", "决策清单", "赋分"}, "physics", []string{"chemistry", "biology"}, "experience", "高一", "6a579cca000000000702ef83", "高中选科里那些老师不说的黑暗内幕…", "木木学长研究生上岸版", 30000, 826, 17000},
	{"shenzhen-note@soulcourse.cn", "深圳一模到高考：语文复盘比刷量更重要", "语文客观题不靠玄学。每次练习后标出设误点，把“做对但思路和解析不同”的题也放进错题本，每一到两周回看一次。\n\n主观题先审清任务，再整理不同题型的答题路径。作文训练可以拆成十分钟审题、列分论点和素材，不必每次都写完整篇。基础分稳定后，再去追求表达亮点。", []string{"广东高考", "语文", "错题复盘"}, "history", []string{"politics", "geography"}, "experience", "高三", "687ef687000000000b01e696", "广东高考语文学习经验分享", "阿布拉科萨斯", 5582, 188, 3841},
	{"foshan-study@soulcourse.cn", "广东英语听说考试，练习要盯住输出链路", "听说训练不要只反复播放材料。一次完整练习应包含听取关键词、组织句子、录音回听和定位失分点。把发音、停顿、信息遗漏分别记录，下一轮只修一个问题。\n\n考前用和正式考试接近的耳机与节奏完成整套模拟，减少设备和流程带来的紧张。", []string{"英语听说", "广东高考", "训练方法"}, "history", []string{"politics", "geography"}, "experience", "高三", "67b05cdf000000002802a7fd", "听说高考满分｜神秘嘉宾经验分享", "无印", 19000, 256, 11000},
	{"yuexuan01@soulcourse.cn", "准高一选科，最常见的三条误区", "误区一是把“覆盖率高”理解成“适合所有人”；误区二是只看单科绝对分，不看校内排名稳定性；误区三是听说某科好赋分就立刻跟进。\n\n建议先下载当届拟在粤招生专业选考要求，用十个自己可能报考的专业做一次真实核对。做完这一步，很多纠结会变得具体。", []string{"准高一", "选科误区", "广东"}, "physics", []string{"chemistry", "politics"}, "experience", "高一", "6a4e20aa00000000210192a0", "热门选科怎么选？准高一选科真的别信谣言…", "木木学长研究生上岸版", 14000, 398, 8385},
	{"gaosan-review@soulcourse.cn", "高分不是方法，能重复的流程才是", "看高分经验时，不要只记作息和刷题数量。真正值得迁移的是：如何发现薄弱点、怎样安排复盘、什么时候寻求老师帮助。\n\n我更建议每周只设一个主攻科目和两个维护科目。周末检查本周真正解决了哪些问题，再决定下一周，而不是把别人的时间表原样搬过来。", []string{"高分经验", "学习节奏", "复盘"}, "physics", []string{"chemistry", "biology"}, "experience", "高三", "6a4f2d66000000001702a979", "广东省屏蔽生：Ask me anything！", "Morniverse", 8296, 1603, 3172},
	{"gd-data@soulcourse.cn", "2027拟在粤招生选科要求，应该怎么看", "广东新高一家庭核对政策时，要先确认文件适用的高考年份，再查院校专业的首选科目和再选科目要求。写着两科的专业通常意味着两科都要满足，不是任选一科。\n\n政策表适合做“不能报什么”的排除，不适合直接替学生决定组合。最终仍要结合成绩稳定性、学校开班和目标专业层次。", []string{"2027高考", "拟在粤招生", "政策核对"}, "physics", []string{"chemistry", "biology"}, "data", "高一", "67c7c966000000000603ed7e", "广东省2027高考选科有重大变化！", "周周聊", 41, 7, 29},
	{"gaosan-review@soulcourse.cn", "高三答疑：先稳住每天能完成的三件事", "高三计划越复杂，越容易在一次考试后全部推倒。每天保留三项可完成任务：一项错题复盘、一项限时训练、一项基础回顾。完成后再追加，不把休息当成失败。\n\n遇到排名波动，先看失分结构和时间分配，至少观察两次考试再调整主线。", []string{"高三", "答疑", "学习计划"}, "physics", []string{"chemistry", "geography"}, "experience", "高三", "6a4f1cd30000000007013d07", "广东26年高考屏蔽生 ｜Ask me anything！", "深中黄学长", 1930, 310, 899},
	{"bayarea-observer@soulcourse.cn", "选科报名提交前，务必做一次双人核对", "科目选择属于低频但高影响操作。提交前把姓名、考生号、首选科目和两门再选科目逐项读出，由学生和家长分别确认一次，并保存学校系统的确认页。\n\n不要依赖口头转达，也不要在最后一分钟修改。发现异常第一时间联系班主任和教务，保留沟通记录。", []string{"报名核对", "选科安全", "家长沟通"}, "history", []string{"politics", "geography"}, "experience", "高一", "6a3fc55a00000000080037c1", "考前三天把地理错报成生物的后续", "紫小鱼", 11000, 687, 1144},
	{"gd-data@soulcourse.cn", "物化绑定第一年，志愿填报为什么更看重专业组", "物化要求变化会重新分配不同专业组的考生数量，但“可能捡漏”不应成为选科依据。填志愿时应按院校专业组逐条核对限制、招生计划和往年位次。\n\n中分段考生尤其要准备冲、稳、保三套方案，不要只盯着某个预测最低分。", []string{"物化绑定", "志愿填报", "专业组"}, "physics", []string{"chemistry", "biology"}, "data", "毕业生", "6672f1d1000000000e031656", "广东高考志愿填报，物化绑定第一年，有漏！", "纯碱不是碱", 831, 640, 443},
	{"lingnan-counselor@soulcourse.cn", "广东3+1+2的12种组合，怎么缩小到两种", "第一步在物理和历史之间做首选科目判断，依据是目标专业硬约束和长期学习表现。第二步从四门再选科目里排除明显跟不上的科目。\n\n最后只保留两种候选组合，用一个月记录每科作业时长、考试排名和学习感受，再和老师讨论。这样比一次性拍板更可靠。", []string{"3+1+2", "12种组合", "选科流程"}, "physics", []string{"chemistry", "geography"}, "experience", "高一", "6878cae2000000002201cffa", "广东新高一选科终极指南｜3+1+2怎么选？", "高途广东", 45, 0, 39},
	{"huizhou-growth@soulcourse.cn", "选科前做一张纸：专业、成绩、成本", "把候选组合各写一列。第一行列专业限制，第二行写最近三次年级排名，第三行估算每周补弱时间，第四行写自己愿意长期学习的程度。\n\n如果一项信息不确定，就标记“待核对”并找到对应老师或官方文件。选择不是靠感觉消除焦虑，而是把未知逐项变成可验证信息。", []string{"选科表格", "信息核对", "高一"}, "physics", []string{"chemistry", "politics"}, "experience", "高一", "68f36977000000000300c13c", "高中一年选科你值得了解", "下班积极分子", 4068, 108, 2378},
	{"gz-physics@soulcourse.cn", "物化政一年后的真实体验", "物化政的优势是保留较多理工方向，同时政治对部分升学与职业路径有帮助。困难也很直接：三科思维切换大，政治不能靠考前突击，物理化学又需要持续训练。\n\n如果选择它，建议每周固定一次政治材料题复盘，避免所有时间都被物化作业占满。", []string{"物化政", "组合体验", "广东"}, "physics", []string{"chemistry", "politics"}, "experience", "高三", "6a4768b10000000016025a8b", "26年广东高考物化政｜Ask me anything", "死循环", 1091, 300, 502},
	{"zhuhai-review@soulcourse.cn", "广东复读决定，先算清四笔账", "复读不只是再学一年。先核对学校是否招收、费用与住宿、现有位次的提升空间、心理和家庭支持。把每一项写成最坏情况和可承受上限。\n\n同时正常完成当年志愿方案，再决定是否放弃录取。情绪最强烈的几天不适合做不可逆选择。", []string{"广东复读", "决策", "心理准备"}, "physics", []string{"chemistry", "biology"}, "question", "毕业生", "6a3b8747000000001603f455", "有没有广东高考复读学校推荐。", "花草茶", 734, 584, 161},
	{"foshan-study@soulcourse.cn", "数学不翻车：先训练时间预算", "数学提升不只是多做难题。每次整卷给选填、前三道大题和后半段分别设时间上限，超时就标记并继续，最后再回头。\n\n复盘时区分“不会”“会但慢”“计算失误”。三类问题的解法不同：补知识、练模型或改书写检查流程。", []string{"数学", "时间分配", "广东高考"}, "physics", []string{"chemistry", "biology"}, "experience", "高三", "6867ed16000000001202d8fe", "广东高考695 数学经验分享", "飞舞", 1018, 26, 764},
	{"shenzhen-note@soulcourse.cn", "作业很多时，时间管理只保留三个区块", "把一天分成课堂吸收、放学后清任务、睡前短复盘三个区块。先完成学校任务中的高反馈部分，再做额外练习。\n\n每天至少留出可恢复的休息，不用分钟级计划填满全天。真正要记录的是任务实际耗时，连续一周后再调整。", []string{"时间管理", "高中学习", "作息"}, "history", []string{"politics", "geography"}, "experience", "高二", "696a40bd000000002103edee", "女高学习指南|如何成为年级第一之时间管理", "柠檬小狗日记", 5166, 38, 2170},
	{"chaoshan-parent@soulcourse.cn", "陪读家长最有用的支持，不是替孩子排满日程", "家长可以每周固定一次二十分钟沟通，只问三件事：这周最困难的是什么、需要哪种帮助、下周想调整什么。其余时间尽量不追问每一道题和每一次排名。\n\n当孩子提出选科想法时，先一起查官方限制，再讨论家庭担忧，避免用亲友案例替代孩子自己的条件。", []string{"家长沟通", "陪读", "选科"}, "history", []string{"politics", "geography"}, "experience", "高二家长", "6a538a7000000000070106d2", "陪跑完高中，给大家几点建议", "望舒倾听", 1389, 48, 2989},
	{"zhuhai-review@soulcourse.cn", "复读值不值得：给自己两周冷静期", "先把失利原因拆成知识缺口、考试状态、志愿信息和长期健康四类。只有能够明确改进路径的问题，才可能通过复读改善。\n\n冷静期内去了解真实学校环境、费用和作息，也准备正常志愿。最终决定要包含退出方案，而不是只靠“不甘心”。", []string{"复读", "高考复盘", "决策"}, "history", []string{"politics", "geography"}, "experience", "毕业生", "6a3b6e5b000000001003e27f", "复读一年的真实感受", "fangyismolly", 2382, 354, 303},
	{"bayarea-observer@soulcourse.cn", "学校不会替你核对的三类选科信息", "第一类是目标年份的专业选考要求，第二类是本校实际开班与师资，第三类是学生本人近半年的排名和投入。三类信息缺一不可。\n\n网上的经验可以提供问题清单，但不能直接给结论。任何“绝对好赋分”“一定有专业”的说法，都应回到官方文件和个人数据复核。", []string{"信息差", "选科核对", "专业要求"}, "physics", []string{"chemistry", "geography"}, "data", "教研", "6a63698d0000000011019b8a", "老师绝对不会告诉你的选科内幕", "酸酸学长", 2024, 44, 1031},
	{"gaosan-review@soulcourse.cn", "高考失常后，先处理生活再处理志愿", "出分后的前几天允许自己难过，但不要同时做十个决定。先保证睡眠和吃饭，找可信任的人一起整理位次、专业限制和志愿时间线。\n\n一次考试不会把所有路径关掉。把“人生完了”改写成三个具体问题，通常就能找到下一步可执行的动作。", []string{"高考失常", "志愿", "心理调节"}, "physics", []string{"chemistry", "biology"}, "experience", "毕业生", "6a3bfa1d000000001102c864", "26年广东高考失常发挥后的亿点感受", "一乘二", 541, 167, 118},
	{"foshan-study@soulcourse.cn", "普通学生也能复制的语数复盘框架", "语文按题型记录设误点和表达缺口，数学按知识、速度、计算三类记录错误。每周各选两个高频问题集中修正，不追求一次覆盖所有薄弱点。\n\n学习经验真正有用的部分，是能够在自己的试卷上验证。连续三周没有改善的方法就停掉，换更具体的训练。", []string{"语文", "数学", "可复制方法"}, "history", []string{"politics", "geography"}, "experience", "高三", "6985e4b2000000001a01c941", "省前十北大光华｜语&数学习经验帖", "鱼曰", 1554, 64, 864},
	{"shenzhen-note@soulcourse.cn", "高三刷题前，先回答为什么做这一套", "整套卷适合训练节奏和暴露综合问题，专题题适合解决明确薄弱点。两者目的不同，不能只用刷套卷的数量衡量努力。\n\n每次训练前写下目标，结束后只整理与目标相关的三道题。这样复盘负担更小，也更容易持续。", []string{"高三刷题", "专题训练", "复盘"}, "physics", []string{"chemistry", "biology"}, "experience", "高三", "6a54a87a00000000150254c6", "广东省屏蔽生：分享第三弹！", "Morniverse", 1214, 74, 732},
	{"gd-data@soulcourse.cn", "为什么不能照搬外省的选科策略", "不同省份的录取模式、赋分办法、专业组设置和学校开班都可能不同。看到外省高赞经验时，先检查它的年份和省份，再判断哪些方法只是通用学习建议。\n\n涉及专业限制的结论，必须回到广东当届拟招生要求；涉及赋分的结论，至少结合本校真实排名。", []string{"外省经验", "广东选科", "信息核验"}, "physics", []string{"chemistry", "politics"}, "data", "教研", "6a638848000000000100f923", "四中孙同学：北京高考选科策略", "孙同学在这里", 19, 3, 13},
	{"gz-physics@soulcourse.cn", "物化生不是默认答案，适合才是", "物化生能保留较多理工农医路径，但三科都需要持续投入。生物并不只是背诵，后期同样重视材料分析和实验思维。\n\n如果化学或生物长期排名靠后，先做四周补弱测试，再决定是否选择。不要因为“别人都选”忽略自己的学习成本。", []string{"物化生", "学习成本", "准高一"}, "physics", []string{"chemistry", "biology"}, "experience", "高一", "6a4c4d19000000001602665a", "初升高选科不能无脑物化生，血泪教训", "舟舟逆袭规划", 960, 17, 1370},
	{"gd-data@soulcourse.cn", "物化组合的机会，不能只看一年的低分专业组", "物化限制可能让部分专业组的竞争结构发生变化，但招生计划和报考热度每年都会调整。所谓“低分机会”更适合用于志愿方案中的冲刺层，而不是选科阶段的承诺。\n\n选科看长期可报范围，填志愿再看当年位次和计划，两个决策不要混在一起。", []string{"物化组合", "中分段", "志愿机会"}, "physics", []string{"chemistry", "geography"}, "data", "教研", "667a54a9000000001d016017", "广东物化低分考生有福了！", "idealism个体", 514, 296, 231},
	{"dongguan-parent@soulcourse.cn", "从物生政限制看，家长沟通哪里出了问题", "“喜欢就好”听起来尊重，但如果完全不核对专业要求，风险会被推迟到填志愿时才出现。家长能做的是陪孩子查限制、列问题、找老师确认，而不是替孩子选。\n\n如果家庭无法提供专业信息，也可以坦诚说不知道，然后一起找官方来源。比反复转述亲友建议更有效。", []string{"物生政", "家长沟通", "专业限制"}, "physics", []string{"biology", "politics"}, "experience", "高一家长", "685def89000000000d01ab8d", "广东高考540分人生完蛋了", "囧囧瓜", 380, 75, 43},
	{"lingnan-counselor@soulcourse.cn", "做一份自己的专业范围清单", "从三个层次各选几所目标院校，把可能报考的专业放进表格，记录首选和再选科目要求。把“可报”“不可报”“待核对”分开。\n\n这份个人清单比网上的组合覆盖率更有用，因为它直接对应你的城市、院校层次和兴趣。每学期更新一次即可。", []string{"专业范围", "选科清单", "院校核对"}, "history", []string{"politics", "geography"}, "data", "高一", "6a3c0779000000001503f69a", "高考选科组合的专业范围", "志愿填报锦鲤本鲤", 17, 0, 10},
	{"dongguan-parent@soulcourse.cn", "惠州高一家长：选科会谈怎么开", "先让孩子说明自己的候选组合和理由，家长只负责记录问题。第二轮一起核对专业限制和学校开班，第三轮再讨论家庭担忧。\n\n一次会谈控制在半小时内，不在考试刚结束时逼结论。无法达成一致时，把争议点带给班主任或生涯老师，而不是继续互相说服。", []string{"惠州", "高一家长", "选科会谈"}, "history", []string{"politics", "geography"}, "question", "高一家长", "68f0c1c00000000004017719", "广东高一家长必看！新高考如何选科", "Dr 陈", 4, 0, 2},
	{"huizhou-growth@soulcourse.cn", "高三外置大脑：每天只记录真正学会的内容", "不要写“今天学习四小时”或“做完练习册十页”，只写真正掌握的知识点和解决的问题。晚上用五分钟回看，第二天优先复习仍说不清的条目。\n\n有整块时间时，可以集中攻克一个难点；碎片时间用于回忆和检查。记录越具体，越容易看到真实进步。", []string{"高三技巧", "有效学习", "知识清单"}, "physics", []string{"chemistry", "geography"}, "experience", "高三", "67ab35de000000001701fa00", "分享一些拯救了我的高三的学习小技巧", "笨鸟不会飞", 1485, 30, 449},
	{"gz-physics@soulcourse.cn", "物理是弱项时，别用总分掩盖问题", "一次高考物理分数不能直接说明能力，但平时如果只记公式、不理解适用条件，综合题会持续失分。先把错题按模型分类，再用口头讲解检验是否真的会。\n\n目标不是每天做最多题，而是让同类错误不再重复。每周请老师确认一到两个关键卡点，比临考突击更稳。", []string{"高中物理", "弱项", "错题分类"}, "physics", []string{"chemistry", "biology"}, "experience", "高三", "6836d5e5000000000f03bc75", "广东高考物理66分上985经验分享", "麻将大王", 321, 114, 41},
}

func seedGuangdongPhaseOne(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var appliedAt string
	err = tx.QueryRow(`SELECT applied_at FROM app_migrations WHERE name = ?`, guangdongPhaseOneMigration).Scan(&appliedAt)
	if err == nil {
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("12345678"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, user := range guangdongUsers {
		if _, err := tx.Exec(`
			INSERT OR IGNORE INTO users (email, password_hash, nickname, role, province, grade, created_at, updated_at)
			VALUES (?, ?, ?, ?, '广东', ?, ?, ?)
		`, user.email, string(passwordHash), user.nickname, user.role, user.grade, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("seed content user %s: %w", user.email, err)
		}
	}

	for index, post := range guangdongPosts {
		var userID int64
		var nickname, role string
		if err := tx.QueryRow(`
			SELECT id, nickname, role FROM users WHERE lower(email) = lower(?) AND deleted_at IS NULL
		`, post.email).Scan(&userID, &nickname, &role); err != nil {
			return fmt.Errorf("load content user %s: %w", post.email, err)
		}

		imageCount := guangdongPostImageCounts[index+1]
		if imageCount == 0 {
			imageCount = 1
		}
		postImages := make([]string, 0, imageCount)
		for imageIndex := 1; imageIndex <= imageCount; imageIndex++ {
			postImages = append(postImages, fmt.Sprintf("/content/guangdong/%02d-%d.webp", index+1, imageIndex))
		}
		imageURLs, _ := json.Marshal(postImages)
		tags, _ := json.Marshal(post.tags)
		electives, _ := json.Marshal(post.electives)
		createdAt := now.Add(-time.Duration(index) * 6 * time.Hour).Format(time.RFC3339Nano)
		result, err := tx.Exec(`
			INSERT INTO posts
				(user_id, author_name, author_role, title, content, image_urls, tags, track, electives, category, grade, province, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '广东', ?, ?)
		`, userID, nickname, role, post.title, post.content, string(imageURLs), string(tags), post.track, string(electives), post.category, post.grade, createdAt, createdAt)
		if err != nil {
			return fmt.Errorf("seed content post %q: %w", post.title, err)
		}
		postID, err := result.LastInsertId()
		if err != nil {
			return err
		}
		sourceURL := "https://www.xiaohongshu.com/explore/" + post.sourceNoteID
		if _, err := tx.Exec(`
			INSERT INTO content_sources
				(post_id, source_platform, source_url, source_note_id, source_title, source_author,
				 source_likes, source_comments, source_favorites, source_format, transformation_note, captured_at)
			VALUES (?, 'xiaohongshu', ?, ?, ?, ?, ?, ?, ?, '图文', ?, ?)
		`, postID, sourceURL, post.sourceNoteID, post.sourceTitle, post.sourceAuthor,
			post.sourceLikes, post.sourceComments, post.sourceFavorites,
			"基于公开笔记主题与讨论信号重新撰写；站内作者为独立 SoulCourse 账号。", now.Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("seed content source %s: %w", post.sourceNoteID, err)
		}
		payload, _ := json.Marshal(map[string]any{
			"postId":      fmt.Sprintf("%d", postID),
			"content":     post.content,
			"track":       post.track,
			"electives":   post.electives,
			"category":    post.category,
			"grade":       post.grade,
			"province":    "广东",
			"imageUrls":   postImages,
			"sourceUrl":   sourceURL,
			"sourceTitle": post.sourceTitle,
		})
		if _, err := tx.Exec(`
			INSERT INTO admin_content_records
				(id, module, title, content_type, status, scope, owner, tags, summary, url,
				 priority, sort_order, payload, created_at, updated_at)
			VALUES (?, 'posts', ?, ?, '已上架', '广东', ?, ?, ?, ?, '常规', ?, ?, ?, ?)
		`, "xhs-"+post.sourceNoteID, post.title, seededPostType(post.category), nickname, string(tags),
			summarizeSeedContent(post.content), fmt.Sprintf("/posts/%d", postID), 200+index,
			string(payload), createdAt, createdAt); err != nil {
			return fmt.Errorf("seed admin post %s: %w", post.sourceNoteID, err)
		}
	}

	if _, err := tx.Exec(`INSERT INTO app_migrations (name, applied_at) VALUES (?, ?)`, guangdongPhaseOneMigration, now.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

func seededPostType(category string) string {
	switch category {
	case "question":
		return "家长提问"
	case "data":
		return "数据建议"
	default:
		return "经验帖"
	}
}

func summarizeSeedContent(content string) string {
	runes := []rune(content)
	if len(runes) <= 120 {
		return content
	}
	return string(runes[:120]) + "..."
}
