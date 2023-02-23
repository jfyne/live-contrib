package livefiber

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/websocket/v2"
	"github.com/jfyne/live"
	"golang.org/x/net/html"
)

var _ live.Engine = &FiberEngine{}

type FiberEngine struct {
	sessionStore *session.Store
	*live.BaseEngine
}

func NewHandler(store *session.Store, h live.Handler) *FiberEngine {
	return &FiberEngine{
		sessionStore: store,
		BaseEngine:   live.NewBaseEngine(h),
	}
}

func (e *FiberEngine) Handlers() []fiber.Handler {
	return []fiber.Handler{e.http, websocket.New(e.ws)}
}

func (e *FiberEngine) http(c *fiber.Ctx) error {
	ctx := contextWithCtx(c.Context(), c)

	// Get Session.
	session, err := getSession(e.sessionStore, c)
	if err != nil {
		e.Error()(ctx, err)
		return err
	}

	upgrade := websocket.IsWebSocketUpgrade(c)
	if upgrade {
		c.Locals("session", session)
		c.Locals("params", NewParamsFromRequest(c))
		c.Locals("views", c.App().Config().Views)
		return c.Next()
	}

	c.Set("Content-Type", "text/html")

	// Get Socket.
	sock := NewSocket(session, e, false)

	// Run mount.
	data, err := e.Mount()(ctx, sock)
	if err != nil {
		e.Error()(ctx, err)
		return err
	}
	sock.Assign(data)

	// Handle any query parameters that are on the page.
	for _, ph := range e.Params() {
		data, err := ph(ctx, sock, NewParamsFromRequest(c))
		if err != nil {
			e.Error()(ctx, err)
			return err
		}
		sock.Assign(data)
	}

	// Render the HTML to display the page.
	render, err := live.RenderSocket(ctx, e, sock)
	if err != nil {
		e.Error()(ctx, err)
		return err
	}
	sock.UpdateRender(render)

	var rendered bytes.Buffer
	html.Render(&rendered, render)

	// Save the session.
	if err := saveSession(e.sessionStore, c, session); err != nil {
		e.Error()(ctx, err)
		return err
	}

	// Output the html.
	if _, err := c.Write(rendered.Bytes()); err != nil {
		e.Error()(ctx, err)
		return err
	}

	return nil
}

func (e *FiberEngine) ws(c *websocket.Conn) {
	err := e._ws(c)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		log.Println(fmt.Errorf("ws closed with status: %w", err))
	}
}

func (e *FiberEngine) _ws(c *websocket.Conn) error {
	// Create live context.
	ctx := contextWithConn(context.Background(), c)

	session, ok := c.Locals("session").(live.Session)
	if !ok {
		return fmt.Errorf("could not get session from locals")
	}

	// Get socket and register with server.
	sock := NewSocket(session, e, true)
	e.AddSocket(sock)
	defer e.DeleteSocket(sock)

	// Internal errors.
	internalErrors := make(chan error)

	// Event errors.
	eventErrors := make(chan live.ErrorEvent)

	// Handle events coming from the websocket connection.
	var (
		t   int
		d   []byte
		err error
	)
	go func() {
		for {
			if t, d, err = c.ReadMessage(); err != nil {
				internalErrors <- err
				break
			}
			switch t {
			case websocket.TextMessage:
				var m live.Event
				if err := json.Unmarshal(d, &m); err != nil {
					internalErrors <- err
					goto stoploop
				}
				switch m.T {
				case live.EventParams:
					if err := e.CallParams(ctx, sock, m); err != nil {
						switch {
						case errors.Is(err, live.ErrNoEventHandler):
							log.Println("event error", m, err)
						default:
							eventErrors <- live.ErrorEvent{Source: m, Err: err.Error()}
						}
					}
				default:
					if err := e.CallEvent(ctx, m.T, sock, m); err != nil {
						switch {
						case errors.Is(err, live.ErrNoEventHandler):
							log.Println("event error", m, err)
						default:
							eventErrors <- live.ErrorEvent{Source: m, Err: err.Error()}
						}
					}
				}
				render, err := live.RenderSocket(ctx, e, sock)
				if err != nil {
					internalErrors <- fmt.Errorf("socket handle error: %w", err)
					goto stoploop
				} else {
					sock.UpdateRender(render)
				}
				if err := sock.Send(live.EventAck, nil, live.WithID(m.ID)); err != nil {
					internalErrors <- fmt.Errorf("socket send error: %w", err)
					goto stoploop
				}
			case websocket.BinaryMessage:
				log.Println("binary messages unhandled")

			}
		}

	stoploop:
		close(internalErrors)
		close(eventErrors)
	}()
	// Run mount again now that eh socket is connected, passing true indicating
	// a connection has been made.
	data, err := e.Mount()(ctx, sock)
	if err != nil {
		return fmt.Errorf("socket mount error: %w", err)
	}
	sock.Assign(data)

	// Run params again now that the socket is connected.
	for _, ph := range e.Params() {
		params, ok := c.Locals("params").(live.Params)
		if !ok {
			return fmt.Errorf("locals params could not be found")
		}
		data, err := ph(ctx, sock, params)
		if err != nil {
			return fmt.Errorf("socket params error: %w", err)
		}
		sock.Assign(data)
	}

	// Run render now that we are connected for the first time and we have just
	// mounted again. This will generate and send any patches if there have
	// been changes.
	render, err := live.RenderSocket(ctx, e, sock)
	if err != nil {
		return fmt.Errorf("socket render error: %w", err)
	}
	sock.UpdateRender(render)

	// Send events to the websocket connection.
	for {
		select {
		case msg := <-sock.Messages():
			c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.WriteJSON(msg); err != nil {
				return fmt.Errorf("writing to socket error: %w", err)
			}
		case ee := <-eventErrors:
			c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			d, err := json.Marshal(ee)
			if err != nil {
				return fmt.Errorf("writing to socket error: %w", err)
			}
			if err := c.WriteJSON(live.Event{T: live.EventError, Data: d}); err != nil {
				return fmt.Errorf("writing to socket error: %w", err)
			}
		case err := <-internalErrors:
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return err
			}

			c.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err != nil {
				d, err := json.Marshal(err.Error())
				if err != nil {
					return fmt.Errorf("marshalling error writing to socket error: %w", err)
				}
				if err := c.WriteJSON(live.Event{T: live.EventError, Data: d}); err != nil {
					return fmt.Errorf("writing to socket error: %w", err)
				}
				// Something catastrophic has happened.
				return fmt.Errorf("internal error: %w", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}
