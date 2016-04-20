package commands

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
)

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

	ns := sbox.Namespace()

	var conn *net.UDPConn
	err = ns.Execute(func(*os.File) error {
		var err error
		conn, err = listenerFactory.ListenUDP("udp", listenAddress)
		return err
	})
	if err != nil {
		return fmt.Errorf("listen udp: %s", err)
	}

	dnsServerRunner := context.DNSServerFactory().New(conn)

	err = sbox.LaunchDNS(dnsServerRunner)
	if err != nil {
		return fmt.Errorf("sandbox launch dns: %s", err)
	}

	return nil
}

func (sd StartDNSServer) String() string {
	return fmt.Sprintf("start dns server in sandbox %s", sd.SandboxName)
}
