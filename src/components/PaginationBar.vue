<script setup lang="ts">
defineProps<{page:number;pages:number;total?:number;busy?:boolean}>();
const emit=defineEmits<{change:[page:number]}>();
</script>
<template><nav class="pagination" aria-label="分页"><button :disabled="page===0||busy" @click="emit('change',page-1)">上一页</button><span>{{page+1}} / {{pages}}<template v-if="total!==undefined"> · 共 {{total}} 项</template></span><button :disabled="page+1>=pages||busy" @click="emit('change',page+1)">下一页</button></nav></template>
