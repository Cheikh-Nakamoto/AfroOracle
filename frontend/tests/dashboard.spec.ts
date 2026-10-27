import { expect, test } from "@playwright/test";

test("golden dashboard renders the published 187 price and excluded fraud proof", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("/");

  await expect(page.locator(".price-value")).toContainText("187");
  await expect(page.locator(".price-bottom")).toContainText("Consensus validé");
  await expect(page.locator(".price-bottom")).toContainText("100%");

  const fraudRow = page.getByRole("row").filter({ hasText: "Agent D" });
  await expect(fraudRow).toContainText("12");
  await expect(fraudRow).toContainText("Écartée · preuve 12");

  await page.getByRole("button", { name: "Lancer le consensus" }).click();
  await expect(page.getByText("Écart 28,3%", { exact: false })).toBeVisible();
  await expect(page.getByText("Consensus pondéré")).toBeVisible();

  await page.getByRole("textbox", { name: "Question à l’oracle" }).fill("Prix du mil à Kaolack ?");
  await page.getByRole("button", { name: "Envoyer la question" }).click();
  await expect(page.locator(".answer-box")).toContainText("187 FCFA/kg");
});

test("dashboard accepts a photo-backed observation and fits a phone viewport", async ({ page }) => {
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/");

  await page.locator('input[type="file"]').setInputFiles({
    name: "ardoise.png",
    mimeType: "image/png",
    buffer: Buffer.from("mock-photo-proof"),
  });
  await page.getByRole("button", { name: "Envoyer l’observation" }).click();
  await expect(page.getByRole("status")).toContainText("preuve 94/100");

  const pageWidth = await page.evaluate(() => document.documentElement.scrollWidth);
  expect(pageWidth).toBeLessThanOrEqual(360);
});
