package livefiber

import "github.com/jfyne/live"

type FiberSocket struct {
	*live.BaseSocket
}

func NewSocket(s live.Session, e live.Engine, connected bool) *FiberSocket {
	return &FiberSocket{
		BaseSocket: live.NewBaseSocket(s, e, connected),
	}
}
