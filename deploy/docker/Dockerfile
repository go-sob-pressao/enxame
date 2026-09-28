# syntax=docker/dockerfile:1

# livro:inicio dockerfile
# Versões fixadas: a imagem de build acompanha o go.mod (Cap. 31).
FROM golang:1.27.1-trixie AS build
WORKDIR /src
# As dependências primeiro: mudam pouco, e a camada fica no cache.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG VERSAO=dev
# CGO desligado: um binário estático, que roda numa imagem sem libc.
# -trimpath e -buildvcs=false: o mesmo código dá o mesmo binário, em
# qualquer máquina e em qualquer diretório.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -buildvcs=false \
      -ldflags="-s -w -X main.versao=${VERSAO}" \
      -o /out/enxamed ./cmd/enxamed

# Sem shell, sem gerenciador de pacotes, sem root: só o binário.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/enxamed /enxamed
USER nonroot:nonroot
# 7233 gRPC (workers e nós) · 8080 API HTTP · 9090 métricas e pprof
EXPOSE 7233 8080 9090
ENTRYPOINT ["/enxamed"]
# livro:fim dockerfile
