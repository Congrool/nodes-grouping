#!/usr/bin/env bash

set -x
set -o errexit
set -o nounset
set -o pipefail

# For all commands, the working directory is the parent directory(repo root).
REPO_ROOT=$(git rev-parse --show-toplevel)
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${REPO_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

export GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
export PATH=$PATH:$GOPATH/bin

echo "${REPO_ROOT}/apis/grouppolicy/v1alpha1"

echo "Generating with register-gen"
"${REPO_ROOT}"/bin/register-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=./pkg/apis/groupmanagement/v1alpha1 \
  --output-package=./pkg/apis/groupmanagement/v1alpha1 \
  --output-file-base=zz_generated.register

# register-gen \
#   --go-header-file hack/boilerplate.go.txt \
#   --input-dirs=./pkg/apis/policy/v1alpha1 \
#   --output-package=./pkg/apis/policy/v1alpha1 \
#   --output-file-base=zz_generated.register

# bash "${REPO_ROOT}"/bin/client-gen \
#   --go-header-file "${CODEGEN_PKG}"/hack/boilerplate.go.txt \
#   --clientset-name versioned \
#   --input-base ./pkg/apis
#   --input group:v1alpha1
#   --input-dirs ./pkg/apis/group/v1alpha1 \
#   --output-package ./pkg/generated/clientset
  
# bash "${CODEGEN_PKG}"/generate-groups.sh \
#   all \
#   github.com/Congrool/nodes-grouping/pkg/generated \
#   github.com/Congrool/nodes-grouping/pkg/apis \
#   "group:v1alpha1" \
#   --go-header-file "${REPO_ROOT}"/hack/boilerplate.go.txt \
#   --output-base "${REPO_ROOT}"/

bash "${CODEGEN_PKG}"/generate-groups.sh \
  all \
  github.com/Congrool/nodes-grouping/pkg/generated \
  github.com/Congrool/nodes-grouping/pkg/apis \
  "groupmanagement:v1alpha1" \
  --go-header-file "${REPO_ROOT}"/hack/boilerplate.go.txt \
  --output-base "${REPO_ROOT}"/ 

bash "${CODEGEN_PKG}"/generate-groups.sh \
  "deepcopy" \
  github.com/Congrool/nodes-grouping/pkg/generated \
  github.com/Congrool/nodes-grouping/pkg/apis \
  "config:v1alpha1" \
  --go-header-file "${REPO_ROOT}"/hack/boilerplate.go.txt \
  --output-base "${REPO_ROOT}"/

mv github.com/Congrool/nodes-grouping/pkg/apis/config/v1alpha1/* pkg/apis/config/v1alpha1
mv github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1/* pkg/apis/groupmanagement/v1alpha1
rm -rf pkg/generated
mv github.com/Congrool/nodes-grouping/pkg/generated pkg
rm -rf github.com
