package utils

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
)

type IPStore struct {
	ips map[string]struct{}
	mu  *sync.Mutex
}

func NewIPStore() *IPStore {
	var mutex sync.Mutex
	return &IPStore{
		mu:  &mutex,
		ips: make(map[string]struct{}),
	}
}

func (s *IPStore) Add(ip string) {
	s.mu.Lock()
	s.ips[ip] = struct{}{}
	s.mu.Unlock()
}

func (s *IPStore) Del(ip string) {
	s.mu.Lock()
	delete(s.ips, ip)
	s.mu.Unlock()
}

func (s *IPStore) IsAllocated(ip string) bool {
	s.mu.Lock()
	_, ok := s.ips[ip]
	s.mu.Unlock()
	if ok {
		return true
	}
	return false
}

func NameToIP(name string) string {
	a := strings.Split(name, "-")
	if a[0] != "ip" {
		return ""
	}
	// IPv4 address
	if len(a) == 5 {
		return fmt.Sprintf("%s.%s.%s.%s", a[1], a[2], a[3], a[4])
	}
	// IPv6 address
	if len(a) == 9 {
		return fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s", a[1], a[2], a[3], a[4], a[5], a[6], a[7], a[8])
	}
	return ""
}

func IPToName(ip string) string {
	addr := net.ParseIP(ip)
	if addr.To16() != nil {
		// IPv6 address
		dst := make([]byte, hex.EncodedLen(len(addr)))
		_ = hex.Encode(dst, addr)
		return fmt.Sprintf(
			string(dst[0:4]) + ":" +
				string(dst[4:8]) + ":" +
				string(dst[8:12]) + ":" +
				string(dst[12:16]) + ":" +
				string(dst[16:20]) + ":" +
				string(dst[20:24]) + ":" +
				string(dst[24:28]) + ":" +
				string(dst[28:]))
	}
	// IPv4 address
	return addr.String()
}
