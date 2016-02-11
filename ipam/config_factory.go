package ipam

import "github.com/appc/cni/pkg/types"

type ConfigFactory struct {
	Config types.IPConfig
}

func (cf *ConfigFactory) Create(networkID string) (types.IPConfig, error) {
	return cf.Config, nil
}
