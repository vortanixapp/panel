# Игровые образы Vortanix

## Четыре рантайма вместо сорока семи образов

Игры запускаются в общих образах-рантаймах — `deploy/images/_runtime/*`:

| Рантайм | Кому | Что внутри |
|---------|------|------------|
| `runtime-steam` | 38 игр: ARK, Rust, Valheim, Palworld, CS2, … | ubuntu 22.04, 32/64-битные библиотеки, steamclient |
| `runtime-srcds` | css, tf2, gmod, cs16 | ubuntu 20.04, 32-битный Source/GoldSrc |
| `runtime-java` | mcjava, mcpaper, mcspigot, mcforge, mcfabric | JRE и mcrcon |
| `runtime-native` | samp, crmp, mta, mcbedrock | нативные бинари без Steam и JVM |

Различия между играми — данные, а не образы. Ключ игры, название, пути к
серверному бинарю и параметры запуска лежат в каталоге
[`pkg/gamecatalog`](../../pkg/gamecatalog) и
приезжают в контейнер переменными `VTX_*`, которые ставит агент.

Так вышло не случайно: из сорока семи прежних entrypoint-скриптов **тридцать
два были байт-в-байт одинаковыми**, кроме трёх строк — ключа игры, названия и
списка путей к бинарю. Отдельный образ на игру означал, что ради новой игры
надо собрать и раздать образ на каждую ноду, а забытая сборка давала
`pull access denied` уже во время оплаченной аренды.

### Сборка

```bash
sh scripts/build-game-images.sh --runtimes
VORTANIX_REGISTRY=ghcr.io/vortanixapp sh scripts/build-game-images.sh --runtimes --push
```

Четыре образа собираются один раз. Новая игра после этого добавляется
записью в каталоге — пересобирать ничего не нужно.

### Переход со старых образов

Агент предпочитает рантайм, но если тот ещё не собран, а собственный образ
игры на ноде есть — берётся он. Поэтому обновление ничего не ломает: нода
со старыми образами продолжает работать, пока рантаймы не собраны.

Переопределение образов — теми же переменными, что и раньше:
`VORTANIX_RUNTIME_STEAM_IMAGE`, `VORTANIX_RUNTIME_SRCDS_IMAGE`,
`VORTANIX_RUNTIME_JAVA_IMAGE`, `VORTANIX_RUNTIME_NATIVE_IMAGE`, плюс общий
префикс реестра `VORTANIX_REGISTRY`.

## Прежние per-game образы

Каталоги `deploy/images/<key>` оставлены для совместимости: на нодах, где
они уже собраны, серверы продолжают работать. Новые игры туда добавлять не
нужно — достаточно записи в каталоге.

```bash
sh scripts/build-game-images.sh mcjava cs2   # выборочно
sh scripts/build-game-images.sh --all        # все старые образы
```

## Что образ ждёт от рантайма

- `/data` — том с файлами сервера, монтируется агентом;
- переменная порта из каталога (`GAME_PORT`, `CS16_PORT`, …);
- `/data/.vtx/startup_params` — строка параметров запуска, если задана;
- `VTX_GAME_KEY`, `VTX_GAME_NAME`, `VTX_SERVER_BINARIES`, `VTX_PORT_ENV`,
  `VTX_PARAM_*` — ставит агент из каталога.

Файлы самого сервера образ **не скачивает** — их кладёт агент в том: из
архива версии, через SteamCMD или сборкой BuildTools. Исключение — Bedrock,
см. ниже.

## Minecraft

Версия игры в админке задаёт, откуда агент возьмёт файлы: ссылка на архив,
свой архив, загруженный в панель, Steam или сборка BuildTools. Архив zip и
tar.gz распаковывается в папку сервера, jar ложится как `server.jar`.

### Java: Vanilla, Paper, Spigot, Forge, Fabric

Образ `mcjava` содержит Java 8, 17, 21 и 25 и сам выбирает нужную по
`server.jar`:

- `version.json` внутри jar — Vanilla 1.18+ и Paper;
- `install.properties` — лаунчер Fabric, по версии Minecraft;
- версия class-файла главного класса — старые Vanilla, Paper, Spigot, Forge.

Выбор виден в логе: `Java 17 (version.json)`. Переопределить можно файлом
`/data/.vtx/java` с номером версии или переменной `JAVA_VERSION`.

Forge и NeoForge ставятся ссылкой на installer: образ при первом старте
запускает `--installServer`, а дальше стартует через
`libraries/.../unix_args.txt` или через старый `forge-*.jar`.

Spigot нельзя раздавать готовым jar, поэтому у версии источник «Сборка
BuildTools»: название версии уходит в `--rev` (`26.2`, `1.21.4`, `latest`).
Агент собирает jar в контейнере `eclipse-temurin:<N>-jdk`, подобрав JDK по
`hub.spigotmc.org/versions/<rev>.json`. Сборка идёт 5–15 минут.

### Bedrock

Образ `mcbedrock` без файлов в томе сам скачивает актуальный сервер: ссылку
отдаёт `net-secondary.web.minecraft-services.net/api/v1.0/download/links`,
версия записывается в `/data/.vtx/bedrock_version`, и при каждом запуске
сервер обновляется до актуальной, не трогая мир, `server.properties`,
`permissions.json` и `allowlist.json`. Переменные `BEDROCK_VERSION`
(`latest`, `preview` или точная версия) и `BEDROCK_DOWNLOAD_URL` меняют
источник. Если файлы положил агент из архива версии, образ их не обновляет.

minecraft.net не отдаёт файлы клиентам со стандартным User-Agent curl и Go,
поэтому и агент, и образ представляются `Mozilla/5.0 (compatible; Vortanix)`.
