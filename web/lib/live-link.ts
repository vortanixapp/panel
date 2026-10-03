import { pollMs } from "@/lib/public-settings";

let linked = false;

export function setLiveLinked(up: boolean) {
  linked = up;
}

export function liveLinked() {
  return linked;
}

export function livePollMs(fastMs: number, slowMs = 60_000) {
  return pollMs(linked ? Math.max(fastMs, slowMs) : fastMs);
}
