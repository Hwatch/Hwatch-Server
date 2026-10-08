import {test,expect} from '@playwright/test';
import {mockAPI} from './fixtures';
import {mkdirSync} from 'node:fs';
const pages=[['/','概览'],['/subscriptions','订阅管理'],['/downloads','下载中心'],['/undownloaded','未下载'],['/library','影视库'],['/excluded','已排除'],['/integrity','媒体完整性'],['/translations','翻译管理'],['/notifications','通知'],['/settings','设置']];
test('每个页面：四种视口、截图、无横向溢出或脚本错误',async({page})=>{
  const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));await mockAPI(page);
  const phase=process.env.HWATCH_SCREENSHOT_PHASE||'after';mkdirSync(`acceptance/${phase}`,{recursive:true});
  for(const width of [390,768,1280,1920]){
    await page.setViewportSize({width,height:900});
    for(const [path,title] of pages){await page.goto(path);await expect(page.locator('main h1')).toHaveText(title);await page.screenshot({path:`acceptance/${phase}/${width}-${path.replaceAll('/','')||'dashboard'}.png`,fullPage:true});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),`${width} ${path} overflow`).toBeTruthy();}
  }
  expect(errors).toEqual([]);
});
test('登录与刷新保持页面',async({page})=>{await mockAPI(page,false);await page.goto('/library');await page.getByRole('textbox').first().fill('test-admin');await page.locator('input[type=password]').fill('test-password-only');await page.getByRole('button',{name:'登录',exact:true}).click();await expect(page.locator('main h1')).toHaveText('影视库');await page.reload();await expect(page.locator('main h1')).toHaveText('影视库');});
