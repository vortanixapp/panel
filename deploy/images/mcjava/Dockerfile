FROM ubuntu:22.04 AS mcrcon-build

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates git build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 https://github.com/Tiiffi/mcrcon.git /tmp/mcrcon \
    && make -C /tmp/mcrcon \
    && install -m 0755 /tmp/mcrcon/mcrcon /usr/local/bin/mcrcon

FROM eclipse-temurin:8-jre AS java8
FROM eclipse-temurin:17-jre AS java17
FROM eclipse-temurin:21-jre AS java21

FROM eclipse-temurin:25-jre

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tini unzip \
    && rm -rf /var/lib/apt/lists/*

COPY --from=java8 /opt/java/openjdk /opt/java/8
COPY --from=java17 /opt/java/openjdk /opt/java/17
COPY --from=java21 /opt/java/openjdk /opt/java/21
RUN ln -s /opt/java/openjdk /opt/java/25

COPY --from=mcrcon-build /usr/local/bin/mcrcon /usr/local/bin/mcrcon

WORKDIR /data

COPY entrypoint.sh /vtx-entrypoint.sh
RUN chmod +x /vtx-entrypoint.sh

EXPOSE 25565/tcp 25565/udp

ENTRYPOINT ["/usr/bin/tini", "--", "/vtx-entrypoint.sh"]
