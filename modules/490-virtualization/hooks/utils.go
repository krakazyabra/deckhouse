package hooks

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
)

func nameToIP(name string) string {
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

func ipToName(ip string) string {
	addr := net.ParseIP(ip)
	if addr.To4() != nil {
		// IPv4 address
		return "ip-" + strings.ReplaceAll(addr.String(), ".", "-")
	}
	if addr.To16() != nil {
		// IPv6 address
		dst := make([]byte, hex.EncodedLen(len(addr)))
		_ = hex.Encode(dst, addr)
		return fmt.Sprintf("ip-" +
			string(dst[0:4]) + "-" +
			string(dst[4:8]) + "-" +
			string(dst[8:12]) + "-" +
			string(dst[12:16]) + "-" +
			string(dst[16:20]) + "-" +
			string(dst[20:24]) + "-" +
			string(dst[24:28]) + "-" +
			string(dst[28:]))
	}
	return ""
}
