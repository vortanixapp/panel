FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libstdc++6 \
        libgcc1 \
        libcurl4 \
        libuuid1 \
        libssl1.1 \
        libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /data
WORKDIR /data

EXPOSE 22003/udp
EXPOSE 22126/udp
EXPOSE 22005/tcp

VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
