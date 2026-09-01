// Package principal carries the authenticated caller's identity through a
// request context so use cases can attribute writes (created_by) without the
// delivery layer leaking auth details into their signatures.
package principal

import "context"

type ctxKey struct{}

// WithUser returns a copy of ctx that carries the authenticated user name.
func WithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, ctxKey{}, user)
}

// User returns the authenticated user name stored in ctx, or "" when the
// request was not authenticated (auth disabled) or no principal was recorded.
func User(ctx context.Context) string {
	user, _ := ctx.Value(ctxKey{}).(string)

	return user
}
