import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
export default defineConfig({ plugins:[vue()], build:{ outDir:'web/dist', emptyOutDir:true }, server:{host:'0.0.0.0',proxy:{'/api':'http://127.0.0.1:8787','/media':'http://127.0.0.1:8787','/thumb':'http://127.0.0.1:8787'}} })
