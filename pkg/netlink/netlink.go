package netlink

import (
	"context"
	"log/slog"
	"net"

	"go.uber.org/fx"
)

// Watcher reports IP addresses assigned to local network interfaces.
type Watcher interface {
	// Watch returns a channel of newly observed addresses. The channel is
	// closed once the watcher stops.
	Watch() (updates <-chan net.IP)
}

type link struct {
	update chan net.IP

	cancel     context.CancelFunc
	shutdowner fx.Shutdowner
}

// New creates a Watcher backed by a platform-specific routing socket.
// shutdowner is used to terminate the application if the underlying socket
// fails after startup.
func New(shutdowner fx.Shutdowner) (Watcher, error) {
	lnk := &link{
		update:     make(chan net.IP),
		shutdowner: shutdowner,
	}
	return lnk, nil
}

// Invoke wires a Watcher's lifecycle into fx, starting and stopping it
// alongside the rest of the application.
func Invoke(lifecycle fx.Lifecycle, watcher Watcher) {
	lnk := watcher.(*link)
	lifecycle.Append(fx.StartStopHook(lnk.start, lnk.stop))
}

func (l *link) start() {
	var ctx context.Context
	ctx, l.cancel = context.WithCancel(context.Background())
	go func() {
		err := l.run(ctx)
		if err != nil {
			slog.Error("shutdown", "error", err)
			l.shutdowner.Shutdown(fx.ExitCode(1))
		}
	}()
}

func (l *link) stop() {
	l.cancel()
}

func (l *link) Watch() (update <-chan net.IP) {
	return l.update
}
