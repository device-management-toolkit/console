package main

const (
	addrWildcardIPv4 = "0.0.0.0"
	addrWildcardIPv6 = "::"
	addrLocalhost    = "localhost"
)

// unbracketHost strips a single pair of surrounding brackets from a literal
// IPv6 host so net.JoinHostPort doesn't double-wrap (e.g. "[::1]" → "::1").
func unbracketHost(host string) string {
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		return host[1 : len(host)-1]
	}

	return host
}

// isWildcardListenHost reports whether host is an all-interfaces bind address
// (empty string, 0.0.0.0, ::, or their bracketed forms). Brackets are stripped
// before comparison so both "::" and "[::]" are recognized uniformly.
func isWildcardListenHost(host string) bool {
	host = unbracketHost(host)

	return host == "" || host == addrWildcardIPv4 || host == addrWildcardIPv6
}

// navigableHost returns a host suitable for constructing a browser URL: it
// strips surrounding IPv6 brackets and maps wildcard bind addresses to
// "localhost".
func navigableHost(host string) string {
	if isWildcardListenHost(host) {
		return addrLocalhost
	}

	return unbracketHost(host)
}
