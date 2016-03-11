package config_test

import (
	"bytes"
	"net"
	"strings"

	"github.com/cloudfoundry-incubator/ducati-daemon/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const fixtureJSON = `
{
	"listen_host": "0.0.0.0",
	"listen_port": 4001,
	"local_subnet": "192.168.${index}.0/24",
	"overlay_network": "192.168.0.0/16",
	"sandbox_dir": "/var/vcap/data/ducati/sandbox",
	"database": {
	  "host": "10.244.16.9",
	  "port": 5432,
	  "username": "ducati_daemon",
	  "password": "some-password",
	  "name": "ducati",
	  "ssl_mode": "disable"
	},
	"index": 9
}
`

var _ = Describe("Daemon config", func() {
	var fixtureDaemon config.Daemon
	BeforeEach(func() {
		fixtureDaemon = config.Daemon{
			ListenHost:     "0.0.0.0",
			ListenPort:     4001,
			LocalSubnet:    "192.168.${index}.0/24",
			OverlayNetwork: "192.168.0.0/16",
			SandboxDir:     "/var/vcap/data/ducati/sandbox",
			Database: config.Database{
				Host:     "10.244.16.9",
				Port:     5432,
				Username: "ducati_daemon",
				Password: "some-password",
				Name:     "ducati",
				SslMode:  "disable",
			},
			Index: 9,
		}
	})

	Describe("serialization and deserialization", func() {
		It("translates between JSON and a config struct", func() {
			configReader := strings.NewReader(fixtureJSON)
			daemonConfig, err := config.Unmarshal(configReader)
			Expect(err).NotTo(HaveOccurred())

			Expect(daemonConfig).To(Equal(fixtureDaemon))

			serializedBytes := &bytes.Buffer{}
			err = daemonConfig.Marshal(serializedBytes)
			Expect(err).NotTo(HaveOccurred())

			Expect(serializedBytes).To(MatchJSON(fixtureJSON))
		})

		Context("when the input is not valid JSON", func() {
			It("returns an error", func() {
				configReader := strings.NewReader(`{{{{{`)
				_, err := config.Unmarshal(configReader)
				Expect(err).To(MatchError("json decode: invalid character '{' looking for beginning of object key string"))
			})
		})
	})

	Describe("parsing and validating the fields", func() {
		It("parses and composes the config into Go types", func() {
			validated, err := fixtureDaemon.ParseAndValidate()
			Expect(err).NotTo(HaveOccurred())

			_, expectedOverlay, _ := net.ParseCIDR("192.168.0.0/16")
			_, expectedLocalSubnet, _ := net.ParseCIDR("192.168.9.0/24")

			dbURL := "postgres://ducati_daemon:some-password@10.244.16.9:5432/ducati?sslmode=disable"

			Expect(validated).To(Equal(config.ValidatedConfig{
				ListenAddress:  "0.0.0.0:4001",
				OverlayNetwork: expectedOverlay,
				LocalSubnet:    expectedLocalSubnet,
				DatabaseURL:    dbURL,
				SandboxRepoDir: "/var/vcap/data/ducati/sandbox",
			}))
		})
	})
})
