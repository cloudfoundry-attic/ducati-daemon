package models

type Container struct {
	ID     string `json:"id"`
	IP     string `json:"ip"`
	MAC    string `json:"mac"`
	HostIP string `json:"host_ip" db:"host_ip"`
}
