# Dahl: MiniMax M2.7 and DeepSeek V4 Flash

[Русский](dahl.md) · [OpenAI selection](openai.en.md) · [Issue #8](https://github.com/pchkauu/want-keep/issues/8)

Sources and runs: 2026-09-07. Only synthetic financial cases were used. The owner's final selection is **Terra Extra High**; Dahl is excluded from production routing, fallback and household-data processing. Other Want Keep features remain unchanged.

## DAHL-E01–E04: access and boundaries

Exact owner-specified IDs `MiniMaxAI/MiniMax-M2.7` and `deepseek-ai/DeepSeek-V4-Flash-0731` were tested at `https://inference.dahl.global/v1/chat/completions`. `/v1/models` returned both IDs; successful completions returned the corresponding model. This establishes route access at inspection time, not immutable identity or guaranteed availability.

DAHL-E01 [API](https://inference.dahl.global/docs/api/), DAHL-E02 [Models](https://inference.dahl.global/docs/models/): Chat Completions, tools and streaming are documented; images are unsupported for these offered models. Photos/PDF were not submitted and extracted text did not substitute for vision/OCR. Without another qualified vision chain these routes cannot replace all mandatory MVP features.

DAHL-E03 [Tokens](https://inference.dahl.global/docs/tokens/), DAHL-E04 [Authentication](https://inference.dahl.global/docs/authentication/): a preallocated pool charges input + output. Documentation quotes approximately USD 0.03/1M, but the key's exact paid tariff was not verified. Billing packages requires an account session and returned 401; bank/browser secrets were not substituted. No purchase, top-up or token reallocation occurred. Public artifacts omit the key, starting balance and private account response.

## DAHL-E05–E06: what high establishes

DAHL-E05 [DeepSeek thinking mode](https://api-docs.deepseek.com/guides/thinking_mode/) describes the direct API: `thinking.type=enabled` and `reasoning_effort=high`, also the default. Both fields were sent through Dahl in high_effort. Dahl returned no applied-effort acknowledgement: HTTP 200 establishes request acceptance, not upstream forwarding. This is not a verified low→high A/B.

DAHL-E06 [MiniMax OpenAI-compatible API](https://platform.minimax.io/docs/api-reference/text-openai-api): M2.x thinking is always enabled and `reasoning_split` changes formatting. M2.7 reasoning_effort control is undocumented. At the owner's request high was submitted and accepted, with no acknowledgement of application. A single run's increased latency/tokens does not establish a causal effect.

## DAHL-E07: measurement

[Machine report](dahl.results.json), [frozen prompt/schema](dahl.contract.json), [runner](../tools/dahl_eval.py), [reporter](../tools/dahl_report.py), [offline tests](../tools/test_dahl_eval.py).

| Model / phase | Text exact | Function | Usage tokens | Reserved tokens |
| --- | --- | --- | --- | --- |
| MiniMaxAI/MiniMax-M2.7 / baseline | 0/0 | 0/0 | 3244 | 0 |
| MiniMaxAI/MiniMax-M2.7 / explicit_schema | 15/20 | 0/0 | 7492 | 0 |
| MiniMaxAI/MiniMax-M2.7 / high_effort | 10/20 | 0/0 | 8421 | 0 |
| deepseek-ai/DeepSeek-V4-Flash-0731 / baseline | 0/0 | 0/0 | 0 | 20259 |
| deepseek-ai/DeepSeek-V4-Flash-0731 / explicit_schema | 0/0 | 0/0 | 0 | 21928 |
| deepseek-ai/DeepSeek-V4-Flash-0731 / streaming_schema | 15/20 | 0/0 | 5589 | 19985 |
| deepseek-ai/DeepSeek-V4-Flash-0731 / high_effort | 9/10 | 0/0 | 2662 | 22208 |

Zero graded cases means no evaluated result, not zero quality or a passing test. The initial screen uses the first 20 cases: one of each archetype, two batches of ten. The remaining 180 variants require a clean screen. Neither model passed; no full 200-case qualification ran. High DeepSeek has only ten graded cases; its second request broke. A separate DeepSeek function probe received 429, leaving tool correctness unverified; no MiniMax function probe ran.

MiniMax default/schema scored 15/20; high requested scored 10/20. High errors include create instead of link for internal movements, lost duplicate fields, financial payload on clarify, and missing report target. DeepSeek default/stream scored 15/20; high requested scored 9/10, retaining financial fields for an unknown account. Other errors and IDs are in JSON. No financial commands ran; valid JSON is not correct accounting.

Dahl default/high after explicit_schema share a frozen prompt/schema from Terra high_receipts. Later final Terra adds self-check/report-target clarifications; different revisions do not constitute a controlled comparison of intrinsic model quality. Gold/Decimal grading is shared and gold is never submitted. MiniMax's high phase also enabled streaming, further limiting latency comparison.

Baseline MiniMax emitted `<think>…</think>` before JSON despite strict response_format. The adapter removes only one complete observed envelope, then strictly parses JSON and grades financial fields. This does not prove server-enforced schema. Streams require finish_reason, usage and `[DONE]`; interrupted streams never become proposals. Raw reasoning is not published. Upstream reasoning-preservation requirements for continuing tool dialogue were not tested: these are single-step proposals without execution.

## DAHL-E08: costs and failures

Observed usage: **27408 tokens**; unresolved/conservative reservations: **84380 tokens**. Each model has a 250000-token cap shared across all phases without resetting earlier calls. 429 reservations remain conservative; HTTP 524 and curl timeout despite HTTP 200 leave charge outcomes unknown. HTTP 200 on an interrupted SSE is not a completed answer. No automatic retries.

The initial Python HTTP client received Cloudflare 403; ordinary curl matching the owner's example worked without protection bypass. Non-stream DeepSeek received 524 after about 120 s; documented SSE produced two answers (27.144/28.186 s), but the later high SSE timed out at 180 s. MiniMax high took 154.070/150.552 s per ten cases. These are call latencies, not UI or production SLAs.

The owner increased the shared test budget to USD 7. OpenAI is accounted separately using exact usage and dated prices; Dahl consumes an existing token pool without a new monetary purchase. No exact combined invoice is claimed: the allocated pool's paid price and final charges for incomplete calls remain unverified. These are limitations of an unselected route, not blockers for choosing Terra.

Dahl's zero-retention statement on its [website](https://inference.dahl.global/) does not establish a complete processing agreement or all upstream operators' terms. Future household-data use requires a separate owner decision, retention/cost/authority/reliability checks and full requalification. This research introduces no production provider contracts.
