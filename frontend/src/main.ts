import { createApp } from 'vue'
import '@/design-system'
import App from './App.vue'
import { router } from './router'

createApp(App).use(router).mount('#app')
