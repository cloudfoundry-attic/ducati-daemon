package models

import "github.com/appc/cni/pkg/types"

type NetworksSetupContainerPayload struct {
	Args               string       `json:"args"`
	ContainerNamespace string       `json:"container_namespace"`
	InterfaceName      string       `json:"interface_name"`
	IPAM               types.Result `json:"ipam"`
}

type NetworksDeleteContainerPayload struct {
	InterfaceName      string `json:"interface_name"`
	ContainerNamespace string `json:"container_namespace"`
}
