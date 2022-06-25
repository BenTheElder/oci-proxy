/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"net/http"
	"regexp"
	"strings"

	"k8s.io/klog/v2"

	"sigs.k8s.io/oci-proxy/pkg/net/cidrs/aws"
)

const (
	// hard coded for now
	infoURL = "https://github.com/kubernetes/k8s.io/tree/main/registry.k8s.io"
)

// MakeHandler returns the root archeio HTTP handler
//
// upstream registry should be the url to the primary registry
// archeio is fronting.
//
// Exact behavior should be documented in docs/request-handling.md
func MakeHandler(upstreamRegistry string) http.Handler {
	blobs := newCachedBlobChecker()
	doV2 := makeV2Handler(upstreamRegistry, blobs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// only allow GET, HEAD
		// this is all a client needs to pull images
		// we do *not* support mutation
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Only GET and HEAD are allowed.", http.StatusMethodNotAllowed)
			return
		}
		// all valid registry requests should be at /v2/
		// v1 API is super old and not supported by GCR anymore.
		path := r.URL.Path
		switch {
		case path == "/v2/":
			w.WriteHeader(http.StatusOK)
			return
		case strings.HasPrefix(path, "/v2/"):
			doV2(w, r)
		case path == "/":
			http.Redirect(w, r, infoURL, http.StatusPermanentRedirect)
		default:
			klog.V(2).InfoS("unknown request", "path", path)
			http.NotFound(w, r)
		}
	})
}

func makeV2Handler(upstreamRegistry string, blobs blobChecker) func(w http.ResponseWriter, r *http.Request) {
	// matches blob requests, captures the requested blob hash
	reBlob := regexp.MustCompile("^/v2/.*/blobs/sha256:([0-9a-f]{64})$")
	// initialize map of clientIP to AWS region
	regionMapper := aws.NewAWSRegionMapper()
	// capture these in a http handler lambda
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		//if path == "/v2/" || path == "/v2"

		// check if blob request
		matches := reBlob.FindStringSubmatch(path)
		if len(matches) != 2 {
			// not a blob request so forward it to the main upstream registry
			redirectPath := path
			// if path is not just /v2/, which is a special endpoint
			if len(path) > 5 {
				redirectPath = "/v2/" + "k8s-artifacts-prod" + strings.TrimPrefix(path, "/v2")
			}
			klog.V(2).InfoS("redirecting non-blob request to upstream registry", "path", path, "redirect", upstreamRegistry+redirectPath)
			http.Redirect(w, r, upstreamRegistry+redirectPath, http.StatusPermanentRedirect)
			return
		}

		// for blob requests, check the client IP and determine the best backend
		clientIP, err := getClientIP(r)
		if err != nil {
			// this should not happen
			klog.ErrorS(err, "failed to get client IP")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// check if client is known to be coming from an AWS region
		awsRegion, ipIsKnown := regionMapper.GetIP(clientIP)
		if !ipIsKnown {
			// no region match, redirect to main upstream registry
			redirectPath := path
			// if path is not just /v2/, which is a special endpoint
			if len(path) > 5 {
				redirectPath = "/v2/" + "k8s-artifacts-prod" + strings.TrimPrefix(path, "/v2")
			}
			klog.V(2).InfoS("redirecting blob request to upstream registry", "path", path, "redirect", upstreamRegistry+redirectPath)
			http.Redirect(w, r, upstreamRegistry+redirectPath, http.StatusPermanentRedirect)
			return
		}

		// check if blob is available in our S3 bucket for the region
		bucketURL := awsRegionToS3URL(awsRegion)
		hash := matches[1]
		// this matches GCR's GCS layout, which we will use for other buckets
		blobURL := bucketURL + "/containers/images/sha256%3A" + hash
		if blobs.BlobExists(blobURL, bucketURL, hash) {
			// blob known to be available in S3, redirect client there
			klog.V(2).InfoS("redirecting blob request to S3", "path", path)
			http.Redirect(w, r, blobURL, http.StatusPermanentRedirect)
			return
		}

		// fall back to redirect to upstream
		redirectPath := path
		// if path is not just /v2/, which is a special endpoint
		if len(path) > 5 {
			redirectPath = "/v2/" + "k8s-artifacts-prod" + strings.TrimPrefix(path, "/v2")
		}
		klog.V(2).InfoS("redirecting blob request to upstream registry", "path", path, "redirect", upstreamRegistry+redirectPath)
		http.Redirect(w, r, upstreamRegistry+redirectPath, http.StatusPermanentRedirect)

	}
}
