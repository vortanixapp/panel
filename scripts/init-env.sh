#!/bin/sh
set -eu

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/deploy/.env.example"
DST="$ROOT/deploy/.env"

DOMAIN="${DOMAIN:-}"

usage() {
  cat <<'TXT'
Готовит deploy/.env: генерирует секреты и настраивает адрес панели.

  sh scripts/init-env.sh                    спросит домен
  sh scripts/init-env.sh panel.example.com  сразу с доменом
  sh scripts/init-env.sh --ip               без домена, по IP машины

С доменом панель работает по https, сертификат Let's Encrypt берётся сам.
Без домена — по http. Домен можно указать позже: перезапустите скрипт.
TXT
}

MODE=""
for arg in "$@"; do
  case "$arg" in
    -h|--help) usage; exit 0 ;;
    --ip) MODE="ip" ;;
    -*) echo "неизвестный флаг: $arg" >&2; exit 2 ;;
    *) DOMAIN="$arg"; MODE="domain" ;;
  esac
done

if ! command -v openssl >/dev/null 2>&1; then
  echo "Нужен openssl: apt-get install -y openssl" >&2
  exit 1
fi
if [ ! -f "$SRC" ]; then
  echo "Не найден $SRC — запускайте из корня репозитория" >&2
  exit 1
fi

if [ -z "$MODE" ]; then
  if [ -t 0 ]; then
    printf 'Домен для панели (Enter — работать по IP): '
    read -r DOMAIN || DOMAIN=""
  fi
  [ -n "$DOMAIN" ] && MODE="domain" || MODE="ip"
fi

if [ "$MODE" = "ip" ]; then
  ADDRESS="$(curl -fsS --max-time 8 https://api.ipify.org 2>/dev/null || true)"
  if [ -z "$ADDRESS" ]; then
    ADDRESS="$(hostname -I 2>/dev/null | awk '{print $1}')"
  fi
  if [ -z "$ADDRESS" ]; then
    echo "Не удалось определить адрес машины. Укажите домен или задайте его руками" >&2
    echo "в deploy/.env: SITE_ADDRESS, PANEL_URL, RELAY_PUBLIC_URL." >&2
    exit 1
  fi
  SITE_ADDRESS="http://$ADDRESS"
  BASE="http://$ADDRESS"
  WS="ws://$ADDRESS"
else
  ADDRESS="$DOMAIN"
  SITE_ADDRESS="$DOMAIN"
  BASE="https://$DOMAIN"
  WS="wss://$DOMAIN"
fi

if [ ! -f "$DST" ]; then
  cp "$SRC" "$DST"
  echo "Создан deploy/.env"
else
  echo "deploy/.env уже есть — заполняю только пустые значения"
fi

put() {
  key="$1"
  value="$2"
  force="${3:-}"
  current="$(grep "^$key=" "$DST" 2>/dev/null | head -1 | cut -d= -f2- || true)"
  if [ -n "$current" ] && [ "$force" != "force" ]; then
    echo "  $key — уже задан, не трогаю"
    return
  fi
  if grep -q "^$key=" "$DST"; then
    awk -v k="$key" -v v="$value" '
      index($0, k "=") == 1 { print k "=" v; next }
      { print }
    ' "$DST" > "$DST.tmp" && mv "$DST.tmp" "$DST"
  else
    printf '%s=%s\n' "$key" "$value" >> "$DST"
  fi
  echo "  $key — записан"
}

put POSTGRES_PASSWORD "$(openssl rand -hex 32)"
put JWT_SECRET "$(openssl rand -base64 32 | tr -d '\n')"
put SECRETS_KEY "$(openssl rand -base64 32 | tr -d '\n')"
put INTERNAL_SECRET "$(openssl rand -base64 32 | tr -d '\n')"

put SITE_ADDRESS "$SITE_ADDRESS" force
put RELAY_PUBLIC_URL "$WS" force
put FRONTEND_URL "$BASE" force

chmod 600 "$DST" 2>/dev/null || true

echo
if [ "$MODE" = "domain" ]; then
  echo "Панель будет отвечать на https://$DOMAIN — сертификат Let's Encrypt"
  echo "берётся автоматически при первом запуске."
  echo
  echo "До запуска убедитесь, что A-запись $DOMAIN указывает на эту машину,"
  echo "а порты 80 и 443 открыты: без них выпустить сертификат нечем."
else
  echo "Панель будет отвечать на http://$ADDRESS — без шифрования."
  echo
  echo "Появится домен — запустите скрипт снова с ним, и панель переедет"
  echo "на https вместе с сертификатом:"
  echo "  sh scripts/init-env.sh panel.example.com"
fi
echo
echo "Секреты не печатаются — они в deploy/.env, права 600."
echo "Сохраните SECRETS_KEY отдельно: им зашифрованы ключи платёжных шлюзов"
echo "и пароли нод, без него резервная копия базы наполовину бесполезна."
