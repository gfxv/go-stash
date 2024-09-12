package senderapp

import (
	"github.com/gfxv/go-stash/internal/sender"
	"github.com/gfxv/go-stash/internal/services"
	"time"
)

type App struct {
	sender        *sender.Client
	checkInterval time.Duration
}

func New(opts *sender.SenderOpts, dht *services.DHTService) *App {
	c := sender.NewClient(opts, dht)
	return &App{sender: c}
}

func (a *App) MustRun(notifyReady chan<- bool) {
	if err := a.Run(notifyReady); err != nil {
		panic(err)
	}
}

func (a *App) Run(notifyReady chan<- bool) error {
	if err := a.sender.Serve(notifyReady); err != nil {
		return err
	}
	return nil
}
