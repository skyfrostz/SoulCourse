import { VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { showAppError } from './lib/appError'
import { reportWebVitals } from './lib/webVitals'
import { router } from './router'
import './styles/app.css'

const app = createApp(App)

app.config.errorHandler = (error) => {
  showAppError(error, 'runtime')
}

app.use(createPinia())
app.use(router)
app.use(VueQueryPlugin, {
  queryClientConfig: {
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        staleTime: 30_000,
        retry: 0,
      },
    },
  },
})

router.onError((error) => {
  showAppError(error, 'chunk')
})

app.mount('#app')
reportWebVitals()
