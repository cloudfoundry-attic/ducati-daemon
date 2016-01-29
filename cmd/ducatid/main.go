package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/cloudfoundry-incubator/ducati-daemon/handlers"
	"github.com/cloudfoundry-incubator/ducati-daemon/marshal"
	"github.com/cloudfoundry-incubator/ducati-daemon/store"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
	"github.com/tedsuo/rata"
)

var address string

const addressFlag = "listenAddr"

func parseFlags() {
	flag.StringVar(&address, addressFlag, "", "")

	flag.Parse()

	if address == "" {
		log.Fatalf("missing required flag %q", addressFlag)
	}
}

func main() {
	parseFlags()

	routes := rata.Routes{
		{Name: "list_containers", Method: "GET", Path: "/containers"},
		{Name: "get_container", Method: "GET", Path: "/containers/:container_id"},
		{Name: "add_container", Method: "POST", Path: "/containers"},
		{Name: "delete_container", Method: "DELETE", Path: "/containers/:container_id"},
	}

	dataStore := store.New()

	listHandler := &handlers.ListHandler{
		Store:     dataStore,
		Marshaler: marshal.MarshalFunc(json.Marshal),
	}

	postHandler := &handlers.PostHandler{
		Store:       dataStore,
		Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
	}

	getHandler := &handlers.GetHandler{
		Store:     dataStore,
		Marshaler: marshal.MarshalFunc(json.Marshal),
	}

	deleteHandler := &handlers.DeleteHandler{
		Store: dataStore,
	}

	handlers := rata.Handlers{
		"list_containers":  listHandler,
		"add_container":    postHandler,
		"get_container":    getHandler,
		"delete_container": deleteHandler,
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
