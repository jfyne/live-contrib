package livefiber

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/jfyne/live"
)

const sessionKey = "_ls"

func getSession(store *session.Store, c *fiber.Ctx) (live.Session, error) {
	s, err := store.Get(c)
	if err != nil {
		return nil, fmt.Errorf("could not get session: %w", err)
	}
	session, ok := s.Get(sessionKey).(live.Session)
	if !ok {
		return live.NewSession(), nil
	}
	return session, nil
}

func saveSession(store *session.Store, c *fiber.Ctx, session live.Session) error {
	s, err := store.Get(c)
	if err != nil {
		return fmt.Errorf("could not get session: %w", err)
	}
	s.Set(sessionKey, session)
	return nil
}

func clearSession(store *session.Store, c *fiber.Ctx) error {
	s, err := store.Get(c)
	if err != nil {
		return fmt.Errorf("could not get session: %w", err)
	}
	return s.Destroy()
}
