#!/bin/sh

set -e

kubectl -n k8sdb-system wait mongodbdatabase --all --for=condition=DatabaseReady --timeout=1m
kubectl -n k8sdb-system wait mongodbuser --all --for=condition=UserReady --timeout=1m
kubectl -n k8sdb-system exec -ti deployment/mongodb mongodb -- mongosh mongodb://localhost:27017/foo --authenticationDatabase=admin -u foo -p password --eval 'db.bar.insert({"foo":"bar"})'
kubectl -n k8sdb-system delete -f ./config/tests/cases/mongodb/user.yaml
! kubectl -n k8sdb-system exec -ti deployment/mongodb mongodb -- mongosh mongodb://localhost:27017/foo --authenticationDatabase=admin -u foo -p password --eval 'db.bar.insert({"foo":"bar"})'