package subscriber

import (
	"fmt"
	"syscall"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	"github.com/pivotal-golang/lager"
	"github.com/vishvananda/netlink"
)

type netlinker interface {
	Subscribe(int, ...uint) (nl.NLSocket, error)
	NeighDeserialize([]byte) (*netlink.Neigh, error)
}

type Subscriber struct {
	Netlinker netlinker
	Logger    lager.Logger
}

func (s *Subscriber) Subscribe(neighChan chan<- *watcher.Neigh, doneChan <-chan struct{}) error {
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
				return
			}

			for _, m := range msgs {
				n, err := s.Netlinker.NeighDeserialize(m.Data)
				if err != nil {
					s.Logger.Error("neighbor deserialize", err)
					return
				}

				if n.IP == nil || (n.HardwareAddr != nil && n.State != netlink.NUD_STALE) {
					continue
				}

				neigh := convertNeigh(n)

				neighChan <- neigh
			}
		}
	}()

	return nil
}

func convertNeigh(input *netlink.Neigh) *watcher.Neigh {
	return &watcher.Neigh{
		LinkIndex:    input.LinkIndex,
		Family:       input.Family,
		State:        input.State,
		Type:         input.Type,
		Flags:        input.Flags,
		IP:           input.IP,
		HardwareAddr: input.HardwareAddr,
	}
}
