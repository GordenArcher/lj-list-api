package models

import "time"

// Metadata carries request-level telemetry. For this API it's a timestamp.
// If we add request tracing later, the trace ID lives here.
type Metadata struct {
	Timestamp string `json:"timestamp"`
}

// APIResponse is the universal envelope for every HTTP response this server
// sends, success, error, documentation, health check, all of it. A single
// struct for every response shape means the frontend can write one parser
// and never special-case an endpoint. The omitempty tags ensure that error
// responses don't carry a null "data" field and success responses don't
// carry a null "errors" field, the wire format stays clean without any
// manual nil checks in the handlers.
type APIResponse struct {
	Status    string              `json:"status"`
	Message   string              `json:"message"`
	Data      any                 `json:"data,omitempty"`
	Errors    map[string][]string `json:"errors,omitempty"`
	Code      string              `json:"code"`
	RequestID string              `json:"request_id"`
	Metadata  Metadata            `json:"metadata"`
}

// NewSuccessResponse constructs a success envelope. Using a constructor
// rather than initializing the struct directly in the handler ensures
// required fields, timestamp, status, code, are never accidentally left
// blank. The requestID comes from middleware; message is human-readable;
// data is the domain payload.
func NewSuccessResponse(requestID, message string, data any) APIResponse {
	return APIResponse{
		Status:    "success",
		Message:   message,
		Data:      data,
		Code:      "OK",
		RequestID: requestID,
		Metadata: Metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// NewErrorResponse constructs a consistent error envelope. The errs map
// accepts field-level validation errors so the frontend can display them
// next to the specific input that caused the problem. For non-validation
// errors (auth failures, internal errors), pass a single key that describes
// the failure domain ("auth", "server") rather than a field name.
func NewErrorResponse(requestID, code, message string, errs map[string][]string) APIResponse {
	return APIResponse{
		Status:    "error",
		Message:   message,
		Errors:    errs,
		Code:      code,
		RequestID: requestID,
		Metadata: Metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// APIDocumentation is the top-level documentation payload returned by the
// "/" endpoint. It describes the service, every available route, and
// operational notes that don't fit neatly into a per-route description.
// A frontend dev can open this in a browser and integrate without reading
// source code.
type APIDocumentation struct {
	Service     string     `json:"service"`
	Version     string     `json:"version"`
	Description string     `json:"description"`
	Routes      []RouteDoc `json:"routes"`
	Notes       []string   `json:"notes"`
}

// RouteDoc describes a single API endpoint in enough detail that a
// developer can integrate without reading any other documentation.
// The curl example field is not optional, a working copy-pasteable
// command is worth more than three paragraphs of prose.
type RouteDoc struct {
	Method          string                 `json:"method"`
	Path            string                 `json:"path"`
	Description     string                 `json:"description"`
	Auth            bool                   `json:"auth"`
	AdminOnly       bool                   `json:"admin_only,omitempty"`
	Headers         map[string]string      `json:"headers,omitempty"`
	Request         any                    `json:"request,omitempty"`
	Response        map[string]any         `json:"response,omitempty"`
	ResponseSuccess APIResponse            `json:"response_success,omitempty"`
	ResponseError   APIResponse            `json:"response_error,omitempty"`
	ErrorExamples   map[string]APIResponse `json:"error_examples,omitempty"`
	Example         string                 `json:"example"`
}

// NewDocResponse wraps the API documentation in the standard envelope.
// The frontend can hit GET / and get a fully typed API reference in the
// same shape as every other response, one parser, zero special cases.
func NewDocResponse(requestID string, doc APIDocumentation) APIResponse {
	return APIResponse{
		Status:    "success",
		Message:   "API documentation",
		Data:      doc,
		Code:      "DOCUMENTATION",
		RequestID: requestID,
		Metadata: Metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
}
