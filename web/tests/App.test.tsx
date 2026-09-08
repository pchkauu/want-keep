import { readFile } from "node:fs/promises";

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { MemoryRouter } from "react-router";
import { AuthLayout } from "@/features/identity/AuthLayout";
import { LocaleProvider } from "@/locales/LocaleProvider";
import { LocaleController } from "@/locales/locale";

describe("application foundation", () => {
  it("renders the centered sign-in composition in the selected language", () => {
    const markup = renderToStaticMarkup(
      <MemoryRouter>
        <LocaleProvider controller={new LocaleController(undefined, "en")}>
          <AuthLayout login>
            <button>Log in with Passkeys</button>
          </AuthLayout>
        </LocaleProvider>
      </MemoryRouter>,
    );

    expect(markup.replace(/<[^>]+>/g, "")).toContain("Want Keep");
    expect(markup).toContain("Log in with Passkeys");
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
