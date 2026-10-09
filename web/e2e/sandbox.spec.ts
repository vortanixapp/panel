import type { Page } from "@playwright/test";
import { expect, test } from "./support/fixtures";
import type { MockApi } from "./support/mock-api";

type Node = {
  id: string;
  name: string;
  sandbox?: { runtime: string; available: boolean; runtimes: string[]; mode: string; installer: boolean };
};

function agentRow(node: Node) {
  return {
    id: node.id,
    name: node.name,
    code: node.id,
    country: "RU",
    region: "eu",
    host: `${node.id}.example.test`,
    state: "online",
    is_online: true,
    labels: [],
    version: "0.1.100",
    outdated: false,
    proto: 2,
    platform: "linux",
    auto_update: true,
    ssh: false,
    maintenance: false,
    resources: {},
    servers: { total: 1, running: 1 },
    ...(node.sandbox ? { sandbox: node.sandbox } : {}),
  };
}

function sandboxItems(values: Record<string, string>) {
  const base = { group: "nodes", section: "sandbox", min: 0, max: 0, unit: "", custom: false };
  return [
    { ...base, key: "agent.sandbox_mode", kind: "enum", default: "off", options: ["off", "selected", "all"], value: values["agent.sandbox_mode"] ?? "off" },
    { ...base, key: "agent.sandbox_games", kind: "string", default: "", max: 400, value: values["agent.sandbox_games"] ?? "" },
    { ...base, key: "agent.sandbox_runtime", kind: "string", default: "runsc", max: 40, value: values["agent.sandbox_runtime"] ?? "runsc" },
    { ...base, key: "agent.sandbox_strict", kind: "bool", default: "0", value: values["agent.sandbox_strict"] ?? "0" },
  ];
}

function setup(api: MockApi, options: { withOldNode?: boolean } = {}) {
  api.user.role = "admin";
  const nodes: Node[] = [
    { id: "n-berlin", name: "Берлин", sandbox: { runtime: "runsc", available: true, runtimes: ["runc", "runsc"], mode: "all", installer: true } },
    { id: "n-moscow", name: "Москва", sandbox: { runtime: "runsc", available: false, runtimes: ["runc"], mode: "off", installer: true } },
    ...(options.withOldNode === false ? [] : [{ id: "n-old", name: "Старый узел" }]),
  ];
  const saved: Record<string, string> = {};
  const state = { saves: [] as Record<string, string>[] };

  api.use(({ method, path, body }) => {
    if (path === "/v1/admin/site-menu") return { json: { menu: null } };
    if (path === "/v1/admin/jobs/feed") {
      return { json: { tasks: [], counts: { active: 0, queued: 0, running: 0, failed_24h: 0 } } };
    }
    if (path === "/v1/admin/settings" && method === "GET") return { json: { values: {}, panel_version: "0.1.100" } };
    if (path === "/v1/admin/daemons" && method === "GET") {
      return {
        json: {
          summary: { total: 3, online: 3 },
          target_version: "0.1.100",
          auto_enabled: true,
          can_write: true,
          agents: nodes.map(agentRow),
        },
      };
    }
    if (path === "/v1/admin/settings/registry" && method === "GET") return { json: { items: sandboxItems(saved) } };
    if (path === "/v1/admin/settings/registry" && method === "PUT") {
      const values = (body as { values: Record<string, string> }).values;
      state.saves.push(values);
      Object.assign(saved, values);
      return { json: { items: sandboxItems(saved) } };
    }
    const sandbox = /^\/v1\/admin\/daemons\/([a-z-]+)\/sandbox$/.exec(path);
    if (sandbox && method === "GET") {
      return { json: { running: null, last: null, supported: nodes.find((n) => n.id === sandbox[1])?.sandbox?.installer === true, online: true } };
    }
    const install = /^\/v1\/admin\/daemons\/([a-z-]+)\/sandbox\/install$/.exec(path);
    if (install && method === "POST") {
      const node = nodes.find((n) => n.id === install[1]);
      if (node?.sandbox) node.sandbox = { ...node.sandbox, available: true, runtimes: ["runc", "runsc"] };
      return { json: { task_id: "task-1" } };
    }
    if (/^\/v1\/admin\/daemons\/[a-z-]+\/tasks\/task-1$/.test(path)) {
      return { json: { task: { id: "task-1", action: "sandbox_install", method: "relay", status: "done", created_at: new Date().toISOString() } } };
    }
    return undefined;
  });
  return state;
}

async function open(page: Page) {
  await page.goto("/admin/settings?tab=nodes");
  await expect(page.getByTestId("sandbox-nodes")).toBeVisible();
}

async function chooseMode(page: Page, mode: "off" | "selected" | "all") {
  await page.locator("select").filter({ has: page.locator('option[value="selected"]') }).selectOption(mode);
}

const strictSwitch = (page: Page) =>
  page
    .getByText("Не запускать без песочницы")
    .locator("xpath=ancestor::*[.//*[@role='switch']][1]")
    .getByRole("switch");

const row = (page: Page, id: string) => page.locator(`[data-node-id="${id}"]`);

test.describe("песочница gVisor в настройках узлов", () => {
  test("у каждого узла показано, есть ли на нём gVisor", async ({ page, signedIn }) => {
    setup(signedIn);
    await open(page);
    await expect(row(page, "n-berlin").getByTestId("sandbox-badge")).toHaveAttribute("data-state", "present");
    await expect(row(page, "n-moscow").getByTestId("sandbox-badge")).toHaveAttribute("data-state", "missing");
    await expect(row(page, "n-old").getByTestId("sandbox-badge")).toHaveAttribute("data-state", "unknown");
    await expect(row(page, "n-berlin").getByRole("button", { name: "Установить gVisor" })).toHaveCount(0);
    await expect(row(page, "n-moscow").getByRole("button", { name: "Установить gVisor" })).toBeEnabled();
    await expect(row(page, "n-old").getByRole("button", { name: "Установить gVisor" })).toBeDisabled();
    expect(signedIn.misses).toEqual([]);
  });

  test("пока песочница выключена, предупреждений нет", async ({ page, signedIn }) => {
    setup(signedIn);
    await open(page);
    await expect(page.getByText("запустятся без песочницы")).toHaveCount(0);
    await expect(page.getByText("не запустятся на узлах без gVisor")).toHaveCount(0);
  });

  test("при включении песочницы панель называет узлы без gVisor", async ({ page, signedIn }) => {
    setup(signedIn);
    await open(page);
    await chooseMode(page, "all");
    await expect(page.getByText(/запустятся без песочницы: Москва/)).toBeVisible();
    await expect(page.getByText(/не применят настройки песочницы: Старый узел/)).toBeVisible();

    await strictSwitch(page).click();
    await expect(
      page.getByText("Строгий режим включён: серверы не запустятся на узлах без gVisor (Москва).")
    ).toBeVisible();
    await expect(page.getByText(/запустятся без песочницы: Москва/)).toHaveCount(0);
  });

  test("gVisor ставится на узел кнопкой из панели", async ({ page, signedIn }) => {
    setup(signedIn);
    await open(page);
    await row(page, "n-moscow").getByRole("button", { name: "Установить gVisor" }).click();

    await expect.poll(() => signedIn.called("POST", "/v1/admin/daemons/n-moscow/sandbox/install").length).toBe(1);
    await expect(row(page, "n-moscow").getByTestId("sandbox-badge")).toHaveAttribute("data-state", "present", { timeout: 15_000 });
    await expect(row(page, "n-moscow").getByRole("button", { name: "Установить gVisor" })).toHaveCount(0);
    expect(signedIn.called("POST", "/v1/admin/daemons/n-berlin/sandbox/install")).toHaveLength(0);
    expect(signedIn.misses).toEqual([]);
  });

  test("сохранение с узлами без gVisor требует подтверждения", async ({ page, signedIn }) => {
    const state = setup(signedIn);
    await open(page);
    await chooseMode(page, "all");
    await page.getByRole("button", { name: "Сохранить" }).click();

    await expect(page.getByText("Не на всех узлах есть gVisor")).toBeVisible();
    await page.getByRole("button", { name: "Отмена" }).click();
    expect(state.saves).toHaveLength(0);

    await page.getByRole("button", { name: "Сохранить" }).click();
    await page.getByRole("button", { name: "Всё равно сохранить" }).click();
    await expect.poll(() => state.saves.length).toBe(1);
    expect(state.saves[0]).toMatchObject({ "agent.sandbox_mode": "all" });
    expect(signedIn.misses).toEqual([]);
  });

  test("если gVisor есть везде, сохранение проходит без вопросов", async ({ page, signedIn }) => {
    const state = setup(signedIn, { withOldNode: false });
    await open(page);
    await row(page, "n-moscow").getByRole("button", { name: "Установить gVisor" }).click();
    await expect(row(page, "n-moscow").getByTestId("sandbox-badge")).toHaveAttribute("data-state", "present", { timeout: 15_000 });

    await chooseMode(page, "selected");
    await expect(page.getByText(/запустятся без песочницы/)).toHaveCount(0);
    await page.getByRole("button", { name: "Сохранить" }).click();
    await expect.poll(() => state.saves.length).toBe(1);
    await expect(page.getByText("Не на всех узлах есть gVisor")).toHaveCount(0);
  });
});
