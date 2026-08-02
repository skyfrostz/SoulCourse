import { useEffect, useRef, useState } from 'react'
import { ArrowUpRight, Menu, Play, X } from 'lucide-react'

const VIDEO_URL =
  'https://d8j0ntlcm91z4.cloudfront.net/user_38xzZboKViGWJOttwIXH07lWA1P/hf_20260328_083109_283f3553-e28f-428b-a723-d639c617eb2b.mp4'
const VIDEO_FALLBACK_URL =
  'https://soulcourse-prod-2026.oss-cn-shenzhen.aliyuncs.com/welcome/hero.mp4'
const ANDROID_DOWNLOAD_URL = '/downloads/android/1.0.0/subject-choice-1.0.0.apk'

const navItems = [
  { label: '选科社区', href: '/community' },
  { label: '选科查询', href: '/requirements' },
  { label: '政策库', href: '/knowledge' },
]

const posts = [
  { id: 89, title: '物化生适合目标不太明确的人吗？', author: '小周同学', tag: '物化生', meta: '128 赞 · 3 评论', tone: 'sage', image: '/welcome/post-images/subject-choice-student.jpg' },
  { id: 39, title: '从最近三届学生看物化地的优劣势', author: '陈老师', tag: '物化地', meta: '212 赞 · 4 评论', tone: 'sky', image: '/welcome/post-images/subject-choice-desk.jpg' },
  { id: 14, title: '选科前做一张纸：专业、成绩、成本', author: '林妈妈', tag: '选科决策', meta: '96 赞 · 2 评论', tone: 'sand', image: '/welcome/post-images/family-discussion.jpg' },
  { id: 11, title: '2026 拟在粤招生选科要求，应该怎么看', author: '选科研究所', tag: '政策核对', meta: '数据建议 · 已复核', tone: 'mint', image: '/welcome/post-images/source-check.jpg' },
]

const capabilities = [
  {
    index: '01',
    title: '从真实经验开始',
    body: '把学生、家长和老师的经验放在同一张桌面上，先看别人如何走过，再判断什么适合自己。',
    link: '浏览选科社区',
    href: '/',
    type: 'feed',
  },
  {
    index: '02',
    title: '查清专业要求',
    body: '按专业、科目和方向筛选，快速找到需要重点核对的选科限制与组合。',
    link: '查看专业目录',
    href: '/requirements',
    type: 'requirements',
  },
  {
    index: '03',
    title: '回到官方来源',
    body: '每条政策和数据建议都保留来源、年份与复核状态，结论可以回到原始依据。',
    link: '查看政策资料',
    href: '/knowledge',
    type: 'source',
  },
]

function App() {
  const videoRef = useRef<HTMLVideoElement>(null)
  const [menuOpen, setMenuOpen] = useState(false)
  const [videoFailed, setVideoFailed] = useState(false)
  const [videoReady, setVideoReady] = useState(false)
  const [videoSrc, setVideoSrc] = useState(VIDEO_URL)

  useEffect(() => {
    const video = videoRef.current
    if (!video) return
    const mediaPreference = window.matchMedia('(prefers-reduced-motion: reduce)')
    const syncMotion = () => {
      if (mediaPreference.matches) {
        video.pause()
        return
      }
      void video.play().catch(() => setVideoFailed(true))
    }
    syncMotion()
    mediaPreference.addEventListener('change', syncMotion)
    return () => mediaPreference.removeEventListener('change', syncMotion)
  }, [])

  return (
    <main className="welcome-page">
      <section className="hero" id="home">
        <div className="hero-media" aria-hidden="true">
          {!videoFailed && (
            <video
              ref={videoRef}
              className={`hero-video ${videoReady ? 'is-ready' : ''}`}
              src={videoSrc}
              muted
              autoPlay
              loop
              playsInline
              preload="metadata"
              onCanPlay={() => setVideoReady(true)}
              onError={() => {
                if (videoSrc !== VIDEO_FALLBACK_URL) {
                  setVideoReady(false)
                  setVideoSrc(VIDEO_FALLBACK_URL)
                } else {
                  setVideoFailed(true)
                }
              }}
            />
          )}
          <div className="hero-fallback" />
          <div className="hero-wash" />
        </div>

        <header className="welcome-header">
          <a className="welcome-brand" href="/welcome" aria-label="SoulCourse，欢迎页">
            <img className="welcome-brand-logo" src="/welcome/soulcourse-logo.jpeg" alt="SoulCourse 开源在线教育系统" />
          </a>
          <nav className="welcome-nav" aria-label="欢迎页导航">
            {navItems.map((item) => <a key={item.href} href={item.href}>{item.label}</a>)}
          </nav>
          <a className="text-link header-download" href={ANDROID_DOWNLOAD_URL} download>下载 Android App <ArrowUpRight size={15} /></a>
          <a className="button button-dark header-cta" href="/community">进入社区 <ArrowUpRight size={15} /></a>
          <button className="menu-button" type="button" aria-label={menuOpen ? '关闭菜单' : '打开菜单'} aria-expanded={menuOpen} onClick={() => setMenuOpen((open) => !open)}>
            {menuOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
        </header>

        <div className={`mobile-menu ${menuOpen ? 'is-open' : ''}`}>
          {navItems.map((item) => <a key={item.href} href={item.href} onClick={() => setMenuOpen(false)}>{item.label}<ArrowUpRight size={16} /></a>)}
        </div>

        <div className="hero-copy">
          <p className="eyebrow">广东新高考 · 选科知谈</p>
          <h1>选科知谈</h1>
          <p className="hero-statement">穿过信息迷雾，找到有据可依的选择。</p>
          <div className="hero-actions">
            <a className="button button-dark" href="/community">进入选科社区 <ArrowUpRight size={16} /></a>
            <a className="text-link" href="/requirements">查询专业要求 <ArrowUpRight size={15} /></a>
          </div>
        </div>

        <a className="scroll-cue" href="#evidence"><span>向下了解</span><i /></a>
      </section>

      <section className="intro-section section-shell" id="evidence">
        <div className="section-kicker"><span>选择之前</span><span>把依据放到一起</span></div>
        <div className="intro-grid">
          <h2>先看事实，<br /><em>再听经验。</em></h2>
          <p>选科不是寻找一个标准答案。我们把专业要求、政策来源和真实讨论放在一起，让每一次判断都有来处，也有可以继续追问的人。</p>
        </div>
      </section>

      <section className="capability-section section-shell" aria-label="产品能力">
        {capabilities.map((item, index) => (
          <article className={`capability-row ${index % 2 ? 'is-reverse' : ''}`} key={item.index}>
            <div className="capability-copy">
              <span className="section-index">{item.index}</span>
              <h2>{item.title}</h2>
              <p>{item.body}</p>
              <a className="text-link" href={item.type === 'feed' ? '/community' : item.href}>{item.link} <ArrowUpRight size={15} /></a>
            </div>
            <CapabilityVisual type={item.type} />
          </article>
        ))}
      </section>

      <section className="evidence-section">
        <div className="section-shell">
          <div className="section-kicker"><span>一条完整的判断路径</span><span>01 — 03</span></div>
          <h2 className="evidence-title">一个问题，三类依据。</h2>
          <div className="evidence-path">
            <div><span>经验</span><strong>别人怎么选</strong><small>真实帖子与讨论</small></div>
            <i aria-hidden="true" />
            <div><span>要求</span><strong>专业需要什么</strong><small>科目与专业目录</small></div>
            <i aria-hidden="true" />
            <div><span>来源</span><strong>依据从哪里来</strong><small>官方政策与数据</small></div>
          </div>
        </div>
      </section>

      <section className="posts-section section-shell" id="community">
        <div className="section-kicker"><span>社区正在讨论</span><a href="/community">浏览全部 <ArrowUpRight size={15} /></a></div>
        <div className="posts-heading"><h2>先从一个真实问题开始。</h2><p>没有被包装成答案的经验，往往更接近你真正想问的事。</p></div>
        <div className="post-grid">
          {posts.map((post) => <PostPreview key={post.id} post={post} />)}
        </div>
      </section>

      <section className="trust-section section-shell" id="knowledge">
        <div className="trust-copy"><span className="section-index">SOURCE / 2026</span><h2>每条结论，<br /><em>都能回到来源。</em></h2><p>官方入口、数据年份和复核状态清楚可见。需要逐校核对的地方，页面会明确告诉你下一步该查什么。</p></div>
        <div className="trust-list"><div><strong>官方来源</strong><span>教育考试院 · 阳光高考</span></div><div><strong>结构化整理</strong><span>政策、专业与科目要求</span></div><div><strong>复核状态</strong><span className="status"><i /> 已完成来源核对</span></div><a className="text-link" href="/knowledge">打开政策资料库 <ArrowUpRight size={15} /></a></div>
      </section>

      <section className="final-section section-shell"><p className="eyebrow">选择开始之前</p><h2>把依据看清楚，<br />再走自己的路。</h2><div className="final-actions"><a className="button button-dark" href="/community">进入选科社区 <ArrowUpRight size={16} /></a><a className="text-link" href={ANDROID_DOWNLOAD_URL} download>下载 Android App <ArrowUpRight size={15} /></a></div></section>

      <footer className="welcome-footer section-shell"><a className="welcome-brand" href="/" aria-label="SoulCourse 首页"><img className="welcome-brand-logo" src="/welcome/soulcourse-logo.jpeg" alt="SoulCourse 开源在线教育系统" /></a><span>广东选科社区</span><nav><a href="/requirements">选科查询</a><a href="/knowledge">政策库</a><a href="/insights">数据中心</a></nav><small>© 2026 SoulCourse</small></footer>
    </main>
  )
}

function CapabilityVisual({ type }: { type: string }) {
  if (type === 'feed') return <div className="visual-frame visual-feed"><div className="visual-topline"><span>选科社区</span><span>推荐　经验　数据</span></div><div className="visual-feed-grid"><div className="mini-post mini-post-tall"><img className="mini-image" src="/welcome/post-images/subject-choice-student.jpg" alt="学生思考选科组合" /><strong>物化生适合目标不太明确的人吗？</strong><small>小周同学　♡ 128</small></div><div className="mini-post"><img className="mini-image" src="/welcome/post-images/subject-choice-desk.jpg" alt="学生整理选科资料" /><strong>从最近三届学生看物化地的优劣势</strong><small>陈老师　♡ 212</small></div><div className="mini-post"><img className="mini-image" src="/welcome/post-images/family-discussion.jpg" alt="家长和学生讨论选科" /><strong>选科前做一张纸</strong><small>林妈妈　♡ 96</small></div></div></div>
  if (type === 'requirements') return <div className="visual-frame visual-requirements"><div className="visual-search"><span>⌕</span><span>搜索专业名称</span></div><div className="visual-result"><div className="major-mark">临床<br />医学</div><div><small>医学与生命科学</small><strong>临床医学</strong><p><b>物理</b><b>化学</b></p></div><span className="verified-mark">已复核</span></div><div className="visual-result faded"><div className="major-mark">软件<br />工程</div><div><small>计算机与电子信息</small><strong>软件工程</strong><p><b>物理</b><b>化学</b></p></div></div><span className="visual-year">2026 / 专业目录</span></div>
  return <div className="visual-frame visual-source"><div className="source-title"><span>政策资料库</span><strong>2026</strong></div><div className="source-chart"><i style={{ height: '52%' }} /><i style={{ height: '78%' }} /><i style={{ height: '64%' }} /><i style={{ height: '91%' }} /><i style={{ height: '70%' }} /></div><div className="source-line"><span>广东省教育考试院</span><b>已复核</b></div><div className="source-line"><span>阳光高考专业目录</span><b>官方来源</b></div></div>
}

function PostPreview({ post }: { post: typeof posts[number] }) {
  return <a className="post-preview" href={`/posts/${post.id}`}><div className={`post-cover ${post.tone}`}><img src={post.image} alt="" /><span>#{post.tag}</span><Play size={17} fill="currentColor" /></div><div className="post-info"><strong>{post.title}</strong><small>{post.author}　{post.meta}</small></div></a>
}

export default App
