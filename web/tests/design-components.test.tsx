import { readFile, readdir } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { resolve, relative, sep } from "node:path";
import { renderToStaticMarkup } from "react-dom/server";
import ts from "typescript";
import { describe, expect, it } from "vitest";
import { MoneyField } from "@/design-system/components/money-field";
import { CalendarDate } from "@/design-system/components/calendar-date";
import { Button } from "@/design-system/components/button";
import { Sidebar } from "@/design-system/components/sidebar";
import { Avatar } from "@/design-system/components/avatar";
import { Progress } from "@/design-system/components/progress";
import { stateScenarios } from "@/design-system/catalog/state-scenarios";

describe("design-components money input", () => {
  for (const assetLabel of ["RUB", "USD", "USDT", "USDC", "BTC", "ETH"]) {
    for (const value of [
      "0",
      "-",
      "0.",
      "1,25",
      "0.123456789012345678",
      "9".repeat(256),
      "9".repeat(257),
    ]) {
      it(`preserves ${assetLabel} draft with ${value.length} characters: ${value.slice(0, 20)}`, () => {
        const html = renderToStaticMarkup(
          <MoneyField
            aria-label="Amount"
            value={value}
            assetLabel={assetLabel}
            onValueChange={() => {}}
          />,
        );
        expect(html).toContain(`value="${value}"`);
        expect(html).toContain(assetLabel);
        expect(html).toContain('type="text"');
        expect(html).not.toContain("maxLength");
        expect(html).not.toContain('type="number"');
      });
    }
  }
});

describe("design-components calendar dates", () => {
  it("keeps empty, invalid and valid dates distinct", () => {
    expect(CalendarDate.parse("", "ru")).toBeNull();
    for (const text of [
      "31.02.2026",
      "29.02.2025",
      "01.13.2026",
      "00.09.2026",
      "1.9.2026",
      "01.01.0000",
      " 01.01.2026",
      "01.01.2026 ",
    ])
      expect(CalendarDate.parse(text, "ru")).toBeUndefined();
    expect(CalendarDate.parse("29.02.2024", "ru")).toBe("2024-02-29");
    expect(CalendarDate.parse("02/29/2024", "en")).toBe("2024-02-29");
    expect(CalendarDate.parse("02/29/2025", "en")).toBeUndefined();
  });
  it("round trips calendar strings without UTC conversion, including early years", () => {
    for (const value of [
      "0001-01-01",
      "0099-12-31",
      "2024-02-29",
      "2026-03-29",
      "2026-10-25",
      "9999-12-31",
    ]) {
      const date = CalendarDate.toDate(value)!;
      expect(CalendarDate.fromDate(date)).toBe(value);
      for (const locale of ["ru", "en"] as const)
        expect(
          CalendarDate.parse(CalendarDate.display(value, locale), locale),
        ).toBe(value);
    }
    expect(() => CalendarDate.toDate("2026-02-31")).toThrow(
      "invalid_calendar_date",
    );
    expect(() => CalendarDate.toDate("2026-9-7")).toThrow(
      "invalid_calendar_date",
    );
  });
});

describe("design-components public surface", () => {
  it("rejects out-of-contract years and invalid Date values before formatting", () => {
    for (const year of [0, -1, 10000]) {
      const date = new Date(2000, 0, 1);
      date.setFullYear(year);
      expect(() => CalendarDate.fromDate(date)).toThrow(
        "invalid_calendar_date",
      );
      expect(date.getFullYear()).toBe(year);
    }
    expect(() => CalendarDate.fromDate(new Date(NaN))).toThrow(
      "invalid_calendar_date",
    );
  });
  it("gives avatars a nameable role and progress a supplied accessible value", () => {
    const avatar = renderToStaticMarkup(
      <Avatar name="Участник А" initials="A" />,
    );
    expect(avatar).toContain('role="img"');
    expect(avatar).toContain('aria-label="Участник А"');
    const progress = renderToStaticMarkup(
      <Progress
        label="Обработка"
        value={null}
        description="Объём неизвестен"
      />,
    );
    expect(progress).toContain('aria-valuetext="Объём неизвестен"');
    expect(progress).not.toContain("aria-valuenow=");
  });
  it("defaults ordinary actions to non-submit buttons", () => {
    expect(renderToStaticMarkup(<Button>Action</Button>)).toContain(
      'type="button"',
    );
    expect(renderToStaticMarkup(<Button type="submit">Save</Button>)).toContain(
      'type="submit"',
    );
  });
  it("exposes navigation context and complete state fixtures", async () => {
    const html = renderToStaticMarkup(
      <Sidebar
        label="Catalog"
        currentId="forms"
        items={[{ id: "forms", label: "Forms", href: "#forms" }]}
      />,
    );
    expect(html).toContain('aria-current="page"');
    expect(html).toContain('href="#forms"');
    const catalog = JSON.parse(
      await readFile(
        new URL("../../spec/001-want-keep-mvp/catalog.json", import.meta.url),
        "utf8",
      ),
    ) as { ui_states: { id: string }[] };
    expect(stateScenarios.map((state) => state.id)).toEqual(
      catalog.ui_states.map((state) => state.id),
    );
    for (const state of stateScenarios)
      for (const locale of ["ru", "en"] as const) {
        expect(state.title[locale]).not.toBe("");
        expect(state.message[locale]).not.toBe("");
        expect(state.action[locale]).not.toBe("");
      }
  });
  it("keeps UI modules independent of application and transport imports", async () => {
    const web = fileURLToPath(new URL("../", import.meta.url));
    const owner = resolve(web, "src/design-system");
    const sourceRoot = resolve(web, "src") + sep;
    const config = ts.readConfigFile(
      resolve(web, "tsconfig.json"),
      ts.sys.readFile,
    );
    const options = ts.parseJsonConfigFileContent(
      config.config,
      ts.sys,
      web,
    ).options;
    const entries = await readdir(owner, {
      recursive: true,
      withFileTypes: true,
    });
    for (const entry of entries) {
      if (!entry.isFile() || !/\.tsx?$/.test(entry.name)) continue;
      const filename = resolve(entry.parentPath, entry.name);
      const source = ts.createSourceFile(
        filename,
        await readFile(filename, "utf8"),
        ts.ScriptTarget.Latest,
        true,
      );
      for (const statement of source.statements) {
        if (
          !ts.isImportDeclaration(statement) ||
          !ts.isStringLiteral(statement.moduleSpecifier)
        )
          continue;
        const module = ts.resolveModuleName(
          statement.moduleSpecifier.text,
          filename,
          options,
          ts.sys,
        ).resolvedModule;
        if (module?.resolvedFileName.startsWith(sourceRoot)) {
          expect(
            relative(owner, module.resolvedFileName).startsWith(".."),
            `${filename} imports ${module.resolvedFileName}`,
          ).toBe(false);
        }
      }
    }
  });
});
