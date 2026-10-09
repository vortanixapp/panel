import { expect, test } from "./support/fixtures";

test.describe("аренда сервера", () => {
  test("заказ нового сервера списывает баланс и открывает страницу сервера", async ({ page, signedIn }) => {
    await page.goto("/rent-server");

    await expect(page.getByRole("heading", { name: "Аренда игрового сервера" })).toBeVisible();
    await page.getByText("Minecraft Java").first().click();
    await page.getByText("Москва").first().click();
    await page.getByText("Оптимальный").first().click();
    await page.getByLabel("Название").fill("Мой новый сервер");

    const pay = page.getByRole("button", { name: "Оплатить и развернуть" });
    await expect(pay).toBeEnabled();
    await pay.click();

    await expect.poll(() => signedIn.called("POST", "/v1/rent-server").length).toBe(1);
    const order = signedIn.called("POST", "/v1/rent-server")[0].body;
    expect(order).toMatchObject({
      node_id: "node-msk",
      game_id: "mc-java",
      tariff_id: "t-opt",
      name: "Мой новый сервер",
      period: 30,
    });
    await expect(page).toHaveURL(/\/servers\/9f1c2d3e-0000-4000-8000-00000000abcd/);
    expect(signedIn.balance).toBe(250);
  });

  test("без выбранного тарифа оплатить нельзя", async ({ page, signedIn }) => {
    await page.goto("/rent-server");
    await page.getByText("Minecraft Java").first().click();
    await expect(page.getByRole("button", { name: /Оплатить|Выберите/ }).last()).toBeDisabled();
    expect(signedIn.called("POST", "/v1/rent-server")).toHaveLength(0);
    expect(signedIn.misses).toEqual([]);
  });
});
