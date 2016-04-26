package models

import "github.com/appc/cni/pkg/types"

type Properties struct {
	AppGUID   string `json:"app_guid"`
	SpaceGUID string `json:"space_guid"`
}

type NetworkPayload struct {
	Properties Properties `json:"properties,omitempty"`
}

type CNIAddPayload struct {
	Args               string         `json:"args"`
	ContainerNamespace string         `json:"container_namespace"`
	InterfaceName      string         `json:"interface_name"`
	IPAM               types.Result   `json:"ipam"`
	Network            NetworkPayload `json:"network"`
	ContainerID        string         `json:"container_id"`
}

type CNIDelPayload struct {
	InterfaceName      string `json:"interface_name"`
	ContainerNamespace string `json:"container_namespace"`
	ContainerID        string `json:"container_id"`
}
