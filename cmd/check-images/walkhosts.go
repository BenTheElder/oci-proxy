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
	"fmt"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/google"
)

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
		if err := writeData(hostsToRefs, dataFileName); err != nil {
			return nil, err
		}
		fmt.Println("Finished host: "+host, time.Now())
	}
	return hostsToRefs, nil
}
