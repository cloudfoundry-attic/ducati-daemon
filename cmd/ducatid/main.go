package main

import (
	"flag"
	"log"
	"net/http"
	"os"

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

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("[]"))
	})

	routes := rata.Routes{
		{Name: "list_containers", Method: "GET", Path: "/containers"},
	}

	handlers := rata.Handlers{
		"list_containers": handler,
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
