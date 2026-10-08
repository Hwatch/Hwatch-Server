import { test, expect } from '@playwright/test';
import { mockAPI, video } from './fixtures';
import { mkdirSync, writeFileSync } from 'node:fs';

test('3400 条媒体与任务：Classic/V2 可比隔离交互指标', async ({ page }, info) => {
  test.setTimeout(90000);
  await mockAPI(page);
  await page.route('**/thumb/**', async route=>{
    await new Promise(resolve=>setTimeout(resolve,150));
    await route.fallback();
  });
  const records = Array.from({length:3400}, (_,i)=>({...video, video_id:`test:${i+1}`, title:`测试长文件名 ${i+1} ${'合成素材'.repeat(30)}`, translated_title:`隔离媒体 ${i+1} ${'长标题'.repeat(25)}`}));
  let calls=0;
  let apiRequests=0;
  page.on('request', request=>{ if(new URL(request.url()).pathname.startsWith('/api/')) apiRequests++; });
  await page.route('**/api/videos?**', async route=>{
    calls++;
    const url=new URL(route.request().url());
    const query=url.searchParams.get('q')||'';
    const selected=records.filter(v=>!query||v.title.includes(query)||v.translated_title.includes(query));
    const offset=Number(url.searchParams.get('offset')||0), limit=Number(url.searchParams.get('limit')||48);
    await new Promise(resolve=>setTimeout(resolve,50));
    await route.fulfill({json:{items:selected.slice(offset,offset+limit),total:selected.length}});
  });
  await page.route('**/api/tasks?**', async route=>{
    calls++;
    const url=new URL(route.request().url()), offset=Number(url.searchParams.get('offset')||0), limit=Number(url.searchParams.get('limit')||48);
    await route.fulfill({json:{items:records.slice(offset,offset+limit).map(v=>({...v,status:'active',download_speed:1024})),total:3400}});
  });
  const results=[];
  for(const [name,base] of [['classic','http://127.0.0.1:5187'],['v2','http://127.0.0.1:5188']]) {
    for(const width of [390,768,1280,1920]) {
      await page.setViewportSize({width,height:900});
      calls=0;
      apiRequests=0;
      const start=Date.now();
      await page.goto(`${base}/library`);
      await expect(page.locator('.library article')).toHaveCount(48);
      const readyMs=Date.now()-start;
      const cards=await page.locator('.library article').count();
      mkdirSync(`acceptance/scale/${info.project.name}`,{recursive:true});
      await page.screenshot({path:`acceptance/scale/${info.project.name}/${name}-${width}-library.png`});
      const imagesBefore=await page.evaluate(()=>Array.from(document.images).filter(i=>i.complete&&i.naturalWidth>0).length);
      const scrollStart=Date.now();
      await page.evaluate(()=>scrollTo(0,document.body.scrollHeight));
      await expect(page.getByRole('button',{name:'下一页',exact:true}).last()).toBeVisible();
      const scrollMs=Date.now()-scrollStart;
      const next=Date.now();
      await page.getByRole('button',{name:'下一页',exact:true}).last().click();
      await expect(page.locator('.library article').first()).toContainText('隔离媒体 49');
      const pageMs=Date.now()-next;
      const searchStart=Date.now();
      await page.getByPlaceholder('搜索标题、译文、作者或标签').fill('隔离媒体 3400');
      await expect(page.locator('.library article')).toHaveCount(1);
      await expect(page.locator('.library article')).toContainText('隔离媒体 3400');
      const searchMs=Date.now()-searchStart;
      const switchStart=Date.now();
      await page.goto(`${base}/downloads`);
      await expect(page.locator('.queue article')).toHaveCount(48);
      const taskCards=await page.locator('.queue article').count();
      const navigationMs=Date.now()-switchStart;
      results.push({name,width,readyMs,scrollMs,pageMs,searchMs,navigationMs,cards,taskCards,imagesBefore,listRequests:calls,apiRequests});
      await expect(page.locator('html')).toHaveJSProperty('scrollWidth',await page.evaluate(()=>innerWidth));
    }
  }
  mkdirSync('acceptance/metrics',{recursive:true});
  writeFileSync(`acceptance/metrics/${info.project.name}.json`,JSON.stringify({dataset:3400,mockLatencyMs:50,thumbnailDelayMs:150,results},null,2));
});
