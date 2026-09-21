<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";

type API = (path: string, init?: RequestInit) => Promise<any>;
type NodeItem = {
  id: string;
  name: string;
  permissions: Record<string, any>;
  capabilities: Record<string, any>;
  version: string;
  platform: string;
  status: string;
  last_seen_at: string;
  created_at: string;
  revoked_at: string;
};
type ComputeJob = {
  id: string;
  type: "dedupe" | "transcode";
  video_id: string;
  status: string;
  node_id: string;
  algorithm_version: string;
  media_relative_path: string;
  progress: number;
  attempts: number;
  lease_until: string;
  error_code: string;
  error_message: string;
  result_json: string;
  created_at: string;
  updated_at: string;
};

const props = defineProps<{ api: API; timezone: string }>();
const nodes = ref<NodeItem[]>([]), jobs = ref<ComputeJob[]>([]), busy = ref(false);
const message = ref(""), nodeName = ref(""), allowDedupe = ref(true), allowTranscode = ref(true);
const createdToken = ref(""), createdNodeID = ref(""), keepOriginal = ref(true);
const jobFilter = ref("active"), videoID = ref(""), newJobType = ref<"dedupe" | "transcode">("dedupe");
let timer: number | undefined;

const activeStatuses = new Set(["pending", "assigned", "analyzing", "matching", "transcoding", "verifying"]);
const visibleJobs = computed(() => jobs.value.filter((job) => jobFilter.value === "all" || (jobFilter.value === "active" ? activeStatuses.has(job.status) : job.status === jobFilter.value)));
const onlineCount = computed(() => nodes.value.filter((node) => node.status === "online").length);
const verifyingCount = computed(() => jobs.value.filter((job) => job.status === "verifying").length);
const statusLabel = (status: string) => ({pending:"等待节点",assigned:"已分派",analyzing:"分析中",matching:"匹配中",transcoding:"转码中",verifying:"等待确认",completed:"已完成",failed:"失败",cancelled:"已取消",online:"在线",offline:"离线",revoked:"已撤销"} as Record<string,string>)[status] || status;
const typeLabel = (type: string) => type === "dedupe" ? "内容去重" : "视频转码";
const time = (raw: string) => {
  if (!raw) return "尚无";
  try { return new Intl.DateTimeFormat("zh-CN", {timeZone: props.timezone || "Asia/Shanghai", dateStyle:"short", timeStyle:"short"}).format(new Date(raw)); }
  catch { return raw; }
};
const capabilityText = (node: NodeItem) => {
  const values: string[] = [];
  if (node.capabilities?.dedupe) values.push(`去重 × ${node.capabilities.max_dedupe_jobs || 1}`);
  if (node.capabilities?.transcode) values.push(`转码 × ${node.capabilities.max_transcode_jobs || 1}`);
  if (node.capabilities?.video_toolbox) values.push("VideoToolbox");
  return values.join(" · ") || "等待能力上报";
};
const jobResult = (job: ComputeJob): any => {
  try { return JSON.parse(job.result_json || "{}"); } catch { return {}; }
};
const relationshipLabel = (value: string) => ({exact:"完全一致",reencoded:"重新编码",contained:"已被长视频包含",partial_overlap:"部分重叠",compilation_contains:"合集包含"} as Record<string,string>)[value] || value;

async function refresh(silent = false) {
  if (!silent) busy.value = true;
  try {
    const [nodeResult, jobResult] = await Promise.all([props.api("/api/nodes"), props.api("/api/compute/jobs?limit=500")]);
    nodes.value = nodeResult.items || [];
    jobs.value = jobResult.items || [];
  } catch (error) { if (!silent) message.value = String(error); }
  finally { if (!silent) busy.value = false; }
}
async function createNode() {
  if (!nodeName.value.trim() || (!allowDedupe.value && !allowTranscode.value)) return;
  busy.value = true;
  try {
    const result = await props.api("/api/nodes", {method:"POST", body:JSON.stringify({name:nodeName.value.trim(), permissions:{dedupe:allowDedupe.value, transcode:allowTranscode.value}})});
    createdToken.value = result.token;
    createdNodeID.value = result.id;
    nodeName.value = "";
    message.value = "节点已创建，请立即保存一次性令牌";
    await refresh(true);
  } catch (error) { message.value = String(error); }
  finally { busy.value = false; }
}
async function copyToken() {
  try { await navigator.clipboard.writeText(createdToken.value); message.value = "节点令牌已复制"; }
  catch { message.value = "复制失败，请手动选择令牌"; }
}
async function revokeNode(node: NodeItem) {
  if (!confirm(`撤销节点“${node.name}”？该节点未完成的任务将等待重新分派。`)) return;
  busy.value = true;
  try { await props.api(`/api/nodes/${encodeURIComponent(node.id)}`, {method:"DELETE"}); message.value = "节点已撤销"; await refresh(true); }
  catch (error) { message.value = String(error); }
  finally { busy.value = false; }
}
async function createJob() {
  if (!videoID.value.trim()) return;
  busy.value = true;
  try {
    const result = await props.api("/api/compute/jobs", {method:"POST", body:JSON.stringify({video_id:videoID.value.trim(), type:newJobType.value})});
    message.value = result.created ? "计算任务已创建" : "相同媒体和算法的任务已经存在";
    videoID.value = "";
    await refresh(true);
  } catch (error) { message.value = String(error); }
  finally { busy.value = false; }
}
async function cancelJob(job: ComputeJob) {
  if (!confirm(`取消 ${typeLabel(job.type)}任务 ${job.id}？`)) return;
  busy.value = true;
  try { await props.api(`/api/compute/jobs/${encodeURIComponent(job.id)}/cancel`, {method:"POST"}); message.value = "任务已取消"; await refresh(true); }
  catch (error) { message.value = String(error); }
  finally { busy.value = false; }
}
async function commitJob(job: ComputeJob) {
  const detail = keepOriginal.value ? "原文件将保存在隐藏备份目录。" : "原文件将在成功替换后删除，此操作不可直接撤销。";
  if (!confirm(`确认提交该转码产物？${detail}`)) return;
  busy.value = true;
  try { await props.api(`/api/compute/jobs/${encodeURIComponent(job.id)}/commit`, {method:"POST", body:JSON.stringify({confirm:true, keep_original:keepOriginal.value})}); message.value = "转码产物已提交"; await refresh(true); }
  catch (error) { message.value = String(error); }
  finally { busy.value = false; }
}

onMounted(() => { refresh(); timer = window.setInterval(() => refresh(true), 10000); });
onBeforeUnmount(() => { if (timer) window.clearInterval(timer); });
</script>

<template>
  <div class="node-page">
    <div class="node-summary">
      <article><small>在线节点</small><b>{{ onlineCount }} / {{ nodes.filter(n=>!n.revoked_at).length }}</b></article>
      <article><small>活动任务</small><b>{{ jobs.filter(j=>activeStatuses.has(j.status)).length }}</b></article>
      <article :class="{warning:verifyingCount}"><small>等待确认</small><b>{{ verifyingCount }}</b></article>
    </div>
    <button v-if="message" class="notice" @click="message=''">{{ message }}</button>

    <section class="node-panel">
      <div class="node-panel-head"><span><h2>计算节点</h2><small>节点令牌只显示一次，服务端仅保存哈希</small></span><button :disabled="busy" @click="refresh()">刷新</button></div>
      <form class="node-create" @submit.prevent="createNode">
        <input v-model="nodeName" maxlength="100" required placeholder="节点名称，例如 Mac mini M4">
        <label><input v-model="allowDedupe" type="checkbox">允许去重</label>
        <label><input v-model="allowTranscode" type="checkbox">允许转码</label>
        <button class="primary" :disabled="busy||(!allowDedupe&&!allowTranscode)">创建节点</button>
      </form>
      <div class="node-list">
        <article v-for="node in nodes" :key="node.id" class="node-card">
          <span class="node-state" :class="node.status"></span>
          <div><div class="node-title"><b>{{ node.name }}</b><span :class="['status-chip',node.status]">{{ statusLabel(node.status) }}</span></div><small>{{ node.platform || '平台未知' }} · {{ node.version || '版本未知' }} · 最近心跳 {{ time(node.last_seen_at) }}</small><p>{{ capabilityText(node) }}</p><code>{{ node.id }}</code></div>
          <button v-if="!node.revoked_at" class="danger" :disabled="busy" @click="revokeNode(node)">撤销</button>
        </article>
        <div v-if="!nodes.length" class="empty compact-empty">还没有计算节点</div>
      </div>
    </section>

    <section class="node-panel">
      <div class="node-panel-head"><span><h2>外部任务</h2><small>输入已下载媒体的视频 ID，或从影视库卡片直接创建</small></span><select v-model="jobFilter"><option value="active">进行中</option><option value="verifying">等待确认</option><option value="failed">失败</option><option value="completed">已完成</option><option value="cancelled">已取消</option><option value="all">全部</option></select></div>
      <form class="job-create" @submit.prevent="createJob"><input v-model="videoID" required placeholder="视频 ID，例如 rule34video:123"><select v-model="newJobType"><option value="dedupe">内容去重</option><option value="transcode">转码 H.265</option></select><button class="primary" :disabled="busy">创建任务</button></form>
      <label v-if="verifyingCount" class="keep-original"><input v-model="keepOriginal" type="checkbox"><span><b>提交后保留原文件</b><small>建议首次使用时保持开启；备份位于 .hwatch-originals。</small></span></label>
      <div class="compute-jobs">
        <article v-for="job in visibleJobs" :key="job.id" class="compute-job">
          <div class="job-main"><div class="job-title"><span :class="['job-kind',job.type]">{{ typeLabel(job.type) }}</span><b>{{ job.video_id }}</b><span :class="['status-chip',job.status]">{{ statusLabel(job.status) }}</span></div><small>{{ job.node_id || '尚未分派节点' }} · {{ job.algorithm_version }} · 尝试 {{ job.attempts }} 次</small><progress :value="job.progress || 0" :max="1"></progress><p v-if="job.error_message" class="error">{{ job.error_code }} · {{ job.error_message }}</p><div v-if="job.type==='dedupe'&&jobResult(job).matches?.length" class="match-results"><span v-for="match in jobResult(job).matches" :key="match.video_id"><b>{{ relationshipLabel(match.relationship) }}</b><small>{{ match.video_id }} · 相似度 {{ Math.round((match.confidence||0)*100) }}%<template v-if="match.coverage"> · 覆盖 {{ Math.round(match.coverage*100) }}%</template></small></span></div><p v-if="job.type==='transcode'&&job.status==='verifying'" class="output-path">待确认产物：{{ jobResult(job).temp_relative_path }}</p><small v-if="!job.error_message">更新时间 {{ time(job.updated_at) }}<template v-if="job.lease_until"> · 租约至 {{ time(job.lease_until) }}</template></small></div>
          <div class="actions"><button v-if="job.status==='verifying'" class="primary" :disabled="busy" @click="commitJob(job)">确认产物</button><button v-if="activeStatuses.has(job.status)" class="danger" :disabled="busy" @click="cancelJob(job)">取消</button></div>
        </article>
        <div v-if="!visibleJobs.length" class="empty compact-empty">当前筛选下没有任务</div>
      </div>
    </section>

    <div v-if="createdToken" class="modal node-token-modal" @click.self="createdToken=''">
      <div class="node-token" role="dialog" aria-modal="true"><h2>保存节点令牌</h2><p>节点 <code>{{ createdNodeID }}</code> 的令牌只显示这一次。关闭后无法再次读取，只能撤销并重建节点。</p><code>{{ createdToken }}</code><div class="actions"><button @click="copyToken">复制令牌</button><button class="primary" @click="createdToken=''">我已保存</button></div></div>
    </div>
  </div>
</template>

<style scoped>
.node-page{display:grid;gap:18px}.node-summary{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px}.node-summary article,.node-panel{background:#1a1d24;border:1px solid #2b303a;border-radius:14px}.node-summary article{padding:16px 18px;display:grid;gap:6px}.node-summary b{font-size:1.5rem;color:#e5b951}.node-summary .warning{border-color:#806c3c}.node-panel{padding:20px}.node-panel-head{display:flex;align-items:center;justify-content:space-between;gap:14px;margin-bottom:16px}.node-panel-head>span{display:grid;gap:4px}.node-panel-head h2{margin:0}.node-panel-head small{margin:0}.node-create,.job-create{display:grid;grid-template-columns:minmax(220px,1fr) auto auto auto;align-items:center;gap:10px;margin-bottom:16px}.job-create{grid-template-columns:minmax(240px,1fr) 170px auto}.node-create label{display:flex;align-items:center;gap:7px;color:#c5cbd5;white-space:nowrap}.node-create input[type=checkbox],.keep-original input{width:auto}.node-list,.compute-jobs{display:grid;gap:9px}.node-card,.compute-job{padding:14px;display:grid;grid-template-columns:auto minmax(0,1fr) auto;align-items:center;gap:13px;border:1px solid #303641;border-radius:11px;background:#14171c}.node-state{width:10px;height:10px;border-radius:50%;background:#737b88;box-shadow:0 0 0 5px #737b8820}.node-state.online{background:#60c694;box-shadow:0 0 0 5px #60c69420}.node-state.revoked{background:#de737b}.node-card>div,.job-main{min-width:0;display:grid;gap:6px}.node-title,.job-title{display:flex;align-items:center;gap:8px;min-width:0}.node-title b,.job-title b{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.node-card p,.compute-job p{margin:0;color:#a8b0bc;font-size:.8rem}.node-card code{color:#777f8d;font-size:.72rem;overflow-wrap:anywhere}.status-chip,.job-kind{padding:3px 8px;border-radius:999px;background:#292e37;color:#c7cdd6;font-size:.7rem;font-weight:750;white-space:nowrap}.status-chip.online,.status-chip.completed{background:#183629;color:#78d7a5}.status-chip.failed,.status-chip.revoked{background:#40252a;color:#ff9d9d}.status-chip.verifying{background:#41341d;color:#f0c56b}.status-chip.analyzing,.status-chip.matching,.status-chip.transcoding,.status-chip.assigned{background:#1c3446;color:#91d3ff}.job-kind.dedupe{color:#b7a3ff;background:#2d2648}.job-kind.transcode{color:#79d8d1;background:#193c3a}.job-main progress{margin-top:2px}.keep-original{display:flex;gap:10px;align-items:center;margin:0 0 14px;padding:12px;border-radius:9px;background:#24271f}.keep-original span{display:grid;gap:3px}.keep-original small{margin:0}.node-token{width:min(620px,calc(100vw - 28px));padding:22px;background:#1a1d24;border:1px solid #806c3c;border-radius:14px}.node-token>code{display:block;padding:14px;margin:14px 0;background:#0f1115;border-radius:8px;color:#e5b951;overflow-wrap:anywhere;user-select:all}.node-token .actions{justify-content:flex-end}.notice{margin:0}.compact-empty{padding:24px}
.match-results{display:grid;gap:5px;padding:9px;border-radius:8px;background:#20232a}.match-results span{display:flex;justify-content:space-between;gap:10px}.match-results small{margin:0;text-align:right}.output-path{overflow-wrap:anywhere}
@media(max-width:850px){.node-create{grid-template-columns:1fr 1fr}.node-create>input{grid-column:1/-1}.job-create{grid-template-columns:1fr 150px}.job-create button{grid-column:1/-1}.node-card,.compute-job{grid-template-columns:auto minmax(0,1fr)}.node-card>button,.compute-job>.actions{grid-column:2;justify-self:start}}
@media(max-width:680px){.node-summary{grid-template-columns:1fr}.node-panel{padding:14px}.node-panel-head{align-items:stretch;flex-direction:column}.node-create,.job-create{grid-template-columns:1fr}.node-create>input,.job-create button{grid-column:auto}.node-card,.compute-job{grid-template-columns:1fr}.node-state{display:none}.node-card>button,.compute-job>.actions{grid-column:1}.node-title,.job-title{align-items:flex-start;flex-wrap:wrap}.job-title b{width:100%}}
</style>
