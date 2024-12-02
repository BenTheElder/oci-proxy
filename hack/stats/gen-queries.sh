#!/usr/bin/env bash

# Copyright 2024 The Kubernetes Authors.
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

# script to generate platform image pull queries
set -o errexit -o nounset -o pipefail

host='registry.k8s.io'
image='kube-proxy'

versions=( 'v1.30.'{0..7} )
declare -A arch_digests
for arch in '"s390x"' '"ppc64le"' '"amd64"' '"arm64"'; do
    arch_digests[${arch}]="("
done
for version in ${versions[@]}; do
    version_image="${host}/${image}:${version}"
    manifest="$(crane manifest "${version_image}")"
    arches=( $(jq '.manifests[].platform.architecture' <<< "${manifest}") )
    for arch in "${arches[@]}"; do
        digest="$(jq '.manifests[] | select(.platform.architecture == '"${arch}"') | .digest' -r <<< "${manifest}")"
        arch_digests[${arch}]="${arch_digests[${arch}]} OR httpRequest.requestUrl=\"https://${host}/v2/${image}/manifests/${digest}\""
    done
done
for arch in ${!arch_digests[@]}; do
    echo "Query for $arch:"
    echo 'resource.type="http_load_balancer"'
    arch_digests[${arch}]=$(echo "${arch_digests[${arch}]}"')' | perl -pe 's/\( OR /\(/')
    echo "${arch_digests[${arch}]}"
    echo ""
done

