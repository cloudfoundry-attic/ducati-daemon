package models

import "github.com/appc/cni/pkg/types"

type NetworksSetupContainerPayload struct {
	Args               string       `json:"args"`
	ContainerNamespace string       `json:"container_namespace"`
	InterfaceName      string       `json:"interface_name"`
	VNI                int          `json:"vni"`
	IPAM               types.Result `json:"ipam"`
}
