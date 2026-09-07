import { readFile } from "node:fs/promises";

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { App } from "@/app/App";

describe("application foundation", () => {
  it("renders an explicit non-product shell", () => {
    const markup = renderToStaticMarkup(<App />);

    expect(markup.replace(/<[^>]+>/g, "")).toContain("WANT KEEP");
    expect(markup).toContain("Product workflows are intentionally unavailable");
  });

  it("configures shadcn for Base UI", async () => {
    const contents = await readFile(
      new URL("../components.json", import.meta.url),
      "utf8",
    );
    const configuration = JSON.parse(contents) as {
      style?: string;
      rsc?: boolean;
    };

    expect(configuration).toMatchObject({ style: "base-nova", rsc: false });
  });
});
