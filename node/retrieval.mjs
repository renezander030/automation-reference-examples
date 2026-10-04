// One total deadline, a deadline per branch, cooperative cancellation.
export async function boundedBranch(fn,ms,parent) {
 const ctl=new AbortController();
 const abort=()=>ctl.abort(parent.reason);
 if(parent?.aborted) return [];
 parent?.addEventListener('abort',abort,{once:true});
 let timer;
 const fallback=new Promise(resolve=>{timer=setTimeout(()=>{ctl.abort();resolve([]);},ms);ctl.signal.addEventListener('abort',()=>resolve([]),{once:true});});
 try { return await Promise.race([Promise.resolve().then(()=>fn(ctl.signal)).catch(()=>[]),fallback]); }
 finally {clearTimeout(timer);parent?.removeEventListener('abort',abort);}
}
export async function retrieve(keyword,vector,{branchMs=200,totalMs=250}={}) {
 if(!Number.isFinite(branchMs)||branchMs<=0||!Number.isFinite(totalMs)||totalMs<=0) throw Error('positive deadlines required');
 const total=new AbortController();const timer=setTimeout(()=>total.abort(),totalMs);
 try { return await Promise.all([boundedBranch(keyword,branchMs,total.signal),boundedBranch(vector,branchMs,total.signal)]); }
 finally {clearTimeout(timer);total.abort();}
}
// A timeout bounds response latency even if fn ignores its signal. That underlying
// operation may keep running: use providers that support cancellation and quotas.
