package graph

import (
	"errors"
	"net/http"

	"graphql_gateway/internal/clients"
)

// gqlError carries an `extensions.code` into the GraphQL response.
type gqlError struct {
	message string
	code    string
}

func newError(message, code string) *gqlError { return &gqlError{message: message, code: code} }

func (e *gqlError) Error() string { return e.message }

// Extensions is picked up by graphql-go and rendered as `extensions`.
func (e *gqlError) Extensions() map[string]interface{} {
	return map[string]interface{}{"code": e.code}
}

func toGraphQLError(err error) error {
	var upstream *clients.UpstreamError
	if !errors.As(err, &upstream) {
		return newError("internal error", "INTERNAL_SERVER_ERROR")
	}
	switch {
	case upstream.Status == 0:
		return newError(upstream.Message, "UPSTREAM_UNAVAILABLE")
	case upstream.Status == http.StatusBadRequest:
		return newError(upstream.Message, "BAD_USER_INPUT")
	default:
		return newError(upstream.Message, "UPSTREAM_ERROR")
	}
}
