import { expect, test } from "./support/fixtures";

test.describe("пополнение баланса", () => {
  test("платёж через тестовый шлюз проходит и зачисляется", async ({ page, signedIn }) => {
    await page.goto("/billing");

    await expect(page.getByText("Способ оплаты")).toBeVisible();
    await page.getByLabel("Своя сумма").fill("700");
    await page.getByRole("button", { name: /Пополнить на 700/ }).click();

    await expect.poll(() => signedIn.called("POST", "/v1/billing/topup/create").length).toBe(1);
    const created = signedIn.called("POST", "/v1/billing/topup/create")[0].body;
    expect(created.amount).toBe(700);
    expect(created.provider).toBe("testpay");
    expect(created.provider_id).toBe("prov-test");

    await expect(page).toHaveURL(/payment=pay-1/);
    await expect(page.getByText("Ожидание оплаты")).toBeVisible();
    await expect(page.getByText("Оплата успешна!")).toBeVisible({ timeout: 20_000 });
    expect(signedIn.balance).toBe(1200);
    expect(signedIn.misses).toEqual([]);
  });

  test("при нулевой сумме кнопка пополнения отключена", async ({ page, signedIn }) => {
    await page.goto("/billing");
    await page.getByLabel("Своя сумма").fill("");
    await expect(page.getByRole("button", { name: /Пополнить на 0/ })).toBeDisabled();
    expect(signedIn.called("POST", "/v1/billing/topup/create")).toHaveLength(0);
    expect(signedIn.misses).toEqual([]);
  });
});
