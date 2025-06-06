#!/bin/sh

set -e

kubectl --context kind-$1 -n db-system wait postgresqldatabase --all --for=condition=DatabaseReady --timeout=3m
kubectl --context kind-$1 -n db-system wait postgresqluser --all --for=condition=UserReady --timeout=3m
kubectl --context kind-$1 -n db-system exec -ti sts/postgresql postgresql -- bash -c "PGPASSWORD=password psql -h localhost -U foo foo -c '\l'"
! kubectl -n postgresql exec -ti sts/postgresql postgresql -- bash -c "PGPASSWORD=password psql -h localhost -U foo foo -c '\l'"
kubectl --context kind-$1  -n db-system delete -f ./config/tests/cases/postgresql/user.yaml
