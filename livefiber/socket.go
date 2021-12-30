package livefiber

import "github.com/jfyne/live"

type FiberSocket struct {
	*live.BaseSocket
}

func NewSocket(s live.Session, h live.Handler, connected bool) *FiberSocket {
	return &FiberSocket{
		BaseSocket: live.NewBaseSocket(s, h, connected),
	}
}
