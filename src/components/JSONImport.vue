<script setup lang="ts">
import {ref} from 'vue';
defineProps<{label:string;disabled?:boolean}>();
const emit=defineEmits<{loaded:[text:string]}>();
const error=ref(''),fileName=ref('');
async function load(event:Event){const input=event.target as HTMLInputElement,file=input.files?.[0];if(!file)return;error.value='';try{if(file.size>10*1024*1024)throw Error('文件超过 10 MB，请拆分后导入。');const text=await file.text();JSON.parse(text);fileName.value=file.name;emit('loaded',text);}catch(e){error.value=e instanceof Error?e.message:String(e);}finally{input.value='';}}
</script>
<template><div class="json-import"><label class="upload" :class="{disabled}">{{label}}<input type="file" accept=".json,application/json" :disabled="disabled" @change="load"></label><small v-if="fileName">{{fileName}}</small><small v-if="error" class="error" role="alert">JSON 无法读取：{{error}}</small></div></template>
