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
	Logger    lager.Logger
	Netlinker netlinker
}

func (s *Subscriber) Subscribe(neighChan chan<- *watcher.Neigh, doneChan <-chan struct{}) error {
	logger := s.Logger.Session("subscribe")
	logger.Info("called")
	defer logger.Info("complete")

	sock, err := s.Netlinker.Subscribe(syscall.NETLINK_ROUTE, syscall.RTNLGRP_NEIGH)
	if err != nil {
		logger.Error("netlink-subscribe-failed", err)
		return fmt.Errorf("failed to acquire netlink socket: %s", err)
	}

	go func() {
		<-doneChan
		logger.Info("closing-netlink-socket")
		sock.Close()
		logger.Info("closed-netlink-socket")
	}()

	go func() {
		defer func() {
			logger.Info("closing-neigh-chan")
			close(neighChan)
			logger.Info("closed-neigh-chan")
		}()

		for {
			msgs, err := sock.Receive()
			logger.Info("receive-message-count", lager.Data{"message-count": len(msgs)})
			if err != nil {
				s.Logger.Error("socket-receive", err)
				return
			}

			for _, m := range msgs {
				n, err := s.Netlinker.NeighDeserialize(m.Data)
				if err != nil {
					s.Logger.Error("neighbor-deserialize", err)
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
