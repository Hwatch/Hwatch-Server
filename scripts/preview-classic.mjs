import {spawn, execFileSync} from 'node:child_process';
import {mkdtempSync, symlinkSync, writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
// Read-only snapshot. No checkout, production config, database or branch changes.
const root=mkdtempSync(join(tmpdir(),'hwatch-classic-'));
const archive=execFileSync('git',['archive','3b06eab']);
execFileSync('tar',['-x','-C',root],{input:archive});
symlinkSync(join(process.cwd(),'node_modules'),join(root,'node_modules'),'dir');
writeFileSync(join(root,'vite.config.ts'), `import {defineConfig} from 'vite';import vue from '@vitejs/plugin-vue';export default defineConfig({plugins:[vue()],server:{host:'127.0.0.1',proxy:{'/api':'http://127.0.0.1:18787','/thumb':'http://127.0.0.1:18787','/media':'http://127.0.0.1:18787'}}});`);
const child=spawn(process.execPath,[join(process.cwd(),'node_modules/vite/bin/vite.js'),'--host','127.0.0.1','--port','5187','--strictPort'],{cwd:root,stdio:'inherit'});
for(const signal of ['SIGINT','SIGTERM'])process.on(signal,()=>child.kill('SIGTERM'));
console.log('Classic 3b06eab: http://127.0.0.1:5187; isolated backend only: 127.0.0.1:18787. Start preview:v2 first.');
