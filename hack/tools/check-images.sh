#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail

# highest bandwith images
images=(
    # top to bottom per justinsb analysis
    dns/k8s-dns-node-cache
    provider-aws/aws-ebs-csi-driver
    csi-secrets-store/driver
    ingress-nginx/controller
    node-problem-detector/node-problem-detector
    sig-storage/csi-node-driver-registrar
    # exclude because system image difficult-to-impossible for user to fix w/ k8s api
    #kube-proxy
    sig-storage/livenessprobe
    # exclude because system image difficult-to-impossible for user to fix w/ k8s api
    #pause
    #etcd
    k8s-dns-node-cache
    # excluding smaller images below but useful for later
    #e2e-test-images/agnhost
    #hpa-example
    #metrics-server/metrics-server
)
readonly old_registry='k8s.gcr.io'
readonly new_registry='registry.k8s.io'

# check images match between hosts
failed=false
total_checked=0
for image in ${images[@]}; do
    echo "Checking ${image} ..."
    # gcrane for GCR, will surface manifest AND tag references
    # crane / portable container registry APIs can only list tags
    # users could be fetching images by digest even for images that don't have tags
    # anymore, in the case of mutable tags in the past
    references=$(gcrane ls "${old_registry}/${image}")
    image_checked=0
    for reference in ${references[@]}; do
        new_reference="${reference/${old_registry}/${new_registry}}"
        # ensure image references have the same digests
        # fetch with retries and backoff
        # TODO: for digest references we could just check that the image exists in the new registry
        old_digest=""
        for i in 1 2 3 4 5; do old_digest=$(crane digest "${reference}") && break || sleep $i; done
        if [ -z "${old_digest}" ]; then
            echo "FAIL: Failed to check ${old_digest}"
            exit 1
        fi
        new_digest=""
        for i in 1 2 3 4 5; do new_digest=$(crane digest "${new_reference}") && break || sleep $i; done
        if [ -z "${new_digest}" ]; then
            echo "FAIL: Failed to check ${old_digest}"
            exit 1
        fi
        if [[ "${new_digest}" != "${old_digest}" ]]; then
            echo "FAIL: Found Non Matching Image!"
            printf "Old: ${reference}\tdigest: ${old_digest}\n"
            printf "New: ${new_reference}\tdigest: ${new_digest}\n"
            failed=true
        fi
        image_checked=$((image_checked +1))
    done
    total_checked=$((total_checked + image_checked))
    echo "All ${image_checked} references matched for ${image}"
done
if [ "$failed" = true ] ; then
    exit 1
else
    echo "PASS: All ${num_checked} image references had identical digests"
fi
