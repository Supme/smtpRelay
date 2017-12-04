package model

import (
	"log"
	"net"
)

// CheckAllow check right for IP and sender email domain
func CheckAllow(ip, domain string) bool {
	return checkAllowIP(ip) && checkAllowDomain(domain)
}

func checkAllowIP(ip string) bool {
	if len(Config.AllowIP) == 0 {
		return true
	}
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		log.Print(err)
		return false
	}
	for i := range Config.AllowIP {
		if Config.AllowIP[i] == host {
			return true
		}
	}
	return false
}

func checkAllowDomain(domain string) bool {
	if len(Config.AllowDomains) == 0 {
		return true
	}
	for i := range Config.AllowDomains {
		if Config.AllowDomains[i] == domain {
			return true
		}
	}
	return false
}
