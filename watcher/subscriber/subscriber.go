package subscriber

import (
	"fmt"
	"net"
	"syscall"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl"
	"github.com/pivotal-golang/lager"
	"github.com/vishvananda/netlink"
)

type Neigh struct {
	LinkIndex    int
	Family       int
	State        int
	Type         int
	Flags        int
	IP           net.IP
	HardwareAddr net.HardwareAddr
}

func convertNeigh(input *netlink.Neigh) *Neigh {
	return &Neigh{
		LinkIndex:    input.LinkIndex,
		Family:       input.Family,
		State:        input.State,
		Type:         input.Type,
		Flags:        input.Flags,
		IP:           input.IP,
		HardwareAddr: input.HardwareAddr,
	}
}
func (n *Neigh) String() string {
	var readableState string
	if n.State&netlink.NUD_INCOMPLETE != 0 {
		readableState = " | " + "INCOMPLETE"
	}
	if n.State&netlink.NUD_REACHABLE != 0 {
		readableState = " | " + "REACHABLE"
	}
	if n.State&netlink.NUD_STALE != 0 {
		readableState = " | " + "STALE"
	}
	if n.State&netlink.NUD_DELAY != 0 {
		readableState = " | " + "DELAY"
	}
	if n.State&netlink.NUD_PROBE != 0 {
		readableState = " | " + "PROBE"
	}
	if n.State&netlink.NUD_FAILED != 0 {
		readableState = " | " + "FAILED"
	}
	if n.State&netlink.NUD_NOARP != 0 {
		readableState = " | " + "NOARP"
	}
	if n.State&netlink.NUD_PERMANENT != 0 {
		readableState = " | " + "PERMANENT"
	}
	return fmt.Sprintf(
		"LinkIndex: %d, Family: %d, State: %s, Type: %d, Flags: %d, IP: %s, HardwareAddr: %s",
		int(n.LinkIndex),
		int(n.Family),
		readableState,
		int(n.Type),
		int(n.Flags),
		n.IP.String(),
		n.HardwareAddr.String(),
	)
}

type netlinker interface {
	Subscribe(int, ...uint) (nl.NLSocket, error)
	NeighDeserialize([]byte) (*netlink.Neigh, error)
}

type Subscriber struct {
	Netlinker netlinker
	Logger    lager.Logger
}

func (s *Subscriber) Subscribe(neighChan chan<- *Neigh, doneChan <-chan struct{}) error {
	sock, err := s.Netlinker.Subscribe(syscall.NETLINK_ROUTE, syscall.RTNLGRP_NEIGH)
	if err != nil {
		return fmt.Errorf("failed to acquire netlink socket: %s", err)
	}

	if doneChan != nil {
		go func() {
			<-doneChan
			sock.Close()
		}()
	}

	go func() {
		defer close(neighChan)
		for {
			msgs, err := sock.Receive()
			if err != nil {
				s.Logger.Error("socket receive", err)
				return
			}

			for _, m := range msgs {
				n, err := s.Netlinker.NeighDeserialize(m.Data)
				if err != nil {
					s.Logger.Error("neighbor deserialize", err)
					return
				}
				neigh := convertNeigh(n)
				neighChan <- neigh
			}
		}
	}()

	return nil
}
