<script setup lang="ts">
defineProps<{title:string;subtitle:string;summary:string;duration:string;src:string;state:string;downloadable:boolean;pending:boolean}>();
const emit=defineEmits<{play:[];download:[];image:[element:unknown];loaded:[];error:[event:Event]}>();
</script>
<template>
  <article>
    <button class="poster" :class="'poster--'+state" :aria-label="'查看 '+title" @click="emit('play')">
      <span class="poster-skeleton" aria-hidden="true"></span>
      <img :ref="el=>emit('image',el)" :class="{loaded:state==='loaded'}" :src="src" :alt="title" loading="lazy" decoding="async" @load="emit('loaded')" @error="emit('error',$event)">
      <span class="poster-duration">{{duration}}</span>
    </button>
    <div class="meta"><b>{{title}}</b><small>{{subtitle}}</small><p>{{summary}}</p></div>
    <div class="library-card-actions"><button class="library-card__play" :disabled="downloadable&&pending" :aria-busy="downloadable&&pending" @click="downloadable?emit('download'):emit('play')">{{downloadable?(pending?'提交中…':'下载'):'播放'}}</button></div>
  </article>
</template>
