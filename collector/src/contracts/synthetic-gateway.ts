import {
  parseCapabilityManifest,
  requireCapabilities,
} from "./capabilities.js";
import type { CapabilityManifest, SyncResult } from "./types.js";
import { fail } from "./validation.js";
import { parseSyncRequest, parseSyncResult } from "./wire.js";

export class SyntheticGateway {
  readonly #manifest: CapabilityManifest;
  readonly #results: readonly unknown[];
  #index = 0;

  constructor(manifest: unknown, results: readonly unknown[]) {
    this.#manifest = parseCapabilityManifest(snapshot(manifest));
    this.#results = snapshot(Array.from(results));
  }

  manifest(): CapabilityManifest {
    return structuredClone(this.#manifest);
  }

  read(request: unknown): SyncResult {
    const expected = parseSyncRequest(request);
    if (this.#manifest.provider !== expected.binding.provider) fail();
    if (this.#index >= this.#results.length) fail();
    const result = parseSyncResult(this.#results[this.#index], expected);
    requireCapabilities(this.#manifest, result, expected);
    this.#index += 1;
    return structuredClone(result);
  }
}

function snapshot<T>(value: T): T {
  try {
    return structuredClone(value);
  } catch {
    fail();
  }
}
