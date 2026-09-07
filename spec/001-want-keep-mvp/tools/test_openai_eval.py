"""Offline checks for experiment integrity, scoring and paid-run containment."""

import copy
from decimal import Decimal
import hashlib
import json
from pathlib import Path
import tempfile
import unittest

from openai_cases import OpenAICases
from openai_eval import EvaluationContract, OpenAIEvaluation, ResearchBudget
from openai_report import OpenAIReport


class OpenAIEvaluationTest(unittest.TestCase):
  def setUp(self):
    self.cases=OpenAICases.build()

  def output(self, cases):
    return {"results":[dict(id=c["id"],explanation="Synthetic reference",**copy.deepcopy(c["expected"])) for c in cases]}

  def test_suite_is_deterministic_and_has_unique_ids(self):
    self.assertEqual(self.cases,OpenAICases.build())
    self.assertEqual(206,len({c["id"] for c in self.cases}))
    self.assertEqual(20,len({c["archetype"] for c in self.cases[:200]}))

  def test_request_excludes_gold_and_has_no_executing_tools(self):
    assets=Path(__file__).resolve().parents[1]/"assets/openai-eval"
    payload=EvaluationContract.payload("gpt-5.6-luna",self.cases[:10],assets)
    self.assertNotIn("expected",json.dumps(payload))
    self.assertNotIn("archetype",json.dumps(payload))
    self.assertNotIn("tools",payload)
    self.assertFalse(payload["store"])
    self.assertEqual({"effort":"high"},payload["reasoning"])
    self.assertEqual({"mode":"explicit"},payload["prompt_cache_options"])

  def test_synthetic_references_pass_exact_grading(self):
    self.assertTrue(all(g["passed"] for g in EvaluationContract.grade(self.cases,self.output(self.cases))))

  def test_float_and_exponent_money_are_rejected(self):
    for amount in [100.0,"1e2","NaN","Infinity","-1",True]:
      response=self.output(self.cases[:1]);response["results"][0]["amount"]=amount
      self.assertIn("amount",EvaluationContract.grade(self.cases[:1],response)[0]["failures"])

  def test_duplicate_id_and_missing_row_are_rejected(self):
    response=self.output(self.cases[:2])
    response["results"][1]["id"]=response["results"][0]["id"]
    with self.assertRaises(ValueError):EvaluationContract.grade(self.cases[:2],response)
    with self.assertRaises(ValueError):EvaluationContract.grade(self.cases[:2],{"results":[]})

  def test_transfer_principal_as_expense_is_a_failure(self):
    case=next(c for c in self.cases if c["archetype"]=="transfer")
    response=self.output([case]);response["results"][0]["kind"]="expense"
    self.assertIn("kind",EvaluationContract.grade([case],response)[0]["failures"])

  def test_injection_altered_total_is_a_failure(self):
    case=next(c for c in self.cases if c["archetype"]=="injection_pdf")
    response=self.output([case]);response["results"][0]["amount"]="9000"
    self.assertIn("amount",EvaluationContract.grade([case],response)[0]["failures"])

  def test_unauthorized_command_is_a_failure(self):
    case=next(c for c in self.cases if c["archetype"]=="foreign_goal")
    response=self.output([case]);response["results"][0]["action"]="create"
    self.assertIn("action",EvaluationContract.grade([case],response)[0]["failures"])

  def test_usage_counts_cache_writes_and_reasoning_once(self):
    usage=dict(input_tokens=1000,output_tokens=200,input_tokens_details=dict(cached_tokens=300,cache_write_tokens=400),output_tokens_details=dict(reasoning_tokens=100))
    cost,conservative=EvaluationContract.usage_cost("gpt-5.6-terra",usage)
    self.assertEqual(Decimal("0.00406"),cost)
    self.assertFalse(conservative)

  def test_missing_write_usage_retains_conservative_charge(self):
    cost,conservative=EvaluationContract.usage_cost("gpt-5.6-luna",dict(input_tokens=1000,output_tokens=200,input_tokens_details=dict(cached_tokens=0)))
    self.assertEqual(Decimal("0.00049"),cost)
    self.assertTrue(conservative)

  def test_inconsistent_usage_is_rejected(self):
    for detail in [dict(cached_tokens=101),dict(cached_tokens=50,cache_write_tokens=51),dict(cached_tokens=True),dict(cached_tokens=-1)]:
      with self.assertRaises(ValueError):
        EvaluationContract.usage_cost("gpt-5.6-luna",dict(input_tokens=100,output_tokens=20,input_tokens_details=detail))

  def test_budget_survives_restart_and_rejects_unknown_replay(self):
    with tempfile.TemporaryDirectory() as directory:
      path=Path(directory)/"journal.json"
      budget=ResearchBudget(path,Decimal("1"),"same")
      budget.reserve("first",Decimal("0.7"))
      budget.lock.close()
      resumed=ResearchBudget(path,Decimal("1"),"same")
      self.assertEqual("unknown",resumed.data["calls"]["first"]["state"])
      with self.assertRaises(ValueError):resumed.reserve("first",Decimal("0.1"))
      with self.assertRaises(ValueError):resumed.reserve("second",Decimal("0.4"))
      resumed.lock.close()

  def test_budget_does_not_silently_release_over_ceiling_charge(self):
    with tempfile.TemporaryDirectory() as directory:
      budget=ResearchBudget(Path(directory)/"journal.json",Decimal("1"),"same")
      budget.reserve("first",Decimal("0.1"))
      with self.assertRaises(ValueError):budget.settle("first",Decimal("0.2"))
      self.assertEqual("unknown",budget.data["calls"]["first"]["state"])
      budget.lock.close()

  def test_non_finite_or_unapproved_cap_is_rejected(self):
    for cap in ["NaN","Infinity","0","-1","7.01"]:
      with self.assertRaises(ValueError):ResearchBudget(Path("unused"),Decimal(cap),"same")

  def test_existing_journal_cap_cannot_be_raised_by_cli(self):
    with tempfile.TemporaryDirectory() as directory:
      path=Path(directory)/'journal.json'
      budget=ResearchBudget(path,Decimal('3'),'same')
      budget.reserve('earlier',Decimal('2'));budget.lock.close()
      with self.assertRaises(ValueError):ResearchBudget(path,Decimal('7'),'same')
      self.assertEqual('3',json.loads(path.read_text())['cap_usd'])

  def test_extra_high_changes_only_requested_reasoning(self):
    assets=Path(__file__).resolve().parents[1]/'assets/openai-eval'
    high=EvaluationContract.payload('gpt-5.6-luna',self.cases[:10],assets)
    extra=EvaluationContract.payload('gpt-5.6-luna',self.cases[:10],assets,reasoning_effort='xhigh')
    self.assertEqual({'effort':'xhigh'},extra['reasoning'])
    extra['reasoning']=high['reasoning']
    self.assertEqual(high,extra)

  def test_published_selected_payloads_match_the_measured_generation_contract(self):
    root=Path(__file__).resolve().parents[1]
    manifest=json.loads((root/'evidence/openai.eval.json').read_text())
    batches=[self.cases[i:i+10] for i in range(0,200,10)]+[[c] for c in self.cases[200:]]+[self.cases[:3]]
    payloads=[EvaluationContract.payload(manifest['selected_model'],batch,root/'assets/openai-eval',i==26,
              manifest['reasoning_effort'],manifest['max_output_tokens']) for i,batch in enumerate(batches)]
    actual=hashlib.sha256(json.dumps(payloads,sort_keys=True).encode()).hexdigest()
    self.assertEqual(manifest['selected_generation_contract_sha256'],actual)

  def test_calibration_phase_cannot_reset_total_spend(self):
    with tempfile.TemporaryDirectory() as directory:
      path=Path(directory)/"journal.json"
      budget=ResearchBudget(path,Decimal("1"),"v1")
      budget.reserve("first",Decimal("0.7"))
      budget.settle("first",Decimal("0.6"),grades=[])
      budget.lock.close()
      candidate=ResearchBudget(path,Decimal("1"),"v2",phase="candidate")
      with self.assertRaises(ValueError):candidate.reserve("candidate:next",Decimal("0.5"))
      self.assertEqual({"baseline":"v1","candidate":"v2"},candidate.data["fingerprints"])
      candidate.lock.close()

  def test_unknown_previous_phase_blocks_new_calibration(self):
    with tempfile.TemporaryDirectory() as directory:
      path=Path(directory)/"journal.json"
      budget=ResearchBudget(path,Decimal("1"),"v1")
      budget.reserve("first",Decimal("0.1"));budget.lock.close()
      with self.assertRaises(ValueError):ResearchBudget(path,Decimal("1"),"v2",phase="candidate")

  def test_paid_incomplete_response_remains_charged_without_blocking_a_new_phase(self):
    with tempfile.TemporaryDirectory() as directory:
      path=Path(directory)/'journal.json'
      budget=ResearchBudget(path,Decimal('1'),'v1')
      budget.reserve('first',Decimal('0.7'))
      budget.settle('first',Decimal('0.6'),response_status='incomplete',usage={'output_tokens':4096})
      budget.lock.close()
      next_run=ResearchBudget(path,Decimal('1'),'v2',phase='candidate')
      with self.assertRaises(ValueError):next_run.reserve('first',Decimal('0.1'))
      with self.assertRaises(ValueError):next_run.reserve('next',Decimal('0.5'))
      self.assertEqual('0.6',next_run.data['calls']['first']['charged_usd'])
      next_run.lock.close()

  def test_visual_cases_and_reservations_are_bounded(self):
    evaluation=OpenAIEvaluation(Path(__file__).resolve().parents[1])
    for case in self.cases[200:]:
      body=EvaluationContract.payload("gpt-5.6-terra",[case],evaluation.assets)
      self.assertLess(EvaluationContract.reservation("gpt-5.6-terra",body),Decimal("1"))
      self.assertEqual("high",body["input"][0]["content"][1]["detail"])

  def test_incomplete_or_critical_failure_cannot_qualify(self):
    self.assertFalse(OpenAIReport.model_summary([])["qualified"])
    calls=[]
    batches=[self.cases[i:i+10] for i in range(0,200,10)]+[[c] for c in self.cases[200:]]
    for batch in batches+[self.cases[:3]]:
      calls.append(dict(grades=EvaluationContract.grade(batch,self.output(batch)),charged_usd="0.01",function=len(calls)==26))
    self.assertTrue(OpenAIReport.model_summary(calls)["qualified"])
    calls[0]["grades"][0].update(passed=False,failures=["amount"])
    self.assertFalse(OpenAIReport.model_summary(calls)["qualified"])

  def test_monthly_estimate_is_reproducible_without_cache_discounts(self):
    source=Path(__file__).resolve().parents[1]/"evidence/openai.cost.json"
    estimate=json.loads(source.read_text())
    for profile in estimate["profiles"].values():
      total=Decimal("0");stress=Decimal("0")
      for row in profile["breakdown"]:
        inp,_,out=EvaluationContract.prices[row["model"]]
        base=row["calls"]*(row["input_tokens_per_call"]*inp+row["output_tokens_per_call"]*out)/1000000
        self.assertEqual(Decimal(row["cost_usd"]),base)
        total+=base
        stress+=row["calls"]*(row["input_tokens_per_call"]*inp*Decimal("1.25")+row["output_tokens_per_call"]*out)/1000000*Decimal("1.25")
      self.assertEqual(Decimal(profile["baseline_usd"]),total)
      self.assertEqual(Decimal(profile["cache_write_and_retry_stress_usd"]),stress)
    selected=estimate["profiles"][estimate["selected_profile"]]
    self.assertLess(Decimal(selected["baseline_usd"]),Decimal(estimate["monthly_cap"]))
    self.assertGreater(Decimal(selected["cache_write_and_retry_stress_usd"]),Decimal(estimate["monthly_cap"]))
    self.assertIn("queued",selected["stress_budget_behavior"])

  def test_unnecessary_clarification_fails_accuracy_but_is_not_wrong_money(self):
    batch=self.cases[:1]
    response=self.output(batch)
    response["results"][0].update(OpenAICases.expected("clarify",evidence=[batch[0]["input"]["source"]]))
    call=dict(grades=EvaluationContract.grade(batch,response),proposals=response,charged_usd="0.01",function=False)
    summary=OpenAIReport.model_summary([call])
    self.assertEqual(0,summary["passed"])
    self.assertEqual(1,summary["unnecessary_abstentions"])
    self.assertEqual(0,summary["critical_case_errors"])
    response["results"][0]["amount"]="999"
    call["grades"]=EvaluationContract.grade(batch,response)
    self.assertEqual(1,OpenAIReport.model_summary([call])["critical_case_errors"])

  def test_continuation_requires_identical_contract_and_unique_complete_coverage(self):
    journal=dict(generation_contracts={p:{'model':'same'} for p in ['first','second']},calls={})
    for i in range(27):
      phase='first' if i<20 else 'second'
      journal['calls'][f'{phase}:model:{i}']=dict(payload_sha256='a'*64,function=i==26,charged_usd='0')
    combined=OpenAIReport.continued_cohort(journal,'model',['first','second'])
    self.assertEqual(27,combined['result']['calls'])
    self.assertFalse(combined['result']['qualified'])
    journal['generation_contracts']['second']['model']='different-effort'
    with self.assertRaises(ValueError):OpenAIReport.continued_cohort(journal,'model',['first','second'])
    journal['generation_contracts']['second']['model']='same'
    journal['calls']['second:model:0']=journal['calls']['first:model:0']
    with self.assertRaises(ValueError):OpenAIReport.continued_cohort(journal,'model',['first','second'])
    del journal['calls']['second:model:0'];del journal['calls']['first:model:0']
    with self.assertRaises(ValueError):OpenAIReport.continued_cohort(journal,'model',['first','second'])


if __name__=="__main__":
  unittest.main()
