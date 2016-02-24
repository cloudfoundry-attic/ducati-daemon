package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"sync"

	"github.com/appc/cni/pkg/types"
	"github.com/cloudfoundry-incubator/ducati-daemon/db"
	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/ipam"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
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

const addressFlag = "listenAddr"
const overlayNetworkFlag = "overlayNetwork"
const localSubnetFlag = "localSubnet"
const databaseURLFlag = "databaseURL"

func parseFlags() {
	flag.StringVar(&address, addressFlag, "", "")
	flag.StringVar(&overlayNetwork, overlayNetworkFlag, "", "")
	flag.StringVar(&localSubnet, localSubnetFlag, "", "")
	flag.StringVar(&databaseURL, databaseURLFlag, "", "")

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

	dbConnectionPool, err := db.GetConnectionPool(databaseURL)
	if err != nil {
		log.Fatalf("db connect: %s", err)
	}

	dataStore, err := store.New(dbConnectionPool)
	if err != nil {
		log.Fatalf("failed to construct datastore: %s", err)
	}

	logger := lager.NewLogger("ducati-d")

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

	// addressManager := &ip.AddressManager{Netlinker: nl.Netlink}
	// routeManager := &ip.RouteManager{Netlinker: nl.Netlink}
	// linkFactory := &links.Factory{Netlinker: nl.Netlink}

	// executor := executor.New(addressManager, routeManager, linkFactory)

	rataHandlers["containers_list"] = &handlers.ContainersList{
		Store:     dataStore,
		Marshaler: marshal.MarshalFunc(json.Marshal),
		Logger:    logger,
	}

	rataHandlers["container_create"] = &handlers.ContainerCreate{
		Store:       dataStore,
		Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
		Logger:      logger,
	}

	rataHandlers["container_get"] = &handlers.ContainerGet{
		Store:     dataStore,
		Marshaler: marshal.MarshalFunc(json.Marshal),
		Logger:    logger,
	}

	rataHandlers["container_delete"] = &handlers.ContainerDelete{
		Store:  dataStore,
		Logger: logger,
	}

	rataHandlers["ipam_allocate"] = &handlers.IPAMAllocate{
		IPAllocator: ipAllocator,
		Marshaler:   marshal.MarshalFunc(json.Marshal),
		Logger:      logger,
	}

	rataHandlers["ipam_release"] = &handlers.IPAMRelease{
		IPAllocator: ipAllocator,
		Marshaler:   marshal.MarshalFunc(json.Marshal),
		Logger:      logger,
	}

	rataHandlers["networks_list_containers"] = &handlers.NetworksListContainers{
		Marshaler: marshal.MarshalFunc(json.Marshal),
		Logger:    logger,
		Datastore: dataStore,
	}

	// rataHandlers["networks_setup_container"] = &handlers.NetworksSetupContainer{
	// 	Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
	// 	Logger:      logger,
	// 	Datastore:   dataStore,
	// 	Executor:    executor,
	// }

	routes := rata.Routes{
		{Name: "containers_list", Method: "GET", Path: "/containers"},
		{Name: "container_get", Method: "GET", Path: "/containers/:container_id"},
		{Name: "container_create", Method: "POST", Path: "/containers"},
		{Name: "container_delete", Method: "DELETE", Path: "/containers/:container_id"},
		{Name: "ipam_allocate", Method: "POST", Path: "/ipam/:network_id/:container_id"},
		{Name: "ipam_release", Method: "DELETE", Path: "/ipam/:network_id/:container_id"},
		{Name: "networks_list_containers", Method: "GET", Path: "/networks/:network_id"},
		// {Name: "networks_setup_container", Method: "POST", Path: "/networks/:network_id/:container_id"},
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
