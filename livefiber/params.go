package livefiber

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jfyne/live"
)

func NewParamsFromRequest(c *fiber.Ctx) live.Params {
	out := live.Params{}
	c.Context().QueryArgs().VisitAll(func(key []byte, value []byte) {
		out[string(key)] = string(value)
	})
	return out
}
