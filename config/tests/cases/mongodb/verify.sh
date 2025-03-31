#!/bin/sh

set -e
kubectl --context kind-$1 -n db-system wait mongodbdatabase --all --for=condition=DatabaseReady --timeout=3m
kubectl --context kind-$1 -n db-system wait mongodbuser --all --for=condition=UserReady --timeout=3m
kubectl --context kind-$1 -n db-system exec -ti deployment/mongodb mongodb -- mongosh mongodb://localhost:27017/foo --authenticationDatabase=admin -u foo -p password --eval 'db.bar.insert({"foo":"bar"})'
! kubectl --context kind-$1 -n db-system exec -ti deployment/mongodb mongodb -- mongosh mongodb://localhost:27017/foo --authenticationDatabase=admin -u foo -p password --eval 'db.bar.insert({"foo":"bar"})'
kubectl --context kind-$1 -n db-system delete -f ./config/tests/cases/mongodb/user.yaml
