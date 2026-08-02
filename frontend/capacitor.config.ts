import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'cn.soulcourse.app',
  appName: '选科π',
  webDir: 'dist-android',
  android: {
    allowMixedContent: false,
    captureInput: false,
  },
  server: {
    androidScheme: 'https',
    cleartext: false,
  },
  plugins: {
    SplashScreen: {
      launchAutoHide: true,
      launchShowDuration: 350,
      backgroundColor: '#f7f8f6',
      showSpinner: false,
    },
    StatusBar: {
      style: 'LIGHT',
      backgroundColor: '#f7f8f6',
    },
  },
}

export default config
