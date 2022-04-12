package awsips

import (
	"net"
	"net/netip"
	"os"
	"testing"

	"sigs.k8s.io/oci-proxy/pkg/net/internal/cidrs"
)

// pre-parsed data from TestMain so benchmark allocation etc is clearer
var parsedPrefixes []netip.Prefix
var parsedIPNet []*net.IPNet

func TestMain(m *testing.M) {
	// parse to netip.Prefix
	c, err := parseRangesNetip(rawRanges)
	if err != nil {
		panic(err)
	}
	parsedPrefixes = c

	// parse to net.IPNet
	c2, err := parseRanges(rawRanges)
	if err != nil {
		panic(err)
	}
	parsedIPNet = c2

	os.Exit(m.Run())
}

func BenchmarkMatchIP(b *testing.B) {
	cidrs, err := parseRanges(rawRanges)
	if err != nil {
		b.Error(err)
	}
	ip := net.ParseIP("52.93.127.172")
	for i := 0; i < b.N; i++ {
		if matchIP(cidrs, ip) != true {
			b.Fatal("ip did not match when it should have")
		}
	}
	/*
		if matchIP(cidrs, net.ParseIP("127.0.0.1")) != false {
			b.Fatal("ip did not match when it should have")
		}
	*/
}

func BenchmarkMatchIPNetip(b *testing.B) {
	cidrs, err := parseRangesNetip(rawRanges)
	if err != nil {
		b.Error(err)
	}
	ip := netip.MustParseAddr("52.93.127.172")
	for i := 0; i < b.N; i++ {
		if matchIPNetip(cidrs, ip) != true {
			b.Fatal("ip did not match when it should have")
		}
	}
	/*
		if matchIP(cidrs, net.ParseIP("127.0.0.1")) != false {
			b.Fatal("ip did not match when it should have")
		}
	*/
}

func BenchmarkParseRanges(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = parseRanges(rawRanges)
	}
}

func BenchmarkParseRangesNetip(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = parseRangesNetip(rawRanges)
	}
}

func BenchmarkMatchIPTrieMemory(b *testing.B) {
	cidrs, err := parseRanges(rawRanges)
	if err != nil {
		b.Error(err)
	}
	for i := 0; i < b.N; i++ {
		_ = cidrsToTrie(cidrs)
	}
}

func BenchmarkMatchCidranger(b *testing.B) {
	trie := cidrsToTrie(parsedIPNet)
	ip := net.ParseIP("52.93.127.172")
	for i := 0; i < b.N; i++ {
		if matchIPTrie(trie, ip) != true {
			b.Fatal("ip did not match when it should have")
		}
	}
	/*
		if matchIP(cidrs, net.ParseIP("127.0.0.1")) != false {
			b.Fatal("ip did not match when it should have")
		}
	*/
}

func BenchmarkMatchCidrsTrieMap(b *testing.B) {
	trieMap := cidrs.NewTrieMap[string]()
	for i := range parsedPrefixes {
		trieMap.Insert(parsedPrefixes[i], "bogus-region")
	}
	ip := netip.MustParseAddr("52.93.127.172")
	for i := 0; i < b.N; i++ {
		if _, contains := trieMap.GetIP(ip); contains != true {
			b.Fatal("ip did not match when it should have")
		}
	}
	/*
		if matchIP(cidrs, net.ParseIP("127.0.0.1")) != false {
			b.Fatal("ip did not match when it should have")
		}
	*/
}
