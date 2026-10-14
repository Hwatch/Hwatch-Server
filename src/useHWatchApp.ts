import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { requestJSON } from "./api-client";
import { createAsyncActions } from "./asyncAction";
import { createToasts } from "./toast";

// Existing business state and lifecycle, extracted without changing API behavior.
export function useHWatchApp() {
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
  type AuditLogRow = {
    id: number;
    actor: string;
    action: string;
    target_type: string;
    target_id: string;
    result: string;
    created_at: string;
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
    subscribeError = ref(""),
    translationFileName = ref(""),
    translationFileSize = ref(0),
    translationStage = ref(""),
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
  const actions = createAsyncActions();
  const toasts = createToasts({ max: 3 });
  function toastType(message: string): "success" | "info" | "error" {
    if (/失败|错误|error|invalid|无效/i.test(message)) return "error";
    if (/正在|已提交|已请求|后台|处理中/.test(message)) return "info";
    return "success";
  }
  // Legacy `notice` assignments become lifecycle-managed toasts.
  watch(notice, (value) => {
    if (!value) return;
    toasts.push({ type: toastType(value), message: value });
    notice.value = "";
  });
  function notify(message: string, type: "success" | "info" | "error") {
    toasts.push({ type, message });
  }
  const subscribePending = computed(() => actions.pending("subscribe"));
  const subscribeStatus = computed(() =>
    subscribePending.value ? "正在保存订阅，请稍候…" : "",
  );
  const translationImportPending = computed(() =>
    actions.pending("translate-import"),
  );
  const translationImportError = computed(() =>
    actions.error("translate-import"),
  );
  const vdfPreviewPending = computed(() => actions.pending("vdf-preview"));
  const vdfExecutePending = computed(() => actions.pending("vdf-execute"));
  const vdfActionError = computed(
    () => actions.error("vdf-preview") || actions.error("vdf-execute"),
  );
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
  const pageDescriptions: Record<string, string> = {
    home: "下载、媒体与存储的实时概况",
    subscriptions: "管理作者与番剧来源，查看采集和下载进度",
    downloads: "查看下载速度、队列和失败原因",
    library: "浏览已下载媒体，搜索原文与译文",
    undownloaded: "筛选发现的作品并提交下载",
    excluded: "查看排除原因，恢复允许下载",
    integrity: "扫描与校验媒体，所有清理均需预览和确认",
    translations: "导出与回导翻译，刷新 Plex 元数据",
    notifications: "系统提醒与请求级操作记录",
    settings: "下载、存储、备份与界面偏好",
  };
  const subscriptionQuery = ref(""),
    subscriptionState = ref("all"),
    exclusionQuery = ref("");
  const pageLoading = ref(false),
    pageError = ref("");
  let pageRequestID = 0;
  const videosLoading = ref(false),
    videosError = ref("");
  const notificationSeverity = ref("all");
  const visibleNotifications = computed(() =>
    notifications.value.filter(
      (item) =>
        notificationSeverity.value === "all" ||
        (notificationSeverity.value === "unread"
          ? !item.read_at
          : item.severity === notificationSeverity.value),
    ),
  );
  const filteredSubscriptionGroups = computed(() =>
    subscriptionGroups.value
      .map((group) => ({
        ...group,
        sources: group.sources.filter(
          (sub) =>
            (subscriptionState.value === "all" ||
              (subscriptionState.value === "enabled"
                ? sub.enabled
                : subscriptionState.value === "paused"
                  ? !sub.enabled
                  : !!sub.last_error)) &&
            `${sub.name} ${sub.platform} ${sub.url}`
              .toLocaleLowerCase()
              .includes(subscriptionQuery.value.trim().toLocaleLowerCase()),
        ),
      }))
      .filter((group) => group.sources.length),
  );
  const filteredExclusions = computed(() =>
    exclusions.value.filter((item) =>
      `${item.title_snapshot} ${item.video_id} ${item.reason}`
        .toLocaleLowerCase()
        .includes(exclusionQuery.value.trim().toLocaleLowerCase()),
    ),
  );
  const vdfFilter = ref("all"),
    vdfPage = ref(0),
    expandedVdfGroups = ref<Record<string, boolean>>({});
  const filteredVdfGroups = computed(
    () =>
      vdfPreviewData.value?.items.filter(
        (group) =>
          vdfFilter.value === "all" || group.status === vdfFilter.value,
      ) || [],
  );
  const visibleVdfGroups = computed(() =>
    filteredVdfGroups.value.slice(vdfPage.value * 25, (vdfPage.value + 1) * 25),
  );
  watch(vdfFilter, () => (vdfPage.value = 0));
  const viewFromPath = () => {
    const raw = window.location.pathname.replace(/\/+$/, "") || "/";
    const path = raw === "/duplicates" ? "/library" : raw;
    return (
      Object.entries(viewRoutes).find(([, route]) => route === path)?.[0] ||
      "home"
    );
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
  // Primary navigation stays minimal (Emby-like); everything else lives in
  // Settings -> 高级管理 (see managementEntries in App.vue).
  const navGroups = [
    { id: "overview", label: "", items: [{ id: "home", label: "概览" }] },
    {
      id: "media",
      label: "媒体库",
      items: [{ id: "library", label: "影视库" }],
    },
    {
      id: "system",
      label: "系统",
      items: [{ id: "notifications", label: "通知" }],
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
      localStorage.setItem(
        "hwatch-nav-groups",
        JSON.stringify(openGroups.value),
      );
    } catch {}
  }
  function toggleGroup(id: string) {
    openGroups.value = {
      ...openGroups.value,
      [id]: openGroups.value[id] === false,
    };
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
  const notificationTab = ref<"system" | "operations">("system");
  const auditLogs = ref<AuditLogRow[]>([]);
  async function loadAuditLogs() {
    const result = await api("/api/audit?limit=100");
    auditLogs.value = result.items || [];
  }
  function auditResultLabel(result: string, action: string): string {
    if (result === "failed") return "失败";
    if (result === "success")
      return action && action.startsWith("POST ") ? "已受理" : "成功";
    return result || "未知";
  }
  async function switchNotificationTab(tab: "system" | "operations") {
    notificationTab.value = tab;
    if (tab === "operations") {
      const result = await actions.run("audit-load", () => loadAuditLogs());
      if (!result.ok && !result.duplicate)
        notice.value = result.error || "操作记录加载失败";
    }
  }
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
  const integrityBusy = computed(
    () =>
      actions.pending("integrity-scan") ||
      actions.pending("integrity-import") ||
      actions.pending("vdf-execute") ||
      actions.pending("apply-legacy"),
  );
  const integrityImportPending = computed(() =>
    actions.pending("integrity-import"),
  );
  const integrityImportError = computed(() =>
    actions.error("integrity-import"),
  );
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
  const planBusy = computed(
    () =>
      actions.pending("plan-preview") ||
      actions.pending("plan-execute") ||
      actions.pending("plan-recover"),
  );
  const planPreviewPending = computed(() => actions.pending("plan-preview"));
  const planExecutePending = computed(() => actions.pending("plan-execute"));
  const planActionError = computed(
    () => actions.error("plan-preview") || actions.error("plan-execute"),
  );
  function actionPending(key: string): boolean {
    return actions.pending(key);
  }
  function videoActionKey(videoId: string, _action: string): string {
    // Key by video only, so two conflicting actions on the same video cannot run
    // at once, while different videos stay independent.
    return "video:" + videoId;
  }
  const settingsBusy = computed(() => actions.pending("settings-save"));
  const backupsBusy = computed(
    () =>
      actions.pending("backup-create") ||
      actions.pending("backups-load") ||
      Object.keys(actions.states).some(
        (k) => k.startsWith("backup-restore:") && actions.pending(k),
      ),
  );
  const importBusy = computed(() => actions.pending("import-html"));
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
  const deleteMode = ref<"delete_and_exclude" | "delete_only">(
    "delete_and_exclude",
  );
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
  // Random discovery: a small, non-blocking sample of the whole library.
  const RANDOM_LIMIT = 6;
  const RANDOM_HISTORY_MAX = 36;
  const randomItems = ref<Video[]>([]);
  const randomError = ref("");
  const randomScope = ref<"all" | "current">("all");
  const randomOpen = ref(true);
  const randomHistory = ref<string[]>([]);
  const pages = computed(() => Math.max(1, Math.ceil(total.value / 48))),
    downloadTasks = computed(() => videos.value),
    unfinishedDownloadTasks = computed(() =>
      downloadTasks.value.filter((video) => video.status !== "completed"),
    ),
    completedDownloadTasks = computed(() =>
      downloadTasks.value.filter((video) => video.status === "completed"),
    ),
    subscriptionGroups = computed(() => {
      const groups = new Map<
        string,
        { key: string; name: string; sources: Sub[]; avatar?: Sub }
      >();
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
    } catch (error) {
      authError.value =
        "无法连接服务：" + String(error).replace(/^Error:\s*/, "");
      auth.value = { configured: true, authenticated: false };
    } finally {
      authReady.value = true;
    }
  }
  async function submitAuth() {
    if (busy.value) return;
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
    const result = await actions.run("notif-read", () =>
      api("/api/notifications/read-all", { method: "POST" }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      data.value.unread_notifications = 0;
      await loadNotifications();
    } else {
      notice.value = "标记已读失败：" + (result.error || "");
    }
  }
  async function loadExclusions() {
    const result = await api("/api/media-exclusions?limit=200");
    exclusions.value = result.items || [];
    mediaExclusionTotal.value = result.total || 0;
  }
  async function restoreExclusion(item: Exclusion) {
    if (
      !confirm(
        "恢复允许下载？之后订阅再次发现该视频时可重新下载，但不会立即创建下载任务。",
      )
    )
      return;
    const result = await actions.run("exclusion-restore:" + item.video_id, () =>
      api("/api/media-exclusions/" + encodeURIComponent(item.video_id), {
        method: "DELETE",
      }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      exclusions.value = exclusions.value.filter(
        (x) => x.video_id !== item.video_id,
      );
      mediaExclusionTotal.value = Math.max(0, mediaExclusionTotal.value - 1);
      notice.value = "已恢复允许下载";
    } else {
      notice.value = "恢复失败：" + (result.error || "");
    }
  }
  async function loadIntegrity() {
    const result = await actions.run("integrity-scan", async () => {
      mediaScan.value = await api("/api/media-integrity/scan");
      await loadDeletionPlans();
    });
    if (!result.ok && !result.duplicate) {
      integrityNotice.value = result.error || "扫描失败";
    }
  }
  async function loadDeletionPlans() {
    try {
      deletionPlans.value =
        (await api("/api/media-integrity/plans")).items || [];
    } catch {}
  }
  async function previewTransRunnerPlan() {
    if (!planText.value.trim()) {
      planNotice.value = "请粘贴 TransRunner 删除计划 JSON";
      return;
    }
    const result = await actions.run("plan-preview", async () => {
      planPreviewData.value = await api("/api/media-integrity/plan/preview", {
        method: "POST",
        body: JSON.stringify({ raw: planText.value }),
      });
      planResult.value = undefined;
    });
    if (result.duplicate) return;
    planNotice.value = result.ok ? "" : result.error || "预览失败";
  }
  async function executeTransRunnerPlan(confirm: boolean) {
    const result = await actions.run("plan-execute", () =>
      api("/api/media-integrity/plan/execute", {
        method: "POST",
        body: JSON.stringify({ raw: planText.value, confirm }),
      }),
    );
    if (result.duplicate) return;
    if (result.ok && result.value) {
      planResult.value = result.value;
      planNotice.value = confirm
        ? `已清理 ${result.value.executed}，跳过 ${result.value.skipped}，失败 ${result.value.failed}，需恢复 ${result.value.needs_recovery}`
        : `Dry run：${planPreviewData.value?.ready ?? 0} 项可清理，预计释放 ${size(result.value.estimated_freed)}`;
      await loadDeletionPlans();
      if (confirm) {
        await loadIntegrity();
        await loadExclusions();
        await previewTransRunnerPlan();
      }
    } else if (!result.ok) {
      planNotice.value = result.error || "执行失败";
    }
  }
  async function recoverPlan(planId: string) {
    const result = await actions.run("plan-recover", () =>
      api(
        "/api/media-integrity/plan/" + encodeURIComponent(planId) + "/recover",
        {
          method: "POST",
        },
      ),
    );
    if (result.duplicate) return;
    if (result.ok && result.value) {
      planNotice.value = `恢复：完成 ${result.value.executed}，需恢复 ${result.value.needs_recovery}`;
      await loadDeletionPlans();
      await loadIntegrity();
    } else if (!result.ok) {
      planNotice.value = result.error || "恢复失败";
    }
  }
  async function importDeletionList() {
    if (!deletionListText.value.trim()) {
      integrityNotice.value = "请粘贴 VDF 删除清单 JSON";
      return;
    }
    const result = await actions.run("integrity-import", () =>
      api("/api/media-integrity/import", {
        method: "POST",
        body: deletionListText.value,
      }),
    );
    if (result.duplicate) return;
    if (result.ok && result.value) {
      if (result.value.format === "vdf_gui") {
        vdfPreviewData.value = result.value;
        cleanupPreview.value = undefined;
      } else {
        cleanupPreview.value = result.value;
        vdfPreviewData.value = undefined;
      }
      integrityNotice.value = "";
    } else if (!result.ok) {
      integrityNotice.value = result.error || "导入失败";
    }
  }
  async function vdfExecute(confirm: boolean) {
    const result = await actions.run("vdf-execute", () =>
      api("/api/media-integrity/vdf/execute", {
        method: "POST",
        body: JSON.stringify({ raw: deletionListText.value, confirm }),
      }),
    );
    if (result.duplicate) return;
    if (result.ok && result.value) {
      integrityNotice.value = confirm
        ? `VDF 清理：执行 ${result.value.executed}，跳过 ${result.value.skipped}，失败 ${result.value.failed}，需恢复 ${result.value.needs_recovery}`
        : `VDF Dry run：${vdfPreviewData.value?.ready_groups ?? 0} 组可清理，预计释放 ${size(result.value.estimated_freed)}`;
      if (confirm) {
        await loadIntegrity();
        await loadExclusions();
        await importDeletionList();
      }
    } else if (!result.ok) {
      integrityNotice.value = result.error || "VDF 执行失败";
    }
  }
  async function applyDeletionList(confirm: boolean) {
    const result = await actions.run("apply-legacy", () =>
      api("/api/media-integrity/apply", {
        method: "POST",
        body: JSON.stringify({ raw: deletionListText.value, confirm }),
      }),
    );
    if (result.duplicate) return;
    if (result.ok && result.value) {
      integrityNotice.value = confirm
        ? `已清理 ${result.value.executed} 项，跳过 ${result.value.skipped}，失败 ${result.value.failed}`
        : `Dry run：${result.value.candidates} 项可加入排除`;
      await loadIntegrity();
      await importDeletionList();
      if (confirm) await loadExclusions();
    } else if (!result.ok) {
      integrityNotice.value = result.error || "执行失败";
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
      await api(
        "/api/videos/" + encodeURIComponent(target.video_id) + "/delete",
        {
          method: "POST",
          body: JSON.stringify({ mode: deleteMode.value }),
        },
      );
      videos.value = videos.value.filter((v) => v.video_id !== target.video_id);
      if (selected.value && selected.value.video_id === target.video_id)
        selected.value = undefined;
      const mode = deleteMode.value;
      deleteTarget.value = undefined;
      notice.value =
        mode === "delete_and_exclude" ? "已删除并加入已排除" : "已删除媒体";
      await refreshDashboard();
    } catch (e) {
      notice.value = "删除失败：" + String(e);
    } finally {
      deleting.value = false;
    }
  }
  async function previewExclusions() {
    const result = await actions.run("exclusions-preview", () =>
      api("/api/exclusions/preview", {
        method: "POST",
        body: JSON.stringify(settings.value),
      }),
    );
    if (result.duplicate) return;
    if (result.ok && result.value) {
      exclusionPreview.value = result.value.items;
      exclusionTotal.value = result.value.total;
      notice.value = `当前规则会排除 ${result.value.total} 项`;
    } else if (!result.ok) {
      notice.value = "预览失败：" + (result.error || "");
    }
  }
  async function loadVideos() {
    if (videoLoadTimer) {
      clearTimeout(videoLoadTimer);
      videoLoadTimer = undefined;
    }
    const requestID = ++videoRequestID;
    videosLoading.value = true;
    videosError.value = "";
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
      if (selectedTags.value.length)
        p.set("tags", selectedTags.value.join(","));
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
      if (requestID === videoRequestID)
        videosError.value = String(error).replace(/^Error:\s*/, "");
    } finally {
      if (requestID === videoRequestID) videosLoading.value = false;
    }
  }
  // Random discovery reuses the library filter semantics only when the user
  // opts into "current filters"; by default it spans the entire library.
  function randomParams(): URLSearchParams {
    const p = new URLSearchParams({
      limit: String(RANDOM_LIMIT),
      language: translationLanguage.value,
    });
    if (randomScope.value === "current") {
      if (query.value) p.set("q", query.value);
      p.set("scope", searchScope.value);
      p.set("tag_mode", tagMode.value);
      if (selectedTags.value.length)
        p.set("tags", selectedTags.value.join(","));
      if (excludedTags.value.length)
        p.set("exclude_tags", excludedTags.value.join(","));
      if (creatorFilter.value) p.set("creator", creatorFilter.value);
      if (codecFilter.value) p.set("codec", codecFilter.value);
      if (minHeight.value) p.set("min_height", String(minHeight.value));
    }
    if (randomHistory.value.length)
      p.set("exclude", randomHistory.value.join(","));
    return p;
  }
  async function loadRandom(allowReset = true) {
    // Keyed action: "换一批" shows loading immediately and rejects repeat clicks.
    const result = await actions.run("library-random", () =>
      api("/api/library/random?" + randomParams()),
    );
    if (result.duplicate) return;
    if (!result.ok) {
      // Keep the previous batch on screen and show a compact error.
      randomError.value = result.error || "随机推荐加载失败";
      return;
    }
    const items: Video[] = result.value?.items || [];
    if (items.length === 0 && randomHistory.value.length && allowReset) {
      // The recent-history exclusion consumed the whole candidate pool (small
      // library). Reset history once and try again instead of showing empty.
      randomHistory.value = [];
      return loadRandom(false);
    }
    randomItems.value = items;
    randomError.value = "";
    const ids = items.map((v) => v.video_id);
    randomHistory.value = [...ids, ...randomHistory.value].slice(
      0,
      RANDOM_HISTORY_MAX,
    );
  }
  async function loadFacets() {
    if (!facets.value.tags.length)
      facets.value = await api("/api/videos/facets");
  }
  async function loadSettings() {
    settings.value = await api("/api/settings");
    await loadBackups();
  }
  async function loadBackups() {
    const result = await actions.run("backups-load", () => api("/api/backups"));
    if (result.duplicate) return;
    if (result.ok && result.value) {
      backups.value = result.value.items;
      backupDirectory.value = result.value.directory;
      backupSensitive.value = result.value.sensitive;
    } else if (!result.ok) {
      notice.value = "备份列表加载失败：" + (result.error || "");
    }
  }
  async function createBackup() {
    const result = await actions.run("backup-create", () =>
      api("/api/backups", { method: "POST" }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      notice.value = "数据库备份已创建";
      await loadSettings();
    } else {
      notice.value = "创建备份失败：" + (result.error || "");
    }
  }
  async function restoreBackup(name: string) {
    if (
      !confirm(
        "确认使用 " + name + " 恢复数据库？下载会暂停，服务随后自动重启。",
      )
    )
      return;
    const result = await actions.run("backup-restore:" + name, () =>
      api("/api/backups/" + encodeURIComponent(name) + "/restore", {
        method: "POST",
        body: JSON.stringify({ confirm: true }),
      }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      // 202: the restore request was accepted; the service restart is not done yet.
      notice.value = "已提交恢复请求：服务将暂停下载并重启";
    } else {
      // Never auto-retry a destructive restore; ask the user to verify the service.
      notice.value =
        "恢复请求结果未知：" +
        (result.error || "") +
        "；请核对服务状态后再决定，勿自动重试";
    }
  }
  async function changeView(next: string, updateHistory = true) {
    if (videoLoadTimer) {
      clearTimeout(videoLoadTimer);
      videoLoadTimer = undefined;
    }
    const request = ++pageRequestID;
    pageLoading.value = true;
    pageError.value = "";
    try {
      if (!viewRoutes[next]) next = "home";
      // Clear transient feedback so stale messages/errors do not leak across views.
      notice.value = "";
      subscribeError.value = "";
      translationStage.value = "";
      planNotice.value = "";
      integrityNotice.value = "";
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
      // Random discovery loads after the library so it never blocks first paint.
      if (next === "library") void loadRandom(false);
      if (next === "settings") await loadSettings();
      if (next === "notifications") {
        await loadNotifications();
        if (notificationTab.value === "operations") await loadAuditLogs();
      }
      if (next === "excluded") await loadExclusions();
      if (next === "integrity") await loadIntegrity();
    } catch (error) {
      if (request === pageRequestID)
        pageError.value = String(error).replace(/^Error:\s*/, "");
    } finally {
      if (request === pageRequestID) pageLoading.value = false;
    }
  }
  async function subscribe() {
    subscribeError.value = "";
    // pending is set synchronously inside run() before the first await, so the
    // button flips to loading immediately.
    const result = await actions.run("subscribe", () =>
      api("/api/subscriptions", {
        method: "POST",
        body: JSON.stringify({
          url: modelURL.value,
          download_existing: downloadExisting.value,
        }),
      }),
    );
    if (result.duplicate) return;
    if (result.ok && result.value) {
      const r = result.value;
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
    } else if (!result.ok) {
      // Never assume the server did not create the subscription on timeout/lost
      // response: keep the input and tell the user to verify by refreshing.
      subscribeError.value =
        (result.error || "添加订阅失败") +
        "；请先刷新订阅列表确认是否已创建，再决定是否重试";
    }
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
  let checkWatchActive = false;
  async function watchCheckCompletion() {
    if (checkWatchActive) return;
    checkWatchActive = true;
    try {
      const deadline = Date.now() + 10 * 60 * 1000;
      while (Date.now() < deadline) {
        await new Promise((r) => window.setTimeout(r, 2500));
        let status: { running?: number } | undefined;
        try {
          status = await api("/api/check/status");
        } catch {
          break;
        }
        if (!status) break;
        if ((status.running ?? 0) === 0) {
          await refreshDashboard();
          if (["downloads", "undownloaded", "library"].includes(view.value))
            await loadVideos();
          notify("订阅检查已完成，状态已刷新", "success");
          return;
        }
      }
      // Timed out watching; refresh once anyway so the latest state shows.
      await refreshDashboard();
    } finally {
      checkWatchActive = false;
    }
  }
  async function check(slug = "") {
    const result = await actions.run("check:" + (slug || "all"), () =>
      api("/api/check" + (slug ? "/" + slug : ""), { method: "POST" }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      // Stay on the current view; refresh subscription status when the
      // background checks finish (they are throttled server-side).
      notify(slug ? "已开始检查该订阅" : "已开始检查全部订阅", "info");
      void watchCheckCompletion();
    } else {
      notice.value = "检查失败：" + (result.error || "");
    }
  }
  async function toggle(sub: Sub) {
    const key = "sub:" + sub.slug;
    const result = await actions.run(key, () =>
      api("/api/subscriptions/" + sub.slug, {
        method: "PATCH",
        body: JSON.stringify({ enabled: !sub.enabled }),
      }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      await refreshDashboard();
    } else {
      notice.value = "更新订阅失败：" + (result.error || "");
      await refreshDashboard();
    }
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
    const result = await actions.run("sub-del:" + sub.slug, () =>
      api("/api/subscriptions/" + sub.slug, { method: "DELETE" }),
    );
    if (result.duplicate) return;
    if (result.ok) await refreshDashboard();
    else notice.value = "删除订阅失败：" + (result.error || "");
  }
  async function videoAction(v: Video, action: string) {
    if (
      action === "delete" &&
      !confirm(
        v.status === "completed"
          ? "确认删除“" +
              displayTitle(v) +
              "”的媒体文件、NFO、海报和下载记录？此操作无法撤销。"
          : "确认删除“" +
              displayTitle(v) +
              "”的下载任务与临时文件？此操作无法撤销。",
      )
    )
      return;
    const labels: Record<string, string> = {
      download: "已提交下载任务",
      retry: "已提交重试",
      pause: "已请求暂停",
      resume: "已请求恢复",
      delete: "已提交删除任务",
    };
    const result = await actions.run(videoActionKey(v.video_id, action), () =>
      api("/api/videos/" + v.video_id + "/" + action, { method: "POST" }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      notice.value = (labels[action] || "操作已提交") + "：" + displayTitle(v);
      await loadVideos();
    } else {
      notice.value =
        (labels[action] || "操作") + "失败：" + (result.error || "");
    }
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
    const result = await actions.run("download-all", () =>
      api("/api/videos/download-all", { method: "POST" }),
    );
    if (result.duplicate) return;
    if (result.ok && result.value) {
      // Only the queue submission is confirmed here; the tasks run in background.
      notice.value =
        "已提交 " + result.value.accepted + " 个下载任务，正在后台处理";
      await changeView("downloads");
    } else if (!result.ok) {
      notice.value = "提交下载失败：" + (result.error || "");
    }
  }
  async function toggleDownloads() {
    const paused = !data.value.downloads_paused;
    const result = await actions.run("downloads-toggle", () =>
      api("/api/downloads/toggle", {
        method: "POST",
        body: JSON.stringify({ paused }),
      }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      data.value.downloads_paused = paused;
      notice.value = paused ? "已暂停全部下载" : "已继续全部下载";
      await loadVideos();
    } else {
      notice.value = "切换下载状态失败：" + (result.error || "");
      await refreshDashboard();
    }
  }
  async function saveSettings() {
    const result = await actions.run("settings-save", () =>
      api("/api/settings", {
        method: "PUT",
        body: JSON.stringify(settings.value),
      }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      // Refresh from the server response only after a confirmed save.
      if (result.value) settings.value = result.value;
      notice.value = "设置已保存";
      await refreshDashboard();
    } else {
      // Keep the user's edited values untouched on failure.
      notice.value = "保存设置失败：" + (result.error || "");
    }
  }
  async function importPage() {
    const result = await actions.run("import-html", () =>
      api("/api/import/html", {
        method: "POST",
        body: JSON.stringify({ html: importHTML.value }),
      }),
    );
    if (result.duplicate) return;
    if (result.ok) {
      notice.value = "HTML 数据已导入";
      importHTML.value = "";
      await loadVideos();
    } else {
      notice.value = "导入失败：" + (result.error || "");
    }
  }
  async function exportTranslation() {
    const language = translationLanguage.value;
    const result = await actions.run('translation-export', async () => {
      const payload = await api('/api/translations/export?language=' + encodeURIComponent(language));
      const url = URL.createObjectURL(new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' }));
      const link = document.createElement('a');
      link.href = url;
      link.download = `hwatch-translations-${language}.json`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    });
    if (!result.ok && !result.duplicate) notice.value = '导出失败：' + (result.error || '');
  }
  async function importTranslation(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    translationFileName.value = file.name;
    translationFileSize.value = file.size;
    translationStage.value = "正在上传并导入…";
    const result = await actions.run("translate-import", async () =>
      api("/api/translations/import", {
        method: "POST",
        body: await file.text(),
      }),
    );
    // Reset on both success and failure so the same file can be selected again.
    input.value = "";
    if (result.duplicate) return;
    if (result.ok && result.value) {
      const r = result.value;
      let message = "已导入 " + r.imported_fields + " 个翻译字段";
      if (typeof r.skipped_items === "number")
        message += "，跳过 " + r.skipped_items + " 条";
      if (typeof r.failed_items === "number")
        message += "，失败 " + r.failed_items + " 条";
      notice.value = message;
      translationStage.value = "导入完成";
    } else if (!result.ok) {
      translationStage.value = "导入失败，可重新选择文件重试";
    }
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
    const result = await actions.run("nfo-refresh", () =>
      api("/api/plex/refresh-nfo", {
        method: "POST",
        body: JSON.stringify({ language: translationLanguage.value }),
      }),
    );
    if (result.duplicate) return;
    if (result.ok && result.value) {
      // Accepted for background processing; not a completion signal.
      notice.value =
        "已提交刷新，后台处理 " + result.value.accepted + " 个 NFO 文件";
    } else if (!result.ok) {
      notice.value = "刷新 NFO 失败：" + (result.error || "");
    }
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
    localStorage.setItem("hwatch-sidebar-collapsed", collapsed ? "1" : "0");
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
    if (deleteTarget.value) {
      if (!deleting.value) deleteTarget.value = undefined;
      return;
    }
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
    toasts.clearAll();
  });
  watch(
    deletionListText,
    () => {
      vdfPreviewData.value = undefined;
      cleanupPreview.value = undefined;
      vdfPage.value = 0;
      expandedVdfGroups.value = {};
    },
    { flush: "sync" },
  );
  watch(
    planText,
    () => {
      planPreviewData.value = undefined;
    },
    { flush: "sync" },
  );
  watch(randomScope, () => {
    if (view.value === "library") void loadRandom(false);
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
      // Navigation resets these fields and explicitly loads once. Do not also
      // schedule a debounced request against the next (possibly non-list) view.
      if (pageLoading.value) return;
      if (["library", "undownloaded", "downloads"].includes(view.value)) {
        if (videoLoadTimer) clearTimeout(videoLoadTimer);
        videoLoadTimer = window.setTimeout(() => void loadVideos(), 300);
      }
      if (
        view.value === "library" &&
        randomScope.value === "current" &&
        randomOpen.value
      )
        void loadRandom(false);
      if (view.value === "home") refreshDashboard();
    },
  );
  return {
    auth,
    authReady,
    authUsername,
    authPassword,
    authPasswordConfirm,
    authError,
    empty,
    data,
    videos,
    total,
    view,
    query,
    status,
    page,
    notice,
    busy,
    subscribeError,
    translationFileName,
    translationFileSize,
    translationStage,
    modelURL,
    selected,
    importHTML,
    translationLanguage,
    creatorQuery,
    creatorResults,
    selectedCreator,
    searchingCreators,
    downloadExisting,
    actions,
    toasts,
    subscribePending,
    subscribeStatus,
    translationImportPending,
    translationImportError,
    vdfExecutePending,
    downloadedStat,
    viewTitles,
    pageDescriptions,
    subscriptionQuery,
    subscriptionState,
    exclusionQuery,
    pageLoading,
    pageError,
    videosLoading,
    videosError,
    notificationSeverity,
    visibleNotifications,
    filteredSubscriptionGroups,
    filteredExclusions,
    vdfFilter,
    vdfPage,
    expandedVdfGroups,
    filteredVdfGroups,
    visibleVdfGroups,
    demoMode,
    sidebarCollapsed,
    sidebarItems,
    navGroups,
    openGroups,
    toggleGroup,
    showOriginal,
    settings,
    backups,
    backupDirectory,
    backupSensitive,
    notifications,
    notificationTab,
    auditLogs,
    auditResultLabel,
    switchNotificationTab,
    exclusions,
    mediaExclusionTotal,
    mediaScan,
    cleanupPreview,
    vdfPreviewData,
    deletionListText,
    integrityNotice,
    integrityBusy,
    integrityImportPending,
    cleanupStatusLabel,
    planText,
    planPreviewData,
    planResult,
    deletionPlans,
    planNotice,
    planBusy,
    planPreviewPending,
    planExecutePending,
    actionPending,
    videoActionKey,
    settingsBusy,
    backupsBusy,
    importBusy,
    planStatusLabel,
    vdfGroupStatusLabel,
    vdfGroupBadgeClass,
    vdfItemStatusLabel,
    deleteTarget,
    deleteMode,
    deleting,
    exclusionPreview,
    exclusionTotal,
    facets,
    showFilters,
    searchScope,
    tagMode,
    selectedTags,
    excludedTags,
    creatorFilter,
    codecFilter,
    minHeight,
    sortBy,
    sortOrder,
    randomItems,
    randomError,
    randomScope,
    randomOpen,
    loadRandom,
    pages,
    unfinishedDownloadTasks,
    completedDownloadTasks,
    showCompletedTasks,
    rlLabel,
    activeRateLimits,
    fmtCountdown,
    sourceLabel,
    rateLimitedReason,
    subscriptionTypeLabel,
    size,
    duration,
    localTime,
    resolution,
    mediaSummary,
    pct,
    api,
    submitAuth,
    logout,
    onCoverError,
    coverState,
    markCoverLoaded,
    registerCoverImg,
    markNotificationsRead,
    restoreExclusion,
    loadIntegrity,
    previewTransRunnerPlan,
    executeTransRunnerPlan,
    recoverPlan,
    importDeletionList,
    vdfExecute,
    applyDeletionList,
    openDelete,
    confirmDelete,
    previewExclusions,
    loadVideos,
    createBackup,
    restoreBackup,
    changeView,
    subscribe,
    searchCreators,
    chooseCreator,
    check,
    toggle,
    removeSub,
    videoAction,
    downloadAll,
    toggleDownloads,
    saveSettings,
    importPage,
    exportTranslation,
    importTranslation,
    refreshNFO,
    displayTitle,
    displaySubtitle,
    displayTags,
    toggleOriginal,
    toggleDemo,
    toggleSidebar,
    setSidebarCollapsed,
    updateSidebarItem,
  };
}
