package executor

import (
	"net"

	"github.com/cloudfoundry-incubator/ducati-dns/resolver"
	"github.com/cloudfoundry-incubator/ducati-dns/runner"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

type DNSFactory struct {
	Logger         lager.Logger
	ExternalServer string
}

func (f *DNSFactory) New(listener net.PacketConn) ifrit.Runner {
	return runner.New( f.Logger, resolver.Config{}, f.ExternalServer, listener)
}
