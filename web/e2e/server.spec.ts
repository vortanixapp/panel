import { expect, test } from "./support/fixtures";
import { SERVER_A } from "./support/data";

const base = `/servers/${SERVER_A}`;

test.describe("страница игрового сервера", () => {
  test("шапка показывает имя, статус и адрес", async ({ page, signedIn }) => {
    await page.goto(base);
    await expect(page.getByRole("heading", { name: "Survival Craft" })).toBeVisible();
    await expect(page.getByText("45.93.200.222:25565")).toBeVisible();
    await expect(page.getByText("Minecraft Java").first()).toBeVisible();
    expect(signedIn.misses).toEqual([]);
  });

  test("главная вкладка показывает нагрузку, игроков и сводку", async ({ page, signedIn }) => {
    await page.goto(base);
    await expect(page.getByText("Steve")).toBeVisible();
    await expect(page.getByText("Alex")).toBeVisible();
    expect(signedIn.misses).toEqual([]);
  });

  const tabs: { name: string; suffix: string; marker: string | RegExp }[] = [
    { name: "Логи", suffix: "/logs", marker: "Failed to save player data" },
    { name: "Метрики", suffix: "/metrics", marker: /CPU/ },
    { name: "SFTP", suffix: "/ftp", marker: "mc_e6fca27e" },
    { name: "Планировщик", suffix: "/cron", marker: "save-all" },
    { name: "Firewall", suffix: "/firewall", marker: "203.0.113.0/24" },
    { name: "Порты", suffix: "/ports", marker: "25565" },
    { name: "Плагины", suffix: "/plugins", marker: "EssentialsX" },
    { name: "Тариф", suffix: "/tariff", marker: "Оптимальный" },
  ];

  for (const tab of tabs) {
    test(`вкладка «${tab.name}» открывается и показывает данные`, async ({ page, signedIn }) => {
      await page.goto(base);
      await page.getByRole("link", { name: tab.name, exact: true }).first().click();
      await expect(page).toHaveURL(new RegExp(`${base}${tab.suffix}$`));
      await expect(page.getByText(tab.marker).first()).toBeVisible();
      expect(signedIn.misses).toEqual([]);
    });
  }

  test("остановка и перезапуск отправляют команды питания", async ({ page, signedIn }) => {
    await page.goto(base);
    await page.getByRole("button", { name: "Перезапустить" }).first().click();
    await expect.poll(() => signedIn.called("POST", `/v1/servers/${SERVER_A}/power`).length).toBe(1);
    expect(signedIn.called("POST", `/v1/servers/${SERVER_A}/power`)[0].body).toEqual({ action: "restart" });

    await page.getByRole("button", { name: "Выключить" }).first().click();
    await expect.poll(() => signedIn.called("POST", `/v1/servers/${SERVER_A}/power`).length).toBe(2);
    expect(signedIn.called("POST", `/v1/servers/${SERVER_A}/power`)[1].body).toEqual({ action: "stop" });
    await expect(page.getByRole("button", { name: "Запустить" }).first()).toBeVisible();
  });

  test("адрес копируется кнопкой в шапке", async ({ page, context, signedIn }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);
    await page.goto(base);
    await page.getByRole("button", { name: /копировать/i }).first().click();
    await expect(page.getByText("Адрес скопирован")).toBeVisible();
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe("45.93.200.222:25565");
    expect(signedIn.misses).toEqual([]);
  });
});
