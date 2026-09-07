"""Offline checks for the separate text-only provider boundary."""

import copy
import json
from pathlib import Path
import tempfile
import unittest

from dahl_eval import DahlContract, DahlEvaluation
from dahl_report import DahlReport
from openai_cases import OpenAICases


class DahlEvaluationTest(unittest.TestCase):
  def test_payload_has_same_cases_without_gold_or_openai_only_parameters(self):
    payload=DahlContract.payload(DahlContract.models[0],OpenAICases.build()[:10])
    self.assertNotIn('expected',json.dumps(payload))
    self.assertNotIn('reasoning',payload)
    self.assertNotIn('reasoning_effort',payload)
    self.assertNotIn('store',payload)
    self.assertTrue(payload['response_format']['json_schema']['strict'])

  def test_visual_cases_are_explicitly_outside_this_model_contract(self):
    with self.assertRaises(ValueError):DahlContract.payload(DahlContract.models[0],OpenAICases.build()[200:201])

  def test_high_phase_is_explicit_and_changes_the_experiment_fingerprint(self):
    for model in DahlContract.models:
      default=DahlEvaluation(Path('unused'),model,'streaming_schema')
      high=DahlEvaluation(Path('unused'),model,'high_effort')
      self.assertNotEqual(default.fingerprint,high.fingerprint)
      self.assertEqual('high',high.payloads[0]['reasoning_effort'])
      self.assertTrue(high.payloads[0]['stream'])
      if model.startswith('deepseek'):self.assertEqual({'type':'enabled'},high.payloads[0]['thinking'])

  def test_incomplete_and_refusal_are_not_proposals(self):
    data={'choices':[{'finish_reason':'length','message':{'content':'{}'}}]}
    with self.assertRaises(ValueError):DahlContract.proposals(data,False)
    data['choices'][0].update(finish_reason='stop',message={'refusal':'no'})
    with self.assertRaises(ValueError):DahlContract.proposals(data,False)

  def test_reasoning_content_is_not_parsed_as_json(self):
    data={'choices':[{'finish_reason':'stop','message':{'reasoning_content':'not a proposal','content':'{"results":[]}'}}]}
    self.assertEqual({'results':[]},DahlContract.proposals(data,False))

  def test_only_complete_observed_reasoning_envelope_is_removed(self):
    self.assertEqual({'results':[]},DahlContract.answer_json('<think>discard</think> {"results":[]}'))
    for text in ['<think>incomplete','preamble {"results":[]}','<think>a</think>b</think>{"results":[]}']:
      with self.assertRaises(ValueError):DahlContract.answer_json(text)

  def test_unexpected_tool_is_rejected(self):
    data={'choices':[{'finish_reason':'tool_calls','message':{'tool_calls':[{'type':'function','function':{'name':'send_payment','arguments':'{}'}}]}}]}
    with self.assertRaises(ValueError):DahlContract.proposals(data,True)

  def test_stream_requires_finish_usage_and_done_and_ignores_reasoning(self):
    chunks=[dict(model=DahlContract.models[0],choices=[dict(index=0,delta={'content':'{"results":','reasoning_content':'discard'})]),
            dict(choices=[dict(index=0,delta={'content':'[]}'},finish_reason='stop')]),
            dict(choices=[],usage=dict(prompt_tokens=100,completion_tokens=200,total_tokens=300))]
    text='\n'.join('data: '+json.dumps(c) for c in chunks)+'\ndata: [DONE]\n'
    data=DahlContract.streamed_response(text)
    self.assertEqual({'results':[]},DahlContract.proposals(data,False))
    self.assertEqual(300,DahlContract.usage(data)['total_tokens'])
    with self.assertRaises(ValueError):DahlContract.streamed_response(text.replace('data: [DONE]',''))

  def test_stream_reassembles_only_the_named_tool(self):
    chunks=[dict(model=DahlContract.models[0],choices=[dict(index=0,delta={'tool_calls':[dict(index=0,id='call-1',type='function',function={'name':'propose_accounting','arguments':'{"results":'})]})]),
            dict(choices=[dict(index=0,delta={'tool_calls':[dict(index=0,function={'arguments':'[]}'})]},finish_reason='tool_calls')]),
            dict(choices=[],usage=dict(prompt_tokens=100,completion_tokens=200,total_tokens=300))]
    text='\n'.join('data: '+json.dumps(c) for c in chunks)+'\ndata: [DONE]'
    self.assertEqual({'results':[]},DahlContract.proposals(DahlContract.streamed_response(text),True))

  def test_usage_includes_output_once_and_rejects_inconsistent_counts(self):
    data={'usage':dict(prompt_tokens=100,completion_tokens=200,total_tokens=300,completion_tokens_details={'reasoning_tokens':50})}
    self.assertEqual(300,DahlContract.usage(data)['total_tokens'])
    for total in [True,-1,301]:
      invalid=copy.deepcopy(data);invalid['usage']['total_tokens']=total
      with self.assertRaises(ValueError):DahlContract.usage(invalid)

  def test_reserved_tokens_survive_reload_and_block_over_cap_or_replay(self):
    with tempfile.TemporaryDirectory() as directory:
      path=Path(directory)/'run.json';evaluation=DahlEvaluation(path,DahlContract.models[0])
      evaluation.data={'calls':{}};evaluation.reserve(0,240000)
      resumed=DahlEvaluation(path,DahlContract.models[0]);resumed.data=json.loads(path.read_text())
      with self.assertRaises(ValueError):resumed.reserve(0,1)
      with self.assertRaises(ValueError):resumed.reserve(1,10001)
      self.assertEqual('unknown',resumed.data['calls']['0']['state'])

  def test_http_success_without_a_complete_stream_is_not_graded_or_acknowledged(self):
    call=dict(state='unknown',http_status='200',charged_or_reserved_tokens=22000,requested_reasoning_effort='high')
    report=DahlReport.phase([call])
    self.assertEqual(0,report['graded_text_cases'])
    self.assertEqual(22000,report['unsettled_reserved_tokens'])
    self.assertEqual([],report['acknowledged_effort'])
    self.assertFalse(report['text_screen_pass'])


if __name__=='__main__':unittest.main()
