// Package tenant carries the request-scoped tenant identifier between the HTTP
// layer and the use cases. It has no internal dependencies so every layer can
// import it without creating a cycle or coupling features to each other.
package tenant

import (
	"context"
	"regexp"
)

// MaxLength bounds the identifier. tenant_id is part of the composite primary
// key on profiles, domains, wirelessconfigs, ieee8021xconfigs and ciraconfigs,
// so an unbounded value becomes a permanently addressable row.
const MaxLength = 64

// Hint describes the accepted format in API error responses.
const (
	Pattern       = `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`
	Hint          = "x-tenant-id must match " + Pattern
	TenantIDClaim = "tenantId"
)

// pattern excludes whitespace, control characters and non-ASCII so two visually
// identical identifiers cannot map to two different primary key values.
var pattern = regexp.MustCompile(Pattern)

type contextKey struct{}

// Valid reports whether tenantID is storable. The empty tenant is the default
// single-tenant value and is always allowed.
func Valid(tenantID string) bool {
	return tenantID == "" || pattern.MatchString(tenantID)
}

// WithContext scopes ctx to tenantID.
func WithContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, contextKey{}, tenantID)
}

// FromContext returns the tenant scoping ctx, or the empty tenant when none was
// set.
func FromContext(ctx context.Context) string {
	tenantID, _ := ctx.Value(contextKey{}).(string)

	return tenantID
}
