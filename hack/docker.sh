#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This script holds docker related functions.

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
REGISTRY=${REGISTRY:-"kubeedge.io"}
VERSION=${VERSION:="latest"}

function build_images() {
  local target="$1"
  docker build -t ${REGISTRY}/${target}:${VERSION} -f ${REPO_ROOT}/artifacts/images/${target}/Dockerfile ${REPO_ROOT}
}

build_images $@
