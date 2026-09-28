#!/bin/sh
# O Teste de Realidade #4 de ponta a ponta, na raiz do repositório:
# cluster kind, duas versões do enxamed, o overlay dev, e a
# atualização da primeira para a segunda com 5.000 jobs e 500
# workflows em andamento.
set -eu
kind get clusters | grep -qx enxame ||
  kind create cluster --config deploy/k8s/kind.yaml
make imagem VERSAO=v1.0.0-rc1 && docker tag enxame:v1.0.0-rc1 enxame:dev
make imagem VERSAO=v1.0.0-rc2
docker build -f examples/cap32/realidade4/Dockerfile -t realidade4:dev .
kind load docker-image -n enxame enxame:dev enxame:v1.0.0-rc2 \
  realidade4:dev
kubectl apply -k deploy/k8s/overlays/dev
kubectl rollout status statefulset/enxame --timeout=5m
kubectl rollout status deploy/worker --timeout=5m
go run ./examples/cap32/realidade4 executar \
  -dsn 'postgres://postgres:enxame@localhost:30432/enxame?sslmode=disable' \
  -aos 15s -csv realidade4.csv \
  -atualizar 'kubectl set image statefulset/enxame enxamed=enxame:v1.0.0-rc2' \
  -esperar 'kubectl rollout status statefulset/enxame --timeout=10m'
