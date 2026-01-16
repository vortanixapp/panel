FROM ubuntu:20.04

RUN dpkg --add-architecture i386 \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates \
        libc6-i386 \
        lib32gcc-s1 \
        libstdc++6:i386 \
        libgcc1:i386 \
        zlib1g:i386 \
        libncurses5:i386 \
        libcurl4:i386 \
        libuuid1:i386 \
    && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /data

WORKDIR /data

EXPOSE 7777/udp

VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
CMD ["./samp03svr"]
