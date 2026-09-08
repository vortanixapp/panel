# Общий рантайм нативных серверов — один образ вместо четырёх
# (samp, crmp, mta, mcbedrock).
#
# ubuntu:20.04 и 32-битные библиотеки: SA-MP и CR-MP до сих пор поставляются
# 32-битными бинарями. Bedrock и MTA 64-битные, им это не мешает.
FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN dpkg --add-architecture i386 \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        libc6 \
        libc6-i386 \
        lib32gcc-s1 \
        lib32stdc++6 \
        libstdc++6 \
        libstdc++6:i386 \
        zlib1g \
        zlib1g:i386 \
        libcurl4 \
        libssl1.1 \
        libncurses5 \
        libncurses5:i386 \
    && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /data
WORKDIR /data

VOLUME ["/data"]

ENTRYPOINT ["/entrypoint.sh"]
