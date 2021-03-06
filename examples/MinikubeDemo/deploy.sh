#/bin/bash
set -x
set -e
minikube start --disk-size 50g --memory 4096 --cpus 4
kubectl apply -f ../../../manifests/0-namespace.yaml
kubectl apply -f ../../../manifests/modeldb/db
kubectl apply -f ../../../manifests/modeldb/backend
kubectl apply -f ../../../manifests/modeldb/frontend
kubectl apply -f ../../../manifests/vizier/db
kubectl apply -f ../../../manifests/vizier/core
kubectl apply -f ../../../manifests/vizier/suggestion/random
kubectl apply -f ../../../manifests/vizier/suggestion/grid
kubectl apply -f ../../../manifests/vizier/suggestion/hyperband
kubectl apply -f ../../../manifests/vizier/earlystopping/medianstopping
kubectl apply -f ../../../manifests/studyjobcontroller/crd.yaml
kubectl apply -f ../../../manifests/studyjobcontroller/rbac.yaml
kubectl apply -f ../../../manifests/studyjobcontroller/mcrbac.yaml
kubectl apply -f ../../../manifests/studyjobcontroller/workerConfigMap.yaml
kubectl apply -f ../../../manifests/studyjobcontroller/metricsControllerConfigMap.yaml
kubectl apply -f ../../../manifests/studycontroller/studycontroller.yaml
