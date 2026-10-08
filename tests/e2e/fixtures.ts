import type { Page } from "@playwright/test";
export const video = {
  video_id: "test:1",
  subscription_slug: "test:creator",
  platform: "test",
  title: "测试素材 · 海边旅行",
  translated_title: "海边旅行",
  translated_tags: [],
  translation_language: "zh-CN",
  description: "合成测试数据，没有真实媒体。",
  page_url: "#",
  published_at: "2026-10-09T00:00:00Z",
  discovered_at: "2026-10-09T00:00:00Z",
  thumbnail_url: "",
  views: 120,
  likes: 12,
  like_rate: 98,
  duration_seconds: 120,
  creator_name: "测试作者",
  uploader_name: "测试上传者",
  categories: [],
  tags: [],
  status: "completed",
  bytes_downloaded: 100000000,
  total_bytes: 100000000,
  download_speed: 0,
  connections: 0,
  video_codec: "H.264",
  audio_codec: "AAC",
  container: "mp4",
  width: 1920,
  height: 1080,
  frame_rate: 30,
  bit_rate: 1000000,
  bit_depth: 8,
  transcode_status: "",
  error: "",
  plex_season: 1,
  plex_episode: 1,
};
export const sub = {
  id: 1,
  slug: "test:creator",
  platform: "test",
  subscription_type: "creator",
  name: "测试作者",
  url: "#",
  avatar_url: "",
  avatar_proxy_url: "",
  external_links: [],
  enabled: true,
  last_checked_at: "2026-10-09T00:00:00Z",
  last_error: "",
  progress: {
    total: 12,
    completed: 8,
    active: 1,
    waiting: 2,
    failed: 1,
    bytes_downloaded: 100,
    total_bytes: 200,
  },
};
export const settings = {
  http_proxy: "",
  download_dir: "/test/media",
  aria_rpc_url: "http://localhost:6800/jsonrpc",
  aria_secret: "",
  downloader_mode: "aria2",
  check_time: "03:00",
  timezone: "Asia/Shanghai",
  max_concurrent: 2,
  connections_per_download: 4,
  download_limit_kbps: 0,
  min_free_bytes: 1000000000,
  require_creator_match: false,
  exclude_title_keywords: "",
  exclude_uploaders: "",
  exclude_categories: "",
  exclude_tags: "",
  max_duration_seconds: 0,
  backup_dir: "/test/backups",
  backup_time: "04:00",
  backup_retention: 7,
  backup_last_success: "",
  backup_last_error: "",
};
export async function mockAPI(page: Page, loggedIn = true) {
  // Page interaction tests do not exercise SSE transport. A completed route.fulfill
  // stream reconnects during navigation and produces WebKit access-control noise.
  // Keep transport deterministic here; real transport belongs to live.spec.ts.
  await page.addInitScript(() => {
    class MockEvents {
      onmessage: ((event: MessageEvent) => void) | null = null;
      onerror: (() => void) | null = null;
      close() {}
      addEventListener() {}
      removeEventListener() {}
    }
    Object.defineProperty(window, 'EventSource', {value: MockEvents});
  });
  let authenticated = loggedIn;
  const calls: string[] = [];
  await page.route("**/api/**", async (route) => {
    const req = route.request(),
      path = new URL(req.url()).pathname;
    calls.push(req.method() + " " + path);
    let body: any = {};
    if (path === "/api/auth/status")
      body = {
        configured: true,
        authenticated,
        username: "测试管理员",
        csrf_token: "test-session",
      };
    else if (path === "/api/auth/login") {
      authenticated = true;
      body = {
        configured: true,
        authenticated: true,
        username: "测试管理员",
        csrf_token: "test-session",
      };
    } else if (path === "/api/dashboard")
      body = {
        subscriptions: [sub],
        videos: [video],
        recently_played: [video],
        counts: {
          videos: 12,
          active: 1,
          waiting: 2,
          completed: 8,
          discovered: 4,
        },
        traffic_bytes: 1200000000,
        downloaded_videos: 8,
        library_bytes: 900000000,
        current_download_speed: 1024000,
        current_connections: 4,
        downloads_paused: false,
        disk_total: 100000000000,
        disk_free: 50000000000,
        aria2_online: true,
        downloader_mode: "aria2",
        downloaders: [
          {
            ID: "aria2",
            Name: "aria2",
            Online: true,
            Active: 1,
            DownloadSpeed: 1024000,
            Connections: 4,
          },
        ],
        health_status: "ok",
        health_warnings: [],
        unread_notifications: 1,
        timezone: "Asia/Shanghai",
      };
    else if (path === "/api/videos/facets")
      body = { creators: [], codecs: [], tags: [] };
    else if (path === "/api/videos") body = { items: [video], total: 1 };
    else if (path === "/api/tasks")
      body = {
        items: [
          {
            ...video,
            status: "active",
            bytes_downloaded: 50000000,
            download_speed: 1024000,
            connections: 4,
          },
          {
            ...video,
            video_id: "test:2",
            status: "failed",
            error: "测试网络错误",
          },
        ],
        total: 2,
      };
    else if (path === "/api/settings") body = settings;
    else if (path === "/api/backups")
      body = {
        items: [
          {
            name: "test-backup.db",
            size: 10240,
            created_at: "2026-10-09T00:00:00Z",
            sha256: "a".repeat(64),
            valid: true,
          },
        ],
        directory: "/test/backups",
        sensitive: false,
      };
    else if (path === "/api/notifications")
      body = {
        items: [
          {
            id: 1,
            code: "test",
            severity: "warning",
            message: "合成通知：存储检查提醒",
            created_at: "2026-10-09T00:00:00Z",
            read_at: "",
          },
        ],
      };
    else if (path === "/api/audit")
      body = {
        items: [
          {
            id: 1,
            actor: "测试管理员",
            action: "POST /api/check",
            target_type: "api",
            target_id: "check",
            result: "accepted",
            created_at: "2026-10-09T00:00:00Z",
          },
        ],
      };
    else if (path === "/api/media-exclusions") body = { items: [], total: 0 };
    else if (path === "/api/media-integrity/scan")
      body = {
        total: 8,
        present: 8,
        missing: 0,
        moved: 0,
        permission_denied: 0,
        storage_available: true,
        items: [],
      };
    else if (path === "/api/media-integrity/plans") body = { items: [] };
    else if (path === "/api/recovery/latest") body = { available: false };
    else if (path === "/api/integrations/transrunner/token")
      body = { configured: false, scopes: [] };
    else if (path === "/api/translations/import")
      body = { imported_fields: 1, skipped_items: 0, failed_items: 0 };
    else if (path === "/api/events")
      return route.fulfill({
        status: 200,
        contentType: "text/event-stream",
        headers: { 'Access-Control-Allow-Origin': new URL(req.url()).origin, 'Access-Control-Allow-Credentials': 'true' },
        body: ": mock\n\n",
      });
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(body),
    });
  });
  await page.route("**/thumb/**", (r) =>
    r.fulfill({
      contentType: "image/svg+xml",
      body: '<svg xmlns="http://www.w3.org/2000/svg" width="640" height="360"><rect width="640" height="360" fill="#35546b"/><circle cx="440" cy="95" r="45" fill="#d6b260"/><path d="M0 360V260L230 130L640 360" fill="#1d3546"/></svg>',
    }),
  );
  await page.route("**/media/**", (r) => r.fulfill({ status: 204, body: "" }));
  return calls;
}
