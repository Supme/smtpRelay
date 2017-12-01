package model

import (
	"log"
	"net"
)

var (
	allowIPs     []string
	allowDomains []string
)

func SetAllowIP(allowIP ...string) {
	allowIPs = append(allowIPs, allowIP...)
}

func SetAllowDomains(allowDomain ...string) {
	allowDomains = append(allowDomains, allowDomain...)
}

func CheckAllow(ip, domain string) bool {
	return checkAllowIP(ip) && checkAllowDomain(domain)
}

func checkAllowIP(ip string) bool {
	if len(allowIPs) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		log.Print(err)
		return false
	}
	for i := range allowIPs {
		if allowIPs[i] == host {
			return true
		}
	}
	return false
}

func checkAllowDomain(domain string) bool {
	if len(allowDomains) == 0 {
		return true
	}
	for i := range allowDomains {
		if allowDomains[i] == domain {
			return true
		}
	}
	return false
}
