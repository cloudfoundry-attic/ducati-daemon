package ipam

type ConfigFactory struct {
	Config Config
}

func (cf *ConfigFactory) Create(networkID string) (Config, error) {
	return cf.Config, nil
}
