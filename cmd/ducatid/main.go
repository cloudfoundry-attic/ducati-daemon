package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"sync"

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
var localSubnet string

const addressFlag = "listenAddr"
const localSubnetFlag = "localSubnet"

func parseFlags() {
	flag.StringVar(&address, addressFlag, "", "")
	flag.StringVar(&localSubnet, localSubnetFlag, "", "")

	flag.Parse()

	if address == "" {
		log.Fatalf("missing required flag %q", addressFlag)
	}

	if localSubnet == "" {
		log.Fatalf("missing required flag %q", localSubnetFlag)
	}
}

func main() {
	parseFlags()

	routes := rata.Routes{
		{Name: "list_containers", Method: "GET", Path: "/containers"},
		{Name: "get_container", Method: "GET", Path: "/containers/:container_id"},
		{Name: "add_container", Method: "POST", Path: "/containers"},
		{Name: "delete_container", Method: "DELETE", Path: "/containers/:container_id"},
		{Name: "allocate_ip", Method: "POST", Path: "/ipam/:network_id/:container_id"},
	}

	dataStore := store.New()

	logger := lager.NewLogger("ducati-d")

	listHandler := &handlers.ListHandler{
		Store:     dataStore,
		Marshaler: marshal.MarshalFunc(json.Marshal),
		Logger:    logger,
	}

	postHandler := &handlers.PostHandler{
		Store:       dataStore,
		Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
		Logger:      logger,
	}

	getHandler := &handlers.GetHandler{
		Store:     dataStore,
		Marshaler: marshal.MarshalFunc(json.Marshal),
		Logger:    logger,
	}

	deleteHandler := &handlers.DeleteHandler{
		Store:  dataStore,
		Logger: logger,
	}

	_, subnet, err := net.ParseCIDR(localSubnet)
	if err != nil {
		panic(err)
	}

	configFactory := &ipam.ConfigFactory{
		Config: ipam.Config{
			Subnet: *subnet,
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

	handlers := rata.Handlers{
		"list_containers":  listHandler,
		"add_container":    postHandler,
		"get_container":    getHandler,
		"delete_container": deleteHandler,
		"allocate_ip":      allocateIPHandler,
	}

	rataHandler, err := rata.NewRouter(routes, handlers)

	httpServer := http_server.New(address, rataHandler)

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
