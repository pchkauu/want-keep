"""Summarize Dahl screening without publishing keys, balances or reasoning text."""

import argparse
from collections import Counter
import hashlib
import json
import math
from pathlib import Path
import statistics


class DahlReport:
  @staticmethod
  def phase(calls):
    text=[g for c in calls if not c.get('function') for g in c.get('grades',[])]
    functions=[g for c in calls if c.get('function') for g in c.get('grades',[])]
    times=sorted(c['seconds'] for c in calls if 'grades' in c)
    return dict(attempts=len(calls),graded_text_cases=len(text),text_passed=sum(g['passed'] for g in text),
                function_cases=len(functions),function_passed=sum(g['passed'] for g in functions),
                unique_archetypes=len({g['archetype'] for g in text}),
                complete_text_evaluation=len(text)==200 and len({g['id'] for g in text})==200,
                text_screen_pass=len(text)>=20 and len({g['archetype'] for g in text})==20 and all(g['passed'] for g in text),
                failed_cases=[g for g in text+functions if not g['passed']],
                http_statuses=dict(Counter(c.get('http_status','pending') for c in calls)),
                observed_models=sorted({c['model'] for c in calls if 'model' in c}),
                requested_effort=sorted({c['requested_reasoning_effort'] for c in calls if c.get('requested_reasoning_effort')}),
                acknowledged_effort=sorted({c['acknowledged_reasoning_effort'] for c in calls if c.get('acknowledged_reasoning_effort')}),
                observed_usage_tokens=sum(c.get('usage',{}).get('total_tokens',0) for c in calls),
                prompt_tokens=sum(c.get('usage',{}).get('prompt_tokens',0) for c in calls),
                completion_tokens=sum(c.get('usage',{}).get('completion_tokens',0) for c in calls),
                ungraded_received_answers=sum(c.get('state')=='received' and 'grades' not in c for c in calls),
                unsettled_reserved_tokens=sum(c['charged_or_reserved_tokens'] for c in calls if c['state']=='unknown'),
                call_latency_seconds=dict(p50=statistics.median(times),p95=times[math.ceil(len(times)*.95)-1]) if times else None)

  @classmethod
  def build(cls,journals):
    models={}
    for path in journals:
      raw=path.read_bytes();journal=json.loads(raw);phases={}
      for phase in ['baseline','explicit_schema','streaming_schema','high_effort']:
        calls=[c for k,c in journal['calls'].items() if (':' not in k if phase=='baseline' else k.startswith(phase+':'))]
        if calls:phases[phase]=cls.phase(calls)
      models[journal['model']]=dict(started_at=journal['started_at'],token_cap=journal['token_cap'],
                    reasoning_effort=journal['reasoning_effort'],fingerprints=journal.get('fingerprints',{}),phases=phases,
                    journal_sha256=hashlib.sha256(raw).hexdigest(),
                    total_charged_or_reserved_tokens=sum(c['charged_or_reserved_tokens'] for c in journal['calls'].values()))
    return dict(kind='synthetic_dahl_text_screen',endpoint='https://inference.dahl.global/v1/chat/completions',
                models=models,images_and_pdf='Not supported by the documented model contract; not sent.',
                direct_mvp_replacement_qualified=False,
                limitations=['Twenty text archetypes are an initial screen, not a completed 200-case qualification.',
                  'Accepted response_format does not establish server-enforced JSON schema; MiniMax emitted a reasoning prefix.',
                  'Only a complete observed think envelope is removed; financial proposals still use the same exact grader.',
                  'Provider-default reasoning is not verified as equivalent to OpenAI high.',
                  'Billing is a preallocated token pool. Usage and outstanding reservations are reported in tokens, not invented USD charges.',
                  'No real household data, images, PDF, financial command execution or production provider change.'])


if __name__=='__main__':
  parser=argparse.ArgumentParser(description=__doc__)
  parser.add_argument('output',type=Path)
  parser.add_argument('journals',type=Path,nargs='+')
  args=parser.parse_args()
  report=DahlReport.build(args.journals)
  args.output.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
  print(json.dumps({model:{phase:{k:s[k] for k in ['graded_text_cases','text_passed','function_cases','function_passed','text_screen_pass','unsettled_reserved_tokens']} for phase,s in value['phases'].items()} for model,value in report['models'].items()}))
