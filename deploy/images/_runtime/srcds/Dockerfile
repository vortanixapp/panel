# Общий рантайм Source и GoldSrc — один образ вместо четырёх
# (css, tf2, gmod, cs16).
#
# ubuntu:20.04, а не 22.04: srcds_run и hlds_linux целиком 32-битные и
# тянут libssl1.1 и libstdc++5, которых в 22.04 уже нет.
#
# Файлы сервера в образ не входят: их кладёт агент в том /data.
FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN dpkg --add-architecture i386 \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libc6-i386 \
        lib32gcc-s1 \
        lib32stdc++6 \
        libstdc++6:i386 \
        libstdc++5:i386 \
        libgcc1:i386 \
        zlib1g:i386 \
        libncurses5:i386 \
        libcurl4:i386 \
        libuuid1:i386 \
        libssl1.1:i386 \
        libsdl2-2.0-0:i386 \
        libopenal1:i386 \
        libpng16-16:i386 \
        libjpeg62:i386 \
    && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /data
WORKDIR /data

VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
