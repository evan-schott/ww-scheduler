// Package ctxutil contains utilities for working with context.Context.
package ctxutil

import (
	"context"
	"time"
)

// FromChan turns a <-chan struct{} into a context.
func FromChan(c <-chan struct{}) context.Context { return C(c) }

type C <-chan struct{}

func (c C) Done() <-chan struct{}                 { return c }
func (C) Deadline() (deadline time.Time, ok bool) { return }
func (c C) Value(interface{}) interface{}         { return nil }

func (c C) Err() error {
	select {
	case <-c.Done():
		return context.Canceled
	default:
		return nil
	}
}
