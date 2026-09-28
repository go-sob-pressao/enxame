# A imagem do worker do Teste de Realidade #4. Construa na raiz:
#	docker build -f examples/cap32/realidade4/Dockerfile -t realidade4 .
FROM golang:1.27.1-trixie AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -o /out/realidade4 \
      ./examples/cap32/realidade4

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/realidade4 /realidade4
USER 65532:65532
ENTRYPOINT ["/realidade4"]
