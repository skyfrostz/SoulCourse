# 选科π Android

Android 端复用本目录的 Vue 应用，通过 Capacitor 生成 `cn.soulcourse.app` 原生壳。正式包不会包含管理后台路由，API 使用 `https://soulcourse.cn/api/v1`。

## 本地开发

```bash
pnpm install
pnpm build:android
pnpm exec cap sync android
cd android
./gradlew assembleDebug
```

构建需要 JDK 17、Android SDK 35 和 API 26 的平台/模拟器。真机或模拟器安装 debug APK 后，生产 API 仍要求可访问 HTTPS；本地接口调试请使用独立的 `VITE_API_BASE_URL` 和仅 debug 的 cleartext 配置，不要把明文地址带入 release 包。

## 生产发布

```bash
VITE_APP_VERSION=1.0.0 pnpm build:android
pnpm exec cap sync android
cd android
./gradlew assembleRelease
```

正式 keystore、`local.properties`、`google-services.json` 和签名密码不得提交。发布文件应放在官网 HTTPS 下载目录，同时生成 SHA-256 和 `latest.json`。App 只提示用户跳转官网，不申请安装未知应用权限。

## 移动会话

移动登录使用 `/api/v1/mobile/auth/*`，令牌由 `SecureSession` Android Keystore 插件保存。服务器部署时必须将 Capacitor WebView 的 `https://localhost` 加入 API 的精确 CORS 白名单；不要使用 `*` 或允许 Cookie 跨域。

正式域名启用 App Links 前，需要把签名证书 SHA-256 写入 `/.well-known/assetlinks.json`，并验证帖子、专业要求、政策和观察详情链接能够唤起 App。
