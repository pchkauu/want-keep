"""Bounded synthetic text/tool evaluation through Dahl Chat Completions."""

import argparse
from datetime import datetime, timezone
import fcntl
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

from openai_cases import OpenAICases
from openai_eval import EvaluationContract


class DahlContract:
  models=["MiniMaxAI/MiniMax-M2.7", "deepseek-ai/DeepSeek-V4-Flash-0731"]
  endpoint="https://inference.dahl.global/v1/chat/completions"

  @staticmethod
  def payload(model, cases, function=False):
    if model not in DahlContract.models or any("attachment" in c for c in cases):
      raise ValueError("Only the two authorized text models and synthetic text cases are supported")
    contract=json.loads((Path(__file__).resolve().parents[1]/'evidence/dahl.contract.json').read_text())
    instructions=contract['prompt']+'\nReturn exactly one JSON object, without Markdown, conforming to this schema:\n'+json.dumps(contract['schema'])
    body=dict(model=model,messages=[dict(role="system",content=instructions),
              dict(role="user",content=json.dumps([dict(id=c["id"],**c["input"]) for c in cases],ensure_ascii=False))],
              max_tokens=8192,stream=False)
    if function:
      body.update(tools=[dict(type="function",function=dict(name="propose_accounting",description="Return proposals only; does not execute accounting.",parameters=contract['schema'],strict=True))],
                  tool_choice=dict(type="function",function=dict(name="propose_accounting")),parallel_tool_calls=False)
    else:
      body['response_format']=dict(type="json_schema",json_schema=dict(name="accounting_proposals",schema=contract['schema'],strict=True))
    return body

  @staticmethod
  def usage(data):
    u=data['usage']
    inp,out,total=[u[k] for k in ['prompt_tokens','completion_tokens','total_tokens']]
    if any(type(v) is not int or v<0 for v in [inp,out,total]) or total!=inp+out or out>8192:
      raise ValueError("Invalid or out-of-bounds usage; retain reservation")
    return dict(prompt_tokens=inp,completion_tokens=out,total_tokens=total,
                completion_tokens_details=u.get('completion_tokens_details'))

  @staticmethod
  def streamed_response(text):
    message={'content':'','tool_calls':[]}
    result={'choices':[{'message':message,'finish_reason':None}]}
    tools={};done=False
    for line in text.splitlines():
      if not line.startswith('data:'):continue
      event=line[5:].strip()
      if event=='[DONE]':done=True;continue
      if done:raise ValueError('Stream data after completion marker')
      chunk=json.loads(event)
      if chunk.get('model'):
        if result.get('model',chunk['model'])!=chunk['model']:raise ValueError('Stream model changed')
        result['model']=chunk['model']
      if chunk.get('usage'):result['usage']=chunk['usage']
      if chunk.get('reasoning_effort'):result['reasoning_effort']=chunk['reasoning_effort']
      for choice in chunk.get('choices',[]):
        if choice.get('index')!=0:raise ValueError('Unexpected stream choice')
        if choice.get('finish_reason'):result['choices'][0]['finish_reason']=choice['finish_reason']
        delta=choice.get('delta',{})
        if delta.get('content'):message['content']+=delta['content']
        if delta.get('refusal'):message['refusal']=delta['refusal']
        for call in delta.get('tool_calls',[]):
          index=call['index']
          if type(index) is not int or not 0<=index<8:raise ValueError('Invalid streamed tool index')
          assembled=tools.setdefault(index,{'function':{'name':'','arguments':''}})
          for field in ['id','type']:
            if call.get(field):assembled[field]=call[field]
          for field in ['name','arguments']:
            if call.get('function',{}).get(field):assembled['function'][field]+=call['function'][field]
    if not done or not result['choices'][0]['finish_reason'] or 'usage' not in result:
      raise ValueError('Incomplete stream or missing usage; retain reservation')
    message['tool_calls']=[tools[k] for k in sorted(tools)]
    return result

  @staticmethod
  def answer_json(content):
    if not isinstance(content,str):raise ValueError('Missing text output')
    content=content.strip()
    if content.startswith('<think>'):
      if content.count('</think>')!=1:raise ValueError('Incomplete or ambiguous reasoning envelope')
      content=content.split('</think>',1)[1].strip()
    return json.loads(content)

  @staticmethod
  def proposals(data, function):
    choices=data['choices']
    if len(choices)!=1:raise ValueError("Expected one completion")
    choice=choices[0];message=choice['message']
    if message.get('refusal'):raise ValueError("Refusal is not an accounting proposal")
    if function:
      calls=message.get('tool_calls',[])
      if choice.get('finish_reason')!='tool_calls' or len(calls)!=1 or calls[0].get('type')!='function' or calls[0]['function']['name']!='propose_accounting':
        raise ValueError("Unexpected or incomplete tool call")
      return json.loads(calls[0]['function']['arguments'])
    if choice.get('finish_reason')!='stop' or message.get('tool_calls'):
      raise ValueError("Incomplete or unexpected output")
    return DahlContract.answer_json(message['content'])


class DahlEvaluation:
  def __init__(self, journal, model, phase='baseline'):
    self.path=journal
    self.model=model
    self.phase=phase
    self.cases=OpenAICases.build()[:200]
    self.batches=[self.cases[i:i+10] for i in range(0,200,10)]+[self.cases[:3]]
    self.payloads=[DahlContract.payload(model,b,i==20) for i,b in enumerate(self.batches)]
    if phase in ['streaming_schema','high_effort']:
      for payload in self.payloads:payload.update(stream=True,stream_options={'include_usage':True})
    if phase=='high_effort':
      for payload in self.payloads:
        payload['reasoning_effort']='high'
        if model=='deepseek-ai/DeepSeek-V4-Flash-0731':payload['thinking']={'type':'enabled'}
    self.fingerprint=hashlib.sha256(json.dumps(self.payloads,sort_keys=True).encode()).hexdigest()

  def save(self):
    temp=self.path.with_suffix('.writing')
    with temp.open('w') as stream:
      json.dump(self.data,stream,ensure_ascii=False,indent=2)
      stream.flush();os.fsync(stream.fileno())
    os.replace(temp,self.path)
    descriptor=os.open(self.path.parent,os.O_RDONLY)
    try:os.fsync(descriptor)
    finally:os.close(descriptor)

  def reserve(self,index,ceiling):
    if str(index) in self.data['calls']:
      raise ValueError("Never replay a recorded request")
    if sum(c['charged_or_reserved_tokens'] for c in self.data['calls'].values())+ceiling>250000:
      raise ValueError("Per-model 250000-token research cap exhausted")
    self.data['calls'][str(index)]=dict(state='unknown',charged_or_reserved_tokens=ceiling,reserved_tokens=ceiling)
    self.save()

  def run(self,key_file,max_calls=None,function_only=False):
    key=key_file.read_text().strip()
    if not key or any(c.isspace() for c in key):raise ValueError("Expected a raw API key")
    escaped=key.replace('\\','\\\\').replace('"','\\"')
    config='header = "Authorization: Bearer '+escaped+'"\n'
    with open(str(self.path)+'.lock','a') as lock:
      fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
      if self.path.exists():
        self.data=json.loads(self.path.read_text())
        fingerprints=self.data.setdefault('fingerprints',{'baseline':self.data['fingerprint']})
        if self.data['model']!=self.model or (self.phase in fingerprints and fingerprints[self.phase]!=self.fingerprint):
          raise ValueError("Configuration drift; preserve the prior experiment")
        if any('grades' not in c and not (c.get('http_status')=='429' and c.get('curl_exit')==0) and not (c.get('state')=='received' and c.get('finish_reasons')==['stop'])
               and not (self.phase in ['streaming_schema','high_effort'] and c.get('http_status')=='524' and c.get('curl_exit')==0) for c in self.data['calls'].values()):
          raise ValueError("Ungraded or unknown prior call; reconcile before continuing")
        fingerprints[self.phase]=self.fingerprint
        self.save()
      else:
        self.data=dict(model=self.model,endpoint=DahlContract.endpoint,started_at=datetime.now(timezone.utc).isoformat(),
                       fingerprint=self.fingerprint,fingerprints={self.phase:self.fingerprint},token_cap=250000,reasoning_effort='provider_default_not_verified_as_high',calls={})
        self.save()
      sent=0
      for index,body in enumerate(self.payloads):
        if function_only and index!=20:continue
        call_key=str(index) if self.phase=='baseline' else self.phase+':'+str(index)
        if call_key in self.data['calls']:
          if 'grades' not in self.data['calls'][call_key]:raise ValueError('Prior call has no valid grade; do not replay it')
          continue
        # Upper bound for this immutable text/schema set, not a general token counter.
        ceiling=len(json.dumps(body,ensure_ascii=False).encode())+4096+body['max_tokens']
        self.reserve(call_key,ceiling)
        started=time.monotonic()
        with tempfile.TemporaryDirectory() as directory:
          payload=Path(directory)/'request.json';response=Path(directory)/'response.json'
          payload.write_text(json.dumps(body,ensure_ascii=False))
          result=subprocess.run(['curl','-q','--config','-','--proto','=https','--silent','--show-error','--max-time','180',
                                 '--header','Content-Type: application/json','--data-binary','@'+str(payload),
                                 '--output',str(response),'--write-out','%{http_code}',DahlContract.endpoint],
                                input=config,text=True,capture_output=True)
          record=self.data['calls'][call_key]
          record.update(seconds=round(time.monotonic()-started,3),curl_exit=result.returncode,http_status=result.stdout,function=index==20)
          self.save()
          if result.returncode or result.stdout!='200':
            raise ValueError('HTTP '+result.stdout+' or transport failure; reservation retained, no retry')
          if response.stat().st_size>8*1024*1024:raise ValueError('Response too large')
          data=DahlContract.streamed_response(response.read_text()) if body['stream'] else json.loads(response.read_text())
        if data.get('model')!=self.model:
          raise ValueError('Unexpected model identity; retain reservation')
        usage=DahlContract.usage(data)
        if usage['total_tokens']>ceiling:raise ValueError('Usage above reservation; stop')
        record.update(state='received',charged_or_reserved_tokens=usage['total_tokens'],usage=usage,model=data['model'],
                      finish_reasons=[c.get('finish_reason') for c in data.get('choices',[])],
                      requested_reasoning_effort=body.get('reasoning_effort'),acknowledged_reasoning_effort=data.get('reasoning_effort'),
                      answers=[{k:v for k,v in c.get('message',{}).items() if k in ['content','tool_calls','refusal']} for c in data.get('choices',[])])
        self.save()
        proposals=DahlContract.proposals(data,index==20)
        record.update(proposals=proposals,grades=EvaluationContract.grade(self.batches[index],proposals))
        self.save()
        print(self.model,index,sum(g['passed'] for g in record['grades']),'/',len(record['grades']),'tokens',usage['total_tokens'],'seconds',record['seconds'],flush=True)
        sent+=1
        if max_calls and sent>=max_calls:return


if __name__=='__main__':
  parser=argparse.ArgumentParser(description=__doc__)
  parser.add_argument('--model',choices=DahlContract.models,required=True)
  parser.add_argument('--journal',type=Path,required=True)
  parser.add_argument('--key-file',type=Path)
  parser.add_argument('--live',action='store_true')
  parser.add_argument('--max-calls',type=int)
  parser.add_argument('--phase',choices=['baseline','explicit_schema','streaming_schema','high_effort'],default='explicit_schema')
  parser.add_argument('--function-only',action='store_true')
  args=parser.parse_args()
  if args.live and not args.key_file:parser.error('Live run requires an owner-supplied key')
  if args.max_calls is not None and args.max_calls<1:parser.error('max-calls must be positive')
  evaluation=DahlEvaluation(args.journal,args.model,args.phase)
  if not args.live:
    print(json.dumps(dict(model=args.model,cases=200,function_cases=3,fingerprint=evaluation.fingerprint,token_cap=250000,network=False)))
  else:
    try:evaluation.run(args.key_file,args.max_calls,args.function_only)
    except (ValueError,KeyError,OSError) as exc:
      print('Stopped:',type(exc).__name__,str(exc) if isinstance(exc,ValueError) else 'Inspect local evidence; no blind retry')
      raise SystemExit(1)
