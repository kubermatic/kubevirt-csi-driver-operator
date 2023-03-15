# KubeVirt CSI Driver Operator

Operator that installs KubeVirt CSI Driver components and initializes storage classes on tenant clusters.

## Build

### Binary
```shell
make build
```

### Docker image
```shell
make docker-image
```

## Manifests

### Generate
```shell
make manifests
```

### Install
Deploy CRDs on k8s cluster
```shell
make install
```

### Deploy
Deploys operator on k8s cluster
```shell
make deploy
```

### Fetch manifests
Fetch CRD manifest
```shell
bin/kustomize build config/crd
```

Fetch operator manifest
```shell
bin/kustomize build config/default
```

## Run integration tests
Integration tests are using testEnv.  
Please read this [Configuring envtest for integration tests](https://book.kubebuilder.io/reference/envtest.html) to setup your env.  

1- To install testenv
```shell
make envtest
```

2- To run locally (not in ci) the integration tests
```shell
make local-test-integration
```
That will:
- copy the needed envtest binaries (etcd, kube-apiserver, kubectl) in your $LOCALBIN folder
- set KUBEBUILDER_ASSETS to point there
- run the integration tests

Note that there is:
```shell
make clean-test-cache
```
to clean to test cache.

