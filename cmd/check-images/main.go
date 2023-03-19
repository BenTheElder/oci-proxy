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
	"fmt"
	"os"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
)

const dataFileName = "imagerefs.json"

func main() {
	hosts := []string{
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

	// get data
	hostsToRefs, err := walkHosts(hosts)
	if err != nil {
		panic(err)
	}
	println("Done Fetching Data ....")
	if err := checkImages(hostsToRefs); err != nil {
		panic(err)
	}
}

func walkHosts(hosts []string) (HostsToRefs, error) {
	// grab past run data from disk if we have it
	hostsToRefs := tryReadData()
	// roughly 5000 RPM, the limit on our registries
	transport := NewRateLimitRoundTripper(83)
	for _, host := range hosts {
		// skip hosts we have from disk
		if _, ok := hostsToRefs[host]; ok {
			continue
		}
		// identify all references => digests for all images in this repo
		repo, err := name.NewRepository(host)
		if err != nil {
			return nil, err
		}
		if err := google.Walk(repo, func(r name.Repository, tags *google.Tags, err error) error {
			if err != nil {
				return err
			}
			// we only care about leaf entries where len(tags.Tags) > 0 and len(tags.Children) == 0
			if len(tags.Tags) == 0 {
				return nil
			}
			hostsToRefs.Add(host, tags)
			return nil
		}, google.WithTransport(transport)); err != nil {
			return nil, err
		}
		// snapshot data to disk after each host, scanning these is *slow*
		if err := writeData(hostsToRefs); err != nil {
			return nil, err
		}
		fmt.Println("Finished host: "+host, time.Now())
	}
	return hostsToRefs, nil
}

func checkImages(h HostsToRefs) error {
	// TODO
	for host := range h {
		refs := len(h[host])
		fmt.Println(refs, host)
	}

	return nil
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
	name := tags.Name
	for digest, metadata := range tags.Manifests {
		digest := digest
		r[name+"@"+digest] = digest
		for _, tag := range metadata.Tags {
			r[name+":"+tag] = digest
		}
	}
}

func writeData(data HostsToRefs) error {
	file, err := os.OpenFile(dataFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(data)
}

func readData() (HostsToRefs, error) {
	file, err := os.Open(dataFileName)
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
	data, err := readData()
	if err != nil {
		return make(HostsToRefs)
	}
	return data
}
