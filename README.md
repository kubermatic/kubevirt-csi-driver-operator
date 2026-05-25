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

## Release Docker Image

1. Visit [Github Actions](https://github.com/kubermatic/kubevirt-csi-driver-operator/actions)
1. Choose the Workflow named `release`
1. Fill in a proper image tag
1. Click `Run workflow`
