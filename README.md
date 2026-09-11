# R34Watch NAS 媒体订阅器

Go + SQLite + aria2 + Vue/TypeScript 构建的 NAS 视频订阅与 Plex 媒体库管理服务。

## 功能

- 创作者页面全分页订阅，首次建立完整基线，后续新作品自动下载
- SQLite 保存订阅、详情元数据、任务、流量和 Plex 季集编号
- aria2 RPC 下载、暂停、继续、重试、断点续传、实时速度和连接数
- 可设置全局下载限速、单任务连接数、并发数和下载目录
- 最高画质临时解析，不在数据库长期保存多档下载地址
- HTTP/HTTPS 代理同时应用于采集和 aria2 下载
- 每日定时检查，默认 Asia/Shanghai 04:00
- Plex TV 目录、tvshow.nfo、单集 NFO、海报和单集封面
- 下载完成后通过 ffprobe 记录 H.264/H.265/AV1、音频编码、分辨率、帧率、码率、位深和容器格式
- 项目内影视库浏览、搜索、分页、预览和播放
- 粘贴视频详情 HTML 导入
- NAS 磁盘空间与累计下载流量统计
- 平台适配器注册表；Rule34Video 已作为独立适配器接入
- WebAI 翻译 JSON 安全导出、原文哈希校验和翻译回导

## 平台适配器

统一接口位于 `internal/adapters`。平台实现只负责识别订阅地址、列出视频、读取详情和解析下载源，不直接操作 SQLite、aria2 或 Plex。内部视频标识采用 `platform:source_id`，例如 `rule34video:4600143`。未来 Hanime1 以独立适配器注册，不需要修改下载中心和影视库。

## WebAI 翻译

在“翻译管理”页面选择目标语言后导出 JSON。导出包只包含平台标识、标题、标签和原文哈希，不包含代理配置、RPC 密钥、Cookie、下载源或文件路径。WebAI 应保留 `key`、`source_hash` 和 `fields`，仅填写 `translations`。回导时原文哈希不一致的条目会被跳过，翻译保存在独立表中，不覆盖原文。

## Docker Compose

先将 compose.yaml 中的 NAS 路径替换为真实 Plex 媒体目录：

    /path/on/nas/Plex-TV:/media

然后运行：

    docker compose up -d --build

访问：

    http://NAS-IP:8787

SQLite 和 aria2 会话位于 data 目录。Plex 将同一个宿主机媒体目录作为电视节目库挂载，
本项目和 Plex 直接查看同一份视频文件，不会复制两份。

## 本机开发

需要 Node.js、Go 1.24、aria2 和 FFmpeg（使用其中的 ffprobe）：

    npm install
    npm run build
    aria2c --enable-rpc=true --rpc-listen-all=false --daemon=true
    go run ./cmd/r34watch

环境变量：

- R34WATCH_ADDR：监听地址，默认 :8787
- R34WATCH_DB：SQLite 路径，默认 r34watch.db
- R34WATCH_WEB：前端构建目录，默认 web/dist
- R34WATCH_MEDIA_ROOT：下载与 Plex 共用的媒体根目录
- R34WATCH_ARIA_RPC：aria2 JSON-RPC 地址
- R34WATCH_ARIA_SECRET：aria2 RPC 密钥

本服务默认没有登录认证，请只在可信局域网使用，不要直接暴露到公网。
