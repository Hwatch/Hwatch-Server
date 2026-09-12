<script setup lang="ts">
import { ref } from 'vue'
import NavIcon from './NavIcon.vue'

type Settings={http_proxy:string;download_dir:string;aria_rpc_url:string;aria_secret:string;downloader_mode:string;check_time:string;timezone:string;max_concurrent:number;connections_per_download:number;download_limit_kbps:number;min_free_bytes:number;require_creator_match:boolean;exclude_title_keywords:string;exclude_uploaders:string;exclude_categories:string;exclude_tags:string;max_duration_seconds:number;backup_dir:string;backup_time:string;backup_retention:number;backup_last_success:string;backup_last_error:string}
type Backup={name:string;size:number;created_at:string;sha256:string;valid:boolean}
type SidebarItems={language:boolean;original:boolean;demo:boolean;aria:boolean}
const props=defineProps<{settings:Settings;busy:boolean;importHtml:string;demoMode:number;sidebarItems:SidebarItems;backups:Backup[];backupDirectory:string;backupSensitive:boolean}>()
const emit=defineEmits<{save:[];import:[];createBackup:[];restoreBackup:[name:string];'update:importHtml':[value:string];toggleDemo:[];updateSidebarItem:[item:keyof SidebarItems,value:boolean]}>()
const size=(n=0)=>n>=1073741824?(n/1073741824).toFixed(2)+' GB':n>=1048576?(n/1048576).toFixed(1)+' MB':(n/1024).toFixed(1)+' KB'
const open=ref('download')
const toggle=(name:string)=>open.value=open.value===name?'':name
</script>

<template>
  <div class="settings-page">
    <section class="setting-card" :class="{open:open==='download'}">
      <button class="setting-head" type="button" :aria-expanded="open==='download'" @click="toggle('download')">
        <span class="setting-symbol"><NavIcon name="downloads"/></span><span><b>下载与存储</b><small>管理下载目录、并发、连接和空间限制</small></span><i>⌄</i>
      </button>
      <form v-if="open==='download'" class="setting-body" @submit.prevent="emit('save')">
        <label>下载器模式<select v-model="props.settings.downloader_mode" disabled><option value="aria2">固定 aria2（当前可用）</option><option value="wget">固定 wget（接入中）</option><option value="auto">自动选择（接入中）</option></select><small>当前任务继续由原下载器处理；正式开放切换后，仅 queued 和新任务采用新模式。</small></label>
        <div class="downloader-capability"><span><b>aria2</b><small>已接入 · 多连接、全局限速、断点续传</small></span><span><b>wget</b><small>进程管理与重启续传接入中</small></span></div>
        <label>下载与 Plex 媒体目录<input v-model="props.settings.download_dir" placeholder="/volume1/media/HWatch"><small>视频直接按 Plex 电视库结构写入，无需复制第二份。</small></label>
        <div class="setting-grid"><label>并发下载任务数<input v-model.number="props.settings.max_concurrent" type="number" min="1" max="20"></label><label>单任务连接数<input v-model.number="props.settings.connections_per_download" type="number" min="1" max="16"></label><label>全局下载限速（KB/s）<input v-model.number="props.settings.download_limit_kbps" type="number" min="0"><small>0 表示不限速</small></label><label>最低保留空间（字节）<input v-model.number="props.settings.min_free_bytes" type="number"></label></div>
        <button class="primary save-setting" :disabled="busy">保存设置</button>
      </form>
    </section>

    <section class="setting-card" :class="{open:open==='backup'}">
      <button class="setting-head" type="button" :aria-expanded="open==='backup'" @click="toggle('backup')">
        <span class="setting-symbol"><NavIcon name="library"/></span><span><b>数据库备份与恢复</b><small>将 SQLite 一致性快照保存到 NFS，不重复备份媒体</small></span><i>⌄</i>
      </button>
      <div v-if="open==='backup'" class="setting-body">
        <form @submit.prevent="emit('save')">
          <label>备份目录<input v-model="props.settings.backup_dir" :placeholder="props.backupDirectory"><small>留空时使用媒体目录下的 .hwatch-backups。</small></label>
          <div class="setting-grid"><label>每日备份时间<input v-model="props.settings.backup_time" type="time"></label><label>保留数量<input v-model.number="props.settings.backup_retention" type="number" min="1" max="365"></label></div>
          <div class="actions"><button class="primary" :disabled="busy">保存备份设置</button><button type="button" :disabled="busy" @click="emit('createBackup')">立即备份</button></div>
        </form>
        <p v-if="backupSensitive" class="error">数据库包含代理或下载器凭据，备份文件属于敏感数据；目录权限已限制为仅服务账号可读写。</p>
        <p v-if="settings.backup_last_error" class="error">最近备份失败：{{settings.backup_last_error}}</p>
        <small>最近成功：{{settings.backup_last_success||'尚无'}} · 位置：{{backupDirectory}}</small>
        <div class="backup-list"><article v-for="item in backups" :key="item.name"><span><b>{{item.name}}</b><small>{{size(item.size)}} · SHA-256 {{item.sha256.slice(0,12)}}…</small></span><button class="danger" :disabled="busy||!item.valid" @click="emit('restoreBackup',item.name)">{{item.valid?'恢复':'校验失败'}}</button></article><div v-if="!backups.length" class="empty compact-empty">还没有数据库备份</div></div>
      </div>
    </section>

    <section class="setting-card" :class="{open:open==='network'}">
      <button class="setting-head" type="button" :aria-expanded="open==='network'" @click="toggle('network')">
        <span class="setting-symbol"><NavIcon name="network"/></span><span><b>网络与调度</b><small>管理代理、aria2、检查时间和时区</small></span><i>⌄</i>
      </button>
      <form v-if="open==='network'" class="setting-body" @submit.prevent="emit('save')">
        <label>HTTP/HTTPS 代理<input v-model="props.settings.http_proxy" placeholder="http://127.0.0.1:7890"></label>
        <label>aria2 RPC 地址<input v-model="props.settings.aria_rpc_url"></label>
        <label>aria2 RPC 密钥<input v-model="props.settings.aria_secret" type="password" autocomplete="new-password"></label>
        <div class="setting-grid"><label>每日检查时间<input v-model="props.settings.check_time" type="time"></label><label>时区<input v-model="props.settings.timezone" placeholder="Asia/Shanghai"></label></div>
        <button class="primary save-setting" :disabled="busy">保存设置</button>
      </form>
    </section>

    <section class="setting-card" :class="{open:open==='exclude'}">
      <button class="setting-head" type="button" :aria-expanded="open==='exclude'" @click="toggle('exclude')">
        <span class="setting-symbol"><NavIcon name="exclude"/></span><span><b>自动排除规则</b><small>过滤错误归属、合集和不需要的内容</small></span><i>⌄</i>
      </button>
      <form v-if="open==='exclude'" class="setting-body" @submit.prevent="emit('save')">
        <label class="setting-check"><input v-model="props.settings.require_creator_match" type="checkbox"><span><b>验证实际创作者</b><small>实际创作者与订阅作者不一致时不下载</small></span></label>
        <label>标题关键词<textarea v-model="props.settings.exclude_title_keywords" rows="2" placeholder="compilation, collection, 合集, 集锦"></textarea></label>
        <div class="setting-grid"><label>排除上传者<input v-model="props.settings.exclude_uploaders" placeholder="逗号分隔"></label><label>排除分类<input v-model="props.settings.exclude_categories" placeholder="逗号分隔"></label><label>排除标签<input v-model="props.settings.exclude_tags" placeholder="逗号分隔"></label><label>最大时长（秒）<input v-model.number="props.settings.max_duration_seconds" type="number" min="0"><small>0 表示不限制</small></label></div>
        <button class="primary save-setting" :disabled="busy">保存规则</button>
      </form>
    </section>

    <section class="setting-card" :class="{open:open==='display'}">
      <button class="setting-head" type="button" :aria-expanded="open==='display'" @click="toggle('display')">
        <span class="setting-symbol"><NavIcon name="demo"/></span><span><b>显示与隐私</b><small>控制页面演示模式和敏感内容显示</small></span><i>⌄</i>
      </button>
      <div v-if="open==='display'" class="setting-body">
        <div class="display-setting"><span><b>演示模式</b><small>{{demoMode===0?'当前正常显示':demoMode===1?'当前已模糊图片和视频':'当前已隐藏敏感文字并加强模糊'}}</small></span><button :class="{primary:demoMode>0}" @click="emit('toggleDemo')">{{demoMode===0?'开启演示模式':demoMode===1?'进入强演示模式':'关闭演示模式'}}</button></div>
        <div class="sidebar-options"><b>侧栏底部项目</b><small>“设置”入口始终保留，避免隐藏后无法恢复。</small><label><input type="checkbox" :checked="sidebarItems.language" @change="emit('updateSidebarItem','language',($event.target as HTMLInputElement).checked)">语言选择</label><label><input type="checkbox" :checked="sidebarItems.original" @change="emit('updateSidebarItem','original',($event.target as HTMLInputElement).checked)">原文 / 译文切换</label><label><input type="checkbox" :checked="sidebarItems.demo" @change="emit('updateSidebarItem','demo',($event.target as HTMLInputElement).checked)">演示模式快捷按钮</label><label><input type="checkbox" :checked="sidebarItems.aria" @change="emit('updateSidebarItem','aria',($event.target as HTMLInputElement).checked)">aria2 在线状态</label></div>
      </div>
    </section>

    <section class="setting-card" :class="{open:open==='import'}">
      <button class="setting-head" type="button" :aria-expanded="open==='import'" @click="toggle('import')">
        <span class="setting-symbol"><NavIcon name="import"/></span><span><b>数据导入</b><small>粘贴视频详情区域的 HTML 数据</small></span><i>⌄</i>
      </button>
      <div v-if="open==='import'" class="setting-body">
        <label>HTML 数据<textarea :value="importHtml" rows="8" placeholder="粘贴视频详情区 HTML" @input="emit('update:importHtml',($event.target as HTMLTextAreaElement).value)"></textarea></label>
        <button :disabled="!importHtml||busy" @click="emit('import')">解析并导入</button>
      </div>
    </section>
  </div>
</template>
