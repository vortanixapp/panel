FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN dpkg --add-architecture i386 \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        coreutils \
        locales \
        libc6:i386 \
        lib32gcc-s1 \
        lib32stdc++6 \
        libstdc++6:i386 \
        libgcc-s1:i386 \
        libcurl4:i386 \
        libtinfo5:i386 \
        libncurses5:i386 \
        libssl1.1:i386 \
        zlib1g:i386 \
        libbz2-1.0:i386 \
        libicu66:i386 \
        libkrb5-3:i386 \
        libgdiplus \
        libcurl4 \
        libssl1.1 \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 -s /bin/bash steam

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh && mkdir -p /data

WORKDIR /data

EXPOSE 27015/udp
EXPOSE 27016/udp

VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
