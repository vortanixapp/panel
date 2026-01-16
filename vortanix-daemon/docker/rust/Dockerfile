FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN dpkg --add-architecture i386 \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        libc6:i386 \
        libgcc-s1:i386 \
        libstdc++6:i386 \
        libgcc1:i386 \
        zlib1g:i386 \
        libcurl4:i386 \
        libuuid1:i386 \
        libssl1.1:i386 \
        lib32gcc-s1 \
        lib32stdc++6 \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 -s /bin/bash steam

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /data
WORKDIR /data

EXPOSE 28015/udp
EXPOSE 28016/udp
EXPOSE 28016/tcp

VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
