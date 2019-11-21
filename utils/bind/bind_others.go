//go:build !linux

package bind

import (
	"errors"
	"net"
)

func NewTCPDialerFromInterface(iFace string) (d *net.Dialer, err error) {
	return nil, errors.New("not linux")
}
