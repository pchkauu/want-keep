"""Publish synthetic evaluation metrics without credentials or private paths."""

import argparse
from collections import Counter
from decimal import Decimal
import hashlib
import json
import math
from pathlib import Path
import re
import statistics


class OpenAIReport:
  @staticmethod
  def model_summary(calls):
    primary=[c for c in calls if not c.get("function")]
    grades=[g for c in primary for g in c.get("grades",[])]
    function=[g for c in calls if c.get("function") for g in c.get("grades",[])]
    visual=[g for g in grades if g["archetype"].endswith(("_png","_pdf"))]
    errors=[g for g in grades if not g["passed"]]
    critical=[]
    safe_abstentions=0
    for call in primary:
      proposals={row["id"]:row for row in call.get("proposals",{}).get("results",[])}
      for grade in call.get("grades",[]):
        if grade["passed"]:
          continue
        proposal=proposals.get(grade["id"],{})
        abstained=(proposal.get("action") in ["clarify","reject"]
                   and all(proposal.get(field) is None for field in ["amount","asset","fee","target","month","kind"])
                   and proposal.get("shares")==[] and proposal.get("items")==[]
                   and not set(grade["failures"]) & {"shape","evidence"})
        if abstained:
          safe_abstentions+=1
        elif set(grade["failures"]) & {"shape","amount","asset","fee","target","month","shares","items","kind","action","evidence"}:
          critical.append(grade)
    invalid_outputs=sum("shape" in g["failures"] for g in errors)
    times=sorted(c["seconds"] for c in calls if "seconds" in c)
    complete=len(calls)==27 and len(grades)==206 and len({g['id'] for g in grades})==206 and len(function)==3 and all("grades" in c for c in calls)
    passed=sum(g["passed"] for g in grades)
    usage=Counter()
    for call in calls:
      u=call.get("usage",{})
      usage.update({k:u.get(k,0) for k in ["input_tokens","output_tokens"]})
      usage.update({k:u.get("input_tokens_details",{}).get(k,0) for k in ["cached_tokens","cache_write_tokens"]})
      usage["reasoning_tokens"]+=u.get("output_tokens_details",{}).get("reasoning_tokens",0)
    return dict(complete=complete,calls=len(calls),primary_cases=len(grades),passed=passed,
                observed_models=sorted({c["model"] for c in calls if "model" in c}),
                response_statuses=dict(Counter(c.get("response_status","unrecorded") for c in calls)),
                exact_case_accuracy=str(Decimal(passed)/Decimal(len(grades))) if grades else None,
                critical_case_errors=len(critical),unnecessary_abstentions=safe_abstentions,invalid_outputs=invalid_outputs,
                visual_passed=sum(g["passed"] for g in visual),visual_cases=len(visual),
                function_passed=sum(g["passed"] for g in function),function_cases=len(function),
                clarifications=sum(g["clarification"] for g in grades),
                qualified=complete and passed>=200 and not critical and not invalid_outputs and len(visual)==6 and all(g["passed"] for g in visual+function),
                usage=dict(usage),max_output_tokens_observed=max((c.get("usage",{}).get("output_tokens",0) for c in calls),default=0),
                cost_usd=str(sum(Decimal(c["charged_usd"]) for c in calls)),
                cost_is_conservative=any(c.get("conservative_cost",True) for c in calls),
                call_latency_seconds=dict(p50=statistics.median(times),p95=times[math.ceil(len(times)*.95)-1]) if times else None,
                failed_cases=errors)

  @classmethod
  def continued_cohort(cls, journal, model, phases):
    contracts=[journal.get('generation_contracts',{}).get(p,{}).get(model) for p in phases]
    if not contracts[0] or len(set(contracts))!=1:
      raise ValueError('Cannot combine runs with different or unknown generation contracts')
    indexed={}
    for phase in phases:
      prefix=phase+':'+model+':'
      for key,call in journal['calls'].items():
        if not key.startswith(prefix):continue
        index=int(key[len(prefix):])
        if index in indexed or not 0<=index<=26:
          raise ValueError('Continuation overlaps or contains unexpected call indices')
        if not re.fullmatch('[0-9a-f]{64}',call.get('payload_sha256','')) or call.get('function')!=(index==26):
          raise ValueError('Continuation lacks per-request payload provenance')
        indexed[index]=call
    if set(indexed)!=set(range(27)):
      raise ValueError('Continuation does not cover the complete evaluation')
    return dict(source_phases=phases,generation_contract_sha256=contracts[0],
                payload_sha256_by_index={str(i):indexed[i]['payload_sha256'] for i in range(27)},
                result=cls.model_summary([indexed[i] for i in range(27)]))

  @classmethod
  def build(cls, journal):
    phases={}
    for phase in ["baseline","candidate","qualification","high","high_contract","high_receipts","high_checked","high_final","high_final_counted","luna_xhigh","terra_xhigh"]:
      models={}
      for model in ["gpt-5.6-luna","gpt-5.6-terra"]:
        prefix=("" if phase=="baseline" else phase+":")+model+":"
        calls=[c for key,c in journal["calls"].items() if key.startswith(prefix)]
        if calls:models[model]=cls.model_summary(calls)
      if models:phases[phase]=models
    cohorts={}
    if 'high_final_counted' in phases:
      cohorts['terra_high_final']=cls.continued_cohort(journal,'gpt-5.6-terra',['high_final','high_final_counted'])
    return dict(kind="measured_synthetic_model_eval",started_at=journal["started_at"],cap_usd=journal["cap_usd"],
                cap_authorizations=journal.get('cap_authorizations',[]),cohorts=cohorts,
                charged_or_reserved_usd=str(sum(Decimal(c["charged_usd"]) for c in journal["calls"].values())),
                unresolved_calls=sum(c.get('state')!='received' for c in journal["calls"].values()),
                ungraded_received_calls=sum(c.get('state')=='received' and 'grades' not in c for c in journal['calls'].values()),
                fingerprints=journal.get("fingerprints",{"baseline":journal["fingerprint"]}),
                phases=phases,limitations=[
                  "Baseline was exploratory: ambiguous proposal instructions and a missing report year were corrected before candidate qualification.",
                  "Terra final text and visual/function continuation have identical generation contracts and non-overlapping request indices; cost is counted once in phases.",
                  "Prompt calibration repeatedly used the same known cases; this is not an independent holdout.",
                  "Text cases use batches of ten; latency is per API call, not per transaction or production request.",
                  "Two hundred text cases are variants of twenty archetypes; six visual cases are three clean documents in two formats.",
                  "Measured usage cost uses the dated pricing snapshot; not an invoice or production-family monthly bill.",
                  "No production commands, real household data, multi-page boundary or daily user-flow acceptance."])


if __name__=="__main__":
  parser=argparse.ArgumentParser(description=__doc__)
  parser.add_argument("journal",type=Path)
  parser.add_argument("output",type=Path)
  args=parser.parse_args()
  raw=args.journal.read_bytes()
  report=OpenAIReport.build(json.loads(raw))
  report["journal_sha256"]=hashlib.sha256(raw).hexdigest()
  args.output.write_text(json.dumps(report,ensure_ascii=False,indent=2)+"\n")
  print(json.dumps({"unresolved_calls":report["unresolved_calls"],"charged_or_reserved_usd":report["charged_or_reserved_usd"],"phases":{phase:{m:{k:s[k] for k in ["complete","passed","primary_cases","qualified"]} for m,s in models.items()} for phase,models in report['phases'].items()}}))
