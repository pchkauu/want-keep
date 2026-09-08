import { readFile } from "node:fs/promises";
import { expect, test, type Page } from "@playwright/test";
import { authenticator } from "./access/authenticator";

class AccountForm {
  private readonly page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  async create(
    name: string,
    amount: string,
    ownership: "personal" | "household" = "personal",
  ) {
    await this.page.getByLabel("Account name", { exact: true }).fill(name);
    await this.page
      .getByRole("textbox", { name: "Tracking start date", exact: true })
      .fill("09/01/2026");
    await this.page.getByLabel("Opening balance", { exact: true }).fill(amount);
    if (ownership === "household") {
      await this.page.getByRole("combobox", { name: "Ownership" }).click();
      await this.page
        .getByRole("option", { name: "Household account", exact: true })
        .click();
    }
    await this.page
      .getByRole("button", { name: "Add account", exact: true })
      .click();
    await expect(
      this.page.getByRole("heading", { name: `Account added: ${name}` }),
    ).toBeVisible();
  }

  async next() {
    await this.page
      .getByRole("button", { name: "Add another account", exact: true })
      .click();
  }
}

test("household view keeps actor, ownership and invitation state separate", async ({
  browser,
}) => {
  test.setTimeout(180_000);
  const origin = process.env.WANT_KEEP_ACCESS_ORIGIN!;
  const bootstrapPath = process.env.WANT_KEEP_ACCESS_BOOTSTRAP_FILE;
  if (!bootstrapPath || process.env.WANT_KEEP_ACCESS_E2E !== "1")
    throw new Error(
      "Run make e2e SCENARIO=family-access with isolated PostgreSQL and API",
    );

  const first = await browser.newContext({
    baseURL: origin,
    locale: "en-US",
    viewport: { width: 1440, height: 900 },
  });
  const second = await browser.newContext({
    baseURL: origin,
    locale: "en-US",
    viewport: { width: 1280, height: 720 },
  });
  const page = await first.newPage();
  const partner = await second.newPage();
  await authenticator(first, page);
  await authenticator(second, partner);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  partner.on("pageerror", (error) => errors.push(error.message));

  try {
    await page.goto("/setup");
    await page
      .getByLabel("Operator token")
      .fill((await readFile(bootstrapPath, "utf8")).trim());
    await page.getByLabel("Your name", { exact: true }).fill("Andrey Test");
    await page
      .getByLabel("Household name", { exact: true })
      .fill("Want Keep family");
    await page
      .getByRole("button", {
        name: "Create household and passkey",
        exact: true,
      })
      .click();
    await page
      .getByRole("button", { name: "Codes saved — continue", exact: true })
      .click();

    await page.goto("/onboarding?view=household");
    const firstAccounts = new AccountForm(page);
    await firstAccounts.create("Andrey cash", "1000");
    await firstAccounts.next();
    await firstAccounts.create("Family cash", "500", "household");

    await page.goto("/settings/household?view=household");
    await expect(
      page.getByRole("heading", {
        level: 1,
        name: "Household and shared access",
      }),
    ).toBeVisible();
    await expect(
      page.getByText("Andrey Test", { exact: true }).first(),
    ).toBeVisible();

    let lostIssue = 0;
    await page.route("**/api/v1/household/invitations", async (route) => {
      if (route.request().method() !== "POST" || lostIssue > 0) {
        await route.continue();
        return;
      }
      lostIssue++;
      const response = await route.fetch();
      expect(response.ok()).toBe(true);
      await route.abort("failed");
    });
    await page
      .getByRole("button", { name: "Create invitation", exact: true })
      .click();
    await expect(
      page.getByText("The change outcome is unknown."),
    ).toBeVisible();
    await page.unroute("**/api/v1/household/invitations");
    await page
      .getByRole("button", { name: "Read current state", exact: true })
      .click();
    await expect(
      page.getByRole("button", { name: "Issue a new invitation", exact: true }),
    ).toBeVisible();

    await page
      .getByRole("button", { name: "Issue a new invitation", exact: true })
      .click();
    await expect(page.getByLabel("Private invitation link")).toBeVisible();

    await page.route("**/api/v1/household/invitations", async (route) => {
      if (route.request().method() !== "POST") {
        await route.continue();
        return;
      }
      const body = route.request().postDataJSON() as {
        expectedRevision: number;
      };
      const response = await route.fetch({
        postData: JSON.stringify({
          expectedRevision: body.expectedRevision - 1,
        }),
      });
      await route.fulfill({ response });
    });
    await page
      .getByRole("button", { name: "Issue a new invitation", exact: true })
      .click();
    await expect(
      page.getByText("The invitation already changed.", { exact: false }),
    ).toBeVisible();
    await page.unroute("**/api/v1/household/invitations");
    await page
      .getByRole("button", { name: "Read current state", exact: true })
      .click();
    await page
      .getByRole("button", { name: "Issue a new invitation", exact: true })
      .click();
    const invitation = await page
      .getByLabel("Private invitation link", { exact: true })
      .inputValue();

    await partner.goto(invitation);
    await partner
      .getByRole("button", { name: "Check invitation", exact: true })
      .click();
    await partner.getByLabel("Your name", { exact: true }).fill("Partner Test");
    await partner
      .getByRole("button", {
        name: "Join with your own passkey",
        exact: true,
      })
      .click();
    await partner
      .getByRole("button", { name: "Codes saved — continue", exact: true })
      .click();
    await partner.goto("/settings/household?view=household");
    await expect(
      partner.getByText("Andrey Test", { exact: true }).first(),
    ).toBeVisible();
    await expect(
      partner.getByText("Partner Test", { exact: true }).first(),
    ).toBeVisible();
    await page.reload();
    await expect(
      page.getByText("Partner Test", { exact: true }).first(),
    ).toBeVisible();

    const firstMe = await (await page.request.get("/api/v1/me")).json();
    const partnerMe = await (await partner.request.get("/api/v1/me")).json();
    const firstId = firstMe.user.id as string;
    const partnerId = partnerMe.user.id as string;

    await partner
      .getByRole("button", { name: "Andrey Test", exact: true })
      .click();
    await expect(partner).toHaveURL(
      new RegExp(`view=member&member=${firstId}`),
    );
    await partner.getByRole("link", { name: "Want Keep", exact: true }).click();
    await expect(partner).toHaveURL(
      new RegExp(`/overview\\?view=member&member=${firstId}`),
    );
    await partner
      .getByRole("link", { name: "Set up tracking" })
      .first()
      .click();
    await expect(partner).toHaveURL(
      new RegExp(`/onboarding\\?view=member&member=${firstId}`),
    );

    let personalOwner = "";
    await partner.route("**/api/v1/accounts", async (route) => {
      if (route.request().method() === "POST") {
        const body = route.request().postDataJSON() as {
          ownership: { personalOwnerId?: string };
        };
        personalOwner = body.ownership.personalOwnerId ?? "";
      }
      await route.continue();
    });
    const partnerAccounts = new AccountForm(partner);
    await partnerAccounts.create("Partner cash", "700");
    expect(personalOwner).toBe(partnerId);
    expect(personalOwner).not.toBe(firstId);
    await partner.unroute("**/api/v1/accounts");
    await partnerAccounts.next();
    await expect(
      partner.getByText("Partner cash", { exact: true }),
    ).toHaveCount(0);
    await partner
      .getByRole("button", { name: "Household", exact: true })
      .click();
    await expect(
      partner.getByText("Partner cash", { exact: true }),
    ).toBeVisible();
    await expect(
      partner.getByText("Andrey cash", { exact: true }),
    ).toBeVisible();
    await expect(
      partner.getByText("Family cash", { exact: true }),
    ).toBeVisible();

    await partner.goto(`/overview?view=member&member=${firstId}&period=month`);
    await partner.getByRole("link", { name: "Money", exact: true }).click();
    await expect(partner).toHaveURL(
      new RegExp(`/accounts\\?view=member&member=${firstId}$`),
    );
    await partner.goBack();
    await expect(partner).toHaveURL(
      new RegExp(
        `view=member&member=${firstId}&period=month|period=month&view=member&member=${firstId}`,
      ),
    );
    await partner.goto("/overview?view=member&member=unknown");
    await expect(partner).toHaveURL(/view=household$/);

    const partnerToggle = partner.getByRole("button", {
      name: "Partner Test",
      exact: true,
    });
    await partnerToggle.focus();
    await partner.keyboard.press("Space");
    await expect(partner).toHaveURL(
      new RegExp(`view=member&member=${partnerId}`),
    );
    await expect(partner.locator(".current-member span")).toHaveText(
      "Partner Test",
    );

    await partner.getByRole("button", { name: "RU", exact: true }).click();
    await expect(
      partner.getByText("Показывать", { exact: true }),
    ).toBeVisible();
    await partner.getByRole("button", { name: "EN", exact: true }).click();
    await partner.emulateMedia({ reducedMotion: "reduce" });
    expect(
      (
        await partnerToggle.evaluate((element) =>
          getComputedStyle(element).getPropertyValue("transition-duration"),
        )
      )
        .split(",")
        .every((duration) => duration.trim() === "0s"),
    ).toBe(true);

    for (const viewport of [
      { width: 1280, height: 720 },
      { width: 1440, height: 900 },
      { width: 640, height: 360 },
      { width: 720, height: 450 },
    ]) {
      await partner.setViewportSize(viewport);
      await partner.goto("/settings/household?view=household");
      await expect(
        partner.getByRole("heading", {
          level: 1,
          name: "Household and shared access",
        }),
      ).toBeVisible();
      expect(
        await partner.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      ).toBe(true);
      await partner
        .getByRole("link", { name: "Open your security settings" })
        .scrollIntoViewIfNeeded();
      await expect(
        partner.getByRole("link", { name: "Open your security settings" }),
      ).toBeInViewport();
    }

    await partner
      .getByRole("button", { name: "Sign out", exact: true })
      .click();
    await expect(partner).toHaveURL(/\/login$/);
    await partner
      .getByRole("button", { name: "Log in with Passkeys", exact: true })
      .click();
    await expect(partner).toHaveURL(/\/overview\?view=household$/);
    expect((await page.request.get("/api/v1/me")).status()).toBe(200);
    expect(errors).toEqual([]);
  } finally {
    await first.close();
    await second.close();
  }
});
