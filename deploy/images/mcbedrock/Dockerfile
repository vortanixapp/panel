FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        unzip \
        libcurl4 \
        libstdc++6 \
        libgcc-s1 \
        tini \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /data

COPY entrypoint.sh /vtx-entrypoint.sh
RUN chmod +x /vtx-entrypoint.sh

EXPOSE 19132/udp

ENTRYPOINT ["/usr/bin/tini", "--", "/vtx-entrypoint.sh"]
