import { test, expect } from '@playwright/test';
import { mockAPI } from './fixtures';

test('已排除恢复需要确认且不会直接创建下载', async ({page})=>{
  const calls=await mockAPI(page);
  await page.route('**/api/media-exclusions?**',r=>r.fulfill({json:{items:[{video_id:'test:excluded',title:'隔离排除项',reason:'duplicate',excluded_at:'2026-10-09T00:00:00Z'}],total:1}}));
  await page.goto('/excluded');
  page.once('dialog',dialog=>dialog.dismiss());
  await page.getByRole('button',{name:/恢复允许下载/}).click();
  expect(calls.filter(c=>c.startsWith('DELETE'))).toHaveLength(0);
  page.once('dialog',dialog=>dialog.accept());
  await page.getByRole('button',{name:/恢复允许下载/}).click();
  await expect(page.locator('.toast')).toContainText('已恢复允许下载');
  expect(calls.filter(c=>c.startsWith('DELETE'))).toHaveLength(1);
  expect(calls.some(c=>c.includes('/api/download'))).toBe(false);
});

test('旧折叠偏好不隐藏导航；主题设置入口与系统主题在离开设置后仍有效', async ({page})=>{
  await mockAPI(page);
  await page.addInitScript(()=>localStorage.setItem('hwatch-nav-groups',JSON.stringify({content:false,media:false,system:false})));
  await page.goto('/');
  await expect(page.getByLabel('外观主题')).toHaveCount(0);
  for(const label of ['订阅管理','下载中心','影视库','翻译管理']) await expect(page.locator('aside').getByRole('button',{name:new RegExp('^'+label)})).toBeVisible();
  await page.locator('aside').getByRole('button',{name:'设置',exact:true}).click();
  await page.getByRole('button',{name:/显示与隐私/}).click();
  await page.getByLabel('外观主题').selectOption('system');
  await page.locator('aside').getByRole('button',{name:'影视库',exact:true}).click();
  await page.emulateMedia({colorScheme:'light'});
  await expect(page.locator('html')).toHaveAttribute('data-theme','light');
  await page.emulateMedia({colorScheme:'dark'});
  await expect(page.locator('html')).toHaveAttribute('data-theme','dark');
});

test('列表导航只加载一次；离开列表取消待执行搜索', async ({page})=>{
  const calls=await mockAPI(page);
  await page.goto('/library');
  await expect(page.locator('.library article')).toHaveCount(1);
  await page.waitForTimeout(450);
  expect(calls.filter(c=>c==='GET /api/videos')).toHaveLength(1);
  await page.getByPlaceholder('搜索标题、译文、作者或标签').fill('pending');
  await page.locator('aside').getByRole('button',{name:'设置',exact:true}).click();
  await page.waitForTimeout(450);
  expect(calls.filter(c=>c==='GET /api/videos')).toHaveLength(1);
});

test('VDF already_done 显示已处理且不能再次清理', async ({page})=>{
  await mockAPI(page);
  await page.route('**/api/media-integrity/import',r=>r.fulfill({json:{format:'vdf_gui',conflicts:[],has_conflicts:false,ready_groups:0,groups:1,already_done_groups:1,remove_already_done:1,items:[{group_id:'done-group',status:'already_done',remove_items:[{path:'synthetic/a.mp4',match_status:'already_done',video_id:'test:1'}],keep_items:[],problems:[]}]}}));
  await page.goto('/integrity');
  await page.getByLabel('VDF 删除清单 JSON').fill('{"version":1}');
  await page.getByRole('button',{name:'预览',exact:true}).click();
  await expect(page.getByRole('button',{name:'二次确认并执行 VDF 清理'})).toBeDisabled();
  await expect(page.getByRole('button',{name:/done-group/})).toContainText('此前已处理');
});
