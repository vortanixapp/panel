#!/bin/sh
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGES_DIR="$ROOT/deploy/images"
REGISTRY="${VORTANIX_REGISTRY:-}"
PUSH=0
ALL=0
RUNTIMES=0
GAMES=""

usage() {
  cat <<'TXT'
Собирает образы игровых серверов из deploy/images.

  sh scripts/build-game-images.sh --runtimes         четыре общих рантайма
  sh scripts/build-game-images.sh --all              все игры
  sh scripts/build-game-images.sh minecraft rust     только указанные
  sh scripts/build-game-images.sh --runtimes --push  собрать и отправить в реестр

Реестр берётся из VORTANIX_REGISTRY. Без него образы остаются локальными,
и ноды их не найдут.
TXT
}

for arg in "$@"; do
  case "$arg" in
    --all) ALL=1 ;;
    --runtimes) RUNTIMES=1 ;;
    --push) PUSH=1 ;;
    -h|--help) usage; exit 0 ;;
    -*)
      echo "неизвестный флаг: $arg" >&2
      exit 2
      ;;
    *) GAMES="$GAMES $arg" ;;
  esac
done

if [ "$RUNTIMES" -eq 1 ]; then
  failed=""
  built=0
  for rt in steam srcds java native; do
    ctx="$IMAGES_DIR/_runtime/$rt"
    image="vortanix/runtime-$rt:latest"
    [ -n "$REGISTRY" ] && image="${REGISTRY%/}/vortanix/runtime-$rt:latest"
    echo "=== рантайм $rt -> $image ==="
    if docker build -t "$image" "$ctx"; then
      built=$((built + 1))
      if [ "$PUSH" -eq 1 ]; then
        docker push "$image" || failed="$failed $rt(push)"
      fi
    else
      failed="$failed $rt"
    fi
  done
  echo
  echo "собрано рантаймов: $built"
  if [ -n "$failed" ]; then
    echo "с ошибками:$failed" >&2
    exit 1
  fi
  exit 0
fi

if [ "$ALL" -eq 1 ]; then
  GAMES="$(ls "$IMAGES_DIR" | grep -v '^README.md$' | grep -v '^_runtime$')"
fi

GAMES="$(printf '%s' "$GAMES" | tr -s ' \n' '\n' | sed '/^$/d')"
if [ -z "$GAMES" ]; then
  echo "укажите --runtimes, игры или --all; доступные игры:" >&2
  ls "$IMAGES_DIR" | grep -v '^README.md$' | grep -v '^_runtime$' | tr '\n' ' ' >&2
  echo >&2
  exit 2
fi

if ! docker info >/dev/null 2>&1; then
  echo "error: docker daemon недоступен" >&2
  exit 1
fi

failed=""
built=0
for game in $GAMES; do
  ctx="$IMAGES_DIR/$game"
  if [ ! -f "$ctx/Dockerfile" ]; then
    echo "!! $game: нет $ctx/Dockerfile — пропуск" >&2
    failed="$failed $game"
    continue
  fi

  image="vortanix/$game:latest"
  [ -n "$REGISTRY" ] && image="${REGISTRY%/}/vortanix/$game:latest"

  echo "=== $game -> $image ==="
  if docker build -t "$image" "$ctx"; then
    built=$((built + 1))
    if [ "$PUSH" -eq 1 ]; then
      docker push "$image" || failed="$failed $game(push)"
    fi
  else
    failed="$failed $game"
  fi
done

echo
echo "собрано: $built"
if [ -n "$failed" ]; then
  echo "с ошибками:$failed" >&2
  exit 1
fi
