package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var address string

func parseFlags() {
	flag.StringVar(&address, "address", "", "")

	flag.Parse()

	if address == "" {
		log.Fatalf("missing require flag address")
	}
}

func main() {
	parseFlags()

	fmt.Println("will listen on " + address)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("[]"))
	})

	members := grouper.Members{
		{"http_server", http_server.New(address, handler)},
	}

	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))

	err := <-monitor.Wait()
	if err != nil {
		panic(err)
	}
}
