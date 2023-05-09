#!/bin/sh

set -e

kubectl -n k8sdb-system wait postgresqldatabase --all --for=condition=DatabaseReady --timeout=1m
kubectl -n k8sdb-system wait postgresqluser --all --for=condition=UserReady --timeout=1m
kubectl -n k8sdb-system exec -ti sts/postgresql postgresql -- bash -c "PGPASSWORD=password psql -h localhost -U foo foo -c '\l'"
kubectl -n k8sdb-system delete -f ./config/tests/cases/postgresql/user.yaml
! kubectl -n postgresql exec -ti sts/postgresql postgresql -- bash -c "PGPASSWORD=password psql -h localhost -U foo foo -c '\l'"
