#!/usr/bin/env bash

# Copyright 2026 The KubeVirt CSI driver Operator Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

source hack/lib.sh

REGISTRY="${REGISTRY:-quay.io/kubermatic}"
IMAGE_NAME="${IMAGE_NAME:-kubevirt-csi-driver-operator}"

# Use git tag if on an exact tag commit, otherwise use the commit hash.
GIT_VERSION="$(git describe --tags --exact-match 2>/dev/null || git rev-parse HEAD)"

IMG="${REGISTRY}/${IMAGE_NAME}:${GIT_VERSION}"

start_docker_daemon_ci

docker login -u $QUAY_IO_USERNAME -p $QUAY_IO_PASSWORD quay.io

echodate "Building image: ${IMG}"
docker build -t "${IMG}" .

echodate "Pushing image: ${IMG}"
docker push "${IMG}"

echodate "Done: ${IMG}"
