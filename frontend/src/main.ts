import { createApp } from 'vue'
import './styles.css'
import App from './App.vue'
import router from './router'

const app = createApp(App)
app.use(router)
app.mount('#app')

if (import.meta.env.VITE_TRACELOG_DESKTOP_E2E === '1') {
  void import('./desktop-smoke')
}
