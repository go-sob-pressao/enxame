#!/bin/sh
# Congela o contêiner do PostgreSQL por N segundos (padrão 2): o banco
# não cai, não recusa conexões — só não responde. É o que um failover
# rápido, uma pausa longa de I/O ou uma rede congestionada parecem do
# lado do cliente.
set -eu
n=${1:-2}
no=$(kubectl get pod postgres-0 -o jsonpath='{.spec.nodeName}')
id=$(kubectl get pod postgres-0 \
  -o jsonpath='{.status.containerStatuses[0].containerID}' | sed 's|.*//||')
docker exec "$no" ctr -n k8s.io task pause "$id"
sleep "$n"
docker exec "$no" ctr -n k8s.io task resume "$id"
