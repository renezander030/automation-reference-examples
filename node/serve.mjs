#!/usr/bin/env node
import { spawnSync } from 'node:child_process';
import { createInterface } from 'node:readline';
// CLI path and command allowlist are operator-controlled, not JSONL fields.
const cli=process.argv[2];
const allowed=new Set((process.env.RUNNER_COMMANDS||'info,lint').split(','));
if (!cli) { console.error('usage: node serve.mjs CLI [--fail-fast]'); process.exit(2); }
const failFast=process.argv.includes('--fail-fast');
let failed=false;
const rl=createInterface({input:process.stdin,crlfDelay:Infinity});
for await (const line of rl) {
 if (!line.trim()) continue;
 let result;
 try {
  const job=JSON.parse(line);
  if (!job || typeof job!=='object' || Array.isArray(job) || typeof job.cmd!=='string' || !allowed.has(job.cmd)) throw Error('invalid or disallowed cmd');
  if (job.project!==undefined && (typeof job.project!=='string' || job.project.includes('\0'))) throw Error('invalid project');
  if (job.args!==undefined && (!Array.isArray(job.args) || job.args.some(x=>typeof x!=='string'||x.includes('\0')))) throw Error('args must be strings without NUL');
  const args=[job.cmd,...(job.project?[job.project]:[]),...(job.args||[])];
  const r=spawnSync(cli,args,{encoding:'utf8',shell:false,timeout:60000,maxBuffer:1024*1024});
  result={ok:!r.error && !r.signal && r.status===0,cmd:job.cmd,status:r.status,signal:r.signal,stdout:r.stdout||'',stderr:r.stderr||'',error:r.error?.code};
 } catch { result={ok:false,status:null,error:'invalid job JSON or schema'}; }
 if (!result.ok) failed=true;
 process.stdout.write(JSON.stringify(result)+'\n');
 if (failFast && failed) break;
}
process.exitCode=failed?1:0;
