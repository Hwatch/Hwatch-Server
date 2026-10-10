import { test, expect } from "@playwright/test";
import { mockAPI } from "./fixtures";
import { mkdirSync } from "node:fs";
const pages = [
  ["/", "概览"],
  ["/subscriptions", "订阅管理"],
  ["/downloads", "下载中心"],
  ["/undownloaded", "未下载"],
  ["/library", "影视库"],
  ["/excluded", "已排除"],
  ["/integrity", "媒体完整性"],
  ["/translations", "翻译管理"],
  ["/notifications", "通知"],
  ["/settings", "设置"],
];
test("每个页面：四种视口、截图、无横向溢出或脚本错误", async ({ page }, testInfo) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  page.on('console',message=>{if(message.type()==='error')errors.push(message.text());});
  await mockAPI(page);
  const phase = `${process.env.HWATCH_SCREENSHOT_PHASE || "after"}/${testInfo.project.name}`;
  mkdirSync(`acceptance/${phase}`, { recursive: true });
  for (const width of [390, 768, 1280, 1920]) {
    await page.setViewportSize({ width, height: 900 });
    await page.goto("/");
    await page.evaluate(
      (w) =>
        localStorage.setItem("hwatch-sidebar-collapsed", w < 681 ? "1" : "0"),
      width,
    );
    for (const [path, title] of pages) {
      await page.goto(path);
      await expect(page.locator("main h1")).toHaveText(title);
      await expect(page.locator(".state-panel.loading")).toHaveCount(0);
      if(width>=768)await expect(page.locator('aside [aria-current="page"]')).toContainText(title==='设置'?'':title);
      if(path==='/library'||path==='/undownloaded'){
        const cover=page.locator('.poster img').first();await cover.scrollIntoViewIfNeeded();
        await expect(cover).toHaveClass(/loaded/);await page.evaluate(()=>scrollTo(0,0));
        await expect(cover).toHaveCSS('opacity','1');
      }
      await page.screenshot({
        path: `acceptance/${phase}/${width}-${path.replaceAll("/", "") || "dashboard"}.png`,
        fullPage: true,
      });
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
        `${width} ${path} overflow`,
      ).toBeTruthy();
    }
  }
  expect(errors).toEqual([]);
});
test("登录与刷新保持页面", async ({ page }) => {
  await mockAPI(page, false);
  await page.goto("/library");
  await page.getByRole("textbox").first().fill("test-admin");
  await page.locator("input[type=password]").fill("test-password-only");
  await page.getByRole("button", { name: "登录", exact: true }).click();
  await expect(page.locator("main h1")).toHaveText("影视库");
  await page.reload();
  await expect(page.locator("main h1")).toHaveText("影视库");
});
test("订阅添加 Loading、防重复、Toast 自动关闭", async ({ page }) => {
  await mockAPI(page);
  let submissions = 0;
  let release!: () => void;
  const gate = new Promise<void>((r) => (release = r));
  await page.route("**/api/subscriptions", async (r) => {
    submissions++;
    await gate;
    await r.fulfill({ json: { slug: "test:new", baselined: 3, queued: 0 } });
  });
  await page.goto("/subscriptions");
  await page.locator("input[type=url]").fill("https://example.test/creator");
  await page.getByRole("button", { name: "订阅", exact: true }).click();
  await expect(page.getByRole("button", { name: "正在添加…" })).toBeDisabled();
  expect(submissions).toBe(1);
  release();
  await expect(page.locator(".toast")).toContainText("已建立");
  await expect(page.locator(".toast")).toHaveCount(0, { timeout: 12000 });
});
test("下载操作只控制单任务并发送 CSRF", async ({ page }) => {
  await mockAPI(page);
  let csrf = "";
  await page.route("**/api/videos/test:1/pause", async (r) => {
    csrf = r.request().headers()["x-csrf-token"];
    await r.fulfill({ json: {} });
  });
  await page.goto("/downloads");
  await page
    .locator(".queue")
    .getByRole("button", { name: "暂停", exact: true })
    .click();
  expect(csrf).toBe("test-session");
  await expect(page.locator(".toast")).toBeVisible();
});
test("翻译导入、文件重选和结果字段", async ({ page }) => {
  await mockAPI(page);
  await page.goto("/translations");
  await page
    .locator("input[type=file]")
    .setInputFiles({
      name: "translation-test.json",
      mimeType: "application/json",
      buffer: Buffer.from('{"translations":[]}'),
    });
  await expect(page.locator(".toast")).toContainText("已导入 1 个翻译字段");
  await expect(page.locator("input[type=file]")).toHaveValue("");
});
test("操作记录明确请求已受理不等于完成", async ({ page }) => {
  await mockAPI(page);
  await page.goto("/notifications");
  await page.getByRole("button", { name: "操作记录", exact: true }).click();
  await expect(page.locator("main")).toContainText("不代表后台任务已完成");
  await expect(page.locator(".notification-list")).toContainText(
    "POST /api/check",
  );
});
test("VDF 跨组冲突禁用执行、折叠显示保留删除项", async ({ page }) => {
  await mockAPI(page);
  await page.route("**/api/media-integrity/import", (r) =>
    r.fulfill({
      json: {
        format: "vdf_gui",
        has_conflicts: true,
        ready_groups: 1,
        conflicts: [
          {
            kind: "keep_remove",
            video_id: "test:1",
            groups: ["g1", "g2"],
            detail: "跨组 Keep / Remove 冲突",
          },
        ],
        items: [
          {
            group_id: "g1",
            status: "conflict",
            subscription_slug: "test:creator",
            remove_items: [
              {
                path: "test/a.mp4",
                match_status: "matched",
                video_id: "test:1",
              },
            ],
            keep_items: [
              {
                path: "test/b.mp4",
                match_status: "matched",
                video_id: "test:2",
              },
            ],
            problems: ["跨组冲突"],
          },
        ],
        groups: 2,
      },
    }),
  );
  await page.goto("/integrity");
  await page.locator("textarea").first().fill('{"version":1}');
  await page.getByRole("button", { name: "预览", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "二次确认并执行 VDF 清理" }),
  ).toBeDisabled();
  await page.getByRole("button", { name: /g1/ }).click();
  await expect(page.locator(".vdf-items")).toContainText("test/a.mp4");
  await expect(page.locator(".vdf-items")).toContainText("test/b.mp4");
  await page.getByLabel('VDF 删除清单 JSON').fill('{"version":2}');
  await expect(page.getByRole('button',{name:'二次确认并执行 VDF 清理'})).toHaveCount(0);
});
test("设置保存失败保留输入与错误反馈", async ({ page }) => {
  await mockAPI(page);
  await page.route("**/api/settings", async (r) => {
    if (r.request().method() === "PUT")
      await r.fulfill({ status: 503, json: { error: "测试保存失败" } });
    else await r.fallback();
  });
  await page.goto("/settings");
  await page.getByLabel("并发下载任务数").fill("3");
  await page.getByRole("button", { name: "保存设置", exact: true }).click();
  await expect(page.locator(".toast")).toContainText("保存设置失败");
  await expect(page.getByLabel("并发下载任务数")).toHaveValue("3");
});
test("备份恢复取消不提交，确认只表示受理", async ({ page }) => {
  const calls = await mockAPI(page);
  await page.goto("/settings");
  await page.getByRole("button", { name: /数据库备份与恢复/ }).click();
  page.once("dialog", (d) => d.dismiss());
  await page.getByRole("button", { name: "恢复", exact: true }).click();
  expect(calls.some((c) => c.endsWith("/restore"))).toBe(false);
  page.once("dialog", (d) => d.accept());
  await page.getByRole("button", { name: "恢复", exact: true }).click();
  await expect(page.locator(".toast")).toContainText("已提交恢复请求");
});
test("弹窗 Tab 焦点、Esc、恢复焦点", async ({ page }) => {
  await mockAPI(page);
  await page.goto("/library");
  const play = page.getByRole("button", { name: "播放", exact: true });
  await play.click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.keyboard.press("Shift+Tab");
  expect(
    await page
      .getByRole("dialog")
      .evaluate((el) => el.contains(document.activeElement)),
  ).toBe(true);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(play).toBeFocused();
});
test("亮色主题持久化和各页截图", async ({ page }) => {
  await mockAPI(page);
  await page.goto("/settings");
  await page.getByRole("button", { name: /显示与隐私/ }).click();
  await page.getByLabel("外观主题").selectOption("light");
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  mkdirSync("acceptance/after", { recursive: true });
  for (const width of [390, 1280]) {
    await page.setViewportSize({ width, height: 900 });
    for (const [path] of pages) {
      await page.goto(path);
      await page.screenshot({
        path: `acceptance/after/light-${width}-${path.replaceAll("/", "") || "dashboard"}.png`,
        fullPage: true,
      });
    }
  }
});
test("手机导航、筛选和侧栏关闭后全宽", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockAPI(page);
  await page.goto("/");
  await page
    .locator(".mobile-tabbar")
    .getByRole("button", { name: "订阅", exact: true })
    .click();
  await page.getByLabel("筛选订阅").fill("不存在");
  await expect(page.locator("main")).toContainText("没有匹配的订阅");
  await page
    .locator(".mobile-tabbar")
    .getByRole("button", { name: "更多", exact: true })
    .click();
  await expect(page.locator("aside")).toBeVisible();
  await page
    .locator("aside")
    .getByRole("button", { name: "已排除", exact: true })
    .click();
  await expect(page.locator("main h1")).toHaveText("已排除");
  expect(
    await page
      .locator("main")
      .evaluate((el) => el.getBoundingClientRect().width),
  ).toBe(390);
});
test("列表失败可重试，不产生脚本错误", async ({ page }) => {
  await mockAPI(page);
  let fail = true;
  await page.route("**/api/videos?**", async (r) => {
    if (fail)
      await r.fulfill({ status: 503, json: { error: "测试列表不可用" } });
    else await r.fallback();
  });
  await page.goto("/library");
  await expect(page.getByRole("alert")).toContainText("测试列表不可用");
  fail = false;
  await page.getByRole("button", { name: "重试加载" }).click();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.locator(".library article")).toHaveCount(1);
});
test("JSON 导入拒绝语法错误", async ({ page }) => {
  await mockAPI(page);
  await page.goto("/integrity");
  await page
    .getByLabel("选择 VDF JSON 文件")
    .setInputFiles({
      name: "invalid.json",
      mimeType: "application/json",
      buffer: Buffer.from("not json"),
    });
  await expect(page.getByRole("alert")).toContainText("JSON 无法读取");
});
test("强演示模式隐藏标题，封面裁剪", async ({ page }) => {
  await mockAPI(page);
  await page.goto("/");
  await page.evaluate(() => localStorage.setItem("r34watch-demo-mode", "2"));
  await page.goto("/library");
  await expect(page.locator(".meta b")).not.toContainText("海边旅行");
  await expect(page.locator(".meta b")).toContainText("媒体项目");
  expect(
    await page
      .locator(".poster")
      .evaluate((el) => getComputedStyle(el).overflow),
  ).toBe("hidden");
});
test("下载分页与手机播放/删除弹窗", async ({ page }) => {
  await mockAPI(page);
  await page.route('**/api/tasks?**',route=>route.fulfill({json:{items:[],total:96}}));
  await page.goto('/downloads');
  const nextRequest=page.waitForRequest(req=>new URL(req.url()).pathname==='/api/tasks'&&new URL(req.url()).searchParams.get('offset')==='48');
  await page.getByRole('button',{name:'下一页',exact:true}).click();
  await nextRequest;
  await expect(page.locator('.pagination')).toContainText('2 / 2');
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/library");
  await page.getByRole("button", { name: "播放", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.screenshot({ path: "acceptance/after/mobile-player.png" });
  await page.getByRole("button", { name: "删除", exact: true }).click();
  await expect(
    page.getByRole("dialog", { name: "删除这个视频" }),
  ).toBeVisible();
  await page.screenshot({ path: "acceptance/after/mobile-delete.png" });
  await page.getByRole("button", { name: "取消", exact: true }).click();
  await expect(page.getByRole("dialog", { name: "删除这个视频" })).toHaveCount(
    0,
  );
});
