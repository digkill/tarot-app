FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/arcana ./cmd/server

# curl is there for the Coolify health check.
FROM debian:bookworm-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 65532 --home-dir /app --create-home nonroot
WORKDIR /app
COPY --from=build /out/arcana /app/arcana
ENV ARCANA_ADDR=:8090
EXPOSE 8090
USER nonroot
CMD ["/app/arcana"]
