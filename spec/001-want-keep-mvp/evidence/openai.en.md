# OpenAI: model selection, quality and cost

[Русский](openai.md) · [task-0.8 / Issue #8](https://github.com/pchkauu/want-keep/issues/8)

Sources checked: 2026-09-07. REQ-018–REQ-020, REQ-022, REQ-051, REQ-060–REQ-061; AC-018–AC-020, AC-022, AC-051, AC-060, AC-069, AC-076. All examples are synthetic.

## Status and selection rule

Research is complete. The MVP selects **`gpt-5.6-terra`, `reasoning.effort=xhigh`**, Standard foreground Responses and a strict proposal schema. The model-selection contract is closed and task-0.10 issued SDD Ready. The full MVP still requires task-5.x/task-8.x runtime and product ACs. Final live evaluation: 206/206 exact cases (100%), no unnecessary clarifications, zero critical errors, 6/6 PNG/PDF and 3/3 function-calling cases. This is bounded synthetic evidence, not a guarantee of perfect real-data accuracy.

Initial comparison used exact API IDs `gpt-5.6-luna` and `gpt-5.6-terra` with `reasoning.effort=low`; subsequent phases evaluated Terra high, Luna xhigh and the owner’s final Terra xhigh selection separately. Current pages offer no separate dated snapshot for these models: do not invent a date suffix or call an ID an immutable snapshot. Record requested/returned model, prompt/schema version, pricing and run date; a model/alias change requires renewed qualification.

**One Terra Extra High route for all MVP AI tasks:** transaction review, receipts, chat and insights. The owner explicitly selected it after comparison. Luna xhigh scored 30/30 in its first three batches, then returned incomplete at 4096 output tokens; remaining cases were not run. MiniMax/DeepSeek through Dahl received separate limited screens, including requested high; applied effort was not confirmed. [RU report](dahl.md) / [EN](dahl.en.md). This does not establish universal model superiority. No automatic model/effort substitution.

Qualification: ≥97% exact-case accuracy, zero critical financial/authority/provenance errors, 6/6 visual cases, 3/3 function cases and valid complete output. Unnecessary clarification remains an accuracy failure; an empty financial payload is not an incorrect amount or posting. Guessing instead of asking, incorrect amounts/links/allocations or exceeded authority are critical. The grader was corrected to reflect this distinction; no failed case became a correct case, and gold/thresholds were not relaxed.

## OAI-E01–E04: models and pricing

Standard, short context, USD per 1M tokens; published prices, not a measured invoice:

| ID | Input | Cached input | Cache write | Output, including reasoning |
| --- | --- | --- | --- | --- |
| gpt-5.6-luna | 0.20 | 0.02 | 0.25 | 1.20 |
| gpt-5.6-terra | 2.00 | 0.20 | 2.50 | 12.00 |
| gpt-5.6-sol | 4.00 | 0.40 | 5.00 | 20.00 |

OAI-E01 [Luna](https://developers.openai.com/api/docs/models/gpt-5.6-luna), OAI-E02 [Terra](https://developers.openai.com/api/docs/models/gpt-5.6-terra): text/image input, text output, Responses, function calling and Structured Outputs are supported; this does not establish receipt accuracy. OAI-E03 [Sol](https://developers.openai.com/api/docs/models/gpt-5.6-sol) is outside the first comparison: first determine whether either cheaper model is sufficient. Its quality is unmeasured.

OAI-E04 [Pricing](https://developers.openai.com/api/docs/pricing): exclude Batch/Flex/priority discounts and credits from the baseline. Above 272K input the entire request costs more; permitted sizes stay below that threshold. No hosted tools are included. Project access, tier, taxes and actual charges require separate verification.

## OAI-E05–E09: API and documents

OAI-E05 [Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs): `text.format` with `type=json_schema`, `strict=true`; all fields required, unknown values nullable, `additionalProperties=false`. Schema compliance does not establish truth. Refusal, incomplete output, JSON error and a valid proposal are distinct outcomes.

OAI-E06 [Function calling](https://developers.openai.com/api/docs/guides/function-calling): expose only allowed application proposals, with `strict=true`, `parallel_tool_calls=false`. The server supplies actor, family, permissions and current revision outside model output. Receiving a function call does not execute accounting. Research `propose_accounting` only returns JSON; evaluation has no real commands.

OAI-E07 [File inputs](https://developers.openai.com/api/docs/guides/file-inputs): PDF sends both text and page images; inline base64 avoids creating Files API objects. The provider's 50 MB limit does not replace Want Keep's 10 MiB/10-page limit. PDF support does not prove successful processing of a ten-page user document.

OAI-E08 [Vision](https://developers.openai.com/api/docs/guides/images-vision): explicitly use `detail=high`; for this family it bounds dimensions to 2048 px and 2500 patches. Apply the image multiplier 1.2 once; auto/original do not impose the same ceiling. This yields approximately 3000 input tokens per page/image, allowing for rounding; text and schema are additional. Unreadable numbers require clarification, not invented amounts.

OAI-E09 [Reasoning](https://developers.openai.com/api/docs/guides/reasoning): `max_output_tokens` covers visible and hidden tokens; an incomplete response with no useful output can still incur cost. Do not add `reasoning_tokens` again to the already complete `output_tokens`.

## OAI-E10–E13: cost and retention

OAI-E10 [Prompt caching](https://developers.openai.com/api/docs/guides/prompt-caching): GPT-5.6 bills cache reads and writes separately. The initial MVP selects `prompt_cache_options.mode=explicit` with no breakpoints: the current contract says this creates no prompt-cache reads/writes. Do not assume the old `prompt_cache_retention=in_memory` parameter. Even with caching disabled, validate actual usage and reserve conservatively.

`cost = ((input − cached − writes) × inputPrice + cached × cachedPrice + writes × 1.25 × inputPrice + output × outputPrice) / 1_000_000`.

If write count is absent, charge every non-cached input at 1.25× and mark the estimate conservative pending reconciliation; absent does not mean zero. Negative/non-integer usage, cached+writes>input or cost exceeding the reservation stops new calls pending investigation.

OAI-E11 [Token counting](https://developers.openai.com/api/docs/guides/token-counting): obtain the full request's input count before a production call, including schema/images/PDF. This is external read-only preprocessing of the same permitted data, not local character counting. If counting is unavailable or its contract unverified, the document waits; never substitute `characters/4`. The research tool uses an inflated bound only for the six inspected, immutable single-page fixtures; it is not a general production PDF counter.

OAI-E12 [Spend limits](https://developers.openai.com/api/docs/guides/spend-limits): alerts and hard limits exist, but enforcement can lag. The application's atomic reservation remains mandatory. Distinguish request-rate and monetary 429 errors; never raise limits automatically.

OAI-E13 [Data controls](https://developers.openai.com/api/docs/guides/your-data): API data is not used for training without opt-in. Abuse monitoring is usually up to 30 days with legal/service-protection exceptions; Responses `store=false` is not ZDR. Prompt caching has separate retention rules, including up to 24 hours when used. Images/files can be retained for prohibited-content review even under special controls. Project ZDR and EU residency are unverified; hosting Want Keep in Germany does not provide them. Use foreground Responses, local conversation history, `store=false` and inline documents; no Conversations/Files/vector stores/Batch/background in this route. Exclude secrets and unrelated data before the API and from telemetry.

## Target route limits

| Purpose | Maximum input tokens | Maximum output tokens | Next action |
| --- | --- | --- | --- |
| One transaction review | 8192 | 2048 | One proposal or clarification |
| One receipt page and context | 16384 | 4096 | Up to 10 pages sequentially, followed by code-validated totals |
| Chat / insight | 16384 | 4096 | Bounded relevant context and calculation references |
| Complex clarification | 32768 | 8192 | At most one additional paid call if reserved funds allow |

A global input ceiling of 262144 protects the pricing boundary. These are ceilings, not the size of every request. Preserve the entire uploaded document; per-page processing must not discard remaining items. Split oversized context along evidence boundaries while preserving revision; never report a truncated document as processed. Existing 10 MiB/10-page limits remain, with boundary runtime checks owned by task-5.3.

At most two concurrent family calls; disable SDK retries. Permit one bounded retry only after a confirmed retryable rejection and a new reservation. Timeout/unknown outcome never triggers a duplicate. HTTP 401/403, refusal, budget exhaustion and version conflicts are not solved by switching models. After one additional analysis, unresolved ambiguity remains a user question.

The $50 budget is shared by the family and uses UTC months: actual + reserved + unknown. Reserve each request/retry first; unresolved costs survive month rollover. Changing project/key/model cannot reset accounting. Ordinary import/manual accounting continues with AI waiting_budget/waiting_provider. Before production, reconcile project funds and any applicable taxes/fees with task-0.9's total estimate; API prices are not bank charges.

## Monthly estimate — not yet measured

[Reproducible calculation](openai.cost.json): 600 transaction reviews, 200 rechecks, 150 receipt pages, 400 chat messages, 60 insights and 50 complex analyses per month. These are family workload assumptions, not observed usage. Full input includes prompt, schema and vision; output includes reasoning.

| Candidate configuration | Baseline, USD | All input as cache writes + 25% retries, USD |
| --- | --- | --- |
| Luna for reviews/rechecks, Terra for the rest | 23.716 | 32.58250 |
| Terra only, earlier token profile | 32.50 | 44.6875 |
| Terra high, preceding profile (+50% output) | 42.25 | 56.875 |
| **Terra Extra High (+75% output), selected** | **47.125** | **62.96875** |

The selected xhigh profile adds 75% to the initial output assumptions, including reasoning once. Call counts and input remain unchanged. This is an explicit planning buffer, not measured household consumption or a fixed xhigh multiplier. Baseline USD 47.125 fits USD 50; stress USD 62.96875 does not: some AI work waits while ordinary accounting continues. Taxes/other project use reduce available reservations at launch. No automatic downgrade or cap increase. The budget bounds spending rather than guaranteeing processing of any workload. Earlier profiles remain as assumption history.

## Evaluation: reproduction and evidence limits

[Manifest](openai.eval.json), [cases](../tools/openai_cases.py), [runner](../tools/openai_eval.py), [runner checks](../tools/test_openai_eval.py). Python standard library; temporary rendering tools are not application dependencies. Six assets live under `assets/openai-eval/`; synthetic PDFs were visually checked through PNG rendering.

200 text cases = 20 archetypes × 10 amount/ID variants, not 200 independent situations. Coverage includes income, expense, transfer, exchange/fee, credit-card repayment, duplicate receipt, ambiguous match, missing account, cross-month refund, shared allocation, partner purchase, mixed receipt, foreign-goal rejection, stale revision, injection, irrelevant document, unknown FX, pending and grounded insight. Six additional cases are a clean receipt, injected receipt and menu, each as PNG/PDF. A separate function-calling probe on three cases checks response form outside the main score.

Gold answers are never sent to the model. Grading compares action, economic kind, exact Decimal amounts, fee, link, month, allocations, items and provenance. Explanations also need manual reading: numeric matches do not prove text quality. Invalid output never becomes a correct clarification. Preserve previous reports on reruns; prompt/fixture changes receive a new fingerprint, and passing runs never overwrite failures.

```sh
python3 spec/001-want-keep-mvp/tools/openai_eval.py
python3 -m unittest discover -s spec/001-want-keep-mvp/tools -p 'test_openai_eval.py'
make docs-check
```

Default execution uses no network. Live execution requires a separately supplied key, `--live`, `--run-cap-usd` ≤7 and `--journal` in a local ignored directory. Supply the key through `--key-file`, not command text/logs. One durable journal retains the reservation until usage is received, rejects parallel runs and unknown-outcome replay, and cannot raise its cap automatically. This CLI is not the production gateway and executes no financial commands.

Live model evaluation, function calling and PNG/PDF ran with the owner's key. Local regression tests check the tool, estimate and verdict correctness separately from model calls. Clean synthetic documents do not establish accuracy for wrinkled/blurred photos, multi-page OCR or long family chats. Full acceptance criteria and the user's 45-minute daily workflow are verified against the application in task-9.1.

## Handoff

- task-5.1: Responses gateway, usage/cache accounting, atomic reservations/UTC/unknown, access and bounded retries; implement the decisions above with production tests.
- task-5.2: only the server validator can apply commands after actor/household/revision/evidence checks; evaluation scores grant no authority.
- task-5.3: MIME/10 MiB/10-page validation, PDF/vision, item totals, matching/dedup and text/image injection; real UI/runtime checks remain required.
- task-5.5: only validated aggregates/references, forecast/partial distinction and explanation checks.
- task-0.10: BLK-08 is closed by the actual report and selected contract; other blockers and the full Ready review remain.

## OAI-E14: measured results and self review

[Usage/errors](openai.results.json), [prompt/schema history](openai.prompts.json), [report generator](../tools/openai_report.py). **301 OpenAI generation requests**, **2329 repeated graded assessments**, not 2329 independent transactions. Total usage cost **USD 3.2926944** fits the owner's increased USD 7 cap. One Luna incomplete response was paid and ungraded; its cost remains recorded. No unresolved OpenAI charges. This is a usage/pricing calculation, not an invoice. Dahl usage/unknown reservations are separate in its [report](dahl.en.md).

| Phase | Model / effort | Exact cases | PNG/PDF | Function | USD |
| --- | --- | --- | --- | --- | --- |
| baseline | gpt-5.6-luna / low | 185/206 | 4/6 | 3/3 | 0.0352258 |
| baseline | gpt-5.6-terra / low | 156/206 | 2/6 | 2/3 | 0.289510 |
| candidate | gpt-5.6-luna / low | 201/206 | 6/6 | 2/3 | 0.0346898 |
| candidate | gpt-5.6-terra / low | 204/206 | 4/6 | 3/3 | 0.308102 |
| qualification | gpt-5.6-terra / low | 204/206 | 6/6 | 3/3 | 0.301718 |
| high | gpt-5.6-terra / high | 204/206 | 6/6 | 3/3 | 0.373790 |
| high_contract | gpt-5.6-terra / high | 205/206 | 6/6 | 3/3 | 0.364784 |
| high_receipts | gpt-5.6-terra / high | 204/206 | 5/6 | 3/3 | 0.372824 |
| high_checked | gpt-5.6-terra / high | 205/206 | 6/6 | 3/3 | 0.375104 |
| high_final | gpt-5.6-terra / high | 200/200 | 0/0 | 0/0 | 0.337004 |
| high_final_counted | gpt-5.6-terra / high | 6/6 | 6/6 | 3/3 | 0.037644 |
| luna_xhigh | gpt-5.6-luna / xhigh | 30/30 | 0/0 | 0/0 | 0.0167428 |
| terra_xhigh | gpt-5.6-terra / xhigh | 206/206 | 6/6 | 3/3 | 0.445556 |

Baseline/candidate clarified ambiguous fees, linked/report fields, year, family, menu skipping and source references. Low qualification passed with two unnecessary safe questions, retained as accuracy failures. Initial high phases failed qualification through missing linked-receipt kind, omitted/incorrect items and absent report target. Prompt changes clarified these duties without relaxing gold or thresholds. Repeated calibration on known cases is not an independent holdout or proof of flawless behavior.

High_final covered 200 text cases and stopped **before submitting** documents because a base64-byte reservation was overly conservative. High_final_counted continued only the remaining six documents and three function cases. Archived generation payloads are identical and indices do not overlap; the report checks these conditions. Combined Terra high: **206/206 +3/3**, cost **USD 0.374648**. This continues one set without cherry-picking successful responses; phase costs are counted once.

**Final separate Terra Extra High / terra_xhigh run: 206/206, PNG/PDF 6/6, function 3/3, zero critical errors or unnecessary clarifications.** Output cap 8192 with input counting before every request. Input **46228**, output **29425**, including **12831 reasoning** already within output. Cached/cache-write tokens: 0/0. Cost **USD 0.445556**; P50 **11.275 s**, P95 **17.221 s** per API call, largest response **1936** tokens. Text batches contain ten cases, documents one: this is not UI latency or a production SLA.

Thirty-five final xhigh explanations were manually read: OAI-001–OAI-020, the other nine monthly insights and OAI-201–OAI-206. Checks covered rationale, forecast/incompleteness, questions, foreign-goal rejection and menu skipping. RU/EN can mix: production supplies locale and its own persistence-status copy. The corrected grader treats an unnecessary empty clarify as an accuracy error and nonempty incorrect financial fields as critical; regression checks preserve that distinction.

Self review covers RU/EN, source/actor/revision boundaries, scoring, usage/cache/reservations, preserved history and rejection of invalid run aggregation. Full acceptance, production gateway/command execution/permissions, queues/rollover/outages, ten-page documents, real photos, load, Chrome/Arc and daily user flow remain task-5.* / task-9.1. Model evaluation does not claim these completed.

## OAI-E15: input counting and selected-mode reproduction

[Input Tokens API reference](https://developers.openai.com/api/reference/python/resources/responses/subresources/input_tokens/methods/count): send only supported input/instructions/model/reasoning/text/tools/tool_choice/parallel_tool_calls fields. Do not copy store/background/max_output_tokens/service_tier/cache options: a probe with extra fields returned 400 before generation. Corrected counting established PNG/PDF/function inputs and all selected xhigh inputs. Count is external preprocessing of the same allowed data; its requests are excluded from generation-request totals.

`reservation = ((counted_input + 32) × inputPrice × 1.25 + max_output_tokens × outputPrice) / 1M`. Reconcile usage against the reservation; unexpected usage stops continuation. The byte-ceiling fallback applies only to immutable synthetic fixtures, not arbitrary user PDFs. The owner separately authorized USD 3→7 without resetting any prior entries; CLI cannot raise an existing cap automatically. Paid incomplete output is neither retried nor treated as a proposal; a new independent phase retains its cost.

Reproduce the selected mode with --phase terra_xhigh --model gpt-5.6-terra --reasoning-effort xhigh --max-output-tokens 8192 --count-input-tokens, plus --live, authorized cap, key-file and ignored journal. A changed runner/prompt/case set needs a new explicitly named phase without overwriting earlier evidence. Never rerun merely to replace a failed result. Research schema is not the complete application command schema.

After evaluation, CLI received an xhigh default and requires an explicit live model. SHA-256 of all 27 generated payloads matches the measured configuration; an offline regression protects this identity. Manifest separates the run-source and published-runner fingerprints.
