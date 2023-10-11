package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var (
	listenAddr      string
	upstreamSpec    string
	defaultUpstream string
	upstreams       map[string]string
	dohClients      map[string]*http.Client
	debug           bool
)

func init() {
	flag.StringVar(&listenAddr, "listen", ":1053", "Address to listen for DNS queries")
	flag.StringVar(&upstreamSpec, "upstreams", "", "Comma-separated list of domain=resolver. E.g. example.com=https://dns.google/dns-query")
	flag.StringVar(&defaultUpstream, "default", "1.1.1.1", "Default upstream resolver (can be regular DNS or DNS over HTTPS)")
	flag.BoolVar(&debug, "debug", false, "Print debug information for queries and responses")
}

func main() {
	flag.Parse()

	upstreams = make(map[string]string)
	dohClients = make(map[string]*http.Client)

	for _, spec := range strings.Split(upstreamSpec, ",") {
		parts := strings.SplitN(spec, "=", 2)
		if len(parts) != 2 {
			fmt.Println("Error in upstream spec:", spec)
			return
		}
		upstreams[parts[0]] = parts[1]
		if strings.HasPrefix(parts[1], "https://") {
			// Resolve the domain of the DoH server using traditional DNS via 1.1.1.1
			dohURL := parts[1]
			domain := strings.TrimPrefix(dohURL, "https://")
			domain = strings.Split(domain, "/")[0]
			ips, err := resolveTraditionalDNS(domain)
			if err != nil {
				fmt.Println("Failed to resolve DoH domain:", domain, "Error:", err)
				return
			}
			if len(ips) == 0 {
				fmt.Println("No IPs returned for DoH domain:", domain)
				return
			}

			// Use the first returned IP
			ip := ips[0]
			dohURL = strings.Replace(dohURL, domain, ip.String(), 1)

			dohClients[dohURL] = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
				},
				Timeout: 5 * time.Second,
			}
		}
	}

	dns.HandleFunc(".", handleRequest)
	server := &dns.Server{Addr: listenAddr, Net: "udp"}
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	domain := r.Question[0].Name

	if debug {
		fmt.Println("Query:", r.Question[0])
	}

	// Resolve the resolver based on domain or use the default one
	upstream := defaultUpstream
	for k, v := range upstreams {
		if strings.HasSuffix(domain, k+".") {
			upstream = v
			break
		}
	}

	var m *dns.Msg
	var err error
	if strings.HasPrefix(upstream, "https://") {
		m, err = resolveDoH(r, upstream)
	} else {
		m, err = resolveTraditionalDNSMsg(r, upstream)
	}

	if err != nil {
		if debug {
			fmt.Println("Failed:", r.Question[0], " error:", err.Error())
		}
		dns.HandleFailed(w, r)
		return
	}

	if debug && m != nil {
		for _, answer := range m.Answer {
			fmt.Println("Answer:", answer)
		}
	}

	w.WriteMsg(m)
}

func resolveDoH(r *dns.Msg, resolver string) (*dns.Msg, error) {
	client, ok := dohClients[resolver]
	if !ok {
		return nil, fmt.Errorf("no DoH client for resolver: %s", resolver)
	}

	data, err := r.Pack()
	if err != nil {
		return nil, err
	}

	resp, err := client.Post(resolver, "application/dns-message", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	msg := new(dns.Msg)
	err = msg.Unpack(body)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func resolveTraditionalDNS(domain string) ([]net.IP, error) {
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	r, _, err := c.Exchange(m, "1.1.1.1:53")
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, answer := range r.Answer {
		if a, ok := answer.(*dns.A); ok {
			ips = append(ips, a.A)
		}
	}

	return ips, nil
}

func resolveTraditionalDNSMsg(r *dns.Msg, resolver string) (*dns.Msg, error) {
	c := new(dns.Client)
	m, _, err := c.Exchange(r, resolver+":53")
	if err != nil {
		return nil, err
	}
	return m, nil
}
