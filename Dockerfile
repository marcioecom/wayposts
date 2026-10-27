# syntax=docker/dockerfile:1
#
# Imagem única com os binários do repositório. Cada workload escolhe o entrypoint.
# Distroless static: sem shell, sem gerenciador de pacotes, roda como nonroot.
FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
ENV CGO_ENABLED=0 GOFLAGS=-trimpath

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Cross-compila nativamente no host do build: não depende de emulação para
# gerar a imagem amd64 do cluster a partir de um notebook arm64.
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/ ./cmd/...

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/ /usr/local/bin/
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/gateway"]
