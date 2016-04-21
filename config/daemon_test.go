package config_test

import (
	"bytes"
	"io/ioutil"
	"net"
	"strings"

	"github.com/cloudfoundry-incubator/ducati-daemon/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const fixtureJSON = `
{
	"listen_host": "0.0.0.0",
	"listen_port": 4001,
	"local_subnet": "192.168.${index}.0/16",
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
	"host_address": "10.244.16.3",
	"index": 9,
	"dns_server": "1.2.3.4",
	"overlay_dns_address": "192.168.255.254"
}
`

var _ = Describe("Daemon config", func() {
	var fixtureDaemon config.Daemon
	BeforeEach(func() {
		fixtureDaemon = config.Daemon{
			ListenHost:     "0.0.0.0",
			ListenPort:     4001,
			LocalSubnet:    "192.168.${index}.0/16",
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
			HostAddress:       "10.244.16.3",
			Index:             9,
			ExternalDNSServer: "1.2.3.4",
			OverlayDNSAddress: "192.168.255.254",
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
			expectedLocalSubnet := &net.IPNet{
				IP:   net.ParseIP("192.168.9.0"),
				Mask: net.CIDRMask(16, 32),
			}

			dbURL := "postgres://ducati_daemon:some-password@10.244.16.9:5432/ducati?sslmode=disable"

			Expect(validated).To(Equal(&config.ValidatedConfig{
				ListenAddress:     "0.0.0.0:4001",
				OverlayNetwork:    expectedOverlay,
				LocalSubnet:       expectedLocalSubnet,
				DatabaseURL:       dbURL,
				SandboxRepoDir:    "/var/vcap/data/ducati/sandbox",
				HostAddress:       net.ParseIP("10.244.16.3"),
				ExternalDNSServer: net.ParseIP("1.2.3.4"),
				OverlayDNSAddress: net.ParseIP("192.168.255.254"),
			}))
		})
	})

	Describe("error cases", func() {
		var conf config.Daemon

		BeforeEach(func() {
			conf = config.Daemon{
				ListenHost:     "127.0.0.1",
				ListenPort:     4001,
				LocalSubnet:    "192.168.${index}.0/24",
				OverlayNetwork: "192.168.0.0/16",
				SandboxDir:     "/some/sandbox/repo/path",
				Database: config.Database{
					Host:     "some-host",
					Port:     1234,
					Username: "some-username",
					Password: "some-password",
					Name:     "some-database-name",
					SslMode:  "some-ssl-mode",
				},
				Index:             9,
				HostAddress:       "10.244.16.3",
				ExternalDNSServer: "1.2.3.4",
				OverlayDNSAddress: "192.168.255.254",
			}
		})

		DescribeTable("missing or invalid config",
			func(expectedError string, corrupter func()) {
				corrupter()
				_, err := conf.ParseAndValidate()
				Expect(err).To(MatchError(err))
			},

			Entry("missing ListenHost", `missing required config "listen_host"`, func() { conf.ListenHost = "" }),
			Entry("missing ListenPort", `missing required config "listen_port"`, func() { conf.ListenPort = 0 }),
			Entry("missing LocalSubnet", `missing required config "local_subnet"`, func() { conf.LocalSubnet = "" }),
			Entry("missing OverlayNetwork", `missing required config "overlay_network"`, func() { conf.OverlayNetwork = "" }),
			Entry("missing SandboxDir", `missing required config "sandbox_dir"`, func() { conf.SandboxDir = "" }),
			Entry("missing Database Host", `missing required config "database.host"`, func() { conf.Database.Host = "" }),
			Entry("missing Database Port", `missing required config "database.port"`, func() { conf.Database.Port = 0 }),
			Entry("missing Database Username", `missing required config "database.username"`, func() { conf.Database.Username = "" }),
			Entry("missing Database Name", `missing required config "database.name"`, func() { conf.Database.Name = "" }),
			Entry("missing Database SslMode", `missing required config "database.ssl_mode"`, func() { conf.Database.SslMode = "" }),
			Entry("unparsable LocalSubnet", `bad config "local_subnet": invalid CIDR address: foo`, func() { conf.LocalSubnet = "foo" }),
			Entry("unparsable OverlayNetwork", `bad config "overlay_network": invalid CIDR address: bar`, func() { conf.OverlayNetwork = "bar" }),
			Entry("missing ExternalDNSServer", `missing required config "dns_server"`, func() { conf.ExternalDNSServer = "" }),
			Entry("unparsable ExternalDNSServer", `bad config "dns_server"`, func() { conf.ExternalDNSServer = "sdfasdf" }),
			Entry("missing OverlayDNSAddress", `missing required config "overlay_dns_address"`, func() { conf.OverlayDNSAddress = "" }),
			Entry("unparsable OverlayDNSAddress", `bad config "overlay_dns_address"`, func() { conf.OverlayDNSAddress = "sdfasdf" }),
			Entry("OverlayDNSAddress has port", `bad config "overlay_dns_address" has port`, func() { conf.OverlayDNSAddress = "192.168.255.254:99" }),
			Entry("OverlayDNSAddress not in overlay network", `bad config "overlay_dns_address" not in overlay network`, func() { conf.OverlayDNSAddress = "1.2.3.4" }),
			Entry("missing HostAddress", `missing required config "host_address"`, func() { conf.HostAddress = "" }),
			Entry("unparsable HostAddress", `bad config "host_address": invalid CIDR address: bar`, func() { conf.HostAddress = "bar" }),
			Entry("zero HostAddress", `bad config "host_address": must be nonzero`, func() { conf.HostAddress = "0.0.0.0" }),
		)

		It("does not complain when the database password is empty", func() {
			conf.Database.Password = ""
			_, err := conf.ParseAndValidate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("loading config from a file", func() {
		It("returns the parsed and validated config", func() {
			configSource := config.Daemon{
				ListenHost:     "127.0.0.1",
				ListenPort:     4001,
				LocalSubnet:    "192.168.${index}.0/24",
				OverlayNetwork: "192.168.0.0/16",
				SandboxDir:     "/some/sandbox/repo/path",
				Database: config.Database{
					Host:     "some-host",
					Port:     1234,
					Username: "some-username",
					Password: "some-password",
					Name:     "some-database-name",
					SslMode:  "some-ssl-mode",
				},
				Index:             9,
				HostAddress:       "10.244.16.3",
				ExternalDNSServer: "1.2.3.4",
				OverlayDNSAddress: "192.168.255.254",
			}

			configFile, err := ioutil.TempFile("", "config")
			Expect(err).NotTo(HaveOccurred())

			Expect(configSource.Marshal(configFile)).To(Succeed())
			configFile.Close()

			conf, err := config.ParseConfigFile(configFile.Name())
			Expect(err).NotTo(HaveOccurred())

			_, overlay, _ := net.ParseCIDR("192.168.0.0/16")

			expectedLocalSubnet := &net.IPNet{
				IP:   net.ParseIP("192.168.9.0"),
				Mask: net.CIDRMask(24, 32),
			}

			Expect(conf).To(Equal(&config.ValidatedConfig{
				ListenAddress:     "127.0.0.1:4001",
				OverlayNetwork:    overlay,
				LocalSubnet:       expectedLocalSubnet,
				DatabaseURL:       "postgres://some-username:some-password@some-host:1234/some-database-name?sslmode=some-ssl-mode",
				SandboxRepoDir:    "/some/sandbox/repo/path",
				HostAddress:       net.ParseIP("10.244.16.3"),
				ExternalDNSServer: net.ParseIP("1.2.3.4"),
				OverlayDNSAddress: net.ParseIP("192.168.255.254"),
			}))
		})

		Context("when configFilePath is not present", func() {
			It("returns an error", func() {
				_, err := config.ParseConfigFile("")
				Expect(err).To(MatchError("missing config file path"))
			})
		})

		Context("when the config file cannot be opened", func() {
			It("returns an error", func() {
				_, err := config.ParseConfigFile("some-path")
				Expect(err).To(MatchError("open some-path: no such file or directory"))
			})
		})

		Context("when config file contents cannot be unmarshaled", func() {
			It("returns an error", func() {
				_, err := config.ParseConfigFile("/dev/null")
				Expect(err).To(MatchError("parsing config: json decode: EOF"))
			})
		})
	})
})
