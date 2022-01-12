#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

export GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
export GOBIN=$(go env GOROOT)/bin

go install ${CODEGEN_PKG}/cmd/register-gen

bash "${CODEGEN_PKG}"/generate-internal-groups.sh \
  "deepcopy,conversion,defaulter" \
  github.com/Congrool/nodes-grouping/pkg/generated \
  github.com/Congrool/nodes-grouping/pkg/apis \
  github.com/Congrool/nodes-grouping/pkg/apis \
  "config:v1beta1" \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt

bash "${CODEGEN_PKG}"/generate-internal-groups.sh \
  "deepcopy,conversion,defaulter" \
  github.com/Congrool/nodes-grouping/pkg/generated \
  github.com/Congrool/nodes-grouping/pkg/apis \
  github.com/Congrool/nodes-grouping/pkg/apis \
  "config:v1beta2" \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt

bash "${CODEGEN_PKG}"/generate-groups.sh \
  all \
  github.com/Congrool/nodes-grouping/pkg/generated \
  github.com/Congrool/nodes-grouping/pkg/apis \
  "groupmanagement:v1alpha1" \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt

echo "Generating register for groupmanagement:v1alpha1"
${GOBIN}/register-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=./pkg/apis/groupmanagement/v1alpha1 \
  --output-package=./pkg/apis/groupmanagement/v1alpha1 \
  --output-file-base=zz_generated.register

echo "Generating register for config:v1beta1"
${GOBIN}/register-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=./pkg/apis/config/v1beta1 \
  --output-package=./pkg/apis/config/v1beta1 \
  --output-file-base=zz_generated.register

echo "Generating register for config:v1beta2"
${GOBIN}/register-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=./pkg/apis/config/v1beta2 \
  --output-package=./pkg/apis/config/v1beta2 \
  --output-file-base=zz_generated.register
