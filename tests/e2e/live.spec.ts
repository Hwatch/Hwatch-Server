import { test, expect } from "@playwright/test";
test("隔离 Go / SQLite 实例：创建管理员、登录、真实 API 与 CSRF", async ({
  page,
}) => {
  test.skip(
    process.env.HWATCH_LIVE_TEST !== "1",
    "Run npm run preview:v2 first, then HWATCH_LIVE_TEST=1 npm run test:e2e",
  );
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto("http://127.0.0.1:5189/");
  await page.getByLabel("用户名").fill("v2-test-admin");
  await page
    .getByLabel("密码", { exact: true })
    .fill("isolated-v2-test-password");
  const confirmation = page.getByLabel("确认密码");
  if (await confirmation.count())
    await confirmation.fill("isolated-v2-test-password");
  await page.locator(".auth-card").getByRole("button").click();
  await expect(page.locator("main h1")).toHaveText("概览");
  if (!(await page.locator("aside").isVisible()))
    await page.locator(".sidebar-toggle").click();
  await page
    .locator("aside")
    .getByRole("button", { name: "设置", exact: true })
    .click();
  const save = page.getByRole("button", { name: "保存设置", exact: true });
  await save.click();
  await expect(page.locator(".toast")).toContainText("设置已保存");
  const status = await page.request.get(
    "http://127.0.0.1:5189/api/auth/status",
  );
  expect((await status.json()).authenticated).toBe(true);
  await page
    .locator("aside")
    .getByRole("button", { name: "退出登录", exact: true })
    .click();
  await expect(page.locator(".auth-card h1")).toHaveText("登录 HWatch");
  expect(errors).toEqual([]);
});
