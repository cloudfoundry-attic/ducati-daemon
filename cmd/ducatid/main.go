package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"sync"

	"github.com/appc/cni/pkg/types"
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

const addressFlag = "listenAddr"
const overlayNetworkFlag = "overlayNetwork"
const localSubnetFlag = "localSubnet"

func parseFlags() {
	flag.StringVar(&address, addressFlag, "", "")
	flag.StringVar(&overlayNetwork, overlayNetworkFlag, "", "")
	flag.StringVar(&localSubnet, localSubnetFlag, "", "")

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

	routes := rata.Routes{
		{Name: "list_containers", Method: "GET", Path: "/containers"},
		{Name: "get_container", Method: "GET", Path: "/containers/:container_id"},
		{Name: "add_container", Method: "POST", Path: "/containers"},
		{Name: "delete_container", Method: "DELETE", Path: "/containers/:container_id"},
		{Name: "allocate_ip", Method: "POST", Path: "/ipam/:network_id/:container_id"},
		{Name: "release_ip", Method: "DELETE", Path: "/ipam/:network_id/:container_id"},
	}

	dataStore := store.New()

	logger := lager.NewLogger("ducati-d")

	rataHandlers := rata.Handlers{}

	listHandler := &handlers.ListHandler{
		Store:     dataStore,
		Marshaler: marshal.MarshalFunc(json.Marshal),
		Logger:    logger,
	}
	rataHandlers["list_containers"] = listHandler

	postHandler := &handlers.PostHandler{
		Store:       dataStore,
		Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
		Logger:      logger,
	}
	rataHandlers["add_container"] = postHandler

	getHandler := &handlers.GetHandler{
		Store:     dataStore,
		Marshaler: marshal.MarshalFunc(json.Marshal),
		Logger:    logger,
	}
	rataHandlers["get_container"] = getHandler

	deleteHandler := &handlers.DeleteHandler{
		Store:  dataStore,
		Logger: logger,
	}
	rataHandlers["delete_container"] = deleteHandler

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

	allocateIPHandler := &handlers.AllocateIPHandler{
		IPAllocator: ipAllocator,
		Marshaler:   marshal.MarshalFunc(json.Marshal),
		Logger:      logger,
	}
	rataHandlers["allocate_ip"] = allocateIPHandler

	releaseIPHandler := &handlers.ReleaseIPHandler{
		IPAllocator: ipAllocator,
		Marshaler:   marshal.MarshalFunc(json.Marshal),
		Logger:      logger,
	}
	rataHandlers["release_ip"] = releaseIPHandler

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
