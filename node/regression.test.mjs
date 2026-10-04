import test from 'node:test';import assert from 'node:assert/strict';
import {mkdtempSync,writeFileSync,chmodSync,rmSync} from 'node:fs';import {tmpdir} from 'node:os';import {join} from 'node:path';import {spawnSync} from 'node:child_process';
import {retrieve} from './retrieval.mjs';
test('hung vector and failed keyword settle within deadline',async()=>{
 const start=Date.now();const result=await retrieve(async()=>['kw'],()=>new Promise(()=>{}),{branchMs:20,totalMs:30});assert.deepEqual(result,[['kw'],[]]);assert.ok(Date.now()-start<500);
 const result2=await retrieve(()=>Promise.reject(Error('offline')),async()=>['vec']);assert.deepEqual(result2,[[],['vec']]);
});
test('total deadline cancels both providers',async()=>{let cancelled=0;const hang=signal=>new Promise(()=>{signal.addEventListener('abort',()=>cancelled++)});await retrieve(hang,hang,{branchMs:1000,totalMs:10});assert.equal(cancelled,2)});
test('JSONL quotes remain arguments, child/schema failures fail batch',()=>{
 const dir=mkdtempSync(join(tmpdir(),'jsonl-'));try{
 const cli=join(dir,'fake');writeFileSync(cli,'#!/usr/bin/env node\nprocess.stdout.write(JSON.stringify(process.argv.slice(2)));if(process.argv[2]==="lint")process.exitCode=3;');chmodSync(cli,0o700);
 const literal='"\n$(touch pwned)';const input=JSON.stringify({cmd:'info',args:[literal]})+'\n'+JSON.stringify({cmd:'lint'})+'\nnull\n';
 const r=spawnSync(process.execPath,[new URL('./serve.mjs',import.meta.url).pathname,cli],{input,encoding:'utf8'});assert.ifError(r.error);assert.equal(r.status,1);const rows=r.stdout.trim().split('\n').map(JSON.parse);assert.equal(rows.length,3);assert.equal(JSON.parse(rows[0].stdout)[1],literal);assert.equal(rows[1].status,3);assert.equal(rows[2].ok,false);
 const missing=spawnSync(process.execPath,[new URL('./serve.mjs',import.meta.url).pathname,join(dir,'missing')],{input:'{"cmd":"info"}\n',encoding:'utf8'});assert.equal(missing.status,1);assert.equal(JSON.parse(missing.stdout).error,'ENOENT');
 }finally{rmSync(dir,{recursive:true,force:true})}
});
test('built-in CLI parser handles equals and terminator',()=>{const r=spawnSync(process.execPath,[new URL('./mini-cli.cjs',import.meta.url).pathname,'echo','--word=x','--','--literal'],{encoding:'utf8'});assert.ifError(r.error);assert.equal(r.status,0);assert.deepEqual(JSON.parse(r.stdout),{word:'x',args:['--literal']})});
