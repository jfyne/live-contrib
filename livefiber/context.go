package livefiber

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type contextKey string

const (
	ctxKey  contextKey = "context_ctx"
	connKey contextKey = "context_conn"
)

func contextWithCtx(ctx context.Context, c *fiber.Ctx) context.Context {
	return context.WithValue(ctx, ctxKey, c)
}

func contextWithConn(ctx context.Context, conn *websocket.Conn) context.Context {
	return context.WithValue(ctx, connKey, conn)
}

// Ctx gets the initialising *fiber.Ctx from the live context. nil if
// not there or the websocket connection has already been established.
func Ctx(ctx context.Context) *fiber.Ctx {
	data := ctx.Value(ctxKey)
	c, ok := data.(*fiber.Ctx)
	if !ok {
		return nil
	}
	return c
}

// Conn gets the initialising *werbsocket.Conn from the live context. nil
// if not there or the websocket connection hasn't been established.
func Conn(ctx context.Context) *websocket.Conn {
	data := ctx.Value(connKey)
	c, ok := data.(*websocket.Conn)
	if !ok {
		return nil
	}
	return c
}
