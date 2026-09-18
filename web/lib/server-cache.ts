type Entry<T> = { at: number; data: T };

export function cachedLoader<T>(ttlMs: number, limit: number) {
  const entries = new Map<string, Entry<T>>();
  const pending = new Map<string, Promise<T>>();

  return function load(
    key: string,
    fetcher: (previous: T | undefined) => Promise<T>,
    fresh = false
  ): Promise<T> {
    const hit = entries.get(key);
    if (!fresh && hit && Date.now() - hit.at < ttlMs) return Promise.resolve(hit.data);
    const running = pending.get(key);
    if (running) return running;
    const next = fetcher(hit?.data)
      .then((data) => {
        if (!entries.has(key) && entries.size >= limit) entries.clear();
        entries.set(key, { at: Date.now(), data });
        return data;
      })
      .finally(() => pending.delete(key));
    pending.set(key, next);
    return next;
  };
}
