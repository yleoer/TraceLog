import { createApp } from 'vue'
import './styles.css'
import App from './App.vue'
import { api } from './api/client'
import router from './router'

const app = createApp(App)
app.use(router)
app.mount('#app')

if (import.meta.env.VITE_TRACELOG_UPDATE_E2E === '1') {
  window.setTimeout(() => {
    void runUpdateE2E()
  }, 1000)
}

async function runUpdateE2E() {
  const info = await api.getUpdateInfo()
  if (!info.has_update) throw new Error('upgrade test expected an available update')

  const result = await api.installUpdate()
  if (!result.started || !result.will_quit) throw new Error(result.message || 'upgrade test did not start')

  window.setTimeout(() => api.quitForUpdate(), 250)
}
