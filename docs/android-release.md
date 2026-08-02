# Android Release Preparation

This branch contains the Android v1 release preparation for `cn.soulcourse.app`.

## Public artifacts

- APK download: `https://soulcourse.cn/downloads/android/1.0.0/subject-choice-1.0.0.apk`
- Update manifest: `https://soulcourse.cn/downloads/android/stable/latest.json`
- Android App Links: `https://soulcourse.cn/.well-known/assetlinks.json`

## Signing policy

The release keystore, signing passwords, APK/AAB binaries, server credentials, and
local build configuration are intentionally excluded from this repository. The
local build reads signing values only from these environment variables:

```text
SOULCOURSE_RELEASE_KEYSTORE
SOULCOURSE_RELEASE_STORE_PASSWORD
SOULCOURSE_RELEASE_KEY_PASSWORD
```

The public App Links certificate fingerprint is maintained in `deploy/assetlinks.json`.
If Google Play App Signing is enabled, add Google's app-signing fingerprint there
before publishing the Play build.

## Build

```bash
VITE_APP_VERSION=1.0.0 pnpm build:android
pnpm exec cap sync android
cd android
./gradlew assembleRelease bundleRelease
```
