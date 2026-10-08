<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import NavIcon from "./NavIcon.vue";
import SettingsView from "./SettingsView.vue";
import { requestJSON } from "./api-client";
type Link = { Name?: string; URL?: string; Image?: string };
type Creator = {
  platform: string;
  source_id: string;
  name: string;
  url: string;
  avatar_url: string;
  video_count: number;
};
type Sub = {
  id: number;
  slug: string;
  platform: string;
  subscription_type: string;
  source_id: string;
  name: string;
  url: string;
  avatar_url: string;
  avatar_proxy_url: string;
  external_links: Link[];
  enabled: boolean;
  last_checked_at: string;
  last_error: string;
  progress: {
    total: number;
    completed: number;
    active: number;
    waiting: number;
    failed: number;
    bytes_downloaded: number;
    total_bytes: number;
  };
};
type Video = {
  video_id: string;
  subscription_slug: string;
  platform: string;
  title: string;
  description: string;
  page_url: string;
  published_at: string;
  discovered_at: string;
  thumbnail_url: string;
  views: number;
  likes: number;
  like_rate: number;
  duration_seconds: number;
  creator_name: string;
  uploader_name: string;
  categories: Link[];
  tags: Link[];
  status: string;
  aria_gid: string;
  plex_path: string;
  bytes_downloaded: number;
  total_bytes: number;
  download_speed: number;
  connections: number;
  video_codec: string;
  audio_codec: string;
  container: string;
  width: number;
  height: number;
  frame_rate: number;
  bit_rate: number;
  bit_depth: number;
  media_probed_at: string;
  transcode_status: string;
  plex_season: number;
  plex_episode: number;
  error: string;
  error_code: string;
  error_retryable: boolean;
  downloaded_at: string;
  translated_title: string;
  translated_tags: string[];
  translation_language: string;
};
type DownloaderHealth = {
  ID: string;
  Name: string;
  Version: string;
  Error: string;
  Online: boolean;
  SupportsMulti: boolean;
  Active: number;
  Connections: number;
  DownloadSpeed: number;
  UptimeSeconds: number;
};
type HealthWarning = {
  code: string;
  severity: "warning" | "error";
  message: string;
};
type Notification = {
  id: number;
  code: string;
  severity: string;
  message: string;
  created_at: string;
  read_at: string;
};
type Exclusion = {
  video_id: string;
  source: string;
  reason: string;
  title_snapshot: string;
  original_url_snapshot: string;
  thumbnail_snapshot: string;
  created_at: string;
};
type Dashboard = {
  subscriptions: Sub[];
  videos: Video[];
  recently_played: Video[];
  counts: {
    videos: number;
    active: number;
    waiting: number;
    completed: number;
    discovered: number;
  };
  traffic_bytes: number;
  downloaded_videos: number;
  library_bytes: number;
  current_download_speed: number;
  current_connections: number;
  downloads_paused: boolean;
  disk_total: number;
  disk_free: number;
  aria2_online: boolean;
  downloader_mode: string;
  downloaders: DownloaderHealth[];
  health_status: string;
  health_warnings: HealthWarning[];
  unread_notifications: number;
  timezone: string;
};
type Settings = {
  http_proxy: string;
  download_dir: string;
  aria_rpc_url: string;
  aria_secret: string;
  downloader_mode: string;
  check_time: string;
  timezone: string;
  max_concurrent: number;
  connections_per_download: number;
  download_limit_kbps: number;
  min_free_bytes: number;
  require_creator_match: boolean;
  exclude_title_keywords: string;
  exclude_uploaders: string;
  exclude_categories: string;
  exclude_tags: string;
  max_duration_seconds: number;
  backup_dir: string;
  backup_time: string;
  backup_retention: number;
  backup_last_success: string;
  backup_last_error: string;
};
type Backup = {
  name: string;
  size: number;
  created_at: string;
  sha256: string;
  valid: boolean;
};
type Facet = { value: string; count: number };
type AuthState = {
  configured: boolean;
  authenticated: boolean;
  username?: string;
  csrf_token?: string;
};
const auth = ref<AuthState>({ configured: true, authenticated: false }),
  authReady = ref(false),
  authUsername = ref(""),
  authPassword = ref(""),
  authPasswordConfirm = ref(""),
  authError = ref("");
const empty: Dashboard = {
  subscriptions: [],
  videos: [],
  recently_played: [],
  counts: { videos: 0, active: 0, waiting: 0, completed: 0, discovered: 0 },
  traffic_bytes: 0,
  downloaded_videos: 0,
  library_bytes: 0,
  current_download_speed: 0,
  current_connections: 0,
  downloads_paused: false,
  disk_total: 0,
  disk_free: 0,
  aria2_online: false,
  downloader_mode: "aria2",
  downloaders: [],
  health_status: "ok",
  health_warnings: [],
  unread_notifications: 0,
  timezone: "Asia/Shanghai",
};
const data = ref<Dashboard>(empty),
  videos = ref<Video[]>([]),
  total = ref(0),
  view = ref("home"),
  query = ref(""),
  status = ref(""),
  page = ref(0),
  notice = ref(""),
  busy = ref(false),
  modelURL = ref(""),
  selected = ref<Video>(),
  importHTML = ref(""),
  translationLanguage = ref(
    localStorage.getItem("r34watch-language") || "zh-CN",
  ),
  creatorQuery = ref(""),
  creatorResults = ref<Creator[]>([]),
  selectedCreator = ref<Creator>(),
  searchingCreators = ref(false),
  downloadExisting = ref(false);
const downloadedStatLabels: Record<string, { label: string; sub: string }> = {
  "zh-CN": { label: "已下载视频", sub: "本地已入库" },
  "zh-TW": { label: "已下載影片", sub: "本機已入庫" },
  en: { label: "Downloaded Videos", sub: "Stored locally" },
  ja: { label: "ダウンロード済み動画", sub: "ローカル保存済み" },
};
const downloadedStat = computed(
  () =>
    downloadedStatLabels[translationLanguage.value] ||
    downloadedStatLabels["zh-CN"],
);
const viewRoutes: Record<string, string> = {
  home: "/",
  subscriptions: "/subscriptions",
  downloads: "/downloads",
  undownloaded: "/undownloaded",
  library: "/library",
  excluded: "/excluded",
  integrity: "/integrity",
  translations: "/translations",
  notifications: "/notifications",
  settings: "/settings",
};
const viewTitles: Record<string, string> = {
  home: "概览",
  subscriptions: "订阅管理",
  downloads: "下载中心",
  undownloaded: "未下载",
  library: "影视库",
  excluded: "已排除",
  integrity: "媒体完整性",
  translations: "翻译管理",
  notifications: "通知",
  settings: "设置",
};
const viewFromPath = () => {
  const raw = window.location.pathname.replace(/\/+$/, "") || "/";
  const path = raw === "/duplicates" ? "/library" : raw;
  return Object.entries(viewRoutes).find(([, route]) => route === path)?.[0] || "home";
};
const demoMode = ref(
  Math.min(
    2,
    Math.max(0, Number(localStorage.getItem("r34watch-demo-mode") || 0)),
  ),
);
const storedSidebarState = localStorage.getItem("hwatch-sidebar-collapsed");
const sidebarCollapsed = ref(
  storedSidebarState === null
    ? window.matchMedia("(max-width: 680px)").matches
    : storedSidebarState === "1",
);
const sidebarItems = ref({
  language: localStorage.getItem("hwatch-sidebar-language") !== "0",
  original: localStorage.getItem("hwatch-sidebar-original") !== "0",
  demo: localStorage.getItem("hwatch-sidebar-demo") !== "0",
  aria: localStorage.getItem("hwatch-sidebar-aria") !== "0",
});
const navGroups = [
  { id: "overview", label: "", items: [{ id: "home", label: "概览" }] },
  {
    id: "content",
    label: "内容获取",
    items: [
      { id: "subscriptions", label: "订阅管理" },
      { id: "downloads", label: "下载中心" },
      { id: "undownloaded", label: "未下载" },
    ],
  },
  {
    id: "media",
    label: "媒体库",
    items: [
      { id: "library", label: "影视库" },
      { id: "excluded", label: "已排除" },
      { id: "integrity", label: "媒体完整性" },
    ],
  },
  {
    id: "system",
    label: "系统",
    items: [
      { id: "translations", label: "翻译管理" },
      { id: "notifications", label: "通知" },
    ],
  },
];
function loadOpenGroups(): Record<string, boolean> {
  try {
    const raw = localStorage.getItem("hwatch-nav-groups");
    if (raw) return JSON.parse(raw);
  } catch {}
  return { overview: true, content: true, media: true, system: true };
}
const openGroups = ref<Record<string, boolean>>(loadOpenGroups());
function persistOpenGroups() {
  try {
    localStorage.setItem("hwatch-nav-groups", JSON.stringify(openGroups.value));
  } catch {}
}
function toggleGroup(id: string) {
  openGroups.value = { ...openGroups.value, [id]: openGroups.value[id] === false };
  persistOpenGroups();
}
function revealGroupFor(viewID: string) {
  const group = navGroups.find((g) => g.items.some((i) => i.id === viewID));
  if (group && group.label && openGroups.value[group.id] === false) {
    openGroups.value = { ...openGroups.value, [group.id]: true };
    persistOpenGroups();
  }
}
const showOriginal = ref(
  localStorage.getItem("r34watch-show-original") === "1",
);
const settings = ref<Settings>({
  http_proxy: "",
  download_dir: "media",
  aria_rpc_url: "http://127.0.0.1:6800/jsonrpc",
  aria_secret: "",
  downloader_mode: "aria2",
  check_time: "04:00",
  timezone: "Asia/Shanghai",
  max_concurrent: 2,
  connections_per_download: 4,
  download_limit_kbps: 0,
  min_free_bytes: 5368709120,
  require_creator_match: true,
  exclude_title_keywords: "compilation,collection,合集,集锦",
  exclude_uploaders: "",
  exclude_categories: "",
  exclude_tags: "",
  max_duration_seconds: 0,
  backup_dir: "",
  backup_time: "03:30",
  backup_retention: 7,
  backup_last_success: "",
  backup_last_error: "",
});
const backups = ref<Backup[]>([]),
  backupDirectory = ref(""),
  backupSensitive = ref(false);
const notifications = ref<Notification[]>([]);
const exclusions = ref<Exclusion[]>([]);
const mediaExclusionTotal = ref(0);
type MediaScanItem = {
  video_id: string;
  platform: string;
  title: string;
  path: string;
  status: string;
  detail?: string;
};
type MediaScan = {
  storage_available: boolean;
  total: number;
  present: number;
  missing: number;
  moved: number;
  permission_denied: number;
  path_invalid: number;
  unreadable: number;
  items: MediaScanItem[];
  scanned_at: string;
};
type CleanupPreviewItem = {
  video_id: string;
  platform: string;
  title: string;
  path: string;
  status: string;
  detail?: string;
};
type CleanupPreview = {
  storage_available: boolean;
  deletions: number;
  candidates: number;
  already_excluded: number;
  skipped: number;
  items: CleanupPreviewItem[];
};
const mediaScan = ref<MediaScan>();
const cleanupPreview = ref<CleanupPreview>();
type VdfItemView = {
  path: string;
  size_bytes: number;
  resolution?: string;
  match_status: string;
  video_id?: string;
  subscription_slug?: string;
  db_path?: string;
  file_size?: number;
  detail?: string;
};
type VdfGroupView = {
  group_id: string;
  reason: string;
  status: string;
  subscription_slug: string;
  problems: string[];
  remove_items: VdfItemView[];
  keep_items: VdfItemView[];
  estimated_freed: number;
};
type VdfConflict = {
  kind: string;
  video_id?: string;
  path?: string;
  groups: string[];
  detail: string;
};
type VdfPreview = {
  format: string;
  storage_available: boolean;
  has_conflicts: boolean;
  groups: number;
  ready_groups: number;
  already_done_groups: number;
  conflict_groups: number;
  rejected_groups: number;
  remove_total: number;
  remove_matched: number;
  remove_already_done: number;
  remove_unmatched: number;
  keep_total: number;
  keep_matched: number;
  keep_missing: number;
  cross_group_conflicts: number;
  estimated_freed: number;
  conflicts: VdfConflict[];
  items: VdfGroupView[];
};
const vdfPreviewData = ref<VdfPreview>();
const deletionListText = ref("");
const integrityNotice = ref("");
const integrityBusy = ref(false);
const cleanupStatusLabels: Record<string, string> = {
  candidate: "可加入排除",
  already_excluded: "已排除",
  already_removed: "已移除",
  unknown_video: "未知 video_id",
  not_completed: "非已完成",
  active: "下载/转码中",
  path_mismatch: "路径不一致",
  file_present: "文件仍存在",
  path_invalid: "路径无效",
  missing: "文件缺失",
  moved: "疑似移动",
  permission_denied: "权限错误",
  unreadable: "无法读取",
  present: "正常",
};
function cleanupStatusLabel(status: string): string {
  return cleanupStatusLabels[status] || status;
}
type PlanItemView = {
  video_id: string;
  kept_video_id: string;
  original_path: string;
  expected_size: number;
  expected_sha256: string;
  status: string;
  reason?: string;
  freed_bytes?: number;
};
type PlanPreview = {
  kind: string;
  plan_version: number;
  scan_id: string;
  subscription_slug: string;
  valid: boolean;
  problems: string[];
  total: number;
  ready: number;
  rejected: number;
  already_excluded: number;
  needs_recovery: number;
  estimated_freed: number;
  items: PlanItemView[];
};
type PlanResult = {
  plan_id: string;
  dry_run: boolean;
  executed: number;
  skipped: number;
  failed: number;
  needs_recovery: number;
  estimated_freed: number;
};
type DeletionPlanRow = {
  plan_id: string;
  kind: string;
  plan_version: number;
  subscription_slug: string;
  scan_id: string;
  status: string;
  created_at: string;
};
const planText = ref("");
const planPreviewData = ref<PlanPreview>();
const planResult = ref<PlanResult>();
const deletionPlans = ref<DeletionPlanRow[]>([]);
const planNotice = ref("");
const planBusy = ref(false);
const planStatusLabels: Record<string, string> = {
  pending: "待处理",
  verified: "已验证",
  rejected: "已拒绝",
  already_excluded: "已排除",
  quarantined: "已隔离",
  db_excluded: "已清理",
  failed: "失败",
  needs_recovery: "需恢复",
  completed: "已完成",
};
function planStatusLabel(status: string): string {
  return planStatusLabels[status] || status;
}
const vdfGroupStatusLabels: Record<string, string> = {
  ready: "可执行",
  already_done: "此前已处理",
  conflict: "冲突",
  rejected: "无法匹配",
};
function vdfGroupStatusLabel(status: string): string {
  return vdfGroupStatusLabels[status] || status;
}
function vdfGroupBadgeClass(status: string): string {
  if (status === "ready") return "verified";
  if (status === "already_done") return "present";
  if (status === "conflict") return "rejected";
  return "needs_recovery";
}
const vdfItemStatusLabels: Record<string, string> = {
  matched: "已匹配",
  absent: "文件已不存在",
  already_done: "此前已处理",
  already_removed: "已被清理",
  unmatched: "未匹配",
  ambiguous: "歧义",
  basename_only: "仅文件名",
  not_completed: "非已完成",
  missing_file: "文件缺失",
  size_mismatch: "大小不符",
};
function vdfItemStatusLabel(status: string): string {
  return vdfItemStatusLabels[status] || status;
}
const deleteTarget = ref<Video | undefined>(undefined);
const deleteMode = ref<"delete_and_exclude" | "delete_only">("delete_and_exclude");
const deleting = ref(false);
const exclusionPreview = ref<any[]>([]);
const exclusionTotal = ref(0);
const facets = ref<{ creators: Facet[]; tags: Facet[]; codecs: Facet[] }>({
    creators: [],
    tags: [],
    codecs: [],
  }),
  showFilters = ref(false),
  searchScope = ref("all"),
  tagMode = ref("all"),
  selectedTags = ref<string[]>([]),
  excludedTags = ref<string[]>([]),
  creatorFilter = ref(""),
  codecFilter = ref(""),
  minHeight = ref(0),
  sortBy = ref("downloaded_at"),
  sortOrder = ref("desc");
const pages = computed(() => Math.max(1, Math.ceil(total.value / 48))),
  downloadTasks = computed(() => videos.value),
  unfinishedDownloadTasks = computed(() =>
    downloadTasks.value.filter((video) => video.status !== "completed"),
  ),
  completedDownloadTasks = computed(() =>
    downloadTasks.value.filter((video) => video.status === "completed"),
  ),
  subscriptionGroups = computed(() => {
    const groups = new Map<string, { key: string; name: string; sources: Sub[]; avatar?: Sub }>();
    for (const sub of data.value.subscriptions) {
      const key = sub.name.trim().toLocaleLowerCase();
      const group = groups.get(key) || { key, name: sub.name, sources: [] };
      group.sources.push(sub);
      if (!group.avatar && sub.avatar_url) group.avatar = sub;
      groups.set(key, group);
    }
    return [...groups.values()].sort((a, b) => a.name.localeCompare(b.name));
  });
const showCompletedTasks = ref(false);
type RateLimit = {
  source: string;
  host: string;
  reason: string;
  backoff_until: string;
  retry_after_seconds: number;
  consecutive_429: number;
  transient: number;
  waiting: number;
};
const rateLimits = ref<RateLimit[]>([]);
const nowMs = ref(Date.now());
const rateLimitLabels: Record<
  string,
  {
    title: string;
    waiting: string;
    other: string;
    taskRow: string;
    taskRowNetwork: string;
    lastError: string;
  }
> = {
  "zh-CN": {
    title: "来源请求受限",
    waiting: "等待任务",
    other: "其他来源下载不受影响",
    taskRow: "等待来源限速解除",
    taskRowNetwork: "等待来源连接恢复",
    lastError: "上次错误",
  },
  "zh-TW": {
    title: "來源請求受限",
    waiting: "等待任務",
    other: "其他來源下載不受影響",
    taskRow: "等待來源限速解除",
    taskRowNetwork: "等待來源連線恢復",
    lastError: "上次錯誤",
  },
  en: {
    title: "Source temporarily unavailable",
    waiting: "Waiting tasks",
    other: "Downloads from other sources continue",
    taskRow: "Waiting for source rate limit",
    taskRowNetwork: "Waiting for source connection",
    lastError: "Last error",
  },
  ja: {
    title: "取得元が一時的に制限中",
    waiting: "待機中",
    other: "他の取得元のダウンロードは継続します",
    taskRow: "取得元の制限解除を待機中",
    taskRowNetwork: "取得元の接続回復を待機中",
    lastError: "前回のエラー",
  },
};
const rlLabel = computed(
  () =>
    rateLimitLabels[translationLanguage.value] || rateLimitLabels["zh-CN"],
);
const activeRateLimits = computed(() =>
  rateLimits.value
    .map((r) => ({
      ...r,
      remaining: Math.max(
        0,
        Math.floor((Date.parse(r.backoff_until) - nowMs.value) / 1000),
      ),
    }))
    .filter((r) => r.remaining > 0),
);
function fmtCountdown(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  return h > 0 ? `${h}:${pad(m)}:${pad(s)}` : `${m}:${pad(s)}`;
}
function sourceLabel(source: string): string {
  if (source === "rule34video") return "Rule34Video";
  if (source === "hanime1") return "Hanime1";
  return source;
}
function rateLimitedSource(platform?: string): boolean {
  return rateLimitedReason(platform) !== "";
}
function rateLimitedReason(platform?: string): string {
  if (!platform) return "";
  const hit = activeRateLimits.value.find((r) => r.source === platform);
  return hit ? hit.reason || "rate_limit" : "";
}
const subscriptionTypeLabel = (type = "") =>
  type === "playlist" ? "番剧订阅" : "作者订阅";
const size = (n = 0) =>
  n >= 1099511627776
    ? (n / 1099511627776).toFixed(2) + " TB"
    : n >= 1073741824
      ? (n / 1073741824).toFixed(2) + " GB"
      : n >= 1048576
        ? (n / 1048576).toFixed(1) + " MB"
        : (n / 1024).toFixed(1) + " KB";
const duration = (n = 0) =>
  n ? Math.floor(n / 60) + ":" + String(n % 60).padStart(2, "0") : "--:--";
const localTime = (raw = "") => {
  if (!raw) return "时间未知";
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) return raw;
  try {
    return new Intl.DateTimeFormat("zh-CN", {
      timeZone: data.value.timezone || "Asia/Shanghai",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    })
      .format(date)
      .replaceAll("/", "-");
  } catch {
    return date.toLocaleString("zh-CN", { hour12: false });
  }
};
const resolution = (v: Video) =>
  v.height ? (v.height >= 2160 ? "2160p / 4K" : v.height + "p") : "";
const mediaSummary = (v: Video) =>
  [resolution(v), v.video_codec, v.bit_depth ? v.bit_depth + "-bit" : ""]
    .filter(Boolean)
    .join(" · ");
const pct = (v: Video) =>
  v.total_bytes
    ? Math.min(100, Math.round((v.bytes_downloaded / v.total_bytes) * 100))
    : 0;
async function api(path: string, init?: RequestInit) {
  return requestJSON<any>(path, init, auth.value.csrf_token, () => {
    auth.value = { configured: true, authenticated: false };
    events?.close();
  });
}
async function loadAuth() {
  try {
    auth.value = await api("/api/auth/status");
    authUsername.value = auth.value.username || "";
  } finally {
    authReady.value = true;
  }
}
async function submitAuth() {
  authError.value = "";
  if (
    !auth.value.configured &&
    authPassword.value !== authPasswordConfirm.value
  ) {
    authError.value = "两次输入的密码不一致";
    return;
  }
  busy.value = true;
  try {
    auth.value = await api(
      auth.value.configured ? "/api/auth/login" : "/api/auth/setup",
      {
        method: "POST",
        body: JSON.stringify({
          username: authUsername.value,
          password: authPassword.value,
        }),
      },
    );
    authPassword.value = "";
    authPasswordConfirm.value = "";
    await startApplication();
    await changeView(viewFromPath(), false);
  } catch (e) {
    authError.value = String(e).replace(/^Error:\s*/, "");
  } finally {
    busy.value = false;
  }
}
async function logout() {
  try {
    await api("/api/auth/logout", { method: "POST" });
  } finally {
    events?.close();
    auth.value = { configured: true, authenticated: false };
    authPassword.value = "";
    data.value = empty;
  }
}
const coverFallback =
  "data:image/svg+xml;charset=utf-8," +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="320" height="180"><rect width="100%" height="100%" fill="#17191f"/><text x="50%" y="50%" fill="#9199a7" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif" font-size="15">封面不可用</text></svg>',
  );
function onCoverError(e: Event, id?: string) {
  const img = e.target as HTMLImageElement;
  if (id) coverStates.value[id] = "error";
  if (img && img.src !== coverFallback) img.src = coverFallback;
}
type CoverState = "loading" | "loaded" | "error";
const coverStates = ref<Record<string, CoverState>>({});
function coverState(id: string): CoverState {
  return coverStates.value[id] || "loading";
}
function markCoverLoaded(id: string) {
  if (coverStates.value[id] !== "loaded") coverStates.value[id] = "loaded";
}
function registerCoverImg(id: string, el: unknown) {
  const img = el as HTMLImageElement | null;
  if (img && img.complete && img.naturalWidth > 0) markCoverLoaded(id);
}
async function refreshDashboard() {
  try {
    data.value = await api(
      "/api/dashboard?language=" +
        encodeURIComponent(translationLanguage.value),
    );
  } catch (e) {
    notice.value = String(e);
  }
}
async function loadNotifications() {
  const result = await api("/api/notifications");
  notifications.value = result.items;
  data.value.unread_notifications = result.unread;
}
async function markNotificationsRead() {
  await api("/api/notifications/read-all", { method: "POST" });
  data.value.unread_notifications = 0;
  await loadNotifications();
}
async function loadExclusions() {
  const result = await api("/api/media-exclusions?limit=200");
  exclusions.value = result.items || [];
  mediaExclusionTotal.value = result.total || 0;
}
async function restoreExclusion(item: Exclusion) {
  await act(async () => {
    await api("/api/media-exclusions/" + encodeURIComponent(item.video_id), { method: "DELETE" });
    exclusions.value = exclusions.value.filter((x) => x.video_id !== item.video_id);
    mediaExclusionTotal.value = Math.max(0, mediaExclusionTotal.value - 1);
    notice.value = "已恢复允许下载";
  });
}
async function loadIntegrity() {
  integrityBusy.value = true;
  try {
    mediaScan.value = await api("/api/media-integrity/scan");
    await loadDeletionPlans();
  } catch (e) {
    integrityNotice.value = String(e);
  } finally {
    integrityBusy.value = false;
  }
}
async function loadDeletionPlans() {
  try {
    deletionPlans.value = (await api("/api/media-integrity/plans")).items || [];
  } catch {}
}
async function previewTransRunnerPlan() {
  if (!planText.value.trim()) {
    planNotice.value = "请粘贴 TransRunner 删除计划 JSON";
    return;
  }
  planBusy.value = true;
  try {
    planPreviewData.value = await api("/api/media-integrity/plan/preview", {
      method: "POST",
      body: JSON.stringify({ raw: planText.value }),
    });
    planResult.value = undefined;
    planNotice.value = "";
  } catch (e) {
    planNotice.value = String(e);
  } finally {
    planBusy.value = false;
  }
}
async function executeTransRunnerPlan(confirm: boolean) {
  planBusy.value = true;
  try {
    const result = await api("/api/media-integrity/plan/execute", {
      method: "POST",
      body: JSON.stringify({ raw: planText.value, confirm }),
    });
    planResult.value = result;
    planNotice.value = confirm
      ? `已清理 ${result.executed}，跳过 ${result.skipped}，失败 ${result.failed}，需恢复 ${result.needs_recovery}`
      : `Dry run：${planPreviewData.value?.ready ?? 0} 项可清理，预计释放 ${size(result.estimated_freed)}`;
    await loadDeletionPlans();
    if (confirm) {
      await loadIntegrity();
      await loadExclusions();
      await previewTransRunnerPlan();
    }
  } catch (e) {
    planNotice.value = String(e);
  } finally {
    planBusy.value = false;
  }
}
async function recoverPlan(planId: string) {
  planBusy.value = true;
  try {
    const result = await api(
      "/api/media-integrity/plan/" + encodeURIComponent(planId) + "/recover",
      { method: "POST" },
    );
    planNotice.value = `恢复：完成 ${result.executed}，需恢复 ${result.needs_recovery}`;
    await loadDeletionPlans();
    await loadIntegrity();
  } catch (e) {
    planNotice.value = String(e);
  } finally {
    planBusy.value = false;
  }
}
async function importDeletionList() {
  if (!deletionListText.value.trim()) {
    integrityNotice.value = "请粘贴 VDF 删除清单 JSON";
    return;
  }
  integrityBusy.value = true;
  try {
    const result = await api("/api/media-integrity/import", {
      method: "POST",
      body: deletionListText.value,
    });
    if (result && result.format === "vdf_gui") {
      vdfPreviewData.value = result;
      cleanupPreview.value = undefined;
    } else {
      cleanupPreview.value = result;
      vdfPreviewData.value = undefined;
    }
    integrityNotice.value = "";
  } catch (e) {
    integrityNotice.value = String(e);
  } finally {
    integrityBusy.value = false;
  }
}
async function vdfExecute(confirm: boolean) {
  integrityBusy.value = true;
  try {
    const result = await api("/api/media-integrity/vdf/execute", {
      method: "POST",
      body: JSON.stringify({ raw: deletionListText.value, confirm }),
    });
    integrityNotice.value = confirm
      ? `VDF 清理：执行 ${result.executed}，跳过 ${result.skipped}，失败 ${result.failed}，需恢复 ${result.needs_recovery}`
      : `VDF Dry run：${vdfPreviewData.value?.ready_groups ?? 0} 组可清理，预计释放 ${size(result.estimated_freed)}`;
    if (confirm) {
      await loadIntegrity();
      await loadExclusions();
      await importDeletionList();
    }
  } catch (e) {
    integrityNotice.value = String(e);
  } finally {
    integrityBusy.value = false;
  }
}
async function applyDeletionList(confirm: boolean) {
  integrityBusy.value = true;
  try {
    const result = await api("/api/media-integrity/apply", {
      method: "POST",
      body: JSON.stringify({ raw: deletionListText.value, confirm }),
    });
    integrityNotice.value = confirm
      ? `已清理 ${result.executed} 项，跳过 ${result.skipped}，失败 ${result.failed}`
      : `Dry run：${result.candidates} 项可加入排除`;
    await loadIntegrity();
    await importDeletionList();
    if (confirm) await loadExclusions();
  } catch (e) {
    integrityNotice.value = String(e);
  } finally {
    integrityBusy.value = false;
  }
}
function openDelete(video: Video) {
  deleteTarget.value = video;
  deleteMode.value = "delete_and_exclude";
}
async function confirmDelete() {
  const target = deleteTarget.value;
  if (!target || deleting.value) return;
  deleting.value = true;
  try {
    await api("/api/videos/" + encodeURIComponent(target.video_id) + "/delete", {
      method: "POST",
      body: JSON.stringify({ mode: deleteMode.value }),
    });
    videos.value = videos.value.filter((v) => v.video_id !== target.video_id);
    if (selected.value && selected.value.video_id === target.video_id) selected.value = undefined;
    const mode = deleteMode.value;
    deleteTarget.value = undefined;
    notice.value = mode === "delete_and_exclude" ? "已删除并加入已排除" : "已删除媒体";
    await refreshDashboard();
  } catch (e) {
    notice.value = "删除失败：" + String(e);
  } finally {
    deleting.value = false;
  }
}
async function previewExclusions() {
  await act(async () => {
    const result = await api("/api/exclusions/preview", { method: "POST", body: JSON.stringify(settings.value) });
    exclusionPreview.value = result.items;
    exclusionTotal.value = result.total;
    notice.value = `当前规则会排除 ${result.total} 项`;
  });
}
async function loadVideos() {
	const requestID = ++videoRequestID;
	videoAbort?.abort();
	videoAbort = new AbortController();
  const p = new URLSearchParams({
    limit: "48",
    offset: String(page.value * 48),
    language: translationLanguage.value,
  });
  if (query.value) p.set("q", query.value);
  if (status.value) p.set("status", status.value);
  if (view.value === "library") {
    p.set("scope", searchScope.value);
    p.set("tag_mode", tagMode.value);
    p.set("sort", sortBy.value);
    p.set("order", sortOrder.value);
    if (selectedTags.value.length) p.set("tags", selectedTags.value.join(","));
    if (excludedTags.value.length)
      p.set("exclude_tags", excludedTags.value.join(","));
    if (creatorFilter.value) p.set("creator", creatorFilter.value);
    if (codecFilter.value) p.set("codec", codecFilter.value);
    if (minHeight.value) p.set("min_height", String(minHeight.value));
  }
	try {
		const result = await api(
			(view.value === "downloads" ? "/api/tasks?" : "/api/videos?") + p,
			{ signal: videoAbort.signal },
		);
		if (requestID !== videoRequestID) return;
		videos.value = result.items;
		total.value = result.total;
		if (view.value === "downloads")
			rateLimits.value = result.rate_limits || [];
	} catch (error) {
		if (error instanceof DOMException && error.name === "AbortError") return;
		throw error;
	}
}
async function loadFacets() {
  if (!facets.value.tags.length) facets.value = await api("/api/videos/facets");
}
async function loadSettings() {
  settings.value = await api("/api/settings");
  await loadBackups();
}
async function loadBackups() {
  const result = await api("/api/backups");
  backups.value = result.items;
  backupDirectory.value = result.directory;
  backupSensitive.value = result.sensitive;
}
async function createBackup() {
  await act(async () => {
    await api("/api/backups", { method: "POST" });
    notice.value = "数据库备份已创建";
    await loadSettings();
  });
}
async function restoreBackup(name: string) {
  if (
    !confirm("确认使用 " + name + " 恢复数据库？下载会暂停，服务随后自动重启。")
  )
    return;
  await act(async () => {
    await api("/api/backups/" + encodeURIComponent(name) + "/restore", {
      method: "POST",
      body: JSON.stringify({ confirm: true }),
    });
    notice.value = "正在恢复数据库并重启服务…";
  });
}
async function changeView(next: string, updateHistory = true) {
  if (!viewRoutes[next]) next = "home";
  view.value = next;
  revealGroupFor(next);
  document.title = (viewTitles[next] || "HWatch") + " · HWatch";
  if (updateHistory && window.location.pathname !== viewRoutes[next])
    window.history.pushState({ view: next }, "", viewRoutes[next]);
  if (window.matchMedia("(max-width: 680px)").matches)
    setSidebarCollapsed(true);
  page.value = 0;
  query.value = "";
  if (next === "library") {
    status.value = "completed";
    await loadFacets();
  } else if (next === "undownloaded") status.value = "discovered";
  else if (next === "downloads") {
    status.value = "";
    showCompletedTasks.value = false;
  }
  if (["library", "undownloaded", "downloads"].includes(next))
    await loadVideos();
  if (next === "settings") await loadSettings();
  if (next === "notifications") await loadNotifications();
  if (next === "excluded") await loadExclusions();
  if (next === "integrity") await loadIntegrity();
}
async function act(run: () => Promise<void>) {
  busy.value = true;
  try {
    await run();
  } catch (e) {
    notice.value = String(e);
  } finally {
    busy.value = false;
  }
}
async function subscribe() {
  await act(async () => {
    const r = await api("/api/subscriptions", {
      method: "POST",
      body: JSON.stringify({
        url: modelURL.value,
        download_existing: downloadExisting.value,
      }),
    });
    notice.value =
      "已建立 " +
      r.slug +
      " 基线，共 " +
      r.baselined +
      " 个视频" +
      (r.queued ? "，已排队 " + r.queued + " 个" : "");
    modelURL.value = "";
    creatorQuery.value = "";
    creatorResults.value = [];
    selectedCreator.value = undefined;
    downloadExisting.value = false;
    await refreshDashboard();
  });
}
async function searchCreators() {
  if (creatorTimer) clearTimeout(creatorTimer);
  creatorTimer = undefined;
  const searchQuery = creatorQuery.value.trim();
  if ([...searchQuery].length < 2) {
    creatorResults.value = [];
    searchingCreators.value = false;
    return;
  }
  const searchID = ++creatorSearchID;
  searchingCreators.value = true;
  try {
    const result = await api(
      "/api/creators/search?platform=rule34video&q=" +
        encodeURIComponent(searchQuery),
    );
    if (searchID === creatorSearchID) creatorResults.value = result.items;
  } catch (e) {
    if (searchID === creatorSearchID) notice.value = String(e);
  } finally {
    if (searchID === creatorSearchID) searchingCreators.value = false;
  }
}
function chooseCreator(creator: Creator) {
  selectedCreator.value = creator;
  modelURL.value = creator.url;
}
async function check(slug = "") {
  await act(async () => {
    await api("/api/check" + (slug ? "/" + slug : ""), { method: "POST" });
    notice.value = "检查任务已开始";
    await changeView("downloads");
  });
}
async function toggle(sub: Sub) {
  await api("/api/subscriptions/" + sub.slug, {
    method: "PATCH",
    body: JSON.stringify({ enabled: !sub.enabled }),
  });
  await refreshDashboard();
}
async function removeSub(sub: Sub) {
  if (
    !confirm(
      "删除 " +
        sub.platform +
        " 的 “" +
        sub.name +
        "” 订阅及其数据库记录？进行中的下载会停止；已下载文件仍保留在磁盘，但不会继续显示在影视库。此操作无法撤销。",
    )
  )
    return;
  await api("/api/subscriptions/" + sub.slug, { method: "DELETE" });
  await refreshDashboard();
}
async function videoAction(v: Video, action: string) {
  if (
    action === "delete" &&
    !confirm(
      v.status === "completed"
        ? "确认删除“" + displayTitle(v) + "”的媒体文件、NFO、海报和下载记录？此操作无法撤销。"
        : "确认删除“" + displayTitle(v) + "”的下载任务与临时文件？此操作无法撤销。",
    )
  )
    return;
  await act(async () => {
    await api("/api/videos/" + v.video_id + "/" + action, { method: "POST" });
    notice.value = "操作已提交";
    await loadVideos();
  });
}
async function downloadAll() {
  if (
    !confirm(
      "确认下载全部 " +
        total.value +
        " 个未下载视频？任务会在后台逐条加入下载队列。",
    )
  )
    return;
  await act(async () => {
    const r = await api("/api/videos/download-all", { method: "POST" });
    notice.value = "正在后台建立 " + r.accepted + " 个下载任务";
    await changeView("downloads");
  });
}
async function toggleDownloads() {
  const paused = !data.value.downloads_paused;
  await act(async () => {
    await api("/api/downloads/toggle", {
      method: "POST",
      body: JSON.stringify({ paused }),
    });
    data.value.downloads_paused = paused;
    notice.value = paused ? "已暂停全部下载" : "已继续全部下载";
    await loadVideos();
  });
}
async function saveSettings() {
  await act(async () => {
    settings.value = await api("/api/settings", {
      method: "PUT",
      body: JSON.stringify(settings.value),
    });
    notice.value = "设置已保存";
    await refreshDashboard();
  });
}
async function importPage() {
  await act(async () => {
    await api("/api/import/html", {
      method: "POST",
      body: JSON.stringify({ html: importHTML.value }),
    });
    notice.value = "HTML 数据已导入";
    importHTML.value = "";
    await loadVideos();
  });
}
function exportTranslation() {
  window.location.href =
    "/api/translations/export?language=" +
    encodeURIComponent(translationLanguage.value);
}
async function importTranslation(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  await act(async () => {
    const result = await api("/api/translations/import", {
      method: "POST",
      body: await file.text(),
    });
    notice.value =
      "已导入 " +
      result.imported_fields +
      " 个翻译字段，跳过 " +
      result.skipped_items +
      " 条";
    (event.target as HTMLInputElement).value = "";
  });
}
async function refreshNFO() {
  if (
    !confirm(
      "使用 " +
        translationLanguage.value +
        " 译文刷新全部已下载视频的 Plex NFO？",
    )
  )
    return;
  await act(async () => {
    const result = await api("/api/plex/refresh-nfo", {
      method: "POST",
      body: JSON.stringify({ language: translationLanguage.value }),
    });
    notice.value = "正在后台刷新 " + result.accepted + " 个 NFO 文件";
  });
}
function privateID(v: Video) {
  return "媒体项目 " + v.video_id.split(":").pop()!.slice(-4).toUpperCase();
}
function displayTitle(v: Video) {
  if (demoMode.value === 2) return privateID(v);
  return !showOriginal.value && v.translated_title
    ? v.translated_title
    : v.title;
}
function displaySubtitle(v: Video, fallback = "本地媒体") {
  return demoMode.value === 2
    ? v.status === "completed"
      ? "已下载 · 本地媒体"
      : v.status === "active"
        ? "下载中 · " + v.connections + " 个连接"
        : v.status === "queued" || v.status === "waiting"
          ? "等待下载 · 已加入队列"
          : v.status === "submitting"
            ? "正在提交 · 已持久化任务"
          : fallback
    : fallback;
}
function displayTags(v: Video) {
  if (demoMode.value === 2) return [];
  return !showOriginal.value && v.translated_tags?.length
    ? v.translated_tags
    : v.tags.map((t) => t.Name || "").filter(Boolean);
}
function toggleOriginal() {
  showOriginal.value = !showOriginal.value;
  localStorage.setItem(
    "r34watch-show-original",
    showOriginal.value ? "1" : "0",
  );
}
function toggleDemo() {
  demoMode.value = (demoMode.value + 1) % 3;
  localStorage.setItem("r34watch-demo-mode", String(demoMode.value));
}
function toggleSidebar() {
  setSidebarCollapsed(!sidebarCollapsed.value);
}
function setSidebarCollapsed(collapsed: boolean) {
  sidebarCollapsed.value = collapsed;
  localStorage.setItem(
    "hwatch-sidebar-collapsed",
    collapsed ? "1" : "0",
  );
}
function updateSidebarItem(
  item: keyof typeof sidebarItems.value,
  value: boolean,
) {
  sidebarItems.value[item] = value;
  localStorage.setItem("hwatch-sidebar-" + item, value ? "1" : "0");
}
function mergeTasks(payload: {
  tasks: Video[];
  download_speed: number;
  connections: number;
}) {
  for (const task of payload.tasks) {
    const i = videos.value.findIndex((v) => v.video_id === task.video_id);
    if (i >= 0)
      videos.value[i] = {
        ...task,
        translated_title: videos.value[i].translated_title,
        translated_tags: videos.value[i].translated_tags,
        translation_language: videos.value[i].translation_language,
      };
  }
  data.value.counts.active = payload.tasks.filter(
    (v) => v.status === "active",
  ).length;
  data.value.counts.waiting = payload.tasks.filter(
    (v) => v.status === "waiting",
  ).length;
  data.value.current_download_speed = payload.download_speed || 0;
  data.value.current_connections = payload.connections || 0;
}
let refreshTimer: number | undefined;
let countdownTimer: number | undefined;
let events: EventSource | undefined;
let creatorTimer: number | undefined;
let creatorSearchID = 0;
let videoLoadTimer: number | undefined;
let videoAbort: AbortController | undefined;
let videoRequestID = 0;
const handlePopState = () => {
  if (auth.value.authenticated) void changeView(viewFromPath(), false);
};
const handleEscape = (event: KeyboardEvent) => {
  if (event.key !== "Escape") return;
  if (selected.value) selected.value = undefined;
  else if (!sidebarCollapsed.value) setSidebarCollapsed(true);
};
const handleVisibility = () => {
  if (!document.hidden && auth.value.authenticated) void refreshDashboard();
};
async function startApplication() {
  await refreshDashboard();
  events?.close();
  events = new EventSource("/api/events");
  events.onmessage = (e) => {
    try {
      mergeTasks(JSON.parse(e.data));
    } catch {}
  };
  if (refreshTimer) clearInterval(refreshTimer);
  refreshTimer = window.setInterval(() => {
    if (document.hidden) return;
    void refreshDashboard();
    if (view.value === "downloads") void loadVideos();
  }, 60000);
  if (countdownTimer) clearInterval(countdownTimer);
  countdownTimer = window.setInterval(() => {
    if (view.value === "downloads") nowMs.value = Date.now();
  }, 1000);
}
onMounted(async () => {
  window.addEventListener("popstate", handlePopState);
  window.addEventListener("keydown", handleEscape);
  document.addEventListener("visibilitychange", handleVisibility);
  await loadAuth();
  if (auth.value.authenticated) {
    await startApplication();
    await changeView(viewFromPath(), false);
  }
});
onBeforeUnmount(() => {
  window.removeEventListener("popstate", handlePopState);
  window.removeEventListener("keydown", handleEscape);
  document.removeEventListener("visibilitychange", handleVisibility);
  events?.close();
  if (refreshTimer) clearInterval(refreshTimer);
  if (countdownTimer) clearInterval(countdownTimer);
  if (creatorTimer) clearTimeout(creatorTimer);
  if (videoLoadTimer) clearTimeout(videoLoadTimer);
  videoAbort?.abort();
});
watch(creatorQuery, () => {
  if (creatorTimer) clearTimeout(creatorTimer);
  creatorSearchID++;
  selectedCreator.value = undefined;
  modelURL.value = "";
  const searchable = [...creatorQuery.value.trim()].length >= 2;
  searchingCreators.value = searchable;
  if (!searchable) {
    creatorResults.value = [];
    return;
  }
  creatorTimer = window.setTimeout(searchCreators, 450);
});
watch(
  [
    query,
    status,
    translationLanguage,
    searchScope,
    tagMode,
    selectedTags,
    excludedTags,
    creatorFilter,
    codecFilter,
    minHeight,
    sortBy,
    sortOrder,
  ],
  () => {
    page.value = 0;
    localStorage.setItem("r34watch-language", translationLanguage.value);
    if (["library", "undownloaded", "downloads"].includes(view.value)) {
      if (videoLoadTimer) clearTimeout(videoLoadTimer);
      videoLoadTimer = window.setTimeout(() => void loadVideos(), 300);
    }
    if (view.value === "home") refreshDashboard();
  },
);
</script>
<template>
  <div v-if="!authReady" class="auth-screen">
    <div class="auth-card">
      <div class="auth-logo">H</div>
      <p>正在载入 HWatch…</p>
    </div>
  </div>
  <div v-else-if="!auth.authenticated" class="auth-screen">
    <form class="auth-card" @submit.prevent="submitAuth">
      <div class="auth-logo">H</div>
      <h1>{{ auth.configured ? "登录 HWatch" : "创建管理员账号" }}</h1>
      <p>
        {{
          auth.configured
            ? "登录后管理订阅、下载与媒体库。"
            : "这是首次启动。该账号将拥有 HWatch 的管理权限。"
        }}
      </p>
      <label
        >用户名<input
          v-model.trim="authUsername"
          autocomplete="username"
          minlength="3"
          maxlength="64"
          required
          autofocus /></label
      ><label
        >密码<input
          v-model="authPassword"
          type="password"
          :autocomplete="auth.configured ? 'current-password' : 'new-password'"
          minlength="10"
          maxlength="256"
          required /></label
      ><label v-if="!auth.configured"
        >确认密码<input
          v-model="authPasswordConfirm"
          type="password"
          autocomplete="new-password"
          minlength="10"
          maxlength="256"
          required
      /></label>
      <p v-if="authError" class="auth-error">{{ authError }}</p>
      <button class="primary" :disabled="busy" type="submit">
        {{ busy ? "请稍候…" : auth.configured ? "登录" : "创建并进入" }}</button
      ><small v-if="!auth.configured"
        >密码至少 10 个字符。账号信息仅保存在本机 SQLite 数据库中。</small
      >
    </form>
  </div>
  <div
    v-if="auth.authenticated"
    class="shell"
    :class="{
      'demo-mode': demoMode > 0,
      'strong-demo': demoMode === 2,
      'sidebar-collapsed': sidebarCollapsed,
    }"
  >
    <button
      class="sidebar-toggle"
      :aria-label="sidebarCollapsed ? '展开侧栏' : '收起侧栏'"
      v-on:click="toggleSidebar"
    >
      <NavIcon name="menu" />
    </button>
    <button
      v-if="!sidebarCollapsed"
      class="sidebar-backdrop"
      aria-label="关闭侧栏"
      @click="setSidebarCollapsed(true)"
    ></button>
    <aside>
      <div class="brand"><span>H</span><b>HWatch</b></div>
      <nav>
        <template v-for="group in navGroups" :key="group.id">
          <button
            v-if="group.label"
            type="button"
            class="nav-group"
            :aria-expanded="openGroups[group.id] !== false"
            @click="toggleGroup(group.id)"
          >
            <span>{{ group.label }}</span
            ><span class="nav-group-caret" :class="{ open: openGroups[group.id] !== false }">⌄</span>
          </button>
          <template v-if="openGroups[group.id] !== false">
            <button
              v-for="item in group.items"
              :key="item.id"
              :class="{ active: view === item.id }"
              @click="changeView(item.id)"
            >
              <NavIcon :name="item.id" /><span class="nav-text">{{
                item.label
              }}</span
              ><i
                v-if="
                  item.id === 'downloads' &&
                  data.counts.active + data.counts.waiting
                "
                >{{ data.counts.active + data.counts.waiting }}</i
              >
              <i v-if="item.id === 'notifications' && data.unread_notifications">{{
                data.unread_notifications
              }}</i>
            </button>
          </template>
        </template>
      </nav>
      <div class="bottom">
        <label v-if="sidebarItems.language" class="sidebar-language"
          ><NavIcon name="language" /><select
            class="language-select"
            v-model="translationLanguage"
            aria-label="内容语言"
          >
            <option value="zh-CN">简中</option>
            <option value="zh-TW">繁中</option>
            <option value="en">EN</option>
            <option value="ja">日本語</option>
          </select></label
        ><button v-if="sidebarItems.original" v-on:click="toggleOriginal">
          <NavIcon name="privacy" /><span>{{
            showOriginal ? "显示译文" : "显示原文"
          }}</span></button
        ><button
          v-if="sidebarItems.demo"
          :class="{ active: demoMode }"
          v-on:click="toggleDemo"
        >
          <NavIcon name="demo" /><span>{{
            demoMode === 0
              ? "开启演示模式"
              : demoMode === 1
                ? "进入强演示模式"
                : "关闭演示模式"
          }}</span></button
        ><button
          :class="{ active: view === 'settings' }"
          v-on:click="changeView('settings')"
        >
          <NavIcon name="settings" /><span>设置</span></button
        ><small v-if="sidebarItems.aria" :class="{ online: data.aria2_online }"
          >● aria2 {{ data.aria2_online ? "在线" : "离线" }}</small
        >
        <div class="sidebar-account">
          <span><small>当前账户</small><b>{{ auth.username }}</b></span>
          <button class="sidebar-logout" aria-label="退出登录" title="退出登录" @click="logout">
            <NavIcon name="logout" /><span>退出登录</span>
          </button>
        </div>
      </div>
    </aside>
    <main>
      <header>
        <div>
          <p class="eyebrow">{{ view.toUpperCase() }}</p>
          <h1>
            {{ viewTitles[view] || "HWatch" }}
          </h1>
        </div>
        <button
          v-if="view === 'undownloaded'"
          class="primary"
          :disabled="busy || !total"
          @click="downloadAll"
        >
          下载全部（{{ total }}）</button
        ><button
          v-else-if="view === 'downloads'"
          class="download-master"
          :class="{ paused: data.downloads_paused }"
          :disabled="busy"
          @click="toggleDownloads"
        >
          {{ data.downloads_paused ? "继续全部" : "暂停全部" }}</button
        ><button
          v-else-if="
            ![
              'settings',
              'translations',
              'notifications',
              'excluded',
              'integrity',
            ].includes(view)
          "
          class="primary"
          :disabled="busy"
          @click="check()"
        >
          检查更新
        </button>
      </header>
      <button v-if="notice" class="notice" role="status" @click="notice = ''">{{ notice }}</button>
      <div v-if="data.health_warnings.length" class="health-alert">
        <b>系统运行提醒</b
        ><span
          v-for="warning in data.health_warnings"
          :key="warning.code"
          :class="warning.severity"
          >{{ warning.message }}</span
        >
      </div>
      <section v-if="view === 'home'">
        <div class="stats">
          <article>
            <small>订阅作者</small><b>{{ data.subscriptions.length }}</b
            ><span
              >启用
              {{ data.subscriptions.filter((s) => s.enabled).length }}</span
            >
          </article>
          <article>
            <small>下载任务</small
            ><b>{{ data.counts.active + data.counts.waiting }}</b
            ><span>{{ data.counts.active }} 个进行中</span>
          </article>
          <article>
            <small>{{ downloadedStat.label }}</small
            ><b>{{ data.downloaded_videos }}</b
            ><span>{{ downloadedStat.sub }}</span>
          </article>
          <article>
            <small>累计流量</small><b>{{ size(data.traffic_bytes) }}</b
            ><span>下载器接收流量</span>
          </article>
          <article>
            <small>影视库</small><b>{{ size(data.library_bytes) }}</b
            ><span>{{ data.counts.videos }} 个记录</span>
          </article>
        </div>
        <div class="storage">
          <div>
            <b>NAS 存储空间</b
            ><span
              >可用 {{ size(data.disk_free) }} / 总计
              {{ size(data.disk_total) }}</span
            >
          </div>
          <progress
            :value="data.disk_total - data.disk_free"
            :max="data.disk_total || 1"
          />
        </div>
        <h2>最近入库</h2>
        <div class="library compact">
          <article
            v-for="v in data.videos"
            :key="v.video_id"
            role="button"
            tabindex="0"
            v-on:click="selected = v"
            @keydown.enter="selected = v"
            @keydown.space.prevent="selected = v"
          >
            <img :src="'/thumb/' + v.video_id" :alt="displayTitle(v)" loading="lazy" />
            <div class="meta">
              <b>{{ displayTitle(v) }}</b
              ><small>{{
                displaySubtitle(v, v.subscription_slug + " · " + v.status)
              }}</small>
            </div>
          </article>
        </div>
        <h2 class="recent-heading">最近播放</h2>
        <div class="library compact">
          <article
            v-for="v in data.recently_played"
            :key="v.video_id"
            role="button"
            tabindex="0"
            v-on:click="selected = v"
            @keydown.enter="selected = v"
            @keydown.space.prevent="selected = v"
          >
            <img :src="'/thumb/' + v.video_id" :alt="displayTitle(v)" loading="lazy" />
            <div class="meta">
              <b>{{ displayTitle(v) }}</b
              ><small>{{
                displaySubtitle(v, v.creator_name || v.subscription_slug)
              }}</small>
            </div>
          </article>
          <div v-if="!data.recently_played.length" class="empty">
            还没有播放记录
          </div>
        </div>
      </section>
      <section v-else-if="view === 'subscriptions'">
        <div class="panel creator-search">
          <h2>通过链接订阅</h2>
          <div class="inline subscription-url">
            <input
              v-model="modelURL"
              type="url"
              autocomplete="off"
              placeholder="粘贴作者页或播放清单链接"
            /><button
              class="primary"
              :disabled="busy || !modelURL.trim()"
              @click="subscribe"
            >
              订阅
            </button>
          </div>
          <label class="download-existing"
            ><input v-model="downloadExisting" type="checkbox" />
            下载全部现有视频</label
          >
          <small>支持 Rule34Video 作者页，以及 Hanime1 用户上传页和播放清单。</small>
          <h2 class="search-heading">搜索 Rule34Video 作者</h2>
          <div class="inline">
            <input
              v-model="creatorQuery"
              type="search"
              autocomplete="off"
              placeholder="输入作者名称，例如 Akt"
            /><button
              :disabled="searchingCreators || creatorQuery.trim().length < 2"
              @click="searchCreators"
            >
              {{ searchingCreators ? "搜索中…" : "搜索" }}
            </button>
          </div>
          <small v-if="creatorQuery.trim().length === 1"
            >至少输入 2 个字符</small
          ><small v-else-if="searchingCreators">正在搜索作者并加载头像…</small>
          <div v-else-if="creatorResults.length" class="creator-results">
            <button
              v-for="creator in creatorResults"
              :key="creator.source_id"
              :class="{
                selected: selectedCreator?.source_id === creator.source_id,
              }"
              @click="chooseCreator(creator)"
            >
              <img
                v-if="creator.avatar_url"
                :src="creator.avatar_url"
                :alt="creator.name"
                loading="lazy"
              /><span v-else class="avatar">{{
                creator.name.slice(0, 1).toUpperCase()
              }}</span
              ><span
                ><b>{{ creator.name }}</b
                ><small>{{ creator.video_count }} 个视频</small></span
              >
            </button>
          </div>
          <div
            v-else-if="creatorQuery.trim().length >= 2 && !searchingCreators"
            class="empty compact-empty"
          >
            没有找到匹配作者
          </div>
          <div v-if="selectedCreator" class="selected-creator">
            <span
              ><small>已选择</small><b>{{ selectedCreator.name }}</b></span
            ><button class="primary" :disabled="busy" v-on:click="subscribe">
              订阅所选作者
            </button>
          </div>
          <small>{{
            downloadExisting
              ? "订阅后会把全部现有视频写入持久下载队列。"
              : "仅建立现有内容基线，之后发现的新视频会自动下载。"
          }}</small>
        </div>
        <div class="rows subscription-groups">
          <article v-for="group in subscriptionGroups" :key="group.key">
            <img
              v-if="group.avatar"
              :src="group.avatar.avatar_proxy_url"
              :alt="group.name"
              loading="lazy"
            />
            <div v-else class="avatar">
              {{ group.name.slice(0, 1).toUpperCase() }}
            </div>
            <div class="subscription-group-body">
              <div class="subscription-group-title">
                <b>{{ group.name }}</b
                ><small v-if="group.sources.length > 1"
                  >{{ group.sources.length }} 个平台来源</small
                >
              </div>
              <div
                v-for="sub in group.sources"
                :key="sub.slug"
                class="subscription-source"
              >
                <div class="subscription-source-info">
                  <span
                    class="subscription-kind"
                    :class="sub.subscription_type === 'playlist' ? 'series' : 'creator'"
                    >{{ subscriptionTypeLabel(sub.subscription_type) }}</span
                  ><span class="platform-badge">{{ sub.platform }}</span
                  ><a :href="sub.url" target="_blank" rel="noreferrer">查看来源</a>
                  <p :class="{ error: sub.last_error }">
                    {{
                      sub.last_error ||
                      "上次检查 " +
                        (sub.last_checked_at ? localTime(sub.last_checked_at) : "尚未")
                    }}
                  </p>
                  <div class="sub-progress">
                    <progress
                      :value="sub.progress.completed"
                      :max="sub.progress.total || 1"
                    ></progress
                    ><small
                      >已完成 {{ sub.progress.completed }} / {{ sub.progress.total }} ·
                      下载中 {{ sub.progress.active }} · 等待 {{ sub.progress.waiting
                      }}<span v-if="sub.progress.failed"> · 失败 {{ sub.progress.failed }}</span></small
                    >
                  </div>
                </div>
                <div class="actions">
                  <button @click="check(sub.slug)">检查</button
                  ><button @click="toggle(sub)">{{ sub.enabled ? "暂停" : "启用" }}</button
                  ><button class="danger" @click="removeSub(sub)">删除</button>
                </div>
              </div>
            </div>
          </article>
          <div v-if="!data.subscriptions.length" class="empty">还没有订阅</div>
        </div>
      </section>
      <section v-else-if="view === 'excluded'">
        <p class="muted">这些视频已被加入“已排除”，订阅或自动发现都不会重新下载。</p>
        <div class="rows">
          <article v-for="item in exclusions" :key="item.video_id">
            <span class="avatar">{{ (item.title_snapshot || item.video_id).slice(0, 1).toUpperCase() }}</span>
            <div>
              <b>{{ item.title_snapshot || item.video_id }}</b>
              <small>{{ item.source || '未知来源' }} · {{ item.video_id }} · {{ localTime(item.created_at) }}</small>
              <small>{{ item.reason || '用户删除并排除' }}</small>
            </div>
            <div class="actions">
              <button :disabled="busy" @click="restoreExclusion(item)">恢复允许下载</button>
            </div>
          </article>
          <div v-if="!exclusions.length" class="empty">没有已排除的视频</div>
        </div>
        <p v-if="mediaExclusionTotal" class="muted">共 {{ mediaExclusionTotal }} 项</p>
      </section>
      <section v-else-if="view === 'integrity'">
        <p class="muted">
          扫描已完成媒体文件完整性。两种清理模式：<b>模式一</b> 外部工具已删除、HWatch 核对后加入“已排除”；
          <b>模式二</b> 导入 TransRunner 删除计划，由 HWatch 将重复文件移入隔离区并更新数据库/排除。
          全部默认 Dry Run，需二次确认；缺失但无计划确认的文件只展示、不自动排除。
        </p>
        <div class="toolbar">
          <button class="primary" :disabled="integrityBusy" @click="loadIntegrity">
            扫描媒体库
          </button>
        </div>
        <div v-if="mediaScan" class="stats">
          <article><small>总媒体数</small><b>{{ mediaScan.total }}</b><span>已完成媒体</span></article>
          <article><small>正常文件</small><b>{{ mediaScan.present }}</b><span>存在且可读</span></article>
          <article><small>确认缺失</small><b>{{ mediaScan.missing }}</b><span>文件不存在</span></article>
          <article>
            <small>存储状态</small><b>{{ mediaScan.storage_available ? "正常" : "不可访问" }}</b
            ><span>{{ mediaScan.storage_available ? "已挂载" : "已阻止缺失判定" }}</span>
          </article>
          <article>
            <small>疑似移动/权限</small
            ><b>{{ mediaScan.moved + mediaScan.permission_denied }}</b><span>需人工确认</span>
          </article>
        </div>
        <div v-if="cleanupPreview" class="stats">
          <article><small>清单条数</small><b>{{ cleanupPreview.deletions }}</b><span>VDF deletions</span></article>
          <article><small>可加入排除</small><b>{{ cleanupPreview.candidates }}</b><span>已验证缺失</span></article>
          <article><small>已排除</small><b>{{ cleanupPreview.already_excluded }}</b><span>幂等跳过</span></article>
          <article><small>跳过</small><b>{{ cleanupPreview.skipped }}</b><span>非候选</span></article>
        </div>
        <div v-if="integrityNotice" class="notice" role="status">{{ integrityNotice }}</div>
        <div class="panel">
          <h2>导入 VDF 删除清单</h2>
          <textarea
            v-model="deletionListText"
            rows="6"
            placeholder='{"version":1,"deletions":[{"video_id":"rule34video:...","original_path":"...","reason":"duplicate","kept_video_id":"..."}]}'
          ></textarea>
          <div class="inline" style="margin-top: 10px">
            <button :disabled="integrityBusy" @click="importDeletionList">预览</button>
            <template v-if="vdfPreviewData">
              <button
                class="primary"
                :disabled="integrityBusy"
                @click="vdfExecute(false)"
              >
                VDF Dry Run
              </button>
              <button
                class="danger"
                :disabled="integrityBusy || !vdfPreviewData.ready_groups"
                @click="vdfExecute(true)"
              >
                二次确认并执行 VDF 清理
              </button>
            </template>
            <template v-else>
              <button
                class="primary"
                :disabled="integrityBusy || !cleanupPreview"
                @click="applyDeletionList(false)"
              >
                Dry Run
              </button>
              <button
                class="danger"
                :disabled="integrityBusy || !cleanupPreview || !cleanupPreview.candidates"
                @click="applyDeletionList(true)"
              >
                确认执行清理
              </button>
            </template>
          </div>
          <div v-if="vdfPreviewData" class="stats" style="margin-top: 12px">
            <article><small>VDF 组</small><b>{{ vdfPreviewData.groups }}</b><span>原生导出</span></article>
            <article><small>可执行组</small><b>{{ vdfPreviewData.ready_groups }}</b><span>尚未执行</span></article>
            <article><small>此前已处理</small><b>{{ vdfPreviewData.already_done_groups }}</b><span>已在排除中</span></article>
            <article><small>冲突组</small><b>{{ vdfPreviewData.conflict_groups }}</b><span>已阻止执行</span></article>
            <article><small>拒绝组</small><b>{{ vdfPreviewData.rejected_groups }}</b><span>无法安全匹配</span></article>
            <article><small>预计释放</small><b>{{ size(vdfPreviewData.estimated_freed) }}</b><span>可回收空间</span></article>
          </div>
          <div v-if="vdfPreviewData" class="stats" style="margin-top: 12px">
            <article>
              <small>删除项</small><b>{{ vdfPreviewData.remove_total }}</b>
              <span>已匹配 {{ vdfPreviewData.remove_matched }} · 已处理 {{ vdfPreviewData.remove_already_done }} · 无法匹配 {{ vdfPreviewData.remove_unmatched }}</span>
            </article>
            <article>
              <small>保留项</small><b>{{ vdfPreviewData.keep_total }}</b>
              <span>正常 {{ vdfPreviewData.keep_matched }} · 缺失/已删 {{ vdfPreviewData.keep_missing }}</span>
            </article>
            <article>
              <small>跨组冲突</small><b>{{ vdfPreviewData.cross_group_conflicts }}</b>
              <span>{{ vdfPreviewData.has_conflicts ? "整个计划已阻止" : "无" }}</span>
            </article>
          </div>
          <div v-if="vdfPreviewData && vdfPreviewData.conflicts.length" class="panel error" style="margin-top: 12px">
            <b>检测到跨组安全冲突，计划不可执行：</b>
            <p v-for="(cf, i) in vdfPreviewData.conflicts" :key="i">
              [{{ cf.kind }}] {{ cf.video_id || cf.path }} · 组 {{ cf.groups.join(", ") }} — {{ cf.detail }}
            </p>
          </div>
          <div v-if="vdfPreviewData" class="rows" style="margin-top: 12px">
            <article
              v-for="g in vdfPreviewData.items.slice(0, 100)"
              :key="g.group_id"
              class="vdf-group"
            >
              <div class="vdf-group-head">
                <b>{{ g.group_id }}</b>
                <span
                  class="integrity-badge"
                  :class="'integrity-' + vdfGroupBadgeClass(g.status)"
                  >{{ vdfGroupStatusLabel(g.status) }}</span
                >
                <span class="muted">{{ g.subscription_slug || "?" }} · 释放 {{ size(g.estimated_freed) }}</span>
              </div>
              <div class="vdf-items">
                <div>
                  <small class="error">删除 ({{ g.remove_items.length }})</small>
                  <p v-for="it in g.remove_items" :key="it.path" class="muted">
                    {{ it.match_status === "matched" || it.match_status === "absent" || it.match_status === "already_done" ? "✓" : "✗" }} {{ it.path }}
                    <em v-if="it.video_id">→ {{ it.video_id }} ({{ vdfItemStatusLabel(it.match_status) }})</em>
                    <em v-else>({{ vdfItemStatusLabel(it.match_status) }})</em>
                  </p>
                </div>
                <div>
                  <small>保留 ({{ g.keep_items.length }})</small>
                  <p v-for="it in g.keep_items" :key="it.path" class="muted">
                    {{ it.match_status === "matched" ? "✓" : "✗" }} {{ it.path }}
                    <em v-if="it.video_id">→ {{ it.video_id }}</em>
                  </p>
                </div>
              </div>
              <p v-if="g.problems.length" class="error">{{ g.problems.join("；") }}</p>
            </article>
          </div>
        </div>
        <div class="panel">
          <h2>模式二：导入 TransRunner 删除计划（HWatch 执行清理）</h2>
          <textarea
            v-model="planText"
            rows="6"
            placeholder='{"kind":"transrunner-duplicate-deletion-plan","plan_version":1,"scan_id":"...","subscription_slug":"rule34video:...","executable":true,"stale":false,"user_confirmed":true,"items":[{"video_id":"...","kept_video_id":"...","original_path":"...","expected_size":123,"expected_sha256":"..."}]}'
          ></textarea>
          <div class="inline" style="margin-top: 10px">
            <button :disabled="planBusy" @click="previewTransRunnerPlan">校验预览</button>
            <button
              class="primary"
              :disabled="planBusy || !planPreviewData"
              @click="executeTransRunnerPlan(false)"
            >
              Dry Run
            </button>
            <button
              class="danger"
              :disabled="planBusy || !planPreviewData || !planPreviewData.ready"
              @click="executeTransRunnerPlan(true)"
            >
              二次确认并执行清理
            </button>
          </div>
          <div v-if="planNotice" class="notice" role="status" style="margin-top: 10px">
            {{ planNotice }}
          </div>
          <div v-if="planPreviewData" class="stats" style="margin-top: 12px">
            <article><small>计划条数</small><b>{{ planPreviewData.total }}</b><span>{{ planPreviewData.subscription_slug }}</span></article>
            <article><small>可清理</small><b>{{ planPreviewData.ready }}</b><span>已验证一致</span></article>
            <article><small>已拒绝</small><b>{{ planPreviewData.rejected }}</b><span>校验不通过</span></article>
            <article><small>需恢复</small><b>{{ planPreviewData.needs_recovery }}</b><span>文件已不在</span></article>
            <article><small>预计释放</small><b>{{ size(planPreviewData.estimated_freed) }}</b><span>可回收空间</span></article>
          </div>
          <div v-if="planResult" class="muted" style="margin-top: 8px">
            {{ planResult.dry_run ? "Dry Run" : "已执行" }} · 清理 {{ planResult.executed }} · 跳过
            {{ planResult.skipped }} · 失败 {{ planResult.failed }} · 需恢复 {{ planResult.needs_recovery }}
          </div>
          <div v-if="planPreviewData" class="rows" style="margin-top: 12px">
            <article
              v-for="item in planPreviewData.items.slice(0, 200)"
              :key="item.video_id"
              class="integrity-row"
            >
              <span class="avatar">{{ item.video_id.slice(-1).toUpperCase() }}</span>
              <div>
                <b>{{ item.video_id }}</b>
                <small>保留: {{ item.kept_video_id }}</small>
                <small>{{ item.original_path }}</small>
                <small v-if="item.reason" class="error">{{ item.reason }}</small>
              </div>
              <span class="integrity-badge" :class="'integrity-' + item.status">{{
                planStatusLabel(item.status)
              }}</span>
            </article>
          </div>
          <div v-if="planPreviewData && !planPreviewData.valid" class="error" style="margin-top: 8px">
            计划不可执行：{{ planPreviewData.problems.join("；") }}
          </div>
        </div>
        <div v-if="deletionPlans.length" class="panel">
          <h2>删除计划执行记录</h2>
          <div class="rows">
            <article v-for="p in deletionPlans" :key="p.plan_id" class="integrity-row">
              <span class="avatar">{{ planStatusLabel(p.status).slice(0, 1) }}</span>
              <div>
                <b>{{ p.scan_id || p.plan_id }}</b>
                <small>{{ p.subscription_slug }}</small>
                <small>{{ localTime(p.created_at) }}</small>
              </div>
              <span class="actions">
                <button :disabled="planBusy" @click="recoverPlan(p.plan_id)">恢复/核对</button>
                <span class="integrity-badge" :class="'integrity-' + p.status">{{
                  planStatusLabel(p.status)
                }}</span>
              </span>
            </article>
          </div>
        </div>
        <div v-if="mediaScan" class="panel">
          <h2>缺失/异常列表</h2>
          <div class="rows">
            <article
              v-for="item in mediaScan.items.slice(0, 200)"
              :key="item.video_id"
              class="integrity-row"
            >
              <span class="avatar">{{ (item.title || item.video_id).slice(0, 1).toUpperCase() }}</span>
              <div>
                <b>{{ item.title || item.video_id }}</b>
                <small>{{ item.platform }} · {{ item.video_id }}</small>
                <small>{{ item.path || "（无路径）" }}</small>
                <small v-if="item.detail" class="error">{{ item.detail }}</small>
              </div>
              <span class="integrity-badge" :class="'integrity-' + item.status">{{
                cleanupStatusLabel(item.status)
              }}</span>
            </article>
            <div v-if="!mediaScan.items.length" class="empty">
              没有发现缺失或异常文件
            </div>
          </div>
        </div>
        <div v-if="cleanupPreview" class="panel">
          <h2>清单校验结果</h2>
          <div class="rows">
            <article
              v-for="item in cleanupPreview.items.slice(0, 200)"
              :key="item.video_id + item.status"
              class="integrity-row"
            >
              <span class="avatar">{{ (item.title || item.video_id).slice(0, 1).toUpperCase() }}</span>
              <div>
                <b>{{ item.title || item.video_id }}</b>
                <small>{{ item.video_id }}</small>
                <small v-if="item.path">{{ item.path }}</small>
                <small v-if="item.detail" class="error">{{ item.detail }}</small>
              </div>
              <span class="integrity-badge" :class="'integrity-' + item.status">{{
                cleanupStatusLabel(item.status)
              }}</span>
            </article>
          </div>
        </div>
      </section>
      <section v-else-if="view === 'notifications'">
        <div class="toolbar">
          <button
            :disabled="!data.unread_notifications"
            @click="markNotificationsRead"
          >
            全部标为已读</button
          ><span>{{ data.unread_notifications }} 条未读</span>
        </div>
        <div class="rows notification-list">
          <article
            v-for="item in notifications"
            :key="item.id"
            :class="{ unread: !item.read_at }"
          >
            <div class="notification-dot" :class="item.severity"></div>
            <div>
              <b>{{ item.message }}</b
              ><small>{{ localTime(item.created_at) }} · {{ item.code }}</small>
            </div>
          </article>
          <div v-if="!notifications.length" class="empty">暂无系统通知</div>
        </div>
      </section>
      <section v-else-if="view === 'downloads'">
        <div class="transfer-total">
          <div
            v-for="engine in data.downloaders"
            :key="engine.ID"
            class="downloader-total"
            :class="{ online: engine.Online }"
          >
            <small
              >{{ engine.Name
              }}{{
                engine.ID === data.downloader_mode ? " · 当前模式" : ""
              }}</small
            ><b
              >{{ engine.Online ? "在线" : "离线" }} ·
              {{ engine.Active || 0 }} 个任务</b
            ><span
              >{{ size(engine.DownloadSpeed || 0) }}/s ·
              {{ engine.Connections || 0 }} 个连接</span
            ><span v-if="engine.Error" class="error">{{ engine.Error }}</span>
          </div>
          <div>
            <small>总下载速度</small
            ><b>{{ size(data.current_download_speed) }}/s</b>
          </div>
          <div>
            <small>总连接数</small><b>{{ data.current_connections }}</b>
          </div>
        </div>
        <div class="toolbar">
          <select v-model="status">
            <option value="">全部状态</option>
            <option value="queued">正在建立任务</option>
            <option value="submitting">正在提交下载器</option>
            <option value="waiting">等待中</option>
            <option value="active">下载中</option>
            <option value="paused">已暂停</option>
            <option value="failed">失败</option>
            <option value="excluded">已排除</option>
            <option value="completed">已完成</option></select
          ><button @click="loadVideos">刷新</button>
        </div>
        <div v-if="activeRateLimits.length" class="rate-limit-banner">
          <b>{{ rlLabel.title }}</b>
          <div
            v-for="r in activeRateLimits"
            :key="r.source"
            class="rate-limit-row"
          >
            <span class="rate-limit-source">{{ sourceLabel(r.source) }}</span>
            <span class="rate-limit-countdown">{{
              fmtCountdown(r.remaining)
            }}</span>
            <span
              >{{ rlLabel.waiting }}: {{ r.waiting }}</span
            >
            <span class="rate-limit-reason">{{
              r.reason === "network" ? rlLabel.taskRowNetwork : rlLabel.taskRow
            }}</span
            >
          </div>
          <small>{{ rlLabel.other }}</small>
        </div>
        <div class="queue">
          <article v-for="v in unfinishedDownloadTasks" :key="v.video_id">
            <img :src="'/thumb/' + v.video_id" :alt="displayTitle(v)" loading="lazy" />
            <div>
              <div class="download-title">
                <b>{{ displayTitle(v) }}</b
                ><span>{{ pct(v) }}%</span>
              </div>
              <small
                >{{ size(v.bytes_downloaded) }} /
                {{ v.total_bytes ? size(v.total_bytes) : "等待获取大小" }} ·
                {{ size(v.download_speed) }}/s ·
                {{ v.connections }} 个连接</small
              ><progress
                :value="v.bytes_downloaded"
                :max="v.total_bytes || 1"
              />
              <p
                v-if="v.status === 'queued' && rateLimitedReason(v.platform)"
                class="rate-limit-hint"
              >
                {{
                  rateLimitedReason(v.platform) === "network"
                    ? rlLabel.taskRowNetwork
                    : rlLabel.taskRow
                }}
              </p>
              <p v-else-if="v.status === 'failed' && v.error" class="error">
                {{ v.error }}
              </p>
              <p
                v-else-if="v.status === 'queued' && v.error"
                class="last-error"
              >
                {{ rlLabel.lastError }}: {{ v.error }}
              </p>
            </div>
            <div class="actions">
              <button
                v-if="v.status === 'active' || v.status === 'waiting'"
                @click="videoAction(v, 'pause')"
              >
                暂停</button
              ><button
                v-if="v.status === 'paused'"
                @click="videoAction(v, 'resume')"
              >
                继续</button
              ><button
                v-if="v.status === 'failed' || v.status === 'excluded'"
                @click="videoAction(v, 'retry')"
              >
                {{ v.status === "excluded" ? "重新判断" : "重试" }}</button
              ><button class="danger" @click="videoAction(v, 'delete')">
                删除
              </button>
            </div>
          </article>
          <div
            v-if="!unfinishedDownloadTasks.length && !completedDownloadTasks.length"
            class="empty"
          >
            下载队列为空
          </div>
        </div>
        <button
          v-if="completedDownloadTasks.length"
          class="completed-tasks-toggle"
          :aria-expanded="showCompletedTasks"
          @click="showCompletedTasks = !showCompletedTasks"
        >
          <span><b>已完成任务</b><small>{{ completedDownloadTasks.length }} 项</small></span>
          <span>{{ showCompletedTasks ? "收起" : "展开" }} {{ showCompletedTasks ? "⌃" : "⌄" }}</span>
        </button>
        <div v-if="showCompletedTasks" class="queue completed-task-list">
          <article v-for="v in completedDownloadTasks" :key="v.video_id">
            <img :src="'/thumb/' + v.video_id" :alt="displayTitle(v)" loading="lazy" />
            <div>
              <div class="download-title">
                <b>{{ displayTitle(v) }}</b><span>100%</span>
              </div>
              <small>{{ v.total_bytes ? size(v.total_bytes) : "已完成" }}</small>
              <progress :value="1" :max="1" />
            </div>
            <div class="actions">
              <button class="danger" @click="videoAction(v, 'delete')">删除</button>
            </div>
          </article>
        </div>
      </section>
      <section v-else-if="view === 'library' || view === 'undownloaded'">
        <div class="toolbar library-toolbar">
          <input
            v-model="query"
            :type="demoMode === 2 ? 'password' : 'search'"
            placeholder="搜索标题、译文、作者或标签"
          /><button
            v-if="view === 'library'"
            :class="{ active: showFilters }"
            v-on:click="showFilters = !showFilters"
          >
            筛选</button
          ><select v-if="view === 'library'" v-model="sortBy">
            <option value="downloaded_at">最近下载</option>
            <option value="published_at">发布时间</option>
            <option value="views">播放量</option>
            <option value="duration">时长</option>
            <option value="size">文件大小</option>
            <option value="resolution">分辨率</option>
            <option value="bit_rate">码率</option>
            <option value="title">标题</option>
            <option value="recent_played">最近播放</option>
            <option value="play_count">播放次数</option></select
          ><select v-if="view === 'library'" v-model="sortOrder">
            <option value="desc">降序</option>
            <option value="asc">升序</option></select
          ><span
            >{{ total }} 个{{
              view === "library" ? "已下载" : "未下载"
            }}视频</span
          >
        </div>
        <div
          v-if="view === 'library' && showFilters"
          class="panel filter-panel"
        >
          <label
            >搜索范围<select v-model="searchScope">
              <option value="all">原文和译文</option>
              <option value="translated">仅当前语言译文</option>
              <option value="original">仅原文</option>
            </select></label
          ><label
            >作者<select v-model="creatorFilter">
              <option value="">全部作者</option>
              <option
                v-for="item in facets.creators"
                :key="item.value"
                :value="item.value"
              >
                {{ item.value }}（{{ item.count }}）
              </option>
            </select></label
          ><label
            >编码<select v-model="codecFilter">
              <option value="">全部编码</option>
              <option
                v-for="item in facets.codecs"
                :key="item.value"
                :value="item.value"
              >
                {{ item.value }}（{{ item.count }}）
              </option>
            </select></label
          ><label
            >最低分辨率<select v-model.number="minHeight">
              <option :value="0">不限</option>
              <option :value="720">720p</option>
              <option :value="1080">1080p</option>
              <option :value="1440">1440p</option>
              <option :value="2160">2160p / 4K</option>
            </select></label
          ><label
            >包含标签<select v-model="selectedTags" multiple>
              <option
                v-for="item in facets.tags"
                :key="item.value"
                :value="item.value"
              >
                {{ item.value }}（{{ item.count }}）
              </option>
            </select></label
          ><label
            >标签关系<select v-model="tagMode">
              <option value="all">全部满足（AND）</option>
              <option value="any">任一满足（OR）</option>
            </select></label
          ><label
            >排除标签<select v-model="excludedTags" multiple>
              <option
                v-for="item in facets.tags"
                :key="item.value"
                :value="item.value"
              >
                {{ item.value }}（{{ item.count }}）
              </option>
            </select></label
          ><button
            v-on:click="
              selectedTags = [];
              excludedTags = [];
              creatorFilter = '';
              codecFilter = '';
              minHeight = 0;
              searchScope = 'all';
              tagMode = 'all';
            "
          >
            清除筛选
          </button>
        </div>
        <div class="library">
          <article v-for="v in videos" :key="v.video_id">
            <button
              class="poster"
              :class="'poster--' + coverState(v.video_id)"
              @click="selected = v"
            >
              <span class="poster-skeleton" aria-hidden="true"></span>
              <img
                :ref="(el) => registerCoverImg(v.video_id, el)"
                :class="{ loaded: coverState(v.video_id) === 'loaded' }"
                :src="'/thumb/' + v.video_id"
                :alt="displayTitle(v)"
                loading="lazy"
                decoding="async"
                @load="markCoverLoaded(v.video_id)"
                @error="onCoverError($event, v.video_id)"
              /><span class="poster-duration">{{
                duration(v.duration_seconds)
              }}</span>
            </button>
            <div class="meta">
              <b>{{ displayTitle(v) }}</b
              ><small>{{
                displaySubtitle(
                  v,
                  (v.creator_name || v.subscription_slug) +
                    " · " +
                    (v.published_at ? localTime(v.published_at) : "日期未知"),
                )
              }}</small>
              <p>
                {{ mediaSummary(v) || v.views.toLocaleString() + " 次观看" }}
              </p>
            </div>
            <div class="library-card-actions">
              <button v-if="view === 'library'" class="library-card__play" @click="selected = v">播放</button>
              <button v-else class="library-card__play" @click="videoAction(v, 'download')">下载</button>
            </div>
          </article>
          <div v-if="!videos.length" class="empty">
            {{ view === "library" ? "影视库暂无已下载视频" : "没有未下载视频" }}
          </div>
        </div>
        <div class="pagination">
          <button
            :disabled="page === 0"
            @click="
              page--;
              loadVideos();
            "
          >
            上一页</button
          ><span>{{ page + 1 }} / {{ pages }}</span
          ><button
            :disabled="page + 1 >= pages"
            @click="
              page++;
              loadVideos();
            "
          >
            下一页
          </button>
        </div>
      </section>
      <section v-else>
        <div v-if="view === 'translations'" class="panel">
          <h2>WebAI 翻译交换</h2>
          <p>
            导出只包含视频标识、标题、标签和原文哈希，不包含代理密码、Cookie、下载地址或本地路径。
          </p>
          <div class="inline">
            <select v-model="translationLanguage">
              <option value="zh-CN">简体中文 zh-CN</option>
              <option value="zh-TW">繁体中文 zh-TW</option>
              <option value="en">英语 en</option>
              <option value="ja">日语 ja</option></select
            ><button class="primary" @click="exportTranslation">
              导出翻译 JSON</button
            ><button v-on:click="refreshNFO">刷新全部 Plex NFO</button
            ><label class="upload"
              >回导翻译 JSON<input
                type="file"
                accept="application/json,.json"
                @change="importTranslation"
            /></label>
          </div>
          <small
            >WebAI 只需填写 translations，并保留 key、source_hash 和
            fields；原文变化的条目会自动跳过。</small
          >
        </div>
        <SettingsView
          v-else
          :settings="settings"
          :api="api"
          :busy="busy"
          :import-html="importHTML"
          :demo-mode="demoMode"
          :sidebar-items="sidebarItems"
          :backups="backups"
          :backup-directory="backupDirectory"
          :backup-sensitive="backupSensitive"
          :exclusion-preview="exclusionPreview"
          :exclusion-total="exclusionTotal"
          @preview-exclusions="previewExclusions"
          v-on:toggle-demo="toggleDemo"
          v-on:update-sidebar-item="updateSidebarItem"
          v-on:save="saveSettings"
          v-on:create-backup="createBackup"
          v-on:restore-backup="restoreBackup"
          v-on:import="importPage"
          v-on:update:import-html="importHTML = $event"
        />
      </section>
    </main>
    <nav class="mobile-tabbar" aria-label="主要导航">
      <button
        v-for="item in [
          { id: 'home', label: '概览' },
          { id: 'subscriptions', label: '订阅' },
          { id: 'downloads', label: '下载' },
          { id: 'library', label: '影视库' },
          { id: 'more', label: '更多' },
        ]"
        :key="item.id"
        :class="{
          active:
            view === item.id ||
            (item.id === 'more' &&
              !['home', 'subscriptions', 'downloads', 'library'].includes(view)),
        }"
        @click="item.id === 'more' ? setSidebarCollapsed(false) : changeView(item.id)"
      >
        <NavIcon :name="item.id" /><span>{{ item.label }}</span
        ><i
          v-if="item.id === 'downloads' && data.counts.active + data.counts.waiting"
          >{{ data.counts.active + data.counts.waiting }}</i
        >
      </button>
    </nav>
  </div>
  <div v-if="selected" class="modal" @click.self="selected = undefined">
    <div class="player" role="dialog" aria-modal="true" :aria-label="displayTitle(selected)">
      <div class="player-top-actions">
        <button class="danger" @click="openDelete(selected)">删除</button>
        <button class="close" @click="selected = undefined">关闭</button>
      </div>
      <h2>{{ displayTitle(selected) }}</h2>
      <video
        v-if="selected.status === 'completed'"
        :src="'/media/' + selected.video_id"
        controls
        playsinline
        preload="metadata"
      /><img v-else :src="'/thumb/' + selected.video_id" :alt="displayTitle(selected)" />
      <div class="detail">
        <span>时长 {{ duration(selected.duration_seconds) }}</span
        ><span>播放 {{ selected.views.toLocaleString() }}</span
        ><span v-if="selected.likes">点赞 {{ selected.likes.toLocaleString() }}</span
        ><span v-if="selected.like_rate">好评 {{ selected.like_rate.toFixed(0) }}%</span
        ><span>S{{ selected.plex_season }}E{{ selected.plex_episode }}</span
        ><span v-if="selected.video_codec">视频 {{ selected.video_codec }}</span
        ><span v-if="selected.width"
          >{{ selected.width }}×{{ selected.height }}</span
        ><span v-if="selected.frame_rate"
          >{{ selected.frame_rate.toFixed(2) }} FPS</span
        ><span v-if="selected.bit_depth">{{ selected.bit_depth }}-bit</span
        ><span v-if="selected.bit_rate"
          >码率 {{ size(selected.bit_rate / 8) }}/s</span
        ><span v-if="selected.audio_codec">音频 {{ selected.audio_codec }}</span
        ><span v-if="selected.container">容器 {{ selected.container }}</span>
      </div>
      <p v-if="selected.description" class="video-description">{{ selected.description }}</p>
      <div class="chips">
        <span v-for="tag in displayTags(selected)" :key="tag">{{ tag }}</span>
      </div>
      <a :href="selected.page_url" target="_blank">打开详情页</a>
    </div>
  </div>
  <div v-if="deleteTarget" class="modal" @click.self="deleteTarget = undefined">
    <div class="player" role="dialog" aria-modal="true" style="max-width:520px">
      <h2>删除这个视频？</h2>
      <p class="muted">{{ displayTitle(deleteTarget) }}</p>
      <div class="delete-options">
        <label class="delete-option" :class="{ selected: deleteMode === 'delete_and_exclude' }">
          <input class="delete-option__radio" type="radio" name="delete-mode" value="delete_and_exclude" v-model="deleteMode">
          <span class="delete-option__content"><b>删除并加入已排除</b><small>删除当前媒体文件，并阻止以后订阅或自动发现时再次下载。</small></span>
        </label>
        <label class="delete-option" :class="{ selected: deleteMode === 'delete_only' }">
          <input class="delete-option__radio" type="radio" name="delete-mode" value="delete_only" v-model="deleteMode">
          <span class="delete-option__content"><b>仅删除媒体</b><small>删除当前媒体文件；以后如果订阅再次发现，仍允许重新下载。</small></span>
        </label>
      </div>
      <div class="actions" style="justify-content:flex-end;margin-top:14px">
        <button :disabled="deleting" @click="deleteTarget = undefined">取消</button>
        <button class="danger" :disabled="deleting" @click="confirmDelete">{{ deleting ? '正在删除…' : '确认删除' }}</button>
      </div>
    </div>
  </div>
</template>
