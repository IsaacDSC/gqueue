package asyncadapter

import (
	"context"
	"encoding/json"
	"fmt"
)

type AdapterType string

type AsyncCtx[T any] struct {
	ctx         context.Context
	payload     T
	bytePayload []byte
}

func (c AsyncCtx[T]) Bytes() []byte {
	return c.bytePayload
}

func (c AsyncCtx[T]) Payload() (T, error) {
	var empty T

	if err := json.Unmarshal(c.bytePayload, &c.payload); err != nil {
		return empty, fmt.Errorf("unmarshal payload: %w", err)
	}

	return c.payload, nil
}

func (c AsyncCtx[T]) Context() context.Context {
	return c.ctx
}

type Handle[T any] struct {
	Event   string
	Handler func(c AsyncCtx[T]) error
}
