package application

import "context"

type commandContextKey struct{}

// CurrentCommandID is audit provenance supplied only by the executor.
func CurrentCommandID(ctx context.Context) string {
	id, _ := ctx.Value(commandContextKey{}).(string)
	return id
}
