import { expect, test } from "./support/fixtures";

const CURSOR = "2026-10-09T10:00:00.500000000Z_8b0f7a2c-0000-4000-8000-000000000001";

function sse(parts: { id?: string; event: string; data: unknown }[]) {
  return parts
    .map((p) => `${p.id ? `id: ${p.id}\n` : ""}event: ${p.event}\ndata: ${JSON.stringify(p.data)}\n\n`)
    .join("");
}

const notification = {
  id: "8b0f7a2c-0000-4000-8000-000000000001",
  type: "server.crashed",
  title: "Сервер упал",
  body: "Survival Craft перестал отвечать",
  group: "servers",
  category: "Серверы",
  icon: "alert",
  severity: "critical",
  tone: "bad",
  quiet: false,
  action: "",
  href: "",
  unread: true,
  read_at: "",
  created_at: "2026-10-09T10:00:00Z",
};

test.describe("живой поток событий", () => {
  test("при переподключении клиент отправляет курсор последних событий", async ({ page, signedIn }) => {
    signedIn.streams = [
      sse([
        { event: "hello", data: { unread: 0, resumed: false, at: 5000 } },
        { id: "6000", event: "invalidate", data: { topic: "server:other" } },
        { id: CURSOR, event: "notification", data: { item: notification, unread: 1 } },
      ]),
      sse([{ event: "hello", data: { unread: 1, resumed: true, at: 7000 } }]),
    ];

    await page.goto("/dashboard");
    await expect.poll(() => signedIn.streamRequests.length, { timeout: 15_000 }).toBeGreaterThanOrEqual(2);
    expect(signedIn.streamRequests[0]).toBe("");
    expect(signedIn.streamRequests[1]).toBe(`6000|${CURSOR}`);
    expect(signedIn.misses).toEqual([]);
  });

  test("после возобновления потока списки серверов не перезагружаются", async ({ page, signedIn }) => {
    signedIn.streams = [
      sse([{ event: "hello", data: { unread: 0, resumed: false, at: 5000 } }]),
      sse([{ event: "hello", data: { unread: 0, resumed: true, at: 6000 } }]),
    ];

    await page.goto("/servers");
    await expect(page.getByText("Survival Craft").first()).toBeVisible();
    await expect.poll(() => signedIn.streamRequests.length, { timeout: 15_000 }).toBeGreaterThanOrEqual(2);
    await page.waitForTimeout(800);
    expect(signedIn.hitCount("/v1/my-servers")).toBe(1);
    expect(signedIn.misses).toEqual([]);
  });

  test("без возобновления список серверов перезагружается", async ({ page, signedIn }) => {
    signedIn.streams = [
      sse([{ event: "hello", data: { unread: 0, resumed: false, at: 5000 } }]),
      sse([{ event: "hello", data: { unread: 0, resumed: false, at: 6000 } }]),
    ];

    await page.goto("/servers");
    await expect(page.getByText("Survival Craft").first()).toBeVisible();
    await expect.poll(() => signedIn.hitCount("/v1/my-servers"), { timeout: 15_000 }).toBeGreaterThanOrEqual(2);
    expect(signedIn.misses).toEqual([]);
  });

  test("уведомление, пришедшее во время разрыва, показывается после возобновления", async ({ page, signedIn }) => {
    signedIn.streams = [
      sse([{ event: "hello", data: { unread: 0, resumed: false, at: 5000 } }]),
      sse([
        { event: "hello", data: { unread: 1, resumed: true, at: 6000 } },
        { id: CURSOR, event: "notification", data: { item: notification, unread: 1 } },
      ]),
    ];

    await page.goto("/dashboard");
    await expect(page.getByText("Сервер упал")).toBeVisible({ timeout: 15_000 });
    expect(signedIn.misses).toEqual([]);
  });
});
