"""Measure actual endpoint output before comparing to a vector-index contract."""
import json,urllib.request

def probe_embedding_width(base_url,model,timeout=10):
    request=urllib.request.Request(base_url.rstrip('/')+'/api/embed',data=json.dumps({'model':model,'input':'embedding width probe'}).encode(),headers={'Content-Type':'application/json'})
    with urllib.request.urlopen(request,timeout=timeout) as response:
        data=response.read(1024*1024+1)
    if len(data)>1024*1024: raise ValueError('embedding response too large')
    vectors=json.loads(data).get('embeddings')
    if not isinstance(vectors,list) or len(vectors)!=1 or not isinstance(vectors[0],list) or not vectors[0] or any(isinstance(x,bool) or not isinstance(x,(int,float)) for x in vectors[0]): raise ValueError('invalid embedding response')
    return len(vectors[0])
