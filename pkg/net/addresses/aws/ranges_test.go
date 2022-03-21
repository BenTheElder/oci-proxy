package aws

import (
	"net"
	"testing"
)

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

func BenchmarkParseRanges(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = parseRanges(rawRanges)
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

func BenchmarkMatchIPTrie(b *testing.B) {
	cidrs, err := parseRanges(rawRanges)
	if err != nil {
		b.Error(err)
	}
	trie := cidrsToTrie(cidrs)
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
