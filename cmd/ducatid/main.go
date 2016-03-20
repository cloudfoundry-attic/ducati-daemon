package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"sync"
	"time"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/config"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/db"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/ip"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/links"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/neigh"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/nl"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/subscriber"
	"github.com/cloudfoundry-incubator/ducati-daemon/locks"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/ossupport"
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
	sandboxRepo, err := namespace.NewRepository(conf.SandboxRepoDir)
	if err != nil {
		log.Fatalf("unable to make repo: %s", err) // not tested
	}

	osThreadLocker := &ossupport.OSLocker{}
	namedMutex := &locks.NamedMutex{}

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
	networkMapper := &ipam.FixedNetworkMapper{VNI: conf.VNI}

	commandBuilder := &container.CommandBuilder{
		SandboxRepo:   sandboxRepo,
		MissWatcher:   missWatcher,
		HostNamespace: namespace.NewNamespace("/proc/self/ns/net"),
	}
	executor := executor.New(addressManager, routeManager, linkFactory, sandboxRepo)
	creator := &container.Creator{
		Executor:       executor,
		SandboxRepo:    sandboxRepo,
		NamedLocker:    namedMutex,
		Watcher:        missWatcher,
		CommandBuilder: commandBuilder,
		HostIP:         conf.HostAddress,
	}
	deletor := &container.Deletor{
		Executor:    executor,
		NamedLocker: namedMutex,
		Watcher:     missWatcher,
	}

	marshaler := marshal.MarshalFunc(json.Marshal)
	unmarshaler := marshal.UnmarshalFunc(json.Unmarshal)

	rataHandlers["networks_list_containers"] = &handlers.NetworksListContainers{
		Marshaler: marshaler,
		Logger:    logger,
		Datastore: dataStore,
	}

	rataHandlers["cni_add"] = &handlers.CNIAdd{
		Unmarshaler:    unmarshaler,
		Logger:         logger,
		Datastore:      dataStore,
		Creator:        creator,
		OSThreadLocker: osThreadLocker,
		IPAllocator:    ipAllocator,
		Marshaler:      marshaler,
		NetworkMapper:  networkMapper,
	}

	rataHandlers["cni_del"] = &handlers.CNIDel{
		Unmarshaler:    unmarshaler,
		Logger:         logger,
		Datastore:      dataStore,
		Deletor:        deletor,
		OSThreadLocker: osThreadLocker,
		SandboxRepo:    sandboxRepo,
		NetworkMapper:  networkMapper,
	}

	routes := rata.Routes{
		{Name: "networks_list_containers", Method: "GET", Path: "/networks/:network_id"},
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
		panic(err)
	}
}
