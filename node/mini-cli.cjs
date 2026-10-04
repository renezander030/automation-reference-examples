#!/usr/bin/env node
'use strict';
// Node 22+, CommonJS. Use built-in parsing rather than an ambiguous DIY grammar.
const { parseArgs }=require('node:util');
async function fetchJSON(url,timeoutMs=15000) {
 const endpoint=new URL(url);
 if(!['http:','https:'].includes(endpoint.protocol)) throw Error('HTTP(S) URL required');
 const res=await fetch(endpoint,{signal:AbortSignal.timeout(timeoutMs)});
 if(!res.ok) throw Error(`HTTP ${res.status}`);
 const reader=res.body.getReader();let size=0;const chunks=[];
 try {
  for (;;) {
   const {done,value}=await reader.read();if(done) break;
   size+=value.length;if(size>1024*1024) throw Error('JSON response exceeds 1 MiB');
   chunks.push(Buffer.from(value));
  }
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
 } finally {await reader.cancel();}
}
async function main(argv=process.argv.slice(2)) {
 const {values,positionals}=parseArgs({args:argv,allowPositionals:true,strict:true,options:{help:{type:'boolean',short:'h'},word:{type:'string'}}});
 const [command,...args]=positionals;
 if(values.help || !command) {console.log('Usage: node mini-cli.js echo [--word=value] [-- literal...] | get URL');return;}
 if(command==='echo') {console.log(JSON.stringify({word:values.word||'',args}));return;}
 if(command==='get' && args.length===1) {console.log(JSON.stringify(await fetchJSON(args[0])));return;}
 throw Error('unknown command or invalid arguments');
}
if(require.main===module) main().catch(()=>{console.error('command failed; check arguments, endpoint and timeout');process.exitCode=1;});
module.exports={main,fetchJSON};
