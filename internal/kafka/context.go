package kafka

import "context"

type metadataKey struct{}

// WithMetadata enriches a context with metadata.
func WithMetadata(ctx context.Context, metadata map[string]string) context.Context {
	return context.WithValue(ctx, metadataKey{}, metadata)
}

// MetadataFromContext returns the metadata from the passed context and a bool
// indicating whether the value is present or not.
func MetadataFromContext(ctx context.Context) (map[string]string, bool) {
	if v := ctx.Value(metadataKey{}); v != nil {
		metadata, ok := v.(map[string]string)
		return metadata, ok
	}
	return nil, false
}

// DetachedContext returns a new context detached from the lifetime
// of ctx, but which still returns the values of ctx.
//
// DetachedContext can be used to maintain the context values required
// to correlate events, but where the operation is "fire-and-forget",
// and should not be affected by the deadline or cancellation of ctx.
func DetachedContext(ctx context.Context) context.Context {
	return &detachedContext{Context: context.Background(), orig: ctx}
}

//nolint:containedctx // we need to detach context
type detachedContext struct {
	context.Context

	orig context.Context
}

// Value returns c.orig.Value(key).
func (c *detachedContext) Value(key any) any {
	return c.orig.Value(key)
}

func Enrich(ctx context.Context, key, value string) context.Context {
	meta, ok := MetadataFromContext(ctx)
	if !ok {
		meta = make(map[string]string)
	}

	meta[key] = value
	return WithMetadata(ctx, meta)
}
