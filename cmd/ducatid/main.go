package main

import (
	"fmt"
	"os"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {

	members := grouper.Members{
		{"http_server", helloWorldRunner{}},
	}

	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))

	err := <-monitor.Wait()
	if err != nil {
		panic(err)
	}
}

type helloWorldRunner struct{}

func (helloWorldRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	fmt.Println("hello world")

	select {
	case <-signals:
		return nil
	}
}
