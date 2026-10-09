import { expect, test } from "./support/fixtures";

test.describe("вход в панель", () => {
  test("без сессии страница панели ведёт на вход", async ({ page, api }) => {
    await page.goto("/dashboard");
    await expect(page).toHaveURL(/\/login/);
    expect(api.calls).toEqual([]);
  });

  test("верный пароль открывает дашборд", async ({ page, api }) => {
    await page.goto("/login");
    await page.getByLabel("Email").fill(api.user.email);
    await page.locator("#password").fill(api.user.password);
    await page.getByRole("button", { name: "Войти в панель" }).click();

    await expect(page).toHaveURL(/\/dashboard/);
    await expect(page.getByRole("heading", { name: "Мои серверы" })).toBeVisible();

    const login = api.called("POST", "/v1/auth/login");
    expect(login).toHaveLength(1);
    expect(login[0].body.email).toBe(api.user.email);
  });

  test("неверный пароль показывает ошибку и остаётся на входе", async ({ page, api }) => {
    await page.goto("/login");
    await page.getByLabel("Email").fill(api.user.email);
    await page.locator("#password").fill("wrong-password");
    await page.getByRole("button", { name: "Войти в панель" }).click();

    await expect(page.getByText("Неверный email или пароль")).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
  });

  test("пустая форма не отправляет запрос", async ({ page, api }) => {
    await page.goto("/login");
    await page.getByRole("button", { name: "Войти в панель" }).click();
    await expect(page.getByText("Введите корректный email")).toBeVisible();
    await expect(page.getByText("Введите пароль")).toBeVisible();
    expect(api.called("POST", "/v1/auth/login")).toHaveLength(0);
  });
});
