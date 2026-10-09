import { test as base, expect } from "@playwright/test";
import { MockApi } from "./mock-api";

type Fixtures = {
  api: MockApi;
  signedIn: MockApi;
};

export const test = base.extend<Fixtures>({
  api: async ({ context, page }, use) => {
    const api = new MockApi(context);
    await api.install(page);
    await use(api);
    expect(api.misses, "запросы к API, для которых нет ответа в моке").toEqual([]);
  },
  signedIn: async ({ api }, use) => {
    await api.signIn();
    await use(api);
  },
});

export { expect };
