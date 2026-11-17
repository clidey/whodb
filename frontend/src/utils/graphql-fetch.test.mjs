import assert from 'node:assert/strict';
import test from 'node:test';
import {graphqlFetch} from './graphql-fetch.ts';
const limit=1<<20;
test('preserves JSON through byte boundary and authenticated multipart above it',async()=>{
 const original=globalThis.fetch;
 try{for(const size of [limit-1,limit,limit+1]){
  const prefix=JSON.stringify({query:'mutation { save }',variables:{text:''}});
  const body=prefix.replace('"text":""','"text":"'+'x'.repeat(size-prefix.length)+'"');
  assert.equal(new Blob([body]).size,size);
  const controller=new AbortController(),init={method:'POST',headers:{'Content-Type':'application/json','Authorization':'Bearer opaque','X-CSRF-Token':'csrf','X-Whodb-App-Id':'app'},credentials:'include',signal:controller.signal,body};
  let calls=0;const response=new Response('{}');
  globalThis.fetch=async(url,actual)=>{
   calls++;assert.equal(url,'/api/query');assert.equal(actual.credentials,'include');assert.equal(actual.signal,controller.signal);
   for(const name of ['Authorization','X-CSRF-Token','X-Whodb-App-Id'])assert.equal(new Headers(actual.headers).get(name),init.headers[name]);
   if(size<=limit)assert.equal(actual,init);else{
    assert.ok(actual.body instanceof FormData);assert.deepEqual([...actual.body.keys()],['operations','map']);
    assert.equal(actual.body.get('operations'),body);assert.equal(actual.body.get('map'),'{}');assert.equal(new Headers(actual.headers).get('content-type'),null);assert.equal(init.headers['Content-Type'],'application/json');
   }return response;
  };
  assert.equal(await graphqlFetch('/api/query',init),response);assert.equal(calls,1);
 }}finally{globalThis.fetch=original;}
});
test('counts UTF-8 bytes and retains exact serialized numbers',async()=>{
 const original=globalThis.fetch;
 try{const body='{"query":"mutation { save }","variables":{"id":9007199254740993,"text":"'+'😀'.repeat(270000)+'"}}';
  assert.ok(body.length<limit&&new Blob([body]).size>limit);
  globalThis.fetch=async(_url,init)=>{assert.equal(init.body.get('operations'),body);return new Response('{}');};
  await graphqlFetch('/api/query',{method:'POST',headers:new Headers({'content-type':'application/json; charset=utf-8'}),body});
 }finally{globalThis.fetch=original;}
});
test('never retries server errors, rejection or cancellation',async()=>{
 const original=globalThis.fetch;
 try{for(const error of [new Error('failed'),new DOMException('aborted','AbortError'),null]){
  let calls=0;const response=new Response('{}',{status:500});globalThis.fetch=async()=>{calls++;if(error)throw error;return response;};
  const result=graphqlFetch('/api/query',{method:'POST',headers:{'Content-Type':'application/json'},body:'x'.repeat(limit+1)});
  if(error)await assert.rejects(result,value=>value===error);else assert.equal(await result,response);assert.equal(calls,1);
 }}finally{globalThis.fetch=original;}
});
test('leaves existing multipart, GET and non-JSON requests unchanged',async()=>{
 const original=globalThis.fetch;
 try{for(const init of [{method:'POST',body:new FormData()},{method:'GET'},{method:'POST',headers:{'Content-Type':'text/plain'},body:'x'.repeat(limit+1)}]){
  globalThis.fetch=async(_url,actual)=>{assert.equal(actual,init);return new Response('{}');};await graphqlFetch('/api/query',init);
 }}finally{globalThis.fetch=original;}
});
