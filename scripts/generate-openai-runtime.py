#!/usr/bin/env python3
"""Generate the deployable OpenAI contract from the qualified SDD record."""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "spec/001-want-keep-mvp/evidence/openai.prompts.json"
SELECTED = "terra_xhigh"


def fingerprint(value: object) -> str:
    payload = json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode()
    return hashlib.sha256(payload).hexdigest()


def build() -> dict[str, object]:
    source = json.loads(SOURCE.read_text())
    if source.get("selected_version") != SELECTED:
        raise SystemExit("qualified OpenAI version is not selected")
    selected = source["versions"].get(SELECTED)
    if not isinstance(selected, dict):
        raise SystemExit("selected OpenAI contract is missing")
    if selected.get("reasoning_effort") != "xhigh" or selected.get("max_output_tokens") != 8192:
        raise SystemExit("selected OpenAI qualification changed")
    prompt = selected["prompt"]
    schema = copy.deepcopy(selected["schema"])
    asset_enum = schema["properties"]["results"]["items"]["properties"]["asset"]["enum"]
    asset_enum.insert(asset_enum.index("BTC"), "USDC")
    routes = {
        "transaction_review": {"maximum_input_tokens": 8192, "maximum_output_tokens": 2048},
        "receipt_page": {"maximum_input_tokens": 16384, "maximum_output_tokens": 4096},
        "chat_insight": {"maximum_input_tokens": 16384, "maximum_output_tokens": 4096},
        "complex_clarification": {"maximum_input_tokens": 32768, "maximum_output_tokens": 8192},
    }
    config = {
        "model": "gpt-5.6-terra",
        "qualification": SELECTED,
        "reasoning_effort": "xhigh",
        "reasoning_mode": "standard",
        "service_tier": "default",
        "store": False,
        "background": False,
        "prompt_cache_mode": "explicit",
        "truncation": "disabled",
        "parallel_tool_calls": False,
        "global_maximum_input_tokens": 262144,
        "monthly_limit_usd": "50",
        "maximum_family_concurrency": 2,
        "runtime_input_shape": "qualified_case_list_v1",
        "pricing_per_million_usd": {
            "input": "2",
            "cached_input": "0.2",
            "cache_write": "2.5",
            "output": "12",
        },
        "routes": routes,
    }
    return {
        "kind": "want_keep_openai_runtime_contract_v1",
        "source_version": SELECTED,
        "source_fingerprint": selected["fingerprint"],
        "prompt_fingerprint": hashlib.sha256(prompt.encode()).hexdigest(),
        "schema_fingerprint": fingerprint(schema),
        "config_fingerprint": fingerprint(config),
        "prompt": prompt,
        "schema": schema,
        **config,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(build(), ensure_ascii=False, indent=2) + "\n")


if __name__ == "__main__":
    main()
