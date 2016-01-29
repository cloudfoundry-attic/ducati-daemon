package handlers

import "github.com/pivotal-golang/lager"

//go:generate counterfeiter -o ../fakes/logger.go --fake-name Logger . Logger
type Logger interface {
	Error(action string, err error, data ...lager.Data)
}
