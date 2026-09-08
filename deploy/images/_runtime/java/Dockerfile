# Общий рантайм Minecraft Java — один образ на mcjava, mcpaper,
# mcspigot, mcforge и mcfabric.
#
# Эти пять «игр» различаются не способом запуска, а файлами сервера:
# ядро приезжает в /data/server.jar из архива нужной версии. Скрипт
# запуска у них и раньше был общим — он параметризован SERVER_JAR и
# переменными памяти, а имени игры не знает вовсе.
FROM ubuntu:22.04 AS mcrcon-build

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates git build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 https://github.com/Tiiffi/mcrcon.git /tmp/mcrcon \
    && make -C /tmp/mcrcon \
    && install -m 0755 /tmp/mcrcon/mcrcon /usr/local/bin/mcrcon

# Java 25 LTS: Minecraft 1.20.5+ требует 21, версии 26.x — уже 25.
FROM eclipse-temurin:25-jre

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tini \
    && rm -rf /var/lib/apt/lists/*

COPY --from=mcrcon-build /usr/local/bin/mcrcon /usr/local/bin/mcrcon

WORKDIR /data

COPY entrypoint.sh /vtx-entrypoint.sh
RUN chmod +x /vtx-entrypoint.sh

EXPOSE 25565/tcp 25565/udp

ENTRYPOINT ["/usr/bin/tini", "--", "/vtx-entrypoint.sh"]
