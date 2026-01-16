import sys
from collections import deque
from datetime import datetime, timezone

LOG_BUFFER = deque(maxlen=500)


async def log(message: str) -> None:
    line = f"{datetime.now(timezone.utc).isoformat()} | {message}"
    print(f"[location-daemon] {message}", file=sys.stderr)
    LOG_BUFFER.append(line)
