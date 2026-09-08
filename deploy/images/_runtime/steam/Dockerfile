# Общий рантайм Steam-игр Vortanix — один образ вместо 38.
#
# Набор пакетов взят из прежних per-game Dockerfile'ов: они были одинаковы
# у всех Steam-игр, различались только строки ENV с ключом игры. Теперь
# ключ приходит переменной от агента, а образ собирается один раз.
#
# Файлы сервера в образ не входят: их кладёт агент в том /data.
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# i386 нужен и «64-битным» играм: часть Steam-сборок тянет 32-битный
# steamclient и его зависимости.
RUN dpkg --add-architecture i386 \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libc6 \
        libc6:i386 \
        libstdc++6 \
        libstdc++6:i386 \
        libgcc-s1 \
        libgcc-s1:i386 \
        zlib1g \
        zlib1g:i386 \
        libcurl4 \
        libcurl4:i386 \
        libuuid1 \
        libssl3 \
        libssl3:i386 \
        libicu70 \
        libicu70:i386 \
        lib32gcc-s1 \
        lib32stdc++6 \
        file \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 -s /bin/bash steam

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /data
WORKDIR /data

VOLUME ["/data"]

# Порт публикует агент через -p: диапазоны у игр разные, и EXPOSE в общем
# образе был бы враньём.
ENTRYPOINT ["/entrypoint.sh"]
