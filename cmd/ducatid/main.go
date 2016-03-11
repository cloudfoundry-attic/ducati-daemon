package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/container"
	"github.com/cloudfoundry-incubator/ducati-daemon/db"
	"github.com/cloudfoundry-incubator/ducati-daemon/executor"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/ip"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/links"
	"github.com/cloudfoundry-incubator/ducati-daemon/lib/namespace"
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

var address string
var overlayNetwork string
var localSubnet string
var databaseURL string
var sandboxRepoDir string

const addressFlag = "listenAddr"
const overlayNetworkFlag = "overlayNetwork"
const localSubnetFlag = "localSubnet"
const databaseURLFlag = "databaseURL"
const sandboxRepoDirFlag = "sandboxRepoDir"

func parseFlags() {
	flag.StringVar(&address, addressFlag, "", "")
	flag.StringVar(&overlayNetwork, overlayNetworkFlag, "", "")
	flag.StringVar(&localSubnet, localSubnetFlag, "", "")
	flag.StringVar(&databaseURL, databaseURLFlag, "", "")
	flag.StringVar(&sandboxRepoDir, sandboxRepoDirFlag, "", "")

	flag.Parse()

	if address == "" {
		log.Fatalf("missing required flag %q", addressFlag)
	}

	if overlayNetwork == "" {
		log.Fatalf("missing required flag %q", overlayNetworkFlag)
	}

	if localSubnet == "" {
		log.Fatalf("missing required flag %q", localSubnetFlag)
	}

	if databaseURL == "" {
		log.Fatalf("missing required flag %q", databaseURLFlag)
	}

	if sandboxRepoDir == "" {
		log.Fatalf("missing required flag %q", sandboxRepoDirFlag)
	}
}

func main() {
	parseFlags()

	_, subnet, err := net.ParseCIDR(localSubnet)
	if err != nil {
		log.Fatalf("invalid CIDR provided for %q: %s", localSubnetFlag, localSubnet)
	}

	_, overlay, err := net.ParseCIDR(overlayNetwork)
	if err != nil {
		log.Fatalf("invalid CIDR provided for %q: %s", overlayNetworkFlag, overlayNetwork)
	}

	if !overlay.Contains(subnet.IP) {
		log.Fatalf("overlay network does not contain local subnet")
	}

	retriableConnector := db.RetriableConnector{
		Connector:     db.GetConnectionPool,
		Sleeper:       db.SleeperFunc(time.Sleep),
		RetryInterval: 3 * time.Second,
		MaxRetries:    10,
	}

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
			Routes: []types.Route{{
				Dst: *overlay,
			}},
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
	sandboxRepo, err := namespace.NewRepository(sandboxRepoDir)
	if err != nil {
		log.Fatalf("unable to make repo: %s", err) // not tested
	}

	osThreadLocker := &ossupport.OSLocker{}
	namedMutex := &locks.NamedMutex{}

	subscriber := &subscriber.Subscriber{
		Logger:    logger.Session("subscriber"),
		Netlinker: nl.Netlink,
	}
	missWatcher := watcher.New(logger, subscriber, &sync.Mutex{})

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
	}
	deletor := &container.Deletor{
		Executor:    executor,
		NamedLocker: namedMutex,
		Watcher:     missWatcher,
	}

	marshaler := marshal.MarshalFunc(json.Marshal)
	unmarshaler := marshal.UnmarshalFunc(json.Unmarshal)
	rataHandlers["containers_list"] = &handlers.ContainersList{
		Store:     dataStore,
		Marshaler: marshaler,
		Logger:    logger,
	}

	rataHandlers["container_create"] = &handlers.ContainerCreate{
		Store:       dataStore,
		Unmarshaler: unmarshaler,
		Logger:      logger,
	}

	rataHandlers["container_get"] = &handlers.ContainerGet{
		Store:     dataStore,
		Marshaler: marshaler,
		Logger:    logger,
	}

	rataHandlers["container_delete"] = &handlers.ContainerDelete{
		Store:  dataStore,
		Logger: logger,
	}

	rataHandlers["networks_list_containers"] = &handlers.NetworksListContainers{
		Marshaler: marshaler,
		Logger:    logger,
		Datastore: dataStore,
	}

	rataHandlers["networks_setup_container"] = &handlers.NetworksSetupContainer{
		Unmarshaler:    unmarshaler,
		Logger:         logger,
		Datastore:      dataStore,
		Creator:        creator,
		OSThreadLocker: osThreadLocker,
		IPAllocator:    ipAllocator,
		Marshaler:      marshaler,
	}

	rataHandlers["networks_delete_container"] = &handlers.NetworksDeleteContainer{
		Unmarshaler:    unmarshaler,
		Logger:         logger,
		Datastore:      dataStore,
		Deletor:        deletor,
		OSThreadLocker: osThreadLocker,
		SandboxRepo:    sandboxRepo,
	}

	routes := rata.Routes{
		{Name: "containers_list", Method: "GET", Path: "/containers"},
		{Name: "container_get", Method: "GET", Path: "/containers/:container_id"},
		{Name: "container_create", Method: "POST", Path: "/containers"},
		{Name: "container_delete", Method: "DELETE", Path: "/containers/:container_id"},
		{Name: "networks_list_containers", Method: "GET", Path: "/networks/:network_id"},
		{Name: "networks_setup_container", Method: "POST", Path: "/networks/:network_id/:container_id"},
		{Name: "networks_delete_container", Method: "DELETE", Path: "/networks/:network_id/:container_id"},
	}

	rataRouter, err := rata.NewRouter(routes, rataHandlers)
	if err != nil {
		log.Fatalf("unable to create rata Router: %s", err) // not tested
	}

	httpServer := http_server.New(address, rataRouter)

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
