package livefiber

import (
	"bytes"
	"context"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/jfyne/live"
)

// WithViewsRenderer set the handler to use the fiber App views renderer.
func WithViewsRenderer(name string, layouts ...string) live.HandlerConfig {
	return func(h *live.Handler) error {
		h.HandleRender(
			func(ctx context.Context, rc *live.RenderContext) (io.Reader, error) {
				v := getViews(ctx)
				var buf bytes.Buffer
				if err := v.Render(&buf, name, rc, layouts...); err != nil {
					return nil, err
				}
				return &buf, nil
			},
		)

		return nil
	}
}

func getViews(ctx context.Context) fiber.Views {
	c := Ctx(ctx)
	if c != nil {
		return c.App().Config().Views
	}
	conn := Conn(ctx)
	return conn.Locals("views").(fiber.Views)
}
