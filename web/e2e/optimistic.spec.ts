import { expect, test } from "./support/fixtures";
import { SERVER_A } from "./support/data";

const powerPath = `/v1/servers/${SERVER_A}/power`;

test.describe("мгновенная реакция интерфейса", () => {
  test("карточка сервера сразу показывает остановку, не дожидаясь ответа", async ({ page, signedIn }) => {
    signedIn.delay(powerPath, 3000);
    await page.goto("/dashboard");
    const card = page.locator(`[data-server-id="${SERVER_A}"]`);
    await expect(card.getByText("Работает")).toBeVisible();

    await card.getByRole("button", { name: "Остановить" }).click();
    await expect(card.getByText("Останавливается")).toBeVisible({ timeout: 1500 });
    await expect(card.getByText("Выключен")).toBeVisible({ timeout: 10_000 });
    expect(signedIn.misses).toEqual([]);
  });

  test("при ошибке ноды статус возвращается и показывается причина", async ({ page, signedIn }) => {
    signedIn.powerFails = true;
    await page.goto("/dashboard");
    const card = page.locator(`[data-server-id="${SERVER_A}"]`);
    await card.getByRole("button", { name: "Остановить" }).click();

    await expect(page.getByText("Нода недоступна")).toBeVisible();
    await expect(card.getByText("Работает")).toBeVisible();
    await expect(card.getByText("Останавливается")).toHaveCount(0);
    expect(signedIn.misses).toEqual([]);
  });

  test("страница сервера сразу показывает перезапуск и откатывается при ошибке", async ({ page, signedIn }) => {
    signedIn.powerFails = true;
    signedIn.delay(powerPath, 1200);
    await page.goto(`/servers/${SERVER_A}`);
    await expect(page.getByRole("button", { name: "Выключить" })).toBeVisible();

    await page.getByRole("button", { name: "Перезапустить" }).click();
    await expect(page.getByText("Запускается").first()).toBeVisible({ timeout: 1000 });
    await expect(page.getByText("Нода недоступна")).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole("button", { name: "Выключить" })).toBeVisible();
    expect(signedIn.misses).toEqual([]);
  });

  test("переключатель автоперезапуска срабатывает сразу", async ({ page, signedIn }) => {
    signedIn.delay(`/v1/servers/${SERVER_A}/auto-start`, 2500);
    await page.goto(`/servers/${SERVER_A}`);
    const toggle = page.getByRole("switch", { name: "Автоматический перезапуск" });
    await expect(toggle).toHaveAttribute("aria-checked", "false");

    await toggle.click();
    await expect(toggle).toHaveAttribute("aria-checked", "true", { timeout: 1000 });
    await expect.poll(() => signedIn.called("POST", `/v1/servers/${SERVER_A}/auto-start`).length).toBe(1);
    expect(signedIn.called("POST", `/v1/servers/${SERVER_A}/auto-start`)[0].body).toEqual({ enabled: true });
    await expect(toggle).toHaveAttribute("aria-checked", "true");
    expect(signedIn.misses).toEqual([]);
  });

  test("дашборд при повторном заходе показывается из локального кэша", async ({ page, signedIn }) => {
    await page.goto("/dashboard");
    await expect(page.getByRole("link", { name: "Survival Craft" })).toBeVisible();
    await page.waitForTimeout(2200);

    signedIn.delay("/v1/dashboard", 6000);
    await page.reload();
    await expect(page.getByRole("link", { name: "Survival Craft" })).toBeVisible({ timeout: 3000 });
    expect(signedIn.misses).toEqual([]);
  });

  test("кэш дашборда сбрасывается, когда сессия истекла", async ({ page, signedIn }) => {
    await page.goto("/dashboard");
    await expect(page.getByRole("link", { name: "Survival Craft" })).toBeVisible();
    await expect
      .poll(() => page.evaluate(() => Object.keys(localStorage).filter((k) => k.startsWith("vx-cache:")).length))
      .toBe(1);

    signedIn.refreshFails = true;
    await page.context().clearCookies();
    await signedIn.signIn(30);
    await page.getByRole("link", { name: "Survival Craft" }).click();
    await expect(page).toHaveURL(/\/login/);
    expect(await page.evaluate(() => Object.keys(localStorage).filter((k) => k.startsWith("vx-cache:")))).toHaveLength(0);
  });
});
