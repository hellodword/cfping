package bind

import (
	"fmt"
	"net"
	"syscall"
)

func GetIPFromInterface(iFace string) (net.IP, error) {
	ief, err := net.InterfaceByName(iFace)
	if err != nil {
		return nil, err
	}

	addrs, err := ief.Addrs()
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		err = fmt.Errorf("Interface.Addrs %d", len(addrs))
		return nil, err
	}

	ip, ok := addrs[0].(*net.IPNet)
	if !ok {
		err = fmt.Errorf("net.IPNet convert fail")
		return nil, err
	}
	return ip.IP, nil
}

func NewDialerControlFromInterface(iFace string) func(string, string, syscall.RawConn) error {
	return func(network string, address string, c syscall.RawConn) error {
		var ctlErr error
		fn := func(fd uintptr) {
			ctlErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, iFace)
		}
		if err := c.Control(fn); err != nil {
			return err
		}
		if ctlErr != nil {
			return ctlErr
		}
		return nil
	}
}

func NewTCPDialerFromInterface(iFace string) (d *net.Dialer, err error) {
	ip, err := GetIPFromInterface(iFace)
	if err != nil {
		return
	}

	d = &net.Dialer{LocalAddr: &net.TCPAddr{IP: ip}}
	d.Control = NewDialerControlFromInterface(iFace)
	return
}
