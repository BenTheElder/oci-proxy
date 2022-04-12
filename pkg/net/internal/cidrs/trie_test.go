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

package cidrs

import (
	"net/netip"
	"testing"
)

func TestTrieMap(t *testing.T) {
	trieMap := NewTrieMap[string]()
	testCIDRS := []netip.Prefix{
		netip.MustParsePrefix("35.180.0.0/16"),
		netip.MustParsePrefix("52.94.76.0/22"),
		netip.MustParsePrefix("52.93.127.170/32"),
		netip.MustParsePrefix("52.93.127.172/31"),
		netip.MustParsePrefix("52.93.127.173/32"),
		netip.MustParsePrefix("52.93.127.174/32"),
		netip.MustParsePrefix("52.93.127.175/32"),
		netip.MustParsePrefix("52.93.127.176/32"),
		netip.MustParsePrefix("52.93.127.177/32"),
		netip.MustParsePrefix("52.93.127.178/32"),
		netip.MustParsePrefix("52.93.127.179/32"),
		// ipv6
		netip.MustParsePrefix("2400:6500:0:9::2/128"),
	}
	testIPs := []netip.Addr{
		netip.MustParseAddr("35.180.1.1"),
		netip.MustParseAddr("35.250.1.1"),
		netip.MustParseAddr("52.94.76.1"),
		netip.MustParseAddr("52.94.77.1"),
		netip.MustParseAddr("52.93.127.172"),
		netip.MustParseAddr("2400:6500:0:9::2"),
	}
	for i := range testCIDRS {
		trieMap.Insert(testCIDRS[i], "foo")
	}
	naiveContainsIP := func(ip netip.Addr) bool {
		for _, cidr := range testCIDRS {
			if cidr.Contains(ip) {
				return true
			}
		}
		return false
	}
	for _, ip := range testIPs {
		_, contains := trieMap.GetIP(ip)
		if contains != naiveContainsIP(ip) {
			t.Fatalf("trie does not match naive for %v", ip)
		}
	}
}
