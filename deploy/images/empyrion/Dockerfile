FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV VTX_GAME_KEY="empyrion"
ENV VTX_GAME_NAME="Empyrion"
ENV GAME_PORT="30000"

RUN dpkg --add-architecture i386 \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
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

EXPOSE 30000/udp
EXPOSE 30000/tcp

VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
