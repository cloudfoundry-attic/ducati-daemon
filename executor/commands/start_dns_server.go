package commands

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

const DNS_INTERFACE_NAME = "dns0"

type StartDNSServer struct {
	ListenAddress string
	SandboxName   string
}

func (sd StartDNSServer) Execute(context executor.Context) error {
	listenerFactory := context.ListenerFactory()

	listenAddress, err := net.ResolveUDPAddr("udp", sd.ListenAddress)
	if err != nil {
		return fmt.Errorf("resolve udp address: %s", err)
	}

	sbox, err := context.SandboxRepository().Get(sd.SandboxName)
	if err != nil {
		return fmt.Errorf("get sandbox: %s", err)
	}

	sbox.Lock()
	defer sbox.Unlock()

	namespace := sbox.Namespace()

	var conn *net.UDPConn
	err = namespace.Execute(func(*os.File) error {
		linkFactory := context.LinkFactory()
		err := linkFactory.CreateDummy(DNS_INTERFACE_NAME)
		if err != nil {
			return fmt.Errorf("create dummy: %s", err)
		}

		dnsAddress := &net.IPNet{
			IP:   listenAddress.IP,
			Mask: net.CIDRMask(32, 32),
		}

		err = context.AddressManager().AddAddress(DNS_INTERFACE_NAME, dnsAddress)
		if err != nil {
			return fmt.Errorf("add address: %s", err)
		}

		err = linkFactory.SetUp(DNS_INTERFACE_NAME)
		if err != nil {
			return fmt.Errorf("set up: %s", err)
		}

		conn, err = listenerFactory.ListenUDP("udp", listenAddress)
		if err != nil {
			return fmt.Errorf("listen udp: %s", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("namespace execute: %s", err)
	}

	dnsServerRunner := context.DNSServerFactory().New(conn, namespace)

	err = sbox.LaunchDNS(dnsServerRunner)
	if err != nil {
		return fmt.Errorf("sandbox launch dns: %s", err)
	}

	return nil
}

func (sd StartDNSServer) String() string {
	return fmt.Sprintf("start dns server in sandbox %s", sd.SandboxName)
}
