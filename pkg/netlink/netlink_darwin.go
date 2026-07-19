//go:build darwin

package netlink

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

// run watches for interface address changes via a PF_ROUTE socket, forwarding
// each newly assigned address (RTM_NEWADDR) on l.update. Address removals
// and link up/down events are intentionally not forwarded.
func (l *link) run(ctx context.Context) (err error) {
	defer close(l.update)

	fd, err := unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, unix.AF_UNSPEC)
	if err != nil {
		return fmt.Errorf("failed to open route socket: %w", err)
	}
	socket := os.NewFile(uintptr(fd), "socket")
	defer socket.Close()

	slog.Info("listening for interface events")

	// Dump the addresses already assigned to interfaces at startup. On BSD
	// systems a NET_RT_IFLIST fetch returns RTM_IFINFO messages interleaved
	// with RTM_NEWADDR messages for each interface's addresses.
	rib, err := route.FetchRIB(unix.AF_UNSPEC, route.RIBTypeInterface, 0)
	if err != nil {
		return fmt.Errorf("interfaces request failed: %w", err)
	}
	msgs, err := route.ParseRIB(route.RIBTypeInterface, rib)
	if err != nil {
		return fmt.Errorf("failed to parse initial interface dump: %w", err)
	}
	if err = l.emitAddrs(ctx, msgs); err != nil {
		return err
	}

	buf := make([]byte, os.Getpagesize())
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		n, err := socket.Read(buf)
		if err != nil {
			return fmt.Errorf("receive failed: %w", err)
		}

		msgs, err := route.ParseRIB(route.RIBTypeRoute, buf[:n])
		if err != nil {
			slog.Error("failed to parse route message", "error", err)
			continue
		}

		if err = l.emitAddrs(ctx, msgs); err != nil {
			return err
		}
	}
}

// emitAddrs forwards the address of every RTM_NEWADDR message in msgs on
// l.update, skipping any other message type (RTM_DELADDR, RTM_IFINFO, ...).
func (l *link) emitAddrs(ctx context.Context, msgs []route.Message) error {
	for _, msg := range msgs {
		am, ok := msg.(*route.InterfaceAddrMessage)
		if !ok || am.Type != unix.RTM_NEWADDR {
			continue
		}

		var ip net.IP
		for _, addr := range am.Addrs {
			switch a := addr.(type) {
			case *route.Inet4Addr:
				ip = net.IP(a.IP[:])
			case *route.Inet6Addr:
				ip = net.IP(a.IP[:])
			}
			if ip != nil {
				break
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
	return nil
}
