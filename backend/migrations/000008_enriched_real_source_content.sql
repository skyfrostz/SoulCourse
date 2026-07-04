DROP TABLE IF EXISTS seed_real_posts;

CREATE TEMP TABLE seed_real_posts (
  slug TEXT NOT NULL,
  author_name TEXT NOT NULL,
  author_role TEXT NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  track TEXT NOT NULL,
  electives TEXT[] NOT NULL,
  category TEXT NOT NULL,
  grade TEXT NOT NULL,
  province TEXT NOT NULL,
  likes_count INTEGER NOT NULL,
  favorites_count INTEGER NOT NULL,
  created_offset INTERVAL NOT NULL,
  image_urls TEXT[] NOT NULL,
  tags TEXT[] NOT NULL,
  source_url TEXT NOT NULL,
  source_title TEXT NOT NULL,
  source_type TEXT NOT NULL,
  content_type TEXT NOT NULL,
  status TEXT NOT NULL,
  sort_order INTEGER NOT NULL,
  topic_slugs TEXT[] NOT NULL
);

INSERT INTO seed_real_posts
  (slug, author_name, author_role, title, content, track, electives, category, grade, province, likes_count, favorites_count, created_offset, image_urls, tags, source_url, source_title, source_type, content_type, status, sort_order, topic_slugs)
VALUES
  ('moe-2026-admission-safety', '招录公开课', 'counselor', '教育部2026招生通知里，我最想提醒家长的3个字：公开性',
   $$昨晚把教育部2026年普通高校招生工作通知重新看了一遍，给家长划一个重点：别只盯“今年会不会变难”，更该盯信息公开、招生纪律、专项计划和随迁子女这些硬规则。我的做法是把目标省份、目标高校招生章程、阳光高考专题放在一个收藏夹里，每周固定复核一次。选科也是同理，先核对公开规则，再谈感觉和热度。资料来源：教育部2026年普通高校招生工作通知。$$,
   'physics', ARRAY['chemistry','politics']::TEXT[], 'data', '高三', '全国', 486, 235, interval '1 hour',
   ARRAY['https://images.unsplash.com/photo-1497366754035-f200968a6e72?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['2026招生','官方来源','家长必看']::TEXT[],
   'https://www.moe.gov.cn/srcsite/A15/moe_776/s3258/202601/t20260121_1427110.html', '教育部关于做好2026年普通高校招生工作的通知', 'official_policy', '数据建议', '已上架', 110, ARRAY['new-gaokao-policy-2026','province-admission-notes']::TEXT[]),

  ('moe-2026-special-types', '特招备忘录', 'counselor', '强基综评特殊类型别只看热闹，先看纪律线',
   $$特殊类型招生每年都很热，但我不建议一上来就问“能不能低分进名校”。先看教育部关于2026年特殊类型招生的管理口径：资格、测试、录取、公示都要经得起追溯。适合收藏这条的人：准备强基、综评、艺术体育、高水平运动队等路径的家庭。我的提醒是，孩子高一高二可以准备能力，高三必须按节点和材料清单走，不要把“听说可以”当依据。资料来源：教育部办公厅特殊类型招生工作通知。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'data', '高二', '全国', 352, 171, interval '2 hours',
   ARRAY['https://images.unsplash.com/photo-1519389950473-47ba0277781c?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['特殊类型招生','强基综评','时间线']::TEXT[],
   'https://www.moe.gov.cn/srcsite/A15/moe_776/tslxzs/202510/t20251031_1418664.html', '教育部办公厅关于做好2026年普通高等学校部分特殊类型招生工作的通知', 'official_policy', '数据建议', '已上架', 120, ARRAY['new-gaokao-policy-2026','grade-eleven-timeline']::TEXT[]),

  ('beijing-2026-3plus3', '北京升学小本', 'counselor', '北京2026高考：等级考三门怎么和录取绑在一起',
   $$北京考生别把“3+3”理解成随便挑三科。北京教育考试院2026招生规定里写得很清楚：本科录取总成绩由语数外三门统一高考成绩，加上考生选考的三门学考等级考成绩构成，满分750。换句话说，选科既影响你能不能报，也影响最后那张成绩单的组成。我的建议：目标还模糊的同学，先列出不能承受的科目，再列想保留的专业门类。资料来源：北京教育考试院2026招生工作规定。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'data', '高一', '北京', 421, 199, interval '3 hours',
   ARRAY['https://images.unsplash.com/photo-1523240795612-9a054b0db644?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['北京高考','3+3','等级考']::TEXT[],
   'https://www.bjeea.cn/html/gkgz/tzgg/2026/0505/88114.html', '北京市2026年普通高等学校招生工作规定', 'official_policy', '数据建议', '已上架', 130, ARRAY['new-gaokao-policy-2026','province-admission-notes']::TEXT[]),

  ('beijing-2026-volunteer-password', '海淀家长手账', 'parent', '志愿填报密码这件小事，真的会变成大事',
   $$北京2026招生规定里有一句很朴素但很重要：志愿填报系统初始密码和报名系统密码相关，考生要自己负责志愿真实性和准确性，截止后不能随意更改。说人话就是，账号密码、志愿草表、最后提交截图都要有家庭流程。我们家准备做三份东西：纸质志愿草稿、电子版核对表、提交后截图归档。别笑，这种小事出错，比选科纠结更让人崩。资料来源：北京教育考试院。$$,
   'history', ARRAY['politics','geography']::TEXT[], 'experience', '高三', '北京', 276, 118, interval '4 hours',
   ARRAY['https://images.unsplash.com/photo-1516321497487-e288fb19713f?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['志愿填报','北京高考','家长经验']::TEXT[],
   'https://www.bjeea.cn/html/gkgz/tzgg/2026/0505/88114.html', '北京市2026年普通高等学校招生工作规定', 'official_policy', '经验帖', '已上架', 140, ARRAY['province-admission-notes','parents-decision-room']::TEXT[]),

  ('guangdong-2026-special-plan', '广东专项观察', 'counselor', '广东专项计划：户籍、学籍、实际就读一个都不能漏',
   $$看广东2026重点高校招生专项计划通知，最大的感受是“细”。户籍、连续年限、学籍、实际就读、资格公示，每一项都不是口头说说。适合农村和脱贫地区相关实施区域家庭重点核对：孩子本人和监护人户籍是否在范围内，孩子高中三年是否在户籍所在县区连续学籍并实际就读。不要等志愿填报时才发现资格没走完。资料来源：广东省教育厅专项计划通知。$$,
   'physics', ARRAY['chemistry','geography']::TEXT[], 'data', '高三', '广东', 398, 186, interval '5 hours',
   ARRAY['https://images.unsplash.com/photo-1434030216411-0b793f4b4173?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['广东高考','专项计划','资格审核']::TEXT[],
   'https://gaokao.chsi.com.cn/gkxx/zc/ss/202603/20260325/2293454298.html', '广东省教育厅关于做好2026年重点高校招生专项计划工作的通知', 'official_policy', '数据建议', '已上架', 150, ARRAY['new-gaokao-policy-2026','province-admission-notes']::TEXT[]),

  ('guangdong-special-plan-process', '羊城升学笔记', 'parent', '广东高校专项2026改成什么流程？给家长画重点',
   $$广东专项计划通知里有个变化值得单独记：2026年起，高校专项计划不再由高校组织报名、资格审核和校测，相关招生院校、专业和计划数通过广东省2026普通高校招生专业目录公布。对家长来说，动作变成三步：先确认自己是不是实施区域和资格人群，再按省报名系统节点提交材料，最后等专业目录和志愿文件。别用往年高校简章习惯套今年。资料来源：广东省教育厅。$$,
   'history', ARRAY['politics','geography']::TEXT[], 'experience', '高三', '广东', 247, 104, interval '7 hours',
   ARRAY['https://images.unsplash.com/photo-1488998427799-e3362cec87c3?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['广东专项','招生目录','家长沟通']::TEXT[],
   'https://gaokao.chsi.com.cn/gkxx/zc/ss/202603/20260325/2293454298.html', '广东省教育厅关于做好2026年重点高校招生专项计划工作的通知', 'official_policy', '经验帖', '已上架', 160, ARRAY['province-admission-notes','parents-decision-room']::TEXT[]),

  ('henan-2026-migrant-children', '豫见高考', 'counselor', '河南随迁子女政策提醒：别把材料拖到最后一周',
   $$河南省教育厅2026招生工作通知提到，要做好符合条件的随迁子女在流入地参加高考工作，并结合居住证制度等情况简化材料、提供便利。但“简化”不等于“不审”，学籍管理、报名资格审核还是重点。家长可以提前做一张材料表：居住证、就业或社保、学籍、实际就读证明、报名节点。越早核对，越少临门一脚的焦虑。资料来源：河南省教育厅。$$,
   'history', ARRAY['politics','biology']::TEXT[], 'data', '高二', '河南', 331, 152, interval '9 hours',
   ARRAY['https://images.unsplash.com/photo-1503676260728-1c00da094a0b?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['河南高考','随迁子女','报名资格']::TEXT[],
   'https://jyt.henan.gov.cn/2026/04-27/3346226.html', '河南省教育厅关于做好2026年普通高等学校招生工作的通知', 'official_policy', '数据建议', '已上架', 170, ARRAY['new-gaokao-policy-2026','province-admission-notes']::TEXT[]),

  ('henan-rural-plan', '中原志愿室', 'counselor', '河南家长看专项计划：资格审核比想象中更细',
   $$河南2026招生通知里强调专项计划、资格审核和学籍管理。我的理解：符合条件的农村和脱贫地区考生，专项计划是机会，但不是“报名了就能用”的福利。家长最好提前问清三件事：户籍是否符合，学籍和实际就读是否连续，学校或县区是否有公示流程。选科层面也别只问“专项能不能降分”，还要看专项计划里的专业组对科目有没有硬要求。资料来源：河南省教育厅。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'data', '高三', '河南', 214, 93, interval '11 hours',
   ARRAY['https://images.unsplash.com/photo-1450101499163-c8848c66ca85?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['河南专项','资格审核','专业组']::TEXT[],
   'https://jyt.henan.gov.cn/2026/04-27/3346226.html', '河南省教育厅关于做好2026年普通高等学校招生工作的通知', 'official_policy', '数据建议', '已上架', 180, ARRAY['province-admission-notes','major-requirement-checklist']::TEXT[]),

  ('sd-2027-subject-warning', '山东选科雷达', 'counselor', '山东2027选科要求：同专业不同学校真的会不一样',
   $$山东省教育招生考试院公布2024通用版、2027通用版选考科目要求时，特别提醒同一专业在不同高校可能会有不同选科要求。这个信息对高一高二太重要了：不要只问“法学是不是不限”“计算机是不是物化”，要具体到学校、专业类和当年招生计划。我的建议是做两列清单：目标专业和目标学校，逐校核对，不要拿一个热门说法盖所有学校。资料来源：山东省教育招生考试院。$$,
   'physics', ARRAY['chemistry','geography']::TEXT[], 'data', '高一', '山东', 522, 266, interval '13 hours',
   ARRAY['https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['山东选科','2027要求','逐校核对']::TEXT[],
   'https://www.sdzk.cn/NewsInfo.aspx?NewsID=6819', '关于公布普通高校拟在山东招生专业（类）选考科目要求的公告', 'official_policy', '数据建议', '已上架', 190, ARRAY['major-requirement-checklist','province-admission-notes']::TEXT[]),

  ('sd-actual-plan', '济南高一家长', 'parent', '山东公告里一句话：要求能报，不代表当年一定招生',
   $$我以前以为查到“符合选科要求”就稳了，后来读山东考试院公告才发现还有一层：高校报送的要求是通用要求，当年并非所有高校、所有列出的专业都会在山东安排招生计划，实际以当年公布计划为准。这个提醒太真实了。现在我给孩子做专业清单会分三档：能报、往年在鲁招生、今年等计划确认。选科能打开门，但志愿填报还要看门后面有没有座位。资料来源：山东省教育招生考试院。$$,
   'history', ARRAY['politics','geography']::TEXT[], 'experience', '高一', '山东', 287, 132, interval '15 hours',
   ARRAY['https://images.unsplash.com/photo-1517245386807-bb43f82c33c4?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['山东高考','招生计划','家长笔记']::TEXT[],
   'https://www.sdzk.cn/NewsInfo.aspx?NewsID=6819', '关于公布普通高校拟在山东招生专业（类）选考科目要求的公告', 'official_policy', '经验帖', '已上架', 200, ARRAY['parents-decision-room','major-requirement-checklist']::TEXT[]),

  ('shanghai-2027-major-groups', '沪上专业组笔记', 'counselor', '上海2027：院校专业组才是志愿里的真实单位',
   $$上海教育考试院2027选考科目要求说明里，我最建议收藏的是“本科阶段仍以院校专业组作为志愿填报与投档录取的基本单位”。这句话能解释很多焦虑：不是只看学校名，也不是只看专业名，而是看专业组里实际放了哪些专业、选科怎么要求。上海考生做目标表时，建议把学校、专业组、组内专业、科目要求四列一起写。资料来源：上海市教育考试院。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'data', '高一', '上海', 463, 221, interval '18 hours',
   ARRAY['https://images.unsplash.com/photo-1518005020951-eccb494ad742?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['上海高考','院校专业组','2027选科']::TEXT[],
   'https://www.shmeea.edu.cn/page/02200/20250221/19114.html', '关于公布2027年普通高校本科专业选考科目要求的通知', 'official_policy', '数据建议', '已上架', 210, ARRAY['major-requirement-checklist','province-admission-notes']::TEXT[]),

  ('shanghai-2027-adjustment', '上海高二记录员', 'student', '上海考生别只收藏目录，还要等当年计划',
   $$今天把上海2027选考科目通知看完，心态稳了一点也更谨慎了。目录是重要参考，但通知里也提醒，实际在沪招生的院校和专业可能调整，2027年当年招生目录才是最终版本。我的理解：高一高二先用目录做方向筛选，高三再用当年计划做志愿确认。现在就把所有话说死，反而容易误伤自己。资料来源：上海市教育考试院。$$,
   'history', ARRAY['politics','geography']::TEXT[], 'experience', '高二', '上海', 218, 86, interval '21 hours',
   ARRAY['https://images.unsplash.com/photo-1497633762265-9d179a990aa6?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['上海2027','招生计划','学生记录']::TEXT[],
   'https://www.shmeea.edu.cn/page/02200/20250221/19114.html', '关于公布2027年普通高校本科专业选考科目要求的通知', 'official_policy', '经验帖', '已上架', 220, ARRAY['grade-eleven-timeline','major-requirement-checklist']::TEXT[]),

  ('jiangsu-3plus12-basic', '苏小招答疑', 'teacher', '江苏3+1+2：为什么我建议先把目标专业写出来',
   $$江苏考试院发布2024拟在苏招生本科专业选考要求时，把3+1+2讲得很直白：首选物理或历史，另外在化学、生物、政治、地理中选两科，填报时必须符合高校专业选科要求。老师视角看，最怕学生只按“哪门分高”选，完全不看目标专业。建议先写10个想保留的专业，再看它们对首选和再选科目的要求，最后才比较赋分优势。资料来源：江苏省教育考试院。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'data', '高一', '江苏', 514, 242, interval '1 day',
   ARRAY['https://images.unsplash.com/photo-1513258496099-48168024aec0?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['江苏选科','3+1+2','专业要求']::TEXT[],
   'https://www.jseea.cn/webfile/index/index_zkxx/2022-01-18/27031.html', '省教育厅关于发布2024年拟在江苏招生的普通高校本科专业选考科目要求的公告', 'official_policy', '数据建议', '已上架', 230, ARRAY['major-requirement-checklist','physics-track-how-to-choose']::TEXT[]),

  ('jiangsu-query-path', '南京班主任', 'teacher', '江苏选科查询，别用截图传来传去',
   $$班里最近又开始互传“某专业要求截图”。我一般会让学生回到江苏省教育考试院的查询入口，按学校、专业、考生选考科目或高校选科要求查原始表。截图最大的问题是看不到年份、地区和专业类备注。选科这事最好形成习惯：所有结论后面都写来源链接和查询日期。这个动作很土，但能避开很多误传。资料来源：江苏省教育考试院公告。$$,
   'history', ARRAY['politics','geography']::TEXT[], 'experience', '高一', '江苏', 299, 121, interval '1 day 2 hours',
   ARRAY['https://images.unsplash.com/photo-1519389950473-47ba0277781c?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['江苏高考','查询方法','老师经验']::TEXT[],
   'https://www.jseea.cn/webfile/index/index_zkxx/2022-01-18/27031.html', '省教育厅关于发布2024年拟在江苏招生的普通高校本科专业选考科目要求的公告', 'official_policy', '经验帖', '已上架', 240, ARRAY['major-requirement-checklist','after-selection-score-up']::TEXT[]),

  ('gansu-four-types', '陇原选科表', 'counselor', '甘肃选科要求四类，最容易误解的是第三类',
   $$甘肃选考科目要求把类型分得很清楚：不提要求、只提首选、只提再选、首选和再选都提。很多家长会误解“只要求化学”这类再选条件，以为和物理历史没关系。实际填报还要看当年计划安排在物理类还是历史类。我的建议：看到“化学一门必须选考”时，不要立刻下结论，继续查该专业组实际类别。资料来源：甘肃选考科目要求说明。$$,
   'physics', ARRAY['chemistry','geography']::TEXT[], 'data', '高一', '甘肃', 372, 161, interval '1 day 5 hours',
   ARRAY['https://images.unsplash.com/photo-1509062522246-3755977927d7?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['甘肃高考','选科类型','化学要求']::TEXT[],
   'https://gaokao.chsi.com.cn/gkxx/zc/ss/202111/20211117/2132710994.html', '甘肃：2024年普通高校招生专业选考科目要求', 'official_policy', '数据建议', '已上架', 250, ARRAY['major-requirement-checklist','province-admission-notes']::TEXT[]),

  ('gansu-physical-chemical', '西北数据同学', 'student', '甘肃数据复盘：物化不是万能钥匙，但少了它很多门会关',
   $$看甘肃2024选科要求统计，含物理、含化学、物理+化学的比例都很高，我第一反应不是“大家都去选物化”，而是“目标理工农医的人别轻易拆开物化”。但如果物理或化学长期在班级后段，也要诚实面对学习成本。物化是打开很多门的钥匙，不是保证成绩的护身符。资料来源：阳光高考甘肃选考要求。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'experience', '高二', '甘肃', 336, 149, interval '1 day 7 hours',
   ARRAY['https://images.unsplash.com/photo-1516979187457-637abb4f9353?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['物化生','甘肃数据','理工农医']::TEXT[],
   'https://gaokao.chsi.com.cn/gkxx/zc/ss/202111/20211117/2132710994.html', '甘肃：2024年普通高校招生专业选考科目要求', 'official_policy', '经验帖', '已上架', 260, ARRAY['physics-track-how-to-choose','is-chemistry-important']::TEXT[]),

  ('zhejiang-computer', '浙江目录翻译器', 'counselor', '在浙江查计算机，我发现物化要求越来越硬',
   $$翻浙江2024普通高校招生专业选考科目要求时，一个直观感受：计算机科学与技术、数据科学与大数据技术、电子信息、机械等大量理工专业都把物理和化学放在一起看。对目标计算机的同学，我不建议只问“我物理还行能不能报”，还要问“化学能不能长期跟上”。如果化学只是暂时掉队，优先补体系；如果长期厌学，再重新评估目标专业。资料来源：浙江省教育考试院目录。$$,
   'physics', ARRAY['chemistry','geography']::TEXT[], 'data', '高一', '浙江', 588, 304, interval '1 day 10 hours',
   ARRAY['https://images.unsplash.com/photo-1515879218367-8466d910aaa4?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['浙江选科','计算机','物理化学']::TEXT[],
   'https://www.zjzs.net/col/xk2024/10733.html', '2024年普通高校招生专业选考科目要求', 'official_policy', '数据建议', '已上架', 270, ARRAY['is-chemistry-important','medical-computer-track']::TEXT[]),

  ('zhejiang-agriculture', '农学不是冷门箱', 'teacher', '农学不等于只背书，浙江目录里很多也是物化',
   $$有同学说“我喜欢植物，应该不用学那么硬的理科吧”。我翻浙江目录看到，农学、园艺、植物保护、动物医学、食品科学与工程等不少专业都要求物理+化学。农学不是简单背书，它和实验、化学、生物、数据分析都有关。喜欢生命科学方向的同学，别因为专业名字听起来“田园”就低估了数理化底座。资料来源：浙江省教育考试院目录。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'experience', '高一', '浙江', 341, 156, interval '1 day 12 hours',
   ARRAY['https://images.unsplash.com/photo-1501004318641-b39e6451bec6?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['农学','物化生','浙江目录']::TEXT[],
   'https://www.zjzs.net/col/xk2024/10733.html', '2024年普通高校招生专业选考科目要求', 'official_policy', '经验帖', '已上架', 280, ARRAY['medical-computer-track','physics-track-how-to-choose']::TEXT[]),

  ('catalog-2026-new-majors', '新专业观察员', 'counselor', '2026新增38个本科专业，别只被名字种草',
   $$教育部公布2026年本科专业目录后，最吸引眼球的是新增38种普通高校本科新专业，还有“交叉学科”门类。我的建议是：看到新专业先不要立刻上头，做三个核对：授予什么学位、核心课程偏理工还是偏文管、首批开设高校有没有相关学科支撑。新专业值得关注，但不是所有新名字都适合每个孩子。资料来源：教育部2026本科专业目录。$$,
   'physics', ARRAY['chemistry','politics']::TEXT[], 'data', '高二', '全国', 647, 352, interval '1 day 15 hours',
   ARRAY['https://images.unsplash.com/photo-1531482615713-2afd69097998?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['2026专业目录','新增专业','交叉学科']::TEXT[],
   'https://www.moe.gov.cn/srcsite/A08/moe_1034/s3882/202604/t20260427_1434931.html', '教育部关于公布《普通高等学校本科专业目录（2026年）》的通知', 'official_policy', '数据建议', '已上架', 290, ARRAY['major-requirement-checklist','medical-computer-track']::TEXT[]),

  ('catalog-embodied-ai', 'AI专业冷静剂', 'student', '具身智能听起来很酷，选科上我会先看数理底子',
   $$具身智能被列入2026本科专业目录后，我朋友圈一堆人说“这就是未来”。我也心动，但冷静想想，它大概率绕不开数学、物理、控制、计算机和工程实践。我的自测题是：我愿不愿意长期写代码、学力学/电路/控制，愿不愿意把AI放到真实设备里调试？如果答案只是“名字很酷”，那还不够。资料来源：教育部2026本科专业目录。$$,
   'physics', ARRAY['chemistry','geography']::TEXT[], 'question', '高二', '全国', 502, 247, interval '1 day 18 hours',
   ARRAY['https://images.unsplash.com/photo-1485827404703-89b55fcc595e?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['具身智能','新专业','自我评估']::TEXT[],
   'https://www.moe.gov.cn/srcsite/A08/moe_1034/s3882/202604/t20260427_1434931.html', '教育部关于公布《普通高等学校本科专业目录（2026年）》的通知', 'official_policy', '提问', '已上架', 300, ARRAY['medical-computer-track','physics-track-how-to-choose']::TEXT[]),

  ('catalog-digital-finance', '数金避坑员', 'teacher', '数字金融不是财经轻松版，数学和计算能力要先问自己',
   $$“数字金融”这类名字很容易让家长以为是传统财经换皮，但新专业背后其实更强调数据、模型、技术和业务交叉。老师角度看，孩子如果喜欢经济新闻但很排斥数学和编程，最好谨慎。选科时不用简单贴“文科/理科”标签，而是看课程画像：数学、统计、计算机、金融基础能不能一起承受。资料来源：教育部2026本科专业目录。$$,
   'history', ARRAY['politics','geography']::TEXT[], 'experience', '高二', '全国', 259, 113, interval '1 day 20 hours',
   ARRAY['https://images.unsplash.com/photo-1554224155-6726b3ff858f?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['数字金融','财经专业','课程画像']::TEXT[],
   'https://www.moe.gov.cn/jyb_xwfb/gzdt_gzdt/s5987/202604/t20260428_1435016.html', '《普通高等学校本科专业目录（2026年）》发布', 'official_policy', '经验帖', '已上架', 310, ARRAY['major-requirement-checklist','history-track-careers']::TEXT[]),

  ('catalog-brain-computer', '医工交叉同学', 'student', '脑机科学与技术：适合喜欢生物但也不怕工程的人吗？',
   $$我对脑机科学与技术很感兴趣，原因不是“听起来科幻”，而是它把生物、医学、电子信息、计算和工程问题拧在一起。看教育部2026本科专业目录后，我给自己写了个提醒：如果只是喜欢生物，不代表一定适合脑机；如果愿意啃数学、编程、物理实验，也许可以继续了解。准备去查首批学校的学院背景和培养方案。资料来源：教育部2026本科专业目录。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'question', '高一', '全国', 437, 226, interval '2 days',
   ARRAY['https://images.unsplash.com/photo-1559757175-0eb30cd8c063?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['脑机科学','医工交叉','物化生']::TEXT[],
   'https://www.moe.gov.cn/srcsite/A08/moe_1034/s3882/202604/t20260427_1434931.html', '教育部关于公布《普通高等学校本科专业目录（2026年）》的通知', 'official_policy', '提问', '已上架', 320, ARRAY['medical-computer-track','is-chemistry-important']::TEXT[]),

  ('discussion-physics-chemistry-biology', '冷静选物化生', 'student', '看了好多物化生讨论，我给自己列了三条冷静标准',
   $$最近刷了很多物化生讨论，大家争得很凶。我最后给自己列三条：第一，物理和化学不能只靠短期突击，要看连续三次考试的相对位置；第二，生物不是“背背就行”，遗传、实验和材料题也会卡人；第三，如果未来专业还没想好，物化生确实保留空间，但前提是我能承受节奏。来源是知乎选科讨论的观点整理，我没有照搬原帖，只把共性问题改成自己的核对表。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'experience', '高一', '河北', 612, 298, interval '2 days 3 hours',
   ARRAY['https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['物化生','公开讨论改写','自测表']::TEXT[],
   'https://www.zhihu.com/question/497583894', '物生地，物化生，物化政，历政地这四个我选哪个比较好？', 'public_discussion_rewritten', '经验帖', '已上架', 330, ARRAY['physics-track-how-to-choose','after-selection-score-up']::TEXT[]),

  ('discussion-physics-chemistry-geography', '地理错题本', 'student', '物化地不是低配物化生，我踩过的误区写下来',
   $$我一开始把物化地想得太简单：不想背生物，就换地理。后来刷到不少物化地讨论，才意识到地理不是纯记忆，图表、区域综合、材料分析都很吃理解。物化地适合什么人？我现在的答案是：物理化学能稳住，地理读图不痛苦，且目标专业不强依赖生物。不要把它当“少背一点的物化生”。来源：知乎物化地/物化生讨论改写整理。$$,
   'physics', ARRAY['chemistry','geography']::TEXT[], 'experience', '高二', '四川', 474, 219, interval '2 days 6 hours',
   ARRAY['https://images.unsplash.com/photo-1524661135-423995f22d0b?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['物化地','地理学习','公开讨论改写']::TEXT[],
   'https://www.zhihu.com/question/580822626', '新高考选课，物化生还是物化地？', 'public_discussion_rewritten', '经验帖', '已上架', 340, ARRAY['physics-track-how-to-choose','after-selection-score-up']::TEXT[]),

  ('discussion-physics-chemistry-politics', '物化政观察组', 'student', '物化政到底香不香？别只拿一场政治高分决定',
   $$贴吧里有同学拿生物和政治分数纠结，我太懂了。我的想法是：政治一次考高，不等于高考长期稳定；生物一次低，也不代表彻底不适合。物化政的优势是给公安、军校、部分法学和考公想象留空间，难点是学科思维跨度大。我的判断顺序会是：目标方向是否真的需要政治，其次看政治能否长期输出，最后看生物能否通过体系补回来。来源：公开贴吧讨论改写。$$,
   'physics', ARRAY['chemistry','politics']::TEXT[], 'question', '高一', '江苏', 389, 174, interval '2 days 9 hours',
   ARRAY['https://images.unsplash.com/photo-1517048676732-d65bc937f952?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['物化政','选科纠结','公开讨论改写']::TEXT[],
   'https://tieba.baidu.com/p/10390633657', '选科怎么选？物化生和物化政怎么抉择？', 'public_discussion_rewritten', '提问', '已上架', 350, ARRAY['physics-track-how-to-choose','parents-decision-room']::TEXT[]),

  ('discussion-history-politics-geography', '文科也要清醒', 'student', '史政地家长最该接受的一件事：路径清楚但边界也清楚',
   $$史政地不是“轻松组合”，只是路径更偏人文社科。看了很多选科讨论后，我觉得家长最需要接受两件事：一是孩子如果材料分析、表达、记忆和时政积累都不错，这条路可以走得很踏实；二是理工农医等方向会明显收窄，不能到高三才后悔。史政地适合目标更清晰的人，不适合“因为理科累所以逃过来”。来源：知乎公开讨论改写。$$,
   'history', ARRAY['politics','geography']::TEXT[], 'experience', '高一', '山东', 455, 206, interval '2 days 11 hours',
   ARRAY['https://images.unsplash.com/photo-1456513080510-7bf3a84b82f8?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['史政地','文科路径','公开讨论改写']::TEXT[],
   'https://www.zhihu.com/question/497583894', '物生地，物化生，物化政，历政地这四个我选哪个比较好？', 'public_discussion_rewritten', '经验帖', '已上架', 360, ARRAY['history-track-careers','parents-decision-room']::TEXT[]),

  ('discussion-single-physics', '走班亲历者', 'student', '单选物理或化学这件事，很多学校的排课成本也要算',
   $$公开讨论里有人提到单选物理或单选化学会遇到走班、拼班、答疑资源不完整等现实问题，我觉得这点常被忽略。政策上能不能选是一回事，学校有没有成熟班型、老师怎么排课、同伴氛围怎么样，是另一回事。对想“保留一点理科但不选物化双科”的同学，建议除了查专业要求，也问年级组往届班型和走班安排。来源：贴吧公开讨论改写。$$,
   'physics', ARRAY['biology','geography']::TEXT[], 'experience', '高一', '上海', 316, 139, interval '2 days 14 hours',
   ARRAY['https://images.unsplash.com/photo-1509062522246-3755977927d7?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['走班','单选物理','公开讨论改写']::TEXT[],
   'https://tieba.baidu.com/p/8706977461', '选科规定：物化双选为主，单选或面临诸多不便', 'public_discussion_rewritten', '经验帖', '已上架', 370, ARRAY['grade-eleven-timeline','physics-track-how-to-choose']::TEXT[]),

  ('tieba-bio-politics-choice', '期末后不拍板', 'student', '生物65、政治82.5，我会怎么拆这个选择题',
   $$看到贴吧里有同学拿生物65、政治82.5纠结物化生还是物化政，我会先暂停“按分数投票”。政治选择题阶段的高分，不一定等于主观题长期优势；生物完整卷低分，也要看是知识漏洞、实验题不会，还是纯粹没兴趣。我的拆法：问目标专业是否需要政治，问生物错题能否在4周内修复，问学校物化政班型是否稳定。来源：贴吧公开讨论改写。$$,
   'physics', ARRAY['chemistry','politics']::TEXT[], 'question', '高一', '江苏', 428, 211, interval '2 days 17 hours',
   ARRAY['https://images.unsplash.com/photo-1513258496099-48168024aec0?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['物化政','生物政治','公开讨论改写']::TEXT[],
   'https://tieba.baidu.com/p/10390633657', '选科怎么选？物化生和物化政怎么抉择？', 'public_discussion_rewritten', '提问', '已上架', 380, ARRAY['parents-decision-room','physics-track-how-to-choose']::TEXT[]),

  ('parent-meeting-no-rush', '慢半拍爸爸', 'parent', '家长会后别马上拍板，先做一张不能报清单',
   $$家长会回来最容易热血上头：老师说物化重要，就马上逼孩子选；同桌说史政地舒服，又开始动摇。我现在给自己定规则：任何选科讨论都先做“不能报清单”。把孩子可能想去的医学、计算机、师范、法学、财经、公安等方向逐个查，标出不满足要求的组合。等硬限制看清楚，再谈兴趣和分数。这个方法来自官方选考要求和很多家长讨论的综合整理。$$,
   'history', ARRAY['politics','geography']::TEXT[], 'experience', '高一', '湖北', 533, 267, interval '3 days',
   ARRAY['https://images.unsplash.com/photo-1522202176988-66273c2fd55f?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['家长会','不能报清单','决策流程']::TEXT[],
   'https://gaokao.chsi.com.cn/gkxx/zc/ss/202201/20220105/2155365943.html', '2024年各省市普通高校本科专业选考科目要求汇总', 'official_policy', '经验帖', '已上架', 390, ARRAY['parents-decision-room','major-requirement-checklist']::TEXT[]),

  ('senior-review-after-selection', '高二复盘本', 'student', '高二选完科后的30天，比纠结阶段更关键',
   $$选科落地以后，我发现真正难的是前30天。新班级、新老师、新节奏，很多人会把不适应误判成“选错了”。我的复盘表只有四列：每科最近一次错因、下周最小动作、需要问老师的问题、是否影响目标专业。先跑满30天再评价组合，不要用第一周的情绪否定半年的判断。资料来源：公开选科经验帖和各省选考要求整理。$$,
   'physics', ARRAY['chemistry','biology']::TEXT[], 'experience', '高二', '湖南', 376, 180, interval '3 days 4 hours',
   ARRAY['https://images.unsplash.com/photo-1517842645767-c639042777db?auto=format&fit=crop&w=1200&q=80']::TEXT[],
   ARRAY['高二复盘','学习方法','选科后']::TEXT[],
   'https://www.zhihu.com/question/521261507', '新高考如何选科？', 'public_discussion_rewritten', '经验帖', '已上架', 400, ARRAY['after-selection-score-up','grade-eleven-timeline']::TEXT[]);

INSERT INTO posts (author_name, author_role, title, content, track, electives, category, grade, province, likes_count, comments_count, favorites_count, image_urls, tags, created_at)
SELECT sp.author_name, sp.author_role, sp.title, sp.content, sp.track, sp.electives, sp.category, sp.grade, sp.province,
       sp.likes_count, 0, sp.favorites_count, sp.image_urls, sp.tags, now() - sp.created_offset
FROM seed_real_posts sp
WHERE NOT EXISTS (
  SELECT 1 FROM posts p WHERE p.title = sp.title AND p.deleted_at IS NULL
);

INSERT INTO topics (slug, title, summary, views_count, posts_count)
VALUES
  ('new-gaokao-policy-2026', '2026招生政策核对', '聚合2026高考招生、特殊类型、专项计划、报名资格和志愿填报政策提醒。', 9800, 0),
  ('major-requirement-checklist', '专业要求逐校核对', '把选考科目要求、院校专业组、当年招生计划和专业目录放到同一张表里核验。', 8700, 0),
  ('parents-decision-room', '家长选科决策室', '家长视角的家庭沟通、政策材料、不能报清单和选科会议复盘。', 6900, 0),
  ('medical-computer-track', '医学计算机与新专业', '围绕医学、计算机、新兴交叉专业和物理化学约束展开讨论。', 7200, 0),
  ('province-admission-notes', '各省招生政策笔记', '按省份整理高考综合改革、录取规则、专项计划和选考要求中的关键细节。', 7600, 0)
ON CONFLICT (slug) DO UPDATE
SET title = EXCLUDED.title,
    summary = EXCLUDED.summary,
    views_count = GREATEST(topics.views_count, EXCLUDED.views_count);

INSERT INTO topic_posts (topic_id, post_id)
SELECT t.id, p.id
FROM seed_real_posts sp
JOIN posts p ON p.title = sp.title AND p.deleted_at IS NULL
JOIN LATERAL unnest(sp.topic_slugs) AS topic_slug(slug) ON true
JOIN topics t ON t.slug = topic_slug.slug
ON CONFLICT DO NOTHING;

DROP TABLE IF EXISTS seed_real_comments;

CREATE TEMP TABLE seed_real_comments (
  post_title TEXT NOT NULL,
  author TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  created_offset INTERVAL NOT NULL
);

INSERT INTO seed_real_comments (post_title, author, role, content, created_offset)
VALUES
  ('教育部2026招生通知里，我最想提醒家长的3个字：公开性', '高三妈妈', 'parent', '收藏了，准备把阳光高考和本省考试院都放到浏览器书签第一排。', interval '25 minutes'),
  ('教育部2026招生通知里，我最想提醒家长的3个字：公开性', '陈老师', 'teacher', '公开来源很重要，尤其是政策类信息不要只看二手截图。', interval '18 minutes'),
  ('强基综评特殊类型别只看热闹，先看纪律线', '竞赛边缘人', 'student', '这条说到我了，之前只问降分，没认真看材料节点。', interval '48 minutes'),
  ('强基综评特殊类型别只看热闹，先看纪律线', '规划师阿青', 'counselor', '特殊类型可以准备，但一定要按招生简章和公示要求走。', interval '41 minutes'),
  ('北京2026高考：等级考三门怎么和录取绑在一起', '北京高一', 'student', '原来等级考不是只影响能不能报，还进总分，突然清醒。', interval '1 hour'),
  ('北京2026高考：等级考三门怎么和录取绑在一起', '西城家长', 'parent', '想看更多北京院校专业组的例子。', interval '52 minutes'),
  ('志愿填报密码这件小事，真的会变成大事', '海淀妈妈', 'parent', '截图归档这点太真实了，家里已经因为密码吵过一次。', interval '2 hours'),
  ('志愿填报密码这件小事，真的会变成大事', '班主任周', 'teacher', '建议最后提交前家长和学生各自独立核一遍志愿顺序。', interval '1 hour 50 minutes'),
  ('广东专项计划：户籍、学籍、实际就读一个都不能漏', '粤东考生', 'student', '我们学校已经开始提醒材料了，确实不能拖。', interval '3 hours'),
  ('广东高校专项2026改成什么流程？给家长画重点', '广州家长', 'parent', '以前习惯去高校官网找简章，今年看来要盯省里的目录。', interval '3 hours 20 minutes'),
  ('河南随迁子女政策提醒：别把材料拖到最后一周', '郑州家长', 'parent', '材料表这个办法好，很多东西临时补真的来不及。', interval '5 hours'),
  ('河南家长看专项计划：资格审核比想象中更细', '县中老师', 'teacher', '专项计划的资格和志愿是两条线，都得提前确认。', interval '6 hours'),
  ('山东2027选科要求：同专业不同学校真的会不一样', '青岛高一', 'student', '法学也要逐校看吗？我之前以为基本不限。', interval '7 hours'),
  ('山东2027选科要求：同专业不同学校真的会不一样', '升学规划师B', 'counselor', '对，同名专业不同培养方向会出现不同要求，要看具体学校。', interval '6 hours 40 minutes'),
  ('山东公告里一句话：要求能报，不代表当年一定招生', '烟台妈妈', 'parent', '“门后面有没有座位”这个比喻太扎心了。', interval '8 hours'),
  ('上海2027：院校专业组才是志愿里的真实单位', '浦东高二', 'student', '所以专业组里混了不想去的专业也要小心对吧？', interval '9 hours'),
  ('上海2027：院校专业组才是志愿里的真实单位', '沪上顾问', 'counselor', '是的，专业组内专业结构和调剂风险都要一起看。', interval '8 hours 50 minutes'),
  ('上海考生别只收藏目录，还要等当年计划', '上海家长', 'parent', '高一先定方向，高三再确认计划，这个节奏我能接受。', interval '10 hours'),
  ('江苏3+1+2：为什么我建议先把目标专业写出来', '苏州学生', 'student', '我先写了8个专业，发现有3个都卡化学。', interval '12 hours'),
  ('江苏选科查询，别用截图传来传去', '无锡老师', 'teacher', '截图误传太常见了，年份和省份经常被截掉。', interval '13 hours'),
  ('甘肃选科要求四类，最容易误解的是第三类', '兰州高一', 'student', '第三类真的容易看错，我以为只要化学就行。', interval '15 hours'),
  ('甘肃数据复盘：物化不是万能钥匙，但少了它很多门会关', '西北家长', 'parent', '孩子化学中等，看来不能只看专业覆盖，还要看能不能学下去。', interval '16 hours'),
  ('在浙江查计算机，我发现物化要求越来越硬', '杭州高一', 'student', '目标计算机，正在补化学，看到这条有压力但也更明确。', interval '18 hours'),
  ('在浙江查计算机，我发现物化要求越来越硬', '信息老师', 'teacher', '计算机不是只写代码，数学和理科基础都很关键。', interval '17 hours 40 minutes'),
  ('农学不等于只背书，浙江目录里很多也是物化', '喜欢植物的同学', 'student', '本来想躲化学去农学，看来还是要认真查。', interval '20 hours'),
  ('2026新增38个本科专业，别只被名字种草', '新专业控', 'student', '名字真的很容易上头，准备先查培养方案。', interval '22 hours'),
  ('具身智能听起来很酷，选科上我会先看数理底子', '机器人社团', 'student', '愿意调试设备这一点很真实，酷和累是一起的。', interval '23 hours'),
  ('数字金融不是财经轻松版，数学和计算能力要先问自己', '文科妈妈', 'parent', '孩子喜欢财经但怕数学，看来要换个角度聊。', interval '1 day'),
  ('脑机科学与技术：适合喜欢生物但也不怕工程的人吗？', '生物课代表', 'student', '我就是喜欢生物但怕代码，先不盲冲了。', interval '1 day 1 hour'),
  ('看了好多物化生讨论，我给自己列了三条冷静标准', '同款纠结', 'student', '连续三次考试相对位置这个比看一次分数靠谱。', interval '1 day 3 hours'),
  ('物化地不是低配物化生，我踩过的误区写下来', '地理满分梦', 'student', '地理材料题真的不是背书能解决的。', interval '1 day 5 hours'),
  ('物化政到底香不香？别只拿一场政治高分决定', '政治82本人', 'student', '被点醒了，我得看看主观题能不能稳。', interval '1 day 7 hours'),
  ('史政地家长最该接受的一件事：路径清楚但边界也清楚', '山东妈妈', 'parent', '这条可以拿给家里老人看，不是文科就没出路，也不是没有边界。', interval '1 day 9 hours'),
  ('单选物理或化学这件事，很多学校的排课成本也要算', '走班学生', 'student', '我们学校冷门组合真的要拼班，氛围影响很大。', interval '1 day 11 hours'),
  ('生物65、政治82.5，我会怎么拆这个选择题', '江苏高一', 'student', '我也是政治高生物低，准备按4周错题修复试一下。', interval '1 day 13 hours'),
  ('家长会后别马上拍板，先做一张不能报清单', '理性爸爸', 'parent', '不能报清单比争论兴趣有效多了。', interval '1 day 15 hours'),
  ('高二选完科后的30天，比纠结阶段更关键', '刚分班', 'student', '第一周确实很慌，决定先跑满一个月再评价。', interval '1 day 17 hours');

INSERT INTO comments (post_id, author, role, content, created_at)
SELECT p.id, sc.author, sc.role, sc.content, now() - sc.created_offset
FROM seed_real_comments sc
JOIN posts p ON p.title = sc.post_title AND p.deleted_at IS NULL
WHERE NOT EXISTS (
  SELECT 1
  FROM comments c
  WHERE c.post_id = p.id
    AND c.author = sc.author
    AND c.content = sc.content
    AND c.deleted_at IS NULL
);

UPDATE posts p
SET comments_count = counts.total,
    updated_at = GREATEST(p.updated_at, counts.latest_comment_at)
FROM (
  SELECT post_id, COUNT(*)::integer AS total, MAX(created_at) AS latest_comment_at
  FROM comments
  WHERE deleted_at IS NULL
  GROUP BY post_id
) counts
WHERE p.id = counts.post_id;

UPDATE posts p
SET comments_count = 0
WHERE NOT EXISTS (
  SELECT 1 FROM comments c WHERE c.post_id = p.id AND c.deleted_at IS NULL
);

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
SELECT 'post-' || sp.slug,
       'posts',
       sp.title,
       sp.content_type,
       sp.status,
       sp.province,
       sp.author_name,
       sp.tags,
       left(sp.content, 220),
       '/posts/' || p.id::text,
       CASE WHEN sp.status = '需复核' THEN '高' ELSE '常规' END,
       sp.sort_order,
       jsonb_build_object(
         'postId', p.id::text,
         'content', sp.content,
         'track', sp.track,
         'electives', sp.electives,
         'category', sp.category,
         'grade', sp.grade,
         'province', sp.province,
         'imageUrls', sp.image_urls,
         'sourceUrl', sp.source_url,
         'sourceTitle', sp.source_title,
         'sourceType', sp.source_type,
         'rewritten', true,
         'syncedFromMigration', '000008_enriched_real_source_content'
       )
FROM seed_real_posts sp
JOIN posts p ON p.title = sp.title AND p.deleted_at IS NULL
ON CONFLICT (id) DO UPDATE
SET module = EXCLUDED.module,
    title = EXCLUDED.title,
    content_type = EXCLUDED.content_type,
    status = EXCLUDED.status,
    scope = EXCLUDED.scope,
    owner = EXCLUDED.owner,
    tags = EXCLUDED.tags,
    summary = EXCLUDED.summary,
    url = EXCLUDED.url,
    priority = EXCLUDED.priority,
    sort_order = EXCLUDED.sort_order,
    payload = admin_content_records.payload || EXCLUDED.payload,
    deleted_at = NULL,
    updated_at = now();

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
SELECT 'post-existing-' || p.id::text,
       'posts',
       p.title,
       CASE p.category
         WHEN 'question' THEN '提问'
         WHEN 'data' THEN '数据建议'
         ELSE '经验帖'
       END,
       '已上架',
       p.province,
       p.author_name,
       p.tags,
       left(p.content, 220),
       '/posts/' || p.id::text,
       '常规',
       500 + p.id::integer,
       jsonb_build_object(
         'postId', p.id::text,
         'content', p.content,
         'track', p.track,
         'electives', p.electives,
         'category', p.category,
         'grade', p.grade,
         'province', p.province,
         'imageUrls', p.image_urls,
         'sourceType', 'legacy_seed',
         'syncedFromMigration', '000008_enriched_real_source_content'
       )
FROM posts p
WHERE p.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM admin_content_records acr
    WHERE acr.module = 'posts'
      AND acr.deleted_at IS NULL
      AND acr.payload->>'postId' = p.id::text
  )
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title,
    content_type = EXCLUDED.content_type,
    status = EXCLUDED.status,
    scope = EXCLUDED.scope,
    owner = EXCLUDED.owner,
    tags = EXCLUDED.tags,
    summary = EXCLUDED.summary,
    url = EXCLUDED.url,
    priority = EXCLUDED.priority,
    sort_order = EXCLUDED.sort_order,
    payload = admin_content_records.payload || EXCLUDED.payload,
    deleted_at = NULL,
    updated_at = now();

WITH linked_posts AS (
  SELECT acr.id, p.id AS post_id
  FROM admin_content_records acr
  JOIN posts p ON p.title = acr.title AND p.deleted_at IS NULL
  WHERE acr.module = 'posts'
    AND acr.deleted_at IS NULL
)
UPDATE admin_content_records acr
SET url = '/posts/' || linked_posts.post_id::text,
    payload = acr.payload || jsonb_build_object('postId', linked_posts.post_id::text),
    updated_at = now()
FROM linked_posts
WHERE acr.id = linked_posts.id;

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
VALUES
  ('policy-moe-2026-admission-work', 'policies', '教育部2026年普通高校招生工作通知', '官方来源', '已上架', '全国', '教育部', ARRAY['2026招生','考试安全','信息公开'], '教育部部署2026年普通高校招生工作，涉及考试安全、招生纪律、招生计划、专项计划和信息公开。', 'https://www.moe.gov.cn/srcsite/A15/moe_776/s3258/202601/t20260121_1427110.html', '高', 1, '{"publisher":"教育部","year":2026,"sourceType":"official_policy"}'),
  ('policy-moe-2026-special-types', 'policies', '2026年普通高校部分特殊类型招生工作通知', '官方来源', '已上架', '全国', '教育部', ARRAY['特殊类型招生','强基综评','招生纪律'], '适合强基、综评、艺术体育等特殊类型路径家庭核对资格、测试、录取和公示规则。', 'https://www.moe.gov.cn/srcsite/A15/moe_776/tslxzs/202510/t20251031_1418664.html', '高', 2, '{"publisher":"教育部","year":2026,"sourceType":"official_policy"}'),
  ('policy-moe-2026-undergrad-catalog', 'policies', '普通高等学校本科专业目录（2026年）', '官方来源', '已上架', '全国', '教育部', ARRAY['本科专业目录','新增专业','交叉学科'], '2026年本科专业目录公布，新增38种本科专业，并增设交叉学科门类。', 'https://www.moe.gov.cn/srcsite/A08/moe_1034/s3882/202604/t20260427_1434931.html', '高', 3, '{"publisher":"教育部","year":2026,"sourceType":"official_policy"}'),
  ('policy-beijing-2026-admission-rules', 'policies', '北京市2026年普通高等学校招生工作规定', '官方来源', '已上架', '北京', '北京教育考试院', ARRAY['北京高考','3+3','志愿填报'], '北京2026招生规定，含统一高考、学考等级考、志愿填报、录取和照顾政策。', 'https://www.bjeea.cn/html/gkgz/tzgg/2026/0505/88114.html', '高', 11, '{"publisher":"北京教育考试院","year":2026,"sourceType":"official_policy"}'),
  ('policy-guangdong-2026-special-plan', 'policies', '广东省2026年重点高校招生专项计划通知', '官方来源', '已上架', '广东', '广东省教育厅', ARRAY['广东高考','专项计划','资格审核'], '广东2026高校专项和地方专项计划报考条件、报名方式、资格审核和公示要求。', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202603/20260325/2293454298.html', '高', 12, '{"publisher":"广东省教育厅","year":2026,"sourceType":"official_policy"}'),
  ('policy-henan-2026-admission-rules', 'policies', '河南省2026年普通高等学校招生工作通知', '官方来源', '已上架', '河南', '河南省教育厅', ARRAY['河南高考','随迁子女','专项计划'], '河南2026招生工作通知，涉及区域公平、专项计划、随迁子女和报名资格审核。', 'https://jyt.henan.gov.cn/2026/04-27/3346226.html', '高', 13, '{"publisher":"河南省教育厅","year":2026,"sourceType":"official_policy"}'),
  ('policy-shandong-2027-subject-requirements', 'policies', '山东2024/2027通用版专业选考科目要求公告', '官方来源', '已上架', '山东', '山东省教育招生考试院', ARRAY['山东选科','2027要求','逐校核对'], '山东公布通用版专业选考科目要求，并提醒同一专业不同高校要求可能不同。', 'https://www.sdzk.cn/NewsInfo.aspx?NewsID=6819', '高', 14, '{"publisher":"山东省教育招生考试院","year":2027,"sourceType":"official_policy"}'),
  ('policy-shanghai-2027-subject-requirements', 'policies', '上海2027年普通高校本科专业选考科目要求', '官方来源', '已上架', '上海', '上海市教育考试院', ARRAY['上海高考','院校专业组','2027要求'], '上海2027年本科专业选考科目要求说明，强调院校专业组和当年招生目录。', 'https://www.shmeea.edu.cn/page/02200/20250221/19114.html', '高', 15, '{"publisher":"上海市教育考试院","year":2027,"sourceType":"official_policy"}'),
  ('policy-jiangsu-2024-subject-requirements', 'policies', '江苏2024年拟在苏招生本科专业选考科目要求公告', '官方来源', '已上架', '江苏', '江苏省教育考试院', ARRAY['江苏选科','3+1+2','专业要求'], '江苏2024拟在苏招生本科专业选考科目要求说明，供选科和志愿核对使用。', 'https://www.jseea.cn/webfile/index/index_zkxx/2022-01-18/27031.html', '常规', 16, '{"publisher":"江苏省教育考试院","year":2024,"sourceType":"official_policy"}'),
  ('policy-gansu-2024-subject-requirements', 'policies', '甘肃2024年普通高校招生专业选考科目要求', '官方来源', '已上架', '甘肃', '阳光高考/甘肃省教育考试院', ARRAY['甘肃选科','3+1+2','物理化学'], '甘肃2024选考科目要求，说明不提要求、首选要求、再选要求和组合要求等类型。', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202111/20211117/2132710994.html', '常规', 17, '{"publisher":"阳光高考/甘肃省教育考试院","year":2024,"sourceType":"official_policy"}')
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title,
    content_type = EXCLUDED.content_type,
    status = EXCLUDED.status,
    scope = EXCLUDED.scope,
    owner = EXCLUDED.owner,
    tags = EXCLUDED.tags,
    summary = EXCLUDED.summary,
    url = EXCLUDED.url,
    priority = EXCLUDED.priority,
    sort_order = EXCLUDED.sort_order,
    payload = admin_content_records.payload || EXCLUDED.payload,
    deleted_at = NULL,
    updated_at = now();

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
VALUES
  ('requirement-embodied-ai', 'requirements', '具身智能', '物理+化学强约束', '需复核', '全国', '专业要求库', ARRAY['2026新增专业','人工智能','物理','化学'], '新设交叉专业，建议重点核对首批高校培养方案、授予学位和物理化学要求。', '/requirements/具身智能', '高', 101, '{"sourceUrl":"https://www.moe.gov.cn/srcsite/A08/moe_1034/s3882/202604/t20260427_1434931.html","sourceType":"official_policy"}'),
  ('requirement-brain-computer-science', 'requirements', '脑机科学与技术', '物理+化学强约束', '需复核', '全国', '专业要求库', ARRAY['2026新增专业','医工交叉','物化生'], '医工交叉倾向明显，适合同时关注生命科学、工程技术和计算能力的学生继续核对。', '/requirements/脑机科学与技术', '高', 102, '{"sourceUrl":"https://www.moe.gov.cn/srcsite/A08/moe_1034/s3882/202604/t20260427_1434931.html","sourceType":"official_policy"}'),
  ('requirement-digital-finance', 'requirements', '数字金融', '需逐校核对', '已上架', '全国', '专业要求库', ARRAY['2026新增专业','财经','数据能力'], '不要按传统财经简单理解，建议核对数学、统计、计算机课程占比和院校要求。', '/requirements/数字金融', '常规', 103, '{"sourceUrl":"https://www.moe.gov.cn/jyb_xwfb/gzdt_gzdt/s5987/202604/t20260428_1435016.html","sourceType":"official_policy"}'),
  ('requirement-low-altitude-safety', 'requirements', '低空安全管理', '需逐校核对', '需复核', '全国', '专业要求库', ARRAY['2026新增专业','低空经济','管理交叉'], '关注低空经济相关新专业时，需要同步核对培养单位、课程结构和选考科目。', '/requirements/低空安全管理', '中', 104, '{"sourceUrl":"https://www.moe.gov.cn/srcsite/A08/moe_1034/s3882/202604/t20260427_1434931.html","sourceType":"official_policy"}'),
  ('requirement-agricultural-robotics', 'requirements', '农业机器人', '物理+化学强约束', '需复核', '全国', '专业要求库', ARRAY['2026新增专业','农业工程','机器人'], '农业机器人属于工农交叉方向，不能只按传统农学或传统机械单线理解。', '/requirements/农业机器人', '高', 105, '{"sourceUrl":"https://www.moe.gov.cn/srcsite/A08/moe_1034/s3882/202604/t20260427_1434931.html","sourceType":"official_policy"}'),
  ('requirement-public-security-politics', 'requirements', '公安学类', '政治强相关', '已上架', '全国', '专业要求库', ARRAY['政治','公安','逐校核对'], '公安、警校相关专业常与政治要求、体检体测和政审等条件叠加，需逐校逐年核对。', '/requirements/公安学类', '高', 106, '{"sourceType":"policy_note"}')
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title,
    content_type = EXCLUDED.content_type,
    status = EXCLUDED.status,
    scope = EXCLUDED.scope,
    owner = EXCLUDED.owner,
    tags = EXCLUDED.tags,
    summary = EXCLUDED.summary,
    url = EXCLUDED.url,
    priority = EXCLUDED.priority,
    sort_order = EXCLUDED.sort_order,
    payload = admin_content_records.payload || EXCLUDED.payload,
    deleted_at = NULL,
    updated_at = now();

INSERT INTO subject_insights (combination, trend, heat, match_rate, advice, details)
VALUES
  ('物理 + 化学 + 政治', '专业覆盖高，学科跨度大', 84, 82.80, '适合数理基础稳定、确有公安法学或公共事务兴趣的学生。', '物化政保留大量理工方向，同时给政治相关路径留下空间。风险在于物理、化学、政治思维差异较大，不能只凭一次政治高分决定。'),
  ('物理 + 政治 + 地理', '理工受限，人文公共方向更清晰', 52, 58.60, '适合物理尚可但化学长期吃力，且目标偏公共管理、法政或地理相关方向的学生。', '缺化学会影响大量理工农医专业，选择前要重点核对目标专业是否要求物理+化学。'),
  ('物理 + 生物 + 政治', '跨向组合，需逐校核对', 49, 56.20, '适合希望保留部分理科与公共事务方向，但不以传统工科医学为主目标的学生。', '缺化学会限制部分医学、药学、工科和农学方向，不能只看物理一科。'),
  ('历史 + 政治 + 生物', '人文社科为主，带一点生命科学兴趣', 46, 52.40, '适合历史政治表达较强、生物兴趣稳定但不追求强医学路径的学生。', '该组合专业边界较明确，适合先确认能接受排除理工农医大方向。'),
  ('历史 + 化学 + 生物', '小众交叉，医学想象需谨慎', 43, 62.30, '需要重点核对目标院校专业组选科要求，避免只凭兴趣判断。', '历史首选叠加化学和生物的组合较小众，部分专业可能仍受首选科目限制。'),
  ('历史 + 生物 + 地理', '压力较均衡，专业弹性有限', 48, 55.70, '适合历史方向明确，且喜欢生命、地理材料分析的学生。', '适合人文、教育、管理、部分地理与生态相关想象，但强理工医学方向要谨慎。')
ON CONFLICT (combination) DO UPDATE
SET trend = EXCLUDED.trend,
    heat = EXCLUDED.heat,
    match_rate = EXCLUDED.match_rate,
    advice = EXCLUDED.advice,
    details = EXCLUDED.details,
    updated_at = now();

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
SELECT 'insight-' || replace(replace(replace(combination, ' + ', '-'), '物理', 'physics'), '历史', 'history'),
       'insights',
       combination,
       '组合趋势',
       '已上架',
       '全国',
       '数据运营',
       ARRAY['热度 ' || heat::text, '匹配 ' || match_rate::text || '%'],
       trend || '。' || advice,
       '/insights/' || id::text,
       '常规',
       100 + id::integer,
       jsonb_build_object('insightId', id::text, 'heat', heat, 'matchRate', match_rate)
FROM subject_insights
WHERE combination IN ('物理 + 化学 + 政治', '物理 + 政治 + 地理', '物理 + 生物 + 政治', '历史 + 政治 + 生物', '历史 + 化学 + 生物', '历史 + 生物 + 地理')
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title,
    content_type = EXCLUDED.content_type,
    status = EXCLUDED.status,
    tags = EXCLUDED.tags,
    summary = EXCLUDED.summary,
    url = EXCLUDED.url,
    payload = admin_content_records.payload || EXCLUDED.payload,
    deleted_at = NULL,
    updated_at = now();

UPDATE topics
SET posts_count = COALESCE(counts.total, 0)
FROM (
  SELECT t.id, COUNT(p.id)::integer AS total
  FROM topics t
  LEFT JOIN topic_posts tp ON tp.topic_id = t.id
  LEFT JOIN posts p ON p.id = tp.post_id AND p.deleted_at IS NULL
  GROUP BY t.id
) counts
WHERE topics.id = counts.id;

DROP TABLE IF EXISTS seed_real_comments;
DROP TABLE IF EXISTS seed_real_posts;
