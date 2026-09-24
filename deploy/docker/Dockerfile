# syntax=docker/dockerfile:1

# Versões fixadas: a imagem de build acompanha o go.mod (Cap. 31).
FROM golang:1.27.1-trixie AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSAO=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.versao=${VERSAO}" \
      -o /out/enxamed ./cmd/enxamed

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/enxamed /enxamed
USER nonroot:nonroot
# 7233 gRPC (workers e nós) · 8080 API HTTP · 9090 métricas e pprof
EXPOSE 7233 8080 9090
ENTRYPOINT ["/enxamed"]
