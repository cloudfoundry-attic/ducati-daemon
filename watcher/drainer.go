package watcher

import "github.com/pivotal-golang/lager"

type Drainer struct {
	Logger   lager.Logger
	Firehose chan Miss
}

func (d *Drainer) Drain() {
	for {
		msg := <-d.Firehose
		d.Logger.Info("sandbox-miss", lager.Data{
			"sandbox": msg.SandboxName,
			"dest_ip": msg.DestIP,
		})
	}
}
