import { expect, test } from "./support/fixtures";
import { SERVER_A, SERVER_B } from "./support/data";

test.describe("дашборд пользователя", () => {
  test("показывает серверы, расходы и быстрый старт", async ({ page, signedIn }) => {
    await page.goto("/dashboard");

    await expect(page.getByRole("heading", { name: "Мои серверы" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Survival Craft" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Creative Hub" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Игроки на всех серверах" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Расходы за 30 дней" })).toBeVisible();
    await expect(page.getByRole("link", { name: /Добавить сервер/ })).toHaveAttribute("href", "/rent-server");
    expect(signedIn.misses).toEqual([]);
  });

  test("карточка сервера ведёт на его страницу", async ({ page, signedIn }) => {
    await page.goto("/dashboard");
    await page.getByRole("link", { name: "Survival Craft" }).click();
    await expect(page).toHaveURL(new RegExp(`/servers/${SERVER_A}`));
    expect(signedIn.misses).toEqual([]);
  });

  test("остановленный сервер запускается кнопкой на карточке", async ({ page, signedIn }) => {
    await page.goto("/dashboard");
    const card = page.locator(`[data-server-id="${SERVER_B}"]`);
    await card.getByRole("button", { name: "Запустить" }).click();

    await expect.poll(() => signedIn.called("POST", `/v1/servers/${SERVER_B}/power`).length).toBe(1);
    expect(signedIn.called("POST", `/v1/servers/${SERVER_B}/power`)[0].body).toEqual({ action: "start" });
    await expect(card.getByRole("button", { name: "Остановить" })).toBeVisible();
  });

  test("работающий сервер останавливается кнопкой на карточке", async ({ page, signedIn }) => {
    await page.goto("/dashboard");
    const card = page.locator(`[data-server-id="${SERVER_A}"]`);
    await card.getByRole("button", { name: "Остановить" }).click();

    await expect.poll(() => signedIn.called("POST", `/v1/servers/${SERVER_A}/power`).length).toBe(1);
    expect(signedIn.called("POST", `/v1/servers/${SERVER_A}/power`)[0].body).toEqual({ action: "stop" });
    await expect(card.getByRole("button", { name: "Запустить" })).toBeVisible();
  });

  test("нагрузка и игроки обновляются потоком без перезагрузки", async ({ page, signedIn }) => {
    signedIn.stream = [
      'event: hello\ndata: {"unread":0}\n\n',
      `event: server\ndata: ${JSON.stringify({ server_id: SERVER_A, cpu_pct: 73, mem_used_mb: 1024, mem_limit_mb: 2048, ts: Math.floor(Date.now() / 1000) })}\n\n`,
      `event: players\ndata: ${JSON.stringify({ server_id: SERVER_A, online_players: 11, max_players: 20, players_online: [], current_map: "world" })}\n\n`,
    ].join("");

    await page.goto("/dashboard");
    const card = page.locator(`[data-server-id="${SERVER_A}"]`);
    await expect(card.getByText("73%")).toBeVisible();
    await expect(card.getByText("11/20")).toBeVisible();
  });
});
