#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# For all commands, the working directory is the parent directory(repo root).
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

export GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
export PATH=$PATH:$GOPATH/bin

echo "${REPO_ROOT}/apis/policy/v1alpha1"

echo "Generating with register-gen"
GO111MODULE=on go install k8s.io/code-generator/cmd/register-gen
register-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=./pkg/apis/group/v1alpha1 \
  --output-package=./pkg/apis/group/v1alpha1 \
  --output-file-base=zz_generated.register

register-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=./pkg/apis/policy/v1alpha1 \
  --output-package=./pkg/apis/policy/v1alpha1 \
  --output-file-base=zz_generated.register
