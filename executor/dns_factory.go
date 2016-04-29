package executor

import (
	"io"
	"net"

	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-dns/resolver"
	"github.com/cloudfoundry-incubator/ducati-dns/runner"
	"github.com/miekg/dns"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
)

//go:generate counterfeiter -o ../fakes/writer_decorator_factory.go --fake-name WriterDecoratorFactory . writerDecoratorFactory
type writerDecoratorFactory interface {
	Decorate(namespace.Namespace) dns.DecorateWriter
}

type WriterDecoratorFactoryFunc func(ns namespace.Namespace) dns.DecorateWriter

func (wdf WriterDecoratorFactoryFunc) Decorate(ns namespace.Namespace) dns.DecorateWriter {
	return wdf(ns)
}

func NamespaceDecoratorFactory(ns namespace.Namespace) dns.DecorateWriter {
	return func(w dns.Writer) dns.Writer {
		return &NamespaceWriter{Namespace: ns, Writer: w}
	}
}

type DNSFactory struct {
	Logger           lager.Logger
	ExternalServer   string
	DucatiAPI        string
	Suffix           string
	DecoratorFactory writerDecoratorFactory
}

//go:generate counterfeiter -o ../fakes/writer.go --fake-name Writer . writer
type writer interface {
	io.Writer
}

func (f *DNSFactory) New(listener net.PacketConn, sandboxNS namespace.Namespace) ifrit.Runner {
	resolverConfig := resolver.Config{
		DucatiSuffix: f.Suffix,
		DucatiAPI:    f.DucatiAPI,
	}

	return runner.New(f.Logger, resolverConfig, f.ExternalServer, listener, f.DecoratorFactory.Decorate(sandboxNS))
}
