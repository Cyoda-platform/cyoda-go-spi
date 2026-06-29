package spi

import "context"

type uniqueKeysCtxKey struct{}

func WithUniqueKeys(ctx context.Context, keys []UniqueKey) context.Context {
	return context.WithValue(ctx, uniqueKeysCtxKey{}, keys)
}

func UniqueKeysFromContext(ctx context.Context) []UniqueKey {
	if v, ok := ctx.Value(uniqueKeysCtxKey{}).([]UniqueKey); ok {
		return v
	}
	return nil
}
