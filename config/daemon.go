package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	ListenHost     string   `json:"listen_host"`
	ListenPort     int      `json:"listen_port"`
	LocalSubnet    string   `json:"local_subnet"`
	OverlayNetwork string   `json:"overlay_network"`
	SandboxDir     string   `json:"sandbox_dir"`
	Database       Database `json:"database"`
	Index          int      `json:"index"`
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
	ListenAddress  string
	OverlayNetwork *net.IPNet
	LocalSubnet    *net.IPNet
	DatabaseURL    string
	SandboxRepoDir string
}

func (d Daemon) ParseAndValidate() (ValidatedConfig, error) {
	db := d.Database
	dbURL := fmt.Sprintf("%s://%s:%s@%s:%d/%s?sslmode=%s",
		"postgres", db.Username, db.Password, db.Host, db.Port, db.Name, db.SslMode)

	_, overlay, err := net.ParseCIDR(d.OverlayNetwork)
	if err != nil {
		panic(err)
	}

	interpolatedSubnet := strings.Replace(d.LocalSubnet, "${index}", fmt.Sprintf("%d", d.Index), -1)
	_, subnet, err := net.ParseCIDR(interpolatedSubnet)
	if err != nil {
		panic(err)
	}

	return ValidatedConfig{
		ListenAddress:  fmt.Sprintf("%s:%d", d.ListenHost, d.ListenPort),
		OverlayNetwork: overlay,
		LocalSubnet:    subnet,
		DatabaseURL:    dbURL,
		SandboxRepoDir: "/var/vcap/data/ducati/sandbox",
	}, nil
}
