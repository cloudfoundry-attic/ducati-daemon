package commands

import "net"

//go:generate counterfeiter --fake-name HardwareAddresser . HardwareAddresser
type HardwareAddresser interface {
	HardwareAddress(linkName string) (net.HardwareAddr, error)
}

type GetHardwareAddress struct {
	LinkName string
	Result   net.HardwareAddr
}

func (cmd *GetHardwareAddress) Execute(context Context) error {
	hwAddr, err := context.HardwareAddresser().HardwareAddress(cmd.LinkName)
	cmd.Result = hwAddr
	return err
}
