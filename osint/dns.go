package osint

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/miekg/dns"

	"github.com/caio-ishikawa/netscout/shared"
)

const (
	unexpectedResponseErr = "received unexpected response from DNS server"
	noDNSServersFoundErr  = "no DNS servers found for domain"
	gettingIPErr          = "could not get ips associated with domain"
	noIPsErr              = "no IPs found for domain"
)

// Attempts to perform a DNS zone transfer for each name server of a given domain
func ZoneTransfer(domain string) ([]url.URL, []error) {
	nameServers, err := getDNSServers(domain)
	if err != nil {
		return []url.URL{}, []error{err}
	}

	var errs []error

	var foundDomains []url.URL
	for _, ns := range nameServers {
		subdomains, errs := performAxfr(domain, ns.String())
		if len(errs) > 0 {
			errs = append(errs, errs...)
		}
		foundDomains = append(foundDomains, subdomains...)
	}

	return foundDomains, errs
}

func getDNSServers(domain string) ([]net.IP, error) {
	nsRecords, err := net.LookupNS(domain)
	if err != nil {
		return []net.IP{}, err
	}

	var nameServers []net.IP
	for _, nsRecord := range nsRecords {
		ipv4, err := getIPV4(strings.TrimSuffix(nsRecord.Host, "."))
		if err != nil {
			return []net.IP{}, err
		}

		nameServers = append(nameServers, ipv4)
	}

	if len(nameServers) == 0 {
		return []net.IP{}, fmt.Errorf(noDNSServersFoundErr)
	}

	return nameServers, nil
}

func getIPV4(domain string) (net.IP, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return net.IP{}, err
	}

	if len(ips) == 0 {
		return net.IP{}, fmt.Errorf(noIPsErr)
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			return ip, nil
		}
	}

	return net.IP{}, fmt.Errorf(noIPsErr)
}

// Sends zone transfer request for spefific name server, and updates foundDomains list in-place
func performAxfr(domain string, nameServerIP string) ([]url.URL, []error) {
	transfer := new(dns.Transfer)
	msg := new(dns.Msg)
	msg.SetAxfr(domain + ".")

	var errs []error

	ch, err := transfer.In(msg, nameServerIP+":53")
	if err != nil {
		return []url.URL{}, []error{err}
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)

	var foundDomains []url.URL
	for env := range ch {
		if env.Error != nil {
			return []url.URL{}, []error{env.Error}
		}

		for _, rr := range env.RR {
			trimmed := strings.TrimSuffix(rr.Header().Name, ".")

			url, err := url.Parse(trimmed)
			if err != nil {
				errs = append(errs, err)
			}

			if !shared.SliceContainsURL(foundDomains, *url) {
				foundDomains = append(foundDomains, *url)
			}
		}
	}

	writer.Flush()

	return foundDomains, errs
}
