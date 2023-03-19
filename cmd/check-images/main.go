/*
Copyright 2023 The Kubernetes Authors.

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

// A small utility to verify images match in backends

package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/v1/google"
)

const dataFileName = "imagerefs.json"

var hosts = []string{
	// https://github.com/kubernetes/k8s.io/tree/main/registry.k8s.io/manifests#argcr-manifests
	// k8s.gcr.io
	"us.gcr.io/k8s-artifacts-prod",
	"eu.gcr.io/k8s-artifacts-prod",
	"asia.gcr.io/k8s-artifacts-prod",
	// registry.k8s.io
	"asia-east1-docker.pkg.dev/k8s-artifacts-prod/images",
	"asia-south1-docker.pkg.dev/k8s-artifacts-prod/images",
	"asia-northeast1-docker.pkg.dev/k8s-artifacts-prod/images",
	"asia-northeast2-docker.pkg.dev/k8s-artifacts-prod/images",
	"australia-southeast1-docker.pkg.dev/k8s-artifacts-prod/images",
	"europe-north1-docker.pkg.dev/k8s-artifacts-prod/images",
	"europe-southwest1-docker.pkg.dev/k8s-artifacts-prod/images",
	"europe-west1-docker.pkg.dev/k8s-artifacts-prod/images",
	"europe-west2-docker.pkg.dev/k8s-artifacts-prod/images",
	"europe-west4-docker.pkg.dev/k8s-artifacts-prod/images",
	"europe-west8-docker.pkg.dev/k8s-artifacts-prod/images",
	"europe-west9-docker.pkg.dev/k8s-artifacts-prod/images",
	"southamerica-west1-docker.pkg.dev/k8s-artifacts-prod/images",
	"us-central1-docker.pkg.dev/k8s-artifacts-prod/images",
	"us-east1-docker.pkg.dev/k8s-artifacts-prod/images",
	"us-east4-docker.pkg.dev/k8s-artifacts-prod/images",
	"us-east5-docker.pkg.dev/k8s-artifacts-prod/images",
	"us-south1-docker.pkg.dev/k8s-artifacts-prod/images",
	"us-west1-docker.pkg.dev/k8s-artifacts-prod/images",
	"us-west2-docker.pkg.dev/k8s-artifacts-prod/images",
}

func main() {
	// get data
	//hostsToRefs, err := walkHosts(hosts)
	// filterNonDanglingDigests(hostToRefs)
	// writeData(hostsToRefs, "filtered-imagerefs.json")
	//println("Done Fetching Data ....")
	hostsToRefs, err := readData("filtered-imagerefs.json")
	if err != nil {
		panic(err)
	}
	if err := checkImages(hostsToRefs); err != nil {
		panic(err)
	}
}

func checkImages(h HostsToRefs) error {
	// TODO: figure out a reasonable way to output the full skew
	// For now: manually altering this to inspect particular set
	//a := "us.gcr.io/k8s-artifacts-prod"
	//b := "eu.gcr.io/k8s-artifacts-prod"
	a := "us-west1-docker.pkg.dev/k8s-artifacts-prod/images"
	b := "us-west2-docker.pkg.dev/k8s-artifacts-prod/images"
	println("missing images:")
	for key := range h[a] {
		if _, ok := h[b][key]; !ok {
			println(b + "/" + key)
		}
	}
	for key := range h[b] {
		if _, ok := h[a][key]; !ok {
			println(a + "/" + key)
		}
	}
	return nil
}

// deletes all refs that are digest refs that are reachable via a tag
// leaving us only with tags, and digests that have no tag pointing to them
// this is the unique subset of the data vs the more explicit and complete
// dataset containing @digest => digest references
func filterNonDanglingDigests(h HostsToRefs) {
	for host := range h {
		for ref, digest := range h[host] {
			if !isDigestRef(ref) {
				digestRef := tagRefToDigestRef(ref, digest)
				delete(h[host], digestRef)
			}
		}
	}
}

// fixup data from before we trimmed refs in walk
func trimPrefixes(h HostsToRefs) {
	for host := range h {
		trimmedHost := make(PartialRefToDigest)
		for ref, digest := range h[host] {
			trimmedHost[trimRefPrefix(ref)] = digest
		}
		h[host] = trimmedHost
	}
}

func trimRefPrefix(ref string) string {
	if strings.HasPrefix(ref, "k8s-artifacts-prod/images/") {
		return strings.TrimPrefix(ref, "k8s-artifacts-prod/images/")
	}
	return strings.TrimPrefix(ref, "k8s-artifacts-prod/")
}

func isDigestRef(ref string) bool {
	return strings.Contains(ref, "@sha256:")
}

func tagRefToDigestRef(ref, digest string) string {
	idx := strings.Index(ref, ":")
	return ref[:idx] + "@" + digest
}

type HostsToRefs map[string]PartialRefToDigest

func (h HostsToRefs) Add(host string, tags *google.Tags) {
	if h[host] == nil {
		h[host] = make(PartialRefToDigest)
	}
	h[host].Add(tags)
}

type PartialRefToDigest map[string]string

func (r PartialRefToDigest) Add(tags *google.Tags) {
	name := trimRefPrefix(tags.Name)
	for digest, metadata := range tags.Manifests {
		digest := digest
		r[name+"@"+digest] = digest
		for _, tag := range metadata.Tags {
			r[name+":"+tag] = digest
		}
	}
}

func writeData(data HostsToRefs, filename string) error {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(data)
}

func readData(filename string) (HostsToRefs, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	data := make(HostsToRefs)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func tryReadData() HostsToRefs {
	data, err := readData(dataFileName)
	if err != nil {
		return make(HostsToRefs)
	}
	return data
}
