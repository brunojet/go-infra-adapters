package net_http_base

// EventType represents the type of security event for header sanitization
type EventType string

const (
	// InboundRequest: headers removed from inbound HTTP request (server middleware)
	InboundRequest EventType = "inbound_request_header_removed"

	// OutboundRequest: headers removed from outbound HTTP request (client middleware)
	OutboundRequest EventType = "outbound_request_header_removed"

	// InboundResponse: headers removed from inbound HTTP response (client middleware)
	InboundResponse EventType = "inbound_response_header_removed"

	// OutboundResponse: headers removed from outbound HTTP response (server middleware)
	OutboundResponse EventType = "outbound_response_header_removed"
)

// String returns the string representation of the EventType
func (et EventType) String() string {
	return string(et)
}
