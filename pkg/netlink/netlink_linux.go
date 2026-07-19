//go:build linux

package netlink

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// run watches for interface address changes via a NETLINK_ROUTE socket,
// forwarding each newly assigned address (RTM_NEWADDR) on l.update. Address
// removals and link up/down events are intentionally not forwarded.
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

			if err = l.emitAddrs(ctx, msgs); err != nil {
				return err
			}
		}
	}
}

// emitAddrs forwards the address of every RTM_NEWADDR message in msgs on
// l.update, skipping any other message type (RTM_DELADDR, link events, ...).
func (l *link) emitAddrs(ctx context.Context, msgs []syscall.NetlinkMessage) error {
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
	return nil
}
