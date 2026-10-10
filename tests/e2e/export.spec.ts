import {test,expect} from '@playwright/test';
import {mockAPI} from './fixtures';
test('翻译导出使用当前语言并产生可下载 JSON，不导航离开页面',async({page})=>{
  await mockAPI(page);
  let language='';
  await page.route('**/api/translations/export?**',route=>{
    language=new URL(route.request().url()).searchParams.get('language')||'';
    return route.fulfill({headers:{'Content-Disposition':'attachment; filename="synthetic-translations.json"'},contentType:'application/json',body:JSON.stringify({items:[]})});
  });
  await page.goto('/translations');
  await page.getByLabel('翻译目标语言').selectOption('ja');
  const download=page.waitForEvent('download');
  await page.getByRole('button',{name:'导出翻译 JSON'}).click();
  expect((await download).suggestedFilename()).toBe('hwatch-translations-ja.json');
  expect(language).toBe('ja');
  await expect(page).toHaveURL(/\/translations$/);
});
