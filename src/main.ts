import { createApp } from 'vue'
import App from './App.vue'
import './style.css'
import './health.css'
import './auth.css'
import './privacy.css'
import './frontend-v2.css'
const savedTheme=localStorage.getItem('hwatch-theme');
document.documentElement.dataset.theme=savedTheme==='light'?'light':savedTheme==='system'?(matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light'):'dark';

const standalone = window.matchMedia('(display-mode: standalone)').matches ||
  ('standalone' in navigator && (navigator as Navigator & { standalone?: boolean }).standalone === true)
document.documentElement.classList.toggle('standalone', standalone)

createApp(App).mount('#app')
