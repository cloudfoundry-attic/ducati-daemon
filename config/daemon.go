package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

type Database struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	SslMode  string `json:"ssl_mode"`
}

type Daemon struct {
	ListenHost        string   `json:"listen_host"`
	ListenPort        int      `json:"listen_port"`
	LocalSubnet       string   `json:"local_subnet"`
	OverlayNetwork    string   `json:"overlay_network"`
	SandboxDir        string   `json:"sandbox_dir"`
	Database          Database `json:"database"`
	Index             int      `json:"index"`
	HostAddress       string   `json:"host_address"`
	OverlayDNSAddress string   `json:"overlay_dns_address"`
	ExternalDNSServer string   `json:"dns_server"`
}

func Unmarshal(input io.Reader) (Daemon, error) {
	c := Daemon{}
	decoder := json.NewDecoder(input)

	err := decoder.Decode(&c)
	if err != nil {
		return c, fmt.Errorf("json decode: %s", err)
	}

	return c, nil
}

func (d Daemon) Marshal(output io.Writer) error {
	encoder := json.NewEncoder(output)

	err := encoder.Encode(&d)
	if err != nil {
		return fmt.Errorf("json encode: %s", err) // not tested
	}

	return nil
}

type ValidatedConfig struct {
	ListenAddress     string
	OverlayNetwork    *net.IPNet
	LocalSubnet       *net.IPNet
	DatabaseURL       string
	SandboxRepoDir    string
	HostAddress       net.IP
	OverlayDNSAddress net.IP
	ExternalDNSServer net.IP
}

func (d Daemon) ParseAndValidate() (*ValidatedConfig, error) {
	db := d.Database
	dbURL := fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=%s",
		"postgres", db.Username, db.Password, db.Host, db.Port, db.Name, db.SslMode)

	if d.ListenHost == "" {
		return nil, errors.New(`missing required config "listen_host"`)
	}

	if d.ListenPort == 0 {
		return nil, errors.New(`missing required config "listen_port"`)
	}

	if d.LocalSubnet == "" {
		return nil, errors.New(`missing required config "local_subnet"`)
	}

	if d.OverlayNetwork == "" {
		return nil, errors.New(`missing required config "overlay_network"`)
	}

	if d.SandboxDir == "" {
		return nil, errors.New(`missing required config "sandbox_dir"`)
	}

	if d.Database.Host == "" {
		return nil, errors.New(`missing required config "database.host"`)
	}

	if d.Database.Port == 0 {
		return nil, errors.New(`missing required config "database.port"`)
	}

	if d.Database.Username == "" {
		return nil, errors.New(`missing required config "database.username"`)
	}

	if d.Database.Name == "" {
		return nil, errors.New(`missing required config "database.name"`)
	}

	if d.Database.SslMode == "" {
		return nil, errors.New(`missing required config "database.ssl_mode"`)
	}

	if d.ExternalDNSServer == "" {
		return nil, errors.New(`missing required config "dns_server"`)
	}

	if d.OverlayDNSAddress == "" {
		return nil, errors.New(`missing required config "overlay_dns_address"`)
	}

	if d.HostAddress == "" {
		return nil, errors.New(`missing required config "host_address"`)
	}

	interpolatedSubnet := strings.Replace(d.LocalSubnet, "${index}", fmt.Sprintf("%d", d.Index), -1)
	startIP, subnet, err := net.ParseCIDR(interpolatedSubnet)
	if err != nil {
		return nil, fmt.Errorf(`bad config "local_subnet": %s`, err)
	}

	localSubnet := &net.IPNet{
		IP:   startIP,
		Mask: subnet.Mask,
	}

	_, overlay, err := net.ParseCIDR(d.OverlayNetwork)
	if err != nil {
		return nil, fmt.Errorf(`bad config "overlay_network": %s`, err)
	}

	overlayDNSAddress := net.ParseIP(d.OverlayDNSAddress)
	if overlayDNSAddress == nil {
		return nil, fmt.Errorf(`bad config "host_address": %s`, err)
	}

	if !overlay.Contains(overlayDNSAddress) {
		return nil, fmt.Errorf(`bad config "overlay_dns_address": not in overlay network`)
	}

	externalDNSServer := net.ParseIP(d.ExternalDNSServer)
	if externalDNSServer == nil {
		return nil, fmt.Errorf(`bad config "dns_server": %s`, err)
	}

	hostAddress := net.ParseIP(d.HostAddress)
	if hostAddress == nil {
		return nil, fmt.Errorf(`bad config "host_address": %s`, err)
	}
	if hostAddress.IsUnspecified() {
		return nil, fmt.Errorf(`bad config "host_address": must be nonzero`)
	}

	return &ValidatedConfig{
		ListenAddress:     fmt.Sprintf("%s:%d", d.ListenHost, d.ListenPort),
		OverlayNetwork:    overlay,
		LocalSubnet:       localSubnet,
		DatabaseURL:       dbURL,
		SandboxRepoDir:    d.SandboxDir,
		HostAddress:       hostAddress,
		OverlayDNSAddress: overlayDNSAddress,
		ExternalDNSServer: externalDNSServer,
	}, nil
}

func ParseConfigFile(configFilePath string) (*ValidatedConfig, error) {
	if configFilePath == "" {
		return nil, fmt.Errorf("missing config file path")
	}

	configFile, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()

	daemonConfig, err := Unmarshal(configFile)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %s", err)
	}

	return daemonConfig.ParseAndValidate()
}
