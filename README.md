# dnsrouter

This DNS Proxy is designed to route DNS queries to specified upstream DNS servers based on domains/zones. It supports both traditional DNS and DNS-over-HTTPS (DoH) as upstream servers.

I initially wrote this when I was working from home and connected to my work VPN to route DNS queries for work-related domains to my work VPN's DNS without routing everything to it.

## Compilation

Prerequisites

- Golang installed on your machine.
- Required package: `github.com/miekg/dns`. Install with `go get github.com/miekg/dns`.

1. Place the DNS proxy code into a file named `dnsrouter.go`.
2. Compile the program using:
   ```
   go build dnsrouter.go
   ```

## Usage

Binaries can be found in [Releases](https://github.com/themicknugget/dnsrouter/releases).

```bash
./dnsrouter -listen [LISTEN_ADDRESS] -upstreams [UPSTREAMS] -default [DEFAULT_UPSTREAM]
```

### Parameters:

- `-listen`: Address and port to listen for DNS queries.
   - Default: `:53`
   - Example: `-listen :5353`

- `-upstreams`: Comma-separated list of domain=resolver configurations.
   - Format: `domain=resolver,domain=resolver,...`
   - Resolver can be a traditional DNS server IP or a DNS-over-HTTPS URL.
   - Example: `-upstreams "example.com=https://dns.google/dns-query,example.net=8.8.4.4"`

- `-default`: Default upstream resolver for all other domains not specified in `-upstreams`.
   - Can be a regular DNS server IP or a DNS-over-HTTPS URL.
   - Default: `1.1.1.1`
   - Example: `-default "https://cloudflare-dns.com/dns-query"`

## Behavior:

- Queries for specified domains in `-upstreams` will be routed to the corresponding resolver.
- All other queries will be routed to the `-default` resolver.
- DNS-over-HTTPS servers specified will first have their domain resolved via traditional DNS using `1.1.1.1`.
- If a traditional DNS server doesn't respond or if a DNS-over-HTTPS server is unreachable, it will be ignored for subsequent queries.

## Example:

To route DNS queries for `example.com` to Google's DNS-over-HTTPS, for `example.net` to `8.8.4.4` (traditional DNS), and all other queries to Cloudflare's DNS-over-HTTPS, you would run:

```bash
./dnsrouter -upstreams "example.com=https://dns.google/dns-query,example.net=8.8.4.4" -default "https://cloudflare-dns.com/dns-query"
```
