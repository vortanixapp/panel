import { expect, test } from "./support/fixtures";
import { SERVER_A } from "./support/data";

const logsPath = `/v1/servers/${SERVER_A}/logs`;
const frame = (data: string) => JSON.stringify({ type: "output", data });

test.describe("вкладка логов", () => {
  test("новые строки приходят потоком консоли без дублей и без опроса", async ({ page, signedIn }) => {
    await page.routeWebSocket(/\/v1\/console/, (ws) => {
      ws.send(frame('[20:57:14 INFO]: Done (5.912s)! For help, type "help"\n'));
      ws.send(frame('[00:08:52 ERROR]: Failed to save player data\n'));
      ws.send(frame('[00:09:30 INFO]: Steve joined the game\n'));
    });

    await page.goto(`/servers/${SERVER_A}/logs`);
    await expect(page.getByText("Steve joined the game")).toHaveCount(1);
    await expect(page.getByText("Failed to save player data")).toHaveCount(1);
    await expect(page.getByText("Done (5.912s)")).toHaveCount(1);

    await page.waitForTimeout(6500);
    expect(signedIn.hitCount(logsPath)).toBe(1);
    expect(signedIn.misses).toEqual([]);
  });

  test("если поток консоли недоступен, логи обновляются опросом", async ({ page, signedIn }) => {
    await page.routeWebSocket(/\/v1\/console/, (ws) => {
      void ws.close({ code: 1011, reason: "unavailable" });
    });

    await page.goto(`/servers/${SERVER_A}/logs`);
    await expect(page.getByText("Failed to save player data")).toBeVisible();
    await expect.poll(() => signedIn.hitCount(logsPath), { timeout: 15_000 }).toBeGreaterThan(1);
    expect(signedIn.misses).toEqual([]);
  });

  test("фильтр по уровню оставляет только ошибки", async ({ page, signedIn }) => {
    await page.routeWebSocket(/\/v1\/console/, (ws) => {
      void ws.close({ code: 1011, reason: "unavailable" });
    });
    await page.goto(`/servers/${SERVER_A}/logs`);
    await expect(page.getByText("Starting minecraft server")).toBeVisible();
    await page.getByRole("button", { name: /Ошибки/ }).click();
    await expect(page.getByText("Failed to save player data")).toBeVisible();
    await expect(page.getByText("Starting minecraft server")).toHaveCount(0);
    expect(signedIn.misses).toEqual([]);
  });
});
