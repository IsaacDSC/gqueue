package asyncadapter

import "context"

func NewAsyncCtx[T any](ctx context.Context, payload []byte) AsyncCtx[T] {
	return AsyncCtx[T]{
		ctx:         ctx,
		bytePayload: payload,
	}
}
