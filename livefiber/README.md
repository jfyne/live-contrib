# ⚡ livefiber

Real-time user experiences with server-rendered HTML in Go using Fiber. Inspired by and
borrowing from Phoenix LiveViews.

Live is intended as a replacement for React, Vue, Angular etc. You can write
an interactive web app just using Go and its templates.

![](https://github.com/jfyne/live-examples/blob/main/chat.gif)

The structures provided in this package are compatible with `github.com/gofiber/fiber/v2`.

See the main repository [here](https://github.com/jfyne/live) for more info and docs.

See the [examples](https://github.com/jfyne/live-examples) for usage.

### First handler

Here is an example demonstrating how we would make a simple thermostat.

[embedmd]:# (example_test.go)
```go
// +build example

package livefiber

import (
	"context"
	"log"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html"
	"github.com/jfyne/live"
)

// Model of our thermostat.
type ThermoModel struct {
	C float32
}

// Helper function to get the model from the socket data.
func NewThermoModel(s live.Socket) *ThermoModel {
	m, ok := s.Assigns().(*ThermoModel)
	// If we haven't already initialised set up.
	if !ok {
		m = &ThermoModel{
			C: 19.5,
		}
	}
	return m
}

// thermoMount initialises the thermostat state. Data returned in the mount function will
// automatically be assigned to the socket.
func thermoMount(ctx context.Context, s live.Socket) (interface{}, error) {
	return NewThermoModel(s), nil
}

// tempUp on the temp up event, increase the thermostat temperature by .1 C. An EventHandler function
// is called with the original request context of the socket, the socket itself containing the current
// state and and params that came from the event. Params contain query string parameters and any
// `live-value-` bindings.
func tempUp(ctx context.Context, s live.Socket, p live.Params) (interface{}, error) {
	model := NewThermoModel(s)
	model.C += 0.1
	return model, nil
}

// tempDown on the temp down event, decrease the thermostat temperature by .1 C.
func tempDown(ctx context.Context, s live.Socket, p live.Params) (interface{}, error) {
	model := NewThermoModel(s)
	model.C -= 0.1
	return model, nil
}

// Example shows a simple temperature control using the
// "live-click" event.
func Example() {

	// Setup the handler.
	h := live.NewHandler(WithViewsRenderer("view"))

	// Mount function is called on initial HTTP load and then initial web
	// socket connection. This should be used to create the initial state,
	// the socket Connected func will be true if the mount call is on a web
	// socket connection.
	h.HandleMount(thermoMount)

	// This handles the `live-click="temp-up"` button. First we load the model from
	// the socket, increment the temperature, and then return the new state of the
	// model. Live will now calculate the diff between the last time it rendered and now,
	// produce a set of diffs and push them to the browser to update.
	h.HandleEvent("temp-up", tempUp)

	// This handles the `live-click="temp-down"` button.
	h.HandleEvent("temp-down", tempDown)

	// Setup fiber.
	app := fiber.New(fiber.Config{
		Views: html.New("./views", ".html"),
	})

	app.Get("/thermostat", NewHandler(session.New(), h).Handlers()...)

	// This serves the JS needed to make live work.
	app.Get("/live.js", adaptor.HTTPHandler(live.Javascript{}))

	log.Fatal(app.Listen(":8080"))
}
```

`views/view.html`
```html
<!doctype html>
<html>
    <body>
        <div>{{.C}}</div>
        <button live-click="temp-up">+</button>
        <button live-click="temp-down">-</button>
        <!-- Include to make live work -->
        <script src="/live.js"></script>
    </body>
</html>
```
