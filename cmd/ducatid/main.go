package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"lib/db"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/cni"
	"github.com/cloudfoundry-incubator/ducati-daemon/config"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/ip"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/links"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/neigh"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/subscriber"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
	"github.com/cloudfoundry-incubator/ducati-daemon/sandbox"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/cloudfoundry-incubator/ducati-daemon/watcher"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
	"github.com/tedsuo/rata"
)

func main() {
	var configFilePath string
	const configFileFlag = "configFile"
	flag.StringVar(&configFilePath, configFileFlag, "", "")
	flag.Parse()

	conf, err := config.ParseConfigFile(configFilePath)
	if err != nil {
		log.Fatalf("parsing config: %s", err)
	}

	subnet := conf.LocalSubnet
	overlay := conf.OverlayNetwork

	if !overlay.Contains(subnet.IP) {
		log.Fatalf("overlay network does not contain local subnet")
	}

	retriableConnector := db.RetriableConnector{
		Connector:     db.GetConnectionPool,
		Sleeper:       db.SleeperFunc(time.Sleep),
		RetryInterval: 3 * time.Second,
		MaxRetries:    10,
	}

	databaseURL := conf.DatabaseURL
	dbConnectionPool, err := retriableConnector.GetConnectionPool(databaseURL)
	if err != nil {
		log.Fatalf("db connect: %s", err)
	}

	dataStore, err := store.New(dbConnectionPool)
	if err != nil {
		log.Fatalf("failed to construct datastore: %s", err)
	}

	logger := lager.NewLogger("ducati-d")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	configFactory := &ipam.ConfigFactory{
		Config: types.IPConfig{
			IP: *subnet,
		},
	}

	ipAllocator := ipam.New(
		&ipam.StoreFactory{},
		&sync.Mutex{},
		configFactory,
		&sync.Mutex{},
	)

	rataHandlers := rata.Handlers{}

	addressManager := &ip.AddressManager{Netlinker: nl.Netlink}
	routeManager := &ip.RouteManager{Netlinker: nl.Netlink}
	linkFactory := &links.Factory{Netlinker: nl.Netlink}
	sandboxNamespaceRepo, err := namespace.NewRepository(conf.SandboxRepoDir)
	if err != nil {
		log.Fatalf("unable to make repo: %s", err) // not tested
	}

	sandboxRepo := sandbox.NewRepository(
		logger,
		&sync.Mutex{},
		sandboxNamespaceRepo,
		sandbox.InvokeFunc(ifrit.Invoke),
		linkFactory,
	)

	osThreadLocker := &ossupport.OSLocker{}

	namespaceOpener := &namespace.PathOpener{}

	subscriber := &subscriber.Subscriber{
		Logger:    logger.Session("subscriber"),
		Netlinker: nl.Netlink,
	}
	resolver := &watcher.Resolver{
		Logger: logger,
		Store:  dataStore,
	}
	arpInserter := &neigh.ARPInserter{
		Logger:         logger,
		Netlinker:      nl.Netlink,
		OSThreadLocker: osThreadLocker,
	}
	missWatcher := watcher.New(
		subscriber,
		&sync.Mutex{},
		resolver,
		arpInserter,
	)
	networkMapper := &ipam.FixedNetworkMapper{}

	hostNamespace, err := namespaceOpener.OpenPath("/proc/self/ns/net")
	if err != nil {
		log.Fatalf("unable to open host namespace: %s", err) // not tested
	}
	commandBuilder := &container.CommandBuilder{
		MissWatcher:   missWatcher,
		HostNamespace: hostNamespace,
	}
	dnsFactory := &executor.DNSFactory{
		Logger:         logger,
		ExternalServer: fmt.Sprintf("%s:%d", conf.ExternalDNSServer, 53),
	}
	executor := executor.New(
		logger,
		addressManager,
		routeManager,
		linkFactory,
		sandboxNamespaceRepo,
		sandboxRepo,
		executor.ListenUDPFunc(net.ListenUDP),
		dnsFactory,
	)
	creator := &container.Creator{
		Executor:        executor,
		SandboxRepo:     sandboxRepo,
		Watcher:         missWatcher,
		CommandBuilder:  commandBuilder,
		DNSAddress:      fmt.Sprintf("%s:%d", conf.OverlayDNSAddress, 53),
		HostIP:          conf.HostAddress,
		NamespaceOpener: namespaceOpener,
	}
	deletor := &container.Deletor{
		Executor:        executor,
		Watcher:         missWatcher,
		NamespaceOpener: namespaceOpener,
	}

	addController := &cni.AddController{
		IPAllocator:    ipAllocator,
		NetworkMapper:  networkMapper,
		Creator:        creator,
		Datastore:      dataStore,
		OSThreadLocker: osThreadLocker,
	}

	delController := &cni.DelController{
		Datastore:      dataStore,
		Deletor:        deletor,
		IPAllocator:    ipAllocator,
		NetworkMapper:  networkMapper,
		OSThreadLocker: osThreadLocker,
	}

	marshaler := marshal.MarshalFunc(json.Marshal)
	unmarshaler := marshal.UnmarshalFunc(json.Unmarshal)

	rataHandlers["get_container"] = &handlers.GetContainer{
		Marshaler: marshaler,
		Logger:    logger,
		Datastore: dataStore,
	}

	rataHandlers["networks_list_containers"] = &handlers.NetworksListContainers{
		Marshaler: marshaler,
		Logger:    logger,
		Datastore: dataStore,
	}

	rataHandlers["list_containers"] = &handlers.ListContainers{
		Marshaler: marshaler,
		Logger:    logger,
		Datastore: dataStore,
	}

	rataHandlers["cni_add"] = &handlers.CNIAdd{
		Logger:      logger,
		Marshaler:   marshaler,
		Unmarshaler: unmarshaler,
		Controller:  addController,
	}

	rataHandlers["cni_del"] = &handlers.CNIDel{
		Logger:      logger,
		Marshaler:   marshaler,
		Unmarshaler: unmarshaler,
		Controller:  delController,
	}

	routes := rata.Routes{
		{Name: "get_container", Method: "GET", Path: "/containers/:container_id"},
		{Name: "networks_list_containers", Method: "GET", Path: "/networks/:network_id"},
		{Name: "list_containers", Method: "GET", Path: "/containers"},
		{Name: "cni_add", Method: "POST", Path: "/cni/add"},
		{Name: "cni_del", Method: "POST", Path: "/cni/del"},
	}

	rataRouter, err := rata.NewRouter(routes, rataHandlers)
	if err != nil {
		log.Fatalf("unable to create rata Router: %s", err) // not tested
	}

	httpServer := http_server.New(conf.ListenAddress, rataRouter)

	members := grouper.Members{
		{"http_server", httpServer},
	}

	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))

	err = <-monitor.Wait()
	if err != nil {
		log.Fatalf("daemon terminated: %s", err)
	}
}
