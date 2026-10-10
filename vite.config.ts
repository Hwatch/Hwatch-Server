import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
const api = process.env.HWATCH_DEV_API || 'http://127.0.0.1:8787';
export default defineConfig({ plugins:[vue()], build:{ outDir:'web/dist', emptyOutDir:true }, server:{host:'0.0.0.0',proxy:{'/api':api,'/media':api,'/thumb':api}} })
