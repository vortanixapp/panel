FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG VORTANIX_VERSION=dev
ARG COMPONENT=vortanix-api
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w -X main.version=${VORTANIX_VERSION}" \
    -o /out/vortanix ./cmd/${COMPONENT}

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 vortanix

WORKDIR /app

COPY --from=build /out/vortanix /usr/local/bin/vortanix
COPY migrations ./migrations

RUN mkdir -p /app/uploads && chown -R vortanix:vortanix /app/uploads

USER vortanix

ENV CORE_MIGRATIONS_DIR=/app/migrations/core \
    UPLOAD_DIR=/app/uploads

ENTRYPOINT ["vortanix"]
