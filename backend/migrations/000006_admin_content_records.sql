CREATE TABLE IF NOT EXISTS admin_content_records (
  id VARCHAR(120) PRIMARY KEY,
  module VARCHAR(40) NOT NULL,
  title VARCHAR(160) NOT NULL,
  content_type VARCHAR(80) NOT NULL DEFAULT '',
  status VARCHAR(30) NOT NULL DEFAULT '草稿',
  scope VARCHAR(80) NOT NULL DEFAULT '全国',
  owner VARCHAR(120) NOT NULL DEFAULT '',
  tags TEXT[] NOT NULL DEFAULT '{}',
  summary TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  priority VARCHAR(20) NOT NULL DEFAULT '常规',
  sort_order INTEGER NOT NULL DEFAULT 0,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_admin_content_module ON admin_content_records (module, sort_order, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_admin_content_status ON admin_content_records (status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_admin_content_tags ON admin_content_records USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_admin_content_payload ON admin_content_records USING GIN(payload);

CREATE TABLE IF NOT EXISTS admin_audit_logs (
  id BIGSERIAL PRIMARY KEY,
  action VARCHAR(80) NOT NULL,
  record_id VARCHAR(120),
  module VARCHAR(40),
  detail TEXT NOT NULL DEFAULT '',
  actor VARCHAR(80) NOT NULL DEFAULT 'admin',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_created_at ON admin_audit_logs (created_at DESC);

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
VALUES
  ('category-experience', 'categories', '经验帖', '帖子分类', '已上架', '全站', '内容运营', ARRAY['首页可见','社区内容'], '学生、老师、规划师的真实经验分享。', '', '常规', 10, '{"frontendRoute":"/","category":"experience"}'),
  ('category-question', 'categories', '提问', '帖子分类', '已上架', '全站', '内容运营', ARRAY['问答','社区内容'], '家长、学生围绕选科风险和专业限制提出问题。', '', '常规', 20, '{"frontendRoute":"/","category":"question"}'),
  ('category-data', 'categories', '数据建议', '帖子分类', '已上架', '全站', '内容运营', ARRAY['数据','专业覆盖'], '政策、专业覆盖、趋势分析类内容。', '', '常规', 30, '{"frontendRoute":"/","category":"data"}'),
  ('nav-requirements', 'categories', '选科查询', '工具入口', '已上架', '全站', '产品运营', ARRAY['专业要求','工具'], '按专业和组合查询选科约束。', '/requirements', '高', 40, '{"frontendRoute":"/requirements"}'),
  ('nav-knowledge', 'categories', '政策库', '主导航入口', '已上架', '全站', '产品运营', ARRAY['官方来源','省份资料'], '省级考试院入口和政策核对资料包。', '/knowledge', '高', 50, '{"frontendRoute":"/knowledge"}'),
  ('nav-advice', 'categories', '建议库', '内容入口', '已上架', '全站', '产品运营', ARRAY['精选笔记','行动建议'], '围绕选科判断、家长沟通、专业核对的结构化建议。', '/advice', '常规', 60, '{"frontendRoute":"/advice"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
VALUES
  ('policy-national-admission', 'policies', '2026年高考各省市招生政策及照顾政策汇总', '官方来源', '已上架', '全国', '阳光高考/学信网', ARRAY['招生政策','照顾政策','官方来源'], '集中查看各省市招生工作规定、实施办法和照顾政策。', 'https://gaokao.chsi.com.cn/z/gkbmfslq/zszc.jsp', '高', 10, '{"publisher":"阳光高考/学信网","reviewCycle":"月度"}'),
  ('policy-national-major-subject', 'policies', '普通高校本科专业选考科目要求汇总', '官方来源', '已上架', '全国', '阳光高考/学信网', ARRAY['选考要求','专业目录','官方来源'], '用于核对不同省份、院校、专业组的选考科目限制。', 'https://gaokao.chsi.com.cn/gkxx/zc/ss/202201/20220105/2155365943.html', '高', 20, '{"publisher":"阳光高考/学信网","reviewCycle":"季度"}'),
  ('policy-national-zyck', 'policies', '阳光志愿信息服务系统', '官方来源', '已上架', '全国', '阳光高考/学信网', ARRAY['志愿填报','高校信息','官方来源'], '整合高校、专业、招生政策和志愿参考信息。', 'https://gaokao.chsi.com.cn/zyck/', '高', 30, '{"publisher":"阳光高考/学信网","reviewCycle":"季度"}'),
  ('policy-beijing', 'policies', '北京省份资料包', '华北', '已上架', '北京', '北京教育考试院', ARRAY['3+3','华北'], '已实施新高考，关注等级考与院校专业组选考要求。', 'https://www.bjeea.cn/', '常规', 101, '{"mode":"3+3"}'),
  ('policy-tianjin', 'policies', '天津省份资料包', '华北', '已上架', '天津', '天津市教育招生考试院', ARRAY['3+3','华北'], '已实施新高考，选科需结合专业组选考要求。', 'https://www.zhaokao.net/', '常规', 102, '{"mode":"3+3"}'),
  ('policy-hebei', 'policies', '河北省份资料包', '华北', '已上架', '河北', '河北省教育考试院', ARRAY['3+1+2','华北'], '已实施“3+1+2”，首选物理/历史影响专业组范围。', 'https://www.hebeea.edu.cn/', '常规', 103, '{"mode":"3+1+2"}'),
  ('policy-shanghai', 'policies', '上海省份资料包', '华东', '已上架', '上海', '上海市教育考试院', ARRAY['3+3','华东'], '已实施“3+3”，院校专业组和选科要求需一起看。', 'https://www.shmeea.edu.cn/', '常规', 109, '{"mode":"3+3"}'),
  ('policy-jiangsu', 'policies', '江苏省份资料包', '华东', '已上架', '江苏', '江苏省教育考试院', ARRAY['3+1+2','华东'], '已实施“3+1+2”。', 'https://www.jseea.cn/', '常规', 110, '{"mode":"3+1+2"}'),
  ('policy-zhejiang', 'policies', '浙江省份资料包', '华东', '已上架', '浙江', '浙江省教育考试院', ARRAY['3+3','华东'], '已实施“3+3”，技术科目与选考目录需特别关注。', 'https://www.zjzs.net/', '常规', 111, '{"mode":"3+3"}'),
  ('policy-guangdong', 'policies', '广东省份资料包', '华南', '已上架', '广东', '广东省教育考试院', ARRAY['3+1+2','华南'], '已实施“3+1+2”。', 'https://eea.gd.gov.cn/', '常规', 119, '{"mode":"3+1+2"}'),
  ('policy-sichuan', 'policies', '四川省份资料包', '西南', '已上架', '四川', '四川省教育考试院', ARRAY['3+1+2','西南'], '第五批改革省份，2025年起新高考落地。', 'https://www.sceea.cn/', '需复核', 123, '{"mode":"3+1+2"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
VALUES
  ('policy-shanxi', 'policies', '山西省份资料包', '华北', '已上架', '山西', '山西省招生考试管理中心', ARRAY['3+1+2','华北'], '第五批改革省份，2025年起新高考落地。', 'https://www.sxkszx.cn/', '需复核', 104, '{"mode":"3+1+2"}'),
  ('policy-neimenggu', 'policies', '内蒙古省份资料包', '华北', '已上架', '内蒙古', '内蒙古自治区教育招生考试中心', ARRAY['3+1+2','华北'], '第五批改革省份，2025年起新高考落地。', 'https://www.nm.zsks.cn/', '需复核', 105, '{"mode":"3+1+2"}'),
  ('policy-liaoning', 'policies', '辽宁省份资料包', '东北', '已上架', '辽宁', '辽宁省招生考试办公室', ARRAY['3+1+2','东北'], '已实施“3+1+2”。', 'https://www.lnzsks.com/', '常规', 106, '{"mode":"3+1+2"}'),
  ('policy-jilin', 'policies', '吉林省份资料包', '东北', '已上架', '吉林', '吉林省教育考试院', ARRAY['3+1+2','东北'], '第四批改革省份，2024年新高考首考。', 'https://www.jleea.com.cn/', '常规', 107, '{"mode":"3+1+2"}'),
  ('policy-heilongjiang', 'policies', '黑龙江省份资料包', '东北', '已上架', '黑龙江', '黑龙江省招生考试院', ARRAY['3+1+2','东北'], '第四批改革省份，2024年新高考首考。', 'https://www.lzk.hl.cn/', '常规', 108, '{"mode":"3+1+2"}'),
  ('policy-anhui', 'policies', '安徽省份资料包', '华东', '已上架', '安徽', '安徽省教育招生考试院', ARRAY['3+1+2','华东'], '第四批改革省份，2024年新高考首考。', 'https://www.ahzsks.cn/', '常规', 112, '{"mode":"3+1+2"}'),
  ('policy-fujian', 'policies', '福建省份资料包', '华东', '已上架', '福建', '福建省教育考试院', ARRAY['3+1+2','华东'], '已实施“3+1+2”。', 'https://www.eeafj.cn/', '常规', 113, '{"mode":"3+1+2"}'),
  ('policy-jiangxi', 'policies', '江西省份资料包', '华东', '已上架', '江西', '江西省教育考试院', ARRAY['3+1+2','华东'], '第四批改革省份，2024年新高考首考。', 'https://www.jxeea.cn/', '常规', 114, '{"mode":"3+1+2"}'),
  ('policy-shandong', 'policies', '山东省份资料包', '华东', '已上架', '山东', '山东省教育招生考试院', ARRAY['3+3','华东'], '已实施“3+3”，选考科目要求按专业核对。', 'https://www.sdzk.cn/', '常规', 115, '{"mode":"3+3"}'),
  ('policy-henan', 'policies', '河南省份资料包', '华中', '已上架', '河南', '河南省教育考试院', ARRAY['3+1+2','华中'], '第五批改革省份，2025年起新高考落地。', 'https://www.haeea.cn/', '需复核', 116, '{"mode":"3+1+2"}'),
  ('policy-hubei', 'policies', '湖北省份资料包', '华中', '已上架', '湖北', '湖北省教育考试院', ARRAY['3+1+2','华中'], '已实施“3+1+2”。', 'https://www.hbea.edu.cn/', '常规', 117, '{"mode":"3+1+2"}'),
  ('policy-hunan', 'policies', '湖南省份资料包', '华中', '已上架', '湖南', '湖南省教育考试院', ARRAY['3+1+2','华中'], '已实施“3+1+2”。', 'https://jyt.hunan.gov.cn/jyt/sjyt/hnsjyksy/', '常规', 118, '{"mode":"3+1+2"}'),
  ('policy-guangxi', 'policies', '广西省份资料包', '华南', '已上架', '广西', '广西壮族自治区招生考试院', ARRAY['3+1+2','华南'], '第四批改革省份，2024年新高考首考。', 'https://www.gxeea.cn/', '常规', 120, '{"mode":"3+1+2"}'),
  ('policy-hainan', 'policies', '海南省份资料包', '华南', '已上架', '海南', '海南省考试局', ARRAY['3+3','华南'], '已实施新高考。', 'https://ea.hainan.gov.cn/', '常规', 121, '{"mode":"3+3"}'),
  ('policy-chongqing', 'policies', '重庆省份资料包', '西南', '已上架', '重庆', '重庆市教育考试院', ARRAY['3+1+2','西南'], '已实施“3+1+2”。', 'https://www.cqksy.cn/', '常规', 122, '{"mode":"3+1+2"}'),
  ('policy-guizhou', 'policies', '贵州省份资料包', '西南', '已上架', '贵州', '贵州省招生考试院', ARRAY['3+1+2','西南'], '第四批改革省份，2024年新高考首考。', 'https://zsksy.guizhou.gov.cn/', '常规', 124, '{"mode":"3+1+2"}'),
  ('policy-yunnan', 'policies', '云南省份资料包', '西南', '已上架', '云南', '云南省招生考试院', ARRAY['3+1+2','西南'], '第五批改革省份，2025年起新高考落地。', 'https://www.ynzs.cn/', '需复核', 125, '{"mode":"3+1+2"}'),
  ('policy-xizang', 'policies', '西藏省份资料包', '西南', '需复核', '西藏', '西藏自治区教育考试院', ARRAY['traditional','西南'], '暂未纳入已落地新高考省份，需以自治区最新公告为准。', 'https://zsks.edu.xizang.gov.cn/', '高', 126, '{"mode":"traditional"}'),
  ('policy-shaanxi', 'policies', '陕西省份资料包', '西北', '已上架', '陕西', '陕西省教育考试院', ARRAY['3+1+2','西北'], '第五批改革省份，2025年起新高考落地。', 'https://www.sneac.com/', '需复核', 127, '{"mode":"3+1+2"}'),
  ('policy-gansu', 'policies', '甘肃省份资料包', '西北', '已上架', '甘肃', '甘肃省教育考试院', ARRAY['3+1+2','西北'], '第四批改革省份，2024年新高考首考。', 'https://www.ganseea.cn/', '常规', 128, '{"mode":"3+1+2"}'),
  ('policy-qinghai', 'policies', '青海省份资料包', '西北', '已上架', '青海', '青海省教育招生考试院', ARRAY['3+1+2','西北'], '第五批改革省份，2025年起新高考落地。', 'https://www.qhjyks.com/', '需复核', 129, '{"mode":"3+1+2"}'),
  ('policy-ningxia', 'policies', '宁夏省份资料包', '西北', '已上架', '宁夏', '宁夏教育考试院', ARRAY['3+1+2','西北'], '第五批改革省份，2025年起新高考落地。', 'https://www.nxjyks.cn/', '需复核', 130, '{"mode":"3+1+2"}'),
  ('policy-xinjiang', 'policies', '新疆省份资料包', '西北', '需复核', '新疆', '新疆维吾尔自治区教育考试院', ARRAY['traditional','西北'], '暂未纳入已落地新高考省份，需以自治区最新公告为准。', 'https://www.xjzk.gov.cn/', '高', 131, '{"mode":"traditional"}'),
  ('policy-hongkong', 'policies', '香港省份资料包', '港澳台', '已上架', '香港', '香港考试及评核局 / 港澳台招生信息网', ARRAY['special','港澳台'], '适用港澳台侨联招、DSE 及高校面向港澳台招生政策。', 'https://www.hkeaa.edu.hk/tc/ipe/jee/', '常规', 132, '{"mode":"special"}'),
  ('policy-macau', 'policies', '澳门省份资料包', '港澳台', '已上架', '澳门', '澳门教育及青年发展局 / 港澳台招生信息网', ARRAY['special','港澳台'], '适用港澳台侨联招澳门考区及高校面向港澳台招生渠道。', 'https://www.gov.mo/zh-hant/services/ps-1717/ps-1717a/', '常规', 133, '{"mode":"special"}'),
  ('policy-taiwan', 'policies', '台湾省份资料包', '港澳台', '已上架', '台湾', '港澳台招生信息网 / 普通高校联合招生办公室', ARRAY['special','港澳台'], '适用港澳台招生信息网、全国联招及高校面向台湾学生招生政策。', 'https://www.gatzs.com.cn/', '常规', 134, '{"mode":"special"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
VALUES
  ('post-question-physics-chemistry-biology', 'posts', '物化生适合目标不太明确的人吗？', '经验/问答', '已上架', '浙江', '小周同学', ARRAY['物化生','专业覆盖'], '我现在数学和物理还可以，想听听大家对物化生后续专业覆盖和学习强度的真实感受。', '/posts/1', '常规', 10, '{"category":"question","track":"physics"}'),
  ('post-parent-history-risk', 'posts', '孩子想选史政地，家长应该怎么判断风险？', '家长提问', '待审核', '山东', '林妈妈', ARRAY['史政地','风险核对'], '孩子文科表达不错，但担心专业选择变窄。', '/posts/2', '中', 20, '{"category":"question","track":"history"}'),
  ('post-data-coverage-2026', 'posts', '2026届各组合专业覆盖率汇总', '数据建议', '需复核', '全国', '选科研究所', ARRAY['数据建议','专业覆盖率'], '基于多省教育考试院公开数据整理。', '/posts/5', '高', 30, '{"category":"data","track":"physics"}'),
  ('requirement-clinical-medicine', 'requirements', '临床医学', '物理+化学强约束', '已上架', '全国', '专业要求库', ARRAY['物理','化学','医学'], '临床医学通常对物理、化学要求较强，需结合目标省份目录、院校专业组和招生章程核对。', '/requirements/临床医学', '高', 10, '{"category":"医学"}'),
  ('requirement-computer-science', 'requirements', '计算机科学与技术', '物理+化学强约束', '已上架', '全国', '专业要求库', ARRAY['物理','化学','计算机'], '计算机类专业在多数新高考省份对物理、化学有较强要求。', '/requirements/计算机科学与技术', '高', 20, '{"category":"计算机与电子信息"}'),
  ('requirement-law', 'requirements', '法学', '需逐校核对', '已上架', '全国', '专业要求库', ARRAY['不限或按院校','人文社科'], '法学通常限制较少，但强校和特定培养方向仍需逐校核对。', '/requirements/法学', '常规', 30, '{"category":"法学与公共管理"}'),
  ('insight-physics-chemistry-biology', 'insights', '物理 + 化学 + 生物', '组合趋势', '已上架', '全国', '数据运营', ARRAY['热度 96','覆盖 96.2%'], '专业覆盖高，学习强度高。', '/insights/1', '常规', 10, '{"heat":96,"coverage":96.2}'),
  ('insight-history-politics-geography', 'insights', '历史 + 政治 + 地理', '组合趋势', '待审核', '全国', '数据运营', ARRAY['热度 81','覆盖 51.2%'], '人文社科清晰，专业边界明确。', '/insights/4', '中', 20, '{"heat":81,"coverage":51.2}'),
  ('advice-note-audit-major', 'advice', '先列不能报考清单，再谈兴趣', '精选建议', '已上架', '全国', '升学规划组', ARRAY['专业限制','家长沟通'], '把硬性选科限制先排清，再讨论兴趣、学习强度和长期目标。', '/advice/audit-major-fit', '常规', 10, '{"cover":"blue"}'),
  ('advice-note-family-meeting', 'advice', '一次家庭选科会议的结构', '精选建议', '已上架', '全国', '升学规划组', ARRAY['沟通','决策流程'], '用事实、选择、风险、下一步四段式降低亲子沟通成本。', '/advice/family-meeting', '常规', 20, '{"cover":"amber"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO admin_content_records (id, module, title, content_type, status, scope, owner, tags, summary, url, priority, sort_order, payload)
VALUES
  ('user-teacher-review', 'users', '陈老师', '老师', '认证中', '广东', '认证系统', ARRAY['教师认证','内容作者'], 'teacher@example.com', '', '中', 30, '{"certification":"teacher","materials":["教师资格证","学校/机构证明"],"permissions":["comment"]}'),
  ('user-counselor-normal', 'users', '升学规划师A', '规划师', '正常', '全国', '认证系统', ARRAY['规划师认证'], 'counselor@example.com', '', '常规', 40, '{"certification":"counselor","materials":["从业证明"],"permissions":["post","comment","review_suggestion"]}')
ON CONFLICT (id) DO NOTHING;
