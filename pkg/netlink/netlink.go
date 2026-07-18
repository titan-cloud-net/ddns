package netlink

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"syscall"
	"unsafe"

	"go.uber.org/fx"
	"golang.org/x/sys/unix"
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

// New creates a Watcher backed by a netlink socket. shutdowner is used to
// terminate the application if the underlying socket fails after startup.
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

func (l *link) run(ctx context.Context) (err error) {
	defer close(l.update)

	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW, unix.NETLINK_ROUTE)
	if err != nil {
		return fmt.Errorf("failed to open netlink socket: %w", err)
	}
	socket := os.NewFile(uintptr(fd), "socket")
	defer socket.Close()

	// This ensures the kernel pushes link up/down/change events to this socket.
	lsa := &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Groups: unix.RTMGRP_IPV4_IFADDR | unix.RTMGRP_IPV6_IFADDR | unix.RTMGRP_LINK,
	}
	if err = unix.Bind(fd, lsa); err != nil {
		return fmt.Errorf("Failed to bind netlink socket: %w", err)
	}

	slog.Info("listening for interface events")

	wb := make([]byte, syscall.NLMSG_HDRLEN+syscall.SizeofIfAddrmsg)

	hdr := (*syscall.NlMsghdr)(unsafe.Pointer(&wb[0]))
	hdr.Len = uint32(len(wb))
	hdr.Type = syscall.RTM_GETADDR                         // Request address configurations
	hdr.Flags = syscall.NLM_F_REQUEST | syscall.NLM_F_DUMP // Request a full table dump
	hdr.Seq = 1
	hdr.Pid = 0

	ifmReq := (*syscall.IfAddrmsg)(unsafe.Pointer(&wb[syscall.NLMSG_HDRLEN]))
	ifmReq.Family = syscall.AF_UNSPEC

	// send the dump request to the kernel
	err = syscall.Sendto(fd, wb, 0, &syscall.SockaddrNetlink{Family: syscall.AF_NETLINK})
	if err != nil {
		return fmt.Errorf("interfaces request failed: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			buf := make([]byte, os.Getpagesize())
			n, err := socket.Read(buf)
			if err != nil {
				return fmt.Errorf("receive failed: %w", err)
			}

			msgs, err := syscall.ParseNetlinkMessage(buf[:n])
			if err != nil {
				return fmt.Errorf("failed to parse netlink message: %w", err)
			}

			for _, msg := range msgs {
				if msg.Header.Type == syscall.NLMSG_ERROR {
					nlErr := (*syscall.NlMsgerr)(unsafe.Pointer(&msg.Data[0]))
					slog.Error("netlink request failed", "errno", syscall.Errno(-nlErr.Error))
					continue
				}

				if msg.Header.Type != syscall.RTM_NEWADDR {
					continue
				}

				attrs, err := syscall.ParseNetlinkRouteAttr(&msg)
				if err != nil {
					slog.Error("failed to parse route attributes", "error", err)
					continue
				}

				var ip net.IP
				for _, attr := range attrs {
					switch attr.Attr.Type {
					case syscall.IFA_LOCAL:
						ip = net.IP(attr.Value)
					case syscall.IFA_ADDRESS:
						if ip == nil {
							ip = net.IP(attr.Value)
						}
					}
				}

				if ip == nil {
					continue
				}

				select {
				case l.update <- ip:
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}
