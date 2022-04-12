package awsips

//go:generate ./internal/cmd/ranges2go/run.sh

import (
	"net"
	"net/netip"

	"github.com/yl2chen/cidranger"
)

func parseRanges(rawCidrs []string) ([]*net.IPNet, error) {
	res := make([]*net.IPNet, len(rawCidrs))
	for i, rawRange := range rawCidrs {
		_, ipNet, err := net.ParseCIDR(rawRange)
		if err != nil {
			return nil, err
		}
		res[i] = ipNet
	}
	return res, nil
}

func parseRangesNetip(rawCidrs []string) ([]netip.Prefix, error) {
	res := make([]netip.Prefix, len(rawCidrs))
	for i, rawRange := range rawCidrs {
		p, err := netip.ParsePrefix(rawRange)
		if err != nil {
			return nil, err
		}
		res[i] = p
	}
	return res, nil
}

func matchIP(cidrs []*net.IPNet, ip net.IP) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func matchIPNetip(cidrs []netip.Prefix, ip netip.Addr) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func cidrsToTrie(cidrs []*net.IPNet) cidranger.Ranger {
	ranger := cidranger.NewPCTrieRanger()
	for _, cidr := range cidrs {
		ranger.Insert(cidranger.NewBasicRangerEntry(*cidr))
	}
	return ranger
}

func matchIPTrie(t cidranger.Ranger, ip net.IP) bool {
	res, _ := t.Contains(ip)
	return res
}
