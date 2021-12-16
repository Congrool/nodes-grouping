#!/usr/bin/env bash

set -x
set -u
set -e

ROOTPATH="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

readonly TARGET_BIN=(
    extender
    controller-manager
)

# $1: target path
# $2: output path
function build_target {
    CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o $2 $1 
}

function build_all {
    local cmd_path="${ROOTPATH}/cmd"
    local output_path="${ROOTPATH}/_output"
    mkdir -p ${output_path}

    for target in ${TARGET_BIN[@]}; do
        target_path="${cmd_path}/${target}"
        $(build_target $target_path $output_path/$target)
    done
}

build_all