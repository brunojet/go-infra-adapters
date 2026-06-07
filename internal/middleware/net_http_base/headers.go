package net_http_base

// HTTP Header names (canonical form per http.CanonicalHeaderKey)
// Centralized definitions to avoid duplication across server and client middlewares

// Exported headers (used outside this package)
const (
	// Content metadata
	HeaderContentType   = "Content-Type"
	HeaderContentLength = "Content-Length"

	// Response metadata (allowed by default for inbound responses)
	HeaderCacheControl = "Cache-Control"
	HeaderExpires      = "Expires"
	HeaderDate         = "Date"
	HeaderLocation     = "Location"
	HeaderRetryAfter   = "Retry-After"
	HeaderServer       = "Server"
)

// Unexported headers (internal use only)
const (
	// Credentials & Authentication
	headerAuthorization      = "Authorization"
	headerCookie             = "Cookie"
	headerProxyAuthorization = "Proxy-Authorization"
	headerSetCookie          = "Set-Cookie"

	// Routing & Context
	headerHost      = "Host"
	headerUserAgent = "User-Agent"

	// Hop-by-hop headers (RFC 7230)
	headerConnection       = "Connection"
	headerKeepAlive        = "Keep-Alive"
	headerProxyAuth        = "Proxy-Authenticate"
	headerTE               = "Te"
	headerTrailers         = "Trailers"
	headerTransferEncoding = "Transfer-Encoding"
	headerUpgrade          = "Upgrade"
	headerProxyConnection  = "Proxy-Connection"

	// Proxy headers
	headerXForwardedFor   = "X-Forwarded-For"
	headerXForwardedProto = "X-Forwarded-Proto"
	headerXForwardedHost  = "X-Forwarded-Host"
	headerXRealIP         = "X-Real-Ip"

	// Encoding
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"

	// Cache validators
	headerIfNoneMatch     = "If-None-Match"
	headerIfModifiedSince = "If-Modified-Since"
)

// DeniedHeadersCommon are headers that should be blocked in BOTH request and response
// Use these as a base for building direction-specific lists
var DeniedHeadersCommon = map[string]bool{
	// Credentials (never forward or expose)
	headerAuthorization:      true,
	headerCookie:             true,
	headerProxyAuthorization: true,

	// Hop-by-hop (managed by transport layer)
	headerConnection:       true,
	headerKeepAlive:        true,
	headerTE:               true,
	headerTrailers:         true,
	headerTransferEncoding: true,
	headerUpgrade:          true,
	headerProxyConnection:  true,

	// Proxy headers (should not propagate)
	headerXForwardedFor:   true,
	headerXForwardedProto: true,
	headerXForwardedHost:  true,
	headerXRealIP:         true,

	// Encoding (managed by each layer: BFF controls own compression)
	headerContentEncoding: true,

	// Content-Length: calculated by client/server (request) or calculated from body (response)
	HeaderContentLength: true,
}

// DeniedHeadersRequestOnly are headers that should only be blocked in REQUEST direction
// (they make sense in responses but not in requests)
var DeniedHeadersRequestOnly = map[string]bool{
	// Routing/context specific to client
	headerHost:      true,
	headerUserAgent: true,

	// Encoding preference (client's compression preference, not applicable for upstream)
	headerAcceptEncoding: true,

	// Cache validators from client (not applicable for upstream)
	headerIfNoneMatch:     true,
	headerIfModifiedSince: true,
}

// DeniedHeadersResponseOnly are headers that should only be blocked in RESPONSE direction
// (they make sense in requests but not in responses, or are upstream-specific)
var DeniedHeadersResponseOnly = map[string]bool{
	// Session/cookies from upstream
	headerSetCookie: true,
}

// BuildDeniedHeaders combines common and direction-specific headers into a complete deny list
func BuildDeniedHeaders(directionSpecific map[string]bool) map[string]bool {
	result := make(map[string]bool, len(DeniedHeadersCommon)+len(directionSpecific))

	// Add common denied headers
	for k, v := range DeniedHeadersCommon {
		result[k] = v
	}

	// Add direction-specific denied headers
	for k, v := range directionSpecific {
		result[k] = v
	}

	return result
}
