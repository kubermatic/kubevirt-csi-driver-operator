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

## Run tests
```shell
make test
```

