<script setup lang="ts">
import { onMounted, ref } from 'vue'
import NavIcon from './NavIcon.vue'
import ThemeControl from './components/ThemeControl.vue'

type Settings={http_proxy:string;download_dir:string;aria_rpc_url:string;aria_secret:string;downloader_mode:string;check_time:string;timezone:string;max_concurrent:number;connections_per_download:number;download_limit_kbps:number;min_free_bytes:number;require_creator_match:boolean;exclude_title_keywords:string;exclude_uploaders:string;exclude_categories:string;exclude_tags:string;max_duration_seconds:number;backup_dir:string;backup_time:string;backup_retention:number;backup_last_success:string;backup_last_error:string}
type Backup={name:string;size:number;created_at:string;sha256:string;valid:boolean}
type ManagementEntry={id:string;label:string;desc:string}
type SidebarItemKey='language'|'original'|'demo'|'aria'|'subscriptions'|'downloads'|'undownloaded'|'excluded'|'integrity'|'translations'|'logs'
type SidebarItems=Record<SidebarItemKey,boolean>
type ExclusionPreview={id:string;title:string;subscriptionSlug:string;reason:string;status:string}
type API=(path:string,init?:RequestInit)=>Promise<any>
const props=defineProps<{managementEntries:ManagementEntry[];settings:Settings;busy:boolean;settingsBusy:boolean;backupsBusy:boolean;importBusy:boolean;importHtml:string;demoMode:number;sidebarItems:SidebarItems;shortsWatchedDays:number;backups:Backup[];backupDirectory:string;backupSensitive:boolean;exclusionPreview:ExclusionPreview[];exclusionTotal:number;api:API}>()
const emit=defineEmits<{save:[];previewExclusions:[];import:[];createBackup:[];restoreBackup:[name:string];'update:importHtml':[value:string];toggleDemo:[];updateSidebarItem:[item:keyof SidebarItems,value:boolean];updateShortsWatchedDays:[value:number]}>()
const size=(n=0)=>n>=1073741824?(n/1073741824).toFixed(2)+' GB':n>=1048576?(n/1048576).toFixed(1)+' MB':(n/1024).toFixed(1)+' KB'
const open=ref('download')
const toggle=(name:string)=>open.value=open.value===name?'':name
const recovery=ref<any>({available:false})
const trStatus=ref<{configured:boolean;created_at:string;last_used_at:string;scopes:string[]}>({configured:false,created_at:'',last_used_at:'',scopes:[]})
const trNewToken=ref(''),trBusy=ref(false),trError=ref('')
const trScopeLabel:Record<string,string>={read_media:'读取媒体',read_dedupe_candidates:'获取去重候选',write_duplicate_results:'写入重复结果',write_transcode_results:'写入转码结果'}
async function loadTRToken(){try{trStatus.value=await props.api('/api/integrations/transrunner/token')}catch{}}
async function createTRToken(){if(trBusy.value)return;trBusy.value=true;trError.value='';try{const result=await props.api('/api/integrations/transrunner/token',{method:'POST',body:JSON.stringify({rotate:trStatus.value.configured})});trNewToken.value=result.token;await loadTRToken()}catch(e){trError.value=String(e)}finally{trBusy.value=false}}
async function revokeTRToken(){if(trBusy.value)return;if(!confirm('撤销 TransRunner 服务令牌？撤销后所有回调请求将立即失效。'))return;trBusy.value=true;try{await props.api('/api/integrations/transrunner/token',{method:'DELETE'});await loadTRToken()}catch(e){trError.value=String(e)}finally{trBusy.value=false}}
async function copyTRToken(){try{await navigator.clipboard.writeText(trNewToken.value);trError.value="令牌已复制"}catch{trError.value="复制失败，请手动复制令牌"}}
onMounted(async()=>{try{const response=await fetch('/api/recovery/latest');if(response.ok)recovery.value=await response.json()}catch{};void loadTRToken()})
</script>

<template>
  <div class="settings-page">
    <section class="setting-card" :class="{open:open==='download'}" id="setting-download">
      <button class="setting-head" type="button" :aria-expanded="open==='download'" @click="toggle('download')">
        <span class="setting-symbol"><NavIcon name="downloads"/></span><span><b>下载与存储</b><small>管理下载目录、并发、连接和空间限制</small></span><i>⌄</i>
      </button>
      <form v-if="open==='download'" class="setting-body" @submit.prevent="emit('save')">
        <label>下载器模式<select v-model="props.settings.downloader_mode"><option value="aria2">固定 aria2</option><option value="wget">固定 wget</option><option value="auto">自动选择</option></select><small>仅 queued 和新任务采用新模式；活动任务保持原下载器。</small></label>
        <div class="downloader-capability"><span><b>aria2</b><small>多连接、全局限速、断点续传</small></span><span><b>wget</b><small>单连接、限速、断点续传</small></span></div>
        <label>下载与 Plex 媒体目录<input v-model="props.settings.download_dir" placeholder="/volume1/media/HWatch"><small>视频直接按 Plex 电视库结构写入，无需复制第二份。</small></label>
        <div class="setting-grid"><label>并发下载任务数<input v-model.number="props.settings.max_concurrent" type="number" min="1" max="20"></label><label>单任务连接数<input v-model.number="props.settings.connections_per_download" type="number" min="1" max="16"></label><label>全局下载限速（KB/s）<input v-model.number="props.settings.download_limit_kbps" type="number" min="0"><small>0 表示不限速</small></label><label>最低保留空间（字节）<input v-model.number="props.settings.min_free_bytes" type="number"></label></div>
        <button class="primary save-setting" :disabled="settingsBusy">{{ settingsBusy ? "正在保存…" : "保存设置" }}</button>
      </form>
    </section>

    <section class="setting-card" :class="{open:open==='backup'}" id="setting-backup">
      <button class="setting-head" type="button" :aria-expanded="open==='backup'" @click="toggle('backup')">
        <span class="setting-symbol"><NavIcon name="library"/></span><span><b>数据库备份与恢复</b><small>将 SQLite 一致性快照保存到 NFS，不重复备份媒体</small></span><i>⌄</i>
      </button>
      <div v-if="open==='backup'" class="setting-body">
        <form @submit.prevent="emit('save')">
          <label>备份目录<input v-model="props.settings.backup_dir" :placeholder="props.backupDirectory"><small>留空时使用媒体目录下的 .hwatch-backups。</small></label>
          <div class="setting-grid"><label>每日备份时间<input v-model="props.settings.backup_time" type="time"></label><label>保留数量<input v-model.number="props.settings.backup_retention" type="number" min="1" max="365"></label></div>
          <div class="actions"><button class="primary" :disabled="settingsBusy">{{ settingsBusy ? "正在保存…" : "保存备份设置" }}</button><button type="button" :disabled="backupsBusy" @click="emit('createBackup')">{{ backupsBusy ? "处理中…" : "立即备份" }}</button></div>
        </form>
        <p v-if="backupSensitive" class="error">数据库包含代理或下载器凭据，备份文件属于敏感数据；目录权限已限制为仅服务账号可读写。</p>
        <p v-if="settings.backup_last_error" class="error">最近备份失败：{{settings.backup_last_error}}</p>
        <small>最近成功：{{settings.backup_last_success||'尚无'}} · 位置：{{backupDirectory}}</small>
        <div v-if="recovery.available" class="recovery-summary">
          <b>最近恢复协调{{recovery.summary.reconciliation_done?'已完成':'等待 aria2 后继续'}}</b>
          <small>{{recovery.summary.completed_at}} · {{recovery.summary.subscriptions}} 个订阅 · {{recovery.summary.completed_media}} 个已下载媒体</small>
          <small>路径映射 {{recovery.summary.paths_remapped}} · 缺失媒体 {{recovery.summary.missing_media}} · 根目录外路径 {{recovery.summary.path_mismatches}}</small>
          <small>aria2 检查 {{recovery.summary.aria_tasks_checked}} · 保留 {{recovery.summary.aria_tasks_kept}} · 重新排队 {{recovery.summary.aria_tasks_requeued}}</small>
          <small v-if="recovery.summary.source_media_root!==recovery.summary.current_media_root">{{recovery.summary.source_media_root||'（原目录未知）'}} → {{recovery.summary.current_media_root}}</small>
          <p v-if="recovery.summary.aria_unavailable" class="error">aria2 暂时不可用，旧 GID 已原样保留，服务会自动重试协调。</p>
        </div>
        <div class="backup-list"><article v-for="item in backups" :key="item.name"><span><b>{{item.name}}</b><small>{{size(item.size)}} · SHA-256 {{item.sha256.slice(0,12)}}…</small></span><button class="danger" :disabled="backupsBusy||!item.valid" @click="emit('restoreBackup',item.name)">{{item.valid?'恢复':'校验失败'}}</button></article><div v-if="!backups.length" class="empty compact-empty">还没有数据库备份</div></div>
      </div>
    </section>

    <section class="setting-card" :class="{open:open==='network'}" id="setting-network">
      <button class="setting-head" type="button" :aria-expanded="open==='network'" @click="toggle('network')">
        <span class="setting-symbol"><NavIcon name="network"/></span><span><b>网络与调度</b><small>管理代理、aria2、检查时间和时区</small></span><i>⌄</i>
      </button>
      <form v-if="open==='network'" class="setting-body" @submit.prevent="emit('save')">
        <label>HTTP/HTTPS 代理<input v-model="props.settings.http_proxy" placeholder="http://127.0.0.1:7890"></label>
        <label>aria2 RPC 地址<input v-model="props.settings.aria_rpc_url"></label>
        <label>aria2 RPC 密钥<input v-model="props.settings.aria_secret" type="password" autocomplete="new-password"></label>
        <div class="setting-grid"><label>每日检查时间<input v-model="props.settings.check_time" type="time"></label><label>时区<input v-model="props.settings.timezone" placeholder="Asia/Shanghai"></label></div>
        <button class="primary save-setting" :disabled="settingsBusy">{{ settingsBusy ? "正在保存…" : "保存设置" }}</button>
      </form>
    </section>

    <section class="setting-card" :class="{open:open==='exclude'}" id="setting-exclude">
      <button class="setting-head" type="button" :aria-expanded="open==='exclude'" @click="toggle('exclude')">
        <span class="setting-symbol"><NavIcon name="exclude"/></span><span><b>自动排除规则</b><small>过滤错误归属、合集和不需要的内容</small></span><i>⌄</i>
      </button>
      <form v-if="open==='exclude'" class="setting-body" @submit.prevent="emit('save')">
        <label class="setting-check"><input v-model="props.settings.require_creator_match" type="checkbox"><span><b>验证实际创作者</b><small>实际创作者与订阅作者不一致时不下载</small></span></label>
        <label>标题关键词<textarea v-model="props.settings.exclude_title_keywords" rows="2" placeholder="compilation, collection, 合集, 集锦"></textarea></label>
        <div class="setting-grid"><label>排除上传者<input v-model="props.settings.exclude_uploaders" placeholder="逗号分隔"></label><label>排除分类<input v-model="props.settings.exclude_categories" placeholder="逗号分隔"></label><label>排除标签<input v-model="props.settings.exclude_tags" placeholder="逗号分隔"></label><label>最大时长（秒）<input v-model.number="props.settings.max_duration_seconds" type="number" min="0"><small>0 表示不限制</small></label></div>
        <div class="actions"><button class="primary save-setting" :disabled="settingsBusy">{{ settingsBusy ? "正在保存…" : "保存规则" }}</button><button type="button" :disabled="settingsBusy" @click="emit('previewExclusions')">预览命中</button></div>
        <div v-if="exclusionPreview.length" class="backup-list"><small>共命中 {{exclusionTotal}} 项，当前显示 {{exclusionPreview.length}} 项</small><article v-for="item in exclusionPreview" :key="item.id"><span><b>{{item.title}}</b><small>{{item.reason}} · {{item.status}}</small></span></article></div>
      </form>
    </section>

    <section class="setting-card" :class="{open:open==='display'}" id="setting-display">
      <button class="setting-head" type="button" :aria-expanded="open==='display'" @click="toggle('display')">
        <span class="setting-symbol"><NavIcon name="demo"/></span><span><b>显示与隐私</b><small>控制页面演示模式和敏感内容显示</small></span><i>⌄</i>
      </button>
      <div v-if="open==='display'" class="setting-body">
        <div class="display-setting"><span><b>外观主题</b><small>仅保存在当前浏览器，切换后立即生效，无需保存设置。</small></span><ThemeControl /></div>
        <div class="display-setting"><span><b>演示模式</b><small>{{demoMode===0?'当前正常显示':demoMode===1?'当前已模糊图片和视频':'当前已隐藏敏感文字并加强模糊'}}</small></span><button :class="{primary:demoMode>0}" @click="emit('toggleDemo')">{{demoMode===0?'开启演示模式':demoMode===1?'进入强演示模式':'关闭演示模式'}}</button></div>
        <div class="sidebar-options"><b>侧栏底部项目</b><small>“设置”入口始终保留，避免隐藏后无法恢复。</small><label><input type="checkbox" :checked="sidebarItems.language" @change="emit('updateSidebarItem','language',($event.target as HTMLInputElement).checked)">语言选择</label><label><input type="checkbox" :checked="sidebarItems.original" @change="emit('updateSidebarItem','original',($event.target as HTMLInputElement).checked)">原文 / 译文切换</label><label><input type="checkbox" :checked="sidebarItems.demo" @change="emit('updateSidebarItem','demo',($event.target as HTMLInputElement).checked)">演示模式快捷按钮</label><label><input type="checkbox" :checked="sidebarItems.aria" @change="emit('updateSidebarItem','aria',($event.target as HTMLInputElement).checked)">aria2 在线状态</label></div>
        <div class="sidebar-options"><b>主页侧栏快捷入口</b><small>勾选后把高级管理里的入口显示在主页侧栏导航中，方便直达。</small><label v-for="m in managementEntries" :key="m.id"><input type="checkbox" :checked="sidebarItems[m.id as SidebarItemKey]" @change="emit('updateSidebarItem',m.id as SidebarItemKey,($event.target as HTMLInputElement).checked)">{{ m.label }}</label></div>
        <div class="sidebar-options"><b>随机浏览</b><small>随机视频流不推送最近看过的内容（按设置的时区自然日计算）。</small><label>最近看过的视频<select :value="shortsWatchedDays" @change="emit('updateShortsWatchedDays',Number(($event.target as HTMLSelectElement).value))"><option :value="0">不限</option><option :value="1">今天</option><option :value="3">3 天</option><option :value="7">7 天</option><option :value="30">30 天</option></select></label></div>
      </div>
    </section>

    <section class="setting-card" :class="{open:open==='transrunner'}" id="setting-transrunner">
      <button class="setting-head" type="button" :aria-expanded="open==='transrunner'" @click="toggle('transrunner')">
        <span class="setting-symbol"><NavIcon name="network"/></span><span><b>TransRunner 集成</b><small>为外部计算服务签发受限的回调令牌</small></span><i>⌄</i>
      </button>
      <div v-if="open==='transrunner'" class="setting-body">
        <p><b>TransRunner 回调 API</b></p>
        <small>HWatch 只负责媒体库：外部服务读取媒体信息与去重候选，并回写重复/转码结果。HWatch 不管理计算节点、任务进度或在线状态。</small>
        <p>状态：{{trStatus.configured?'已创建':'未创建'}}<template v-if="trStatus.configured"> · 创建于 {{trStatus.created_at}} · 最近使用 {{trStatus.last_used_at||'尚未'}}</template></p>
        <div class="display-setting"><span><b>权限</b></span><small>✓ 读取媒体 · ✓ 获取去重候选 · ✓ 写入重复结果 · ✓ 写入转码结果</small></div>
        <p v-if="trError" class="error">{{trError}}</p>
        <div class="actions">
          <button class="primary" :disabled="trBusy" @click="createTRToken">{{trStatus.configured?'轮换令牌':'生成 Token'}}</button>
          <button v-if="trStatus.configured" class="danger" :disabled="trBusy" @click="revokeTRToken">撤销</button>
        </div>
      </div>
    </section>

    <section class="setting-card" :class="{open:open==='import'}" id="setting-import">
      <button class="setting-head" type="button" :aria-expanded="open==='import'" @click="toggle('import')">
        <span class="setting-symbol"><NavIcon name="import"/></span><span><b>数据导入</b><small>粘贴视频详情区域的 HTML 数据</small></span><i>⌄</i>
      </button>
      <div v-if="open==='import'" class="setting-body">
        <label>HTML 数据<textarea :value="importHtml" rows="8" placeholder="粘贴视频详情区 HTML" @input="emit('update:importHtml',($event.target as HTMLTextAreaElement).value)"></textarea></label>
        <button :disabled="!importHtml||importBusy" @click="emit('import')">{{ importBusy ? "导入中…" : "解析并导入" }}</button>
      </div>
    </section>

    <div v-if="trNewToken" class="modal" @click.self="trNewToken=''">
      <div class="player" style="max-width:560px">
        <button class="close" @click="trNewToken=''">关闭</button>
        <h2>服务令牌只显示一次</h2>
        <p>请立即复制并保存到 TransRunner。关闭后无法再次查看。</p>
        <code style="display:block;word-break:break-all;background:#0f1115;padding:14px;border-radius:8px;color:#e5b951">{{trNewToken}}</code>
        <div class="actions" style="margin-top:14px"><button class="primary" @click="copyTRToken">复制</button></div>
      </div>
    </div>
  </div>
</template>
