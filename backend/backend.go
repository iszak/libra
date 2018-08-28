package backend

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/underarmour/libra/config"
	"github.com/underarmour/libra/structs"
)

// ConfiguredBackends struct
type ConfiguredBackends map[string]ConfiguredBackend
type ConfiguredBackend struct {
	Backend structs.Backender
	Kind    string
}

// InitializeBackends loads valid backends into a map
func InitializeBackends(backends map[string]structs.Backend) (ConfiguredBackends, error) {
	configuredBackends := make(ConfiguredBackends, 0)

	for name, backend := range backends {
		backendType := backend.Kind

		if backendType == "" {
			return nil, fmt.Errorf("Missing backend type for '%s'", name)
		}

		switch backendType {
		case "cloudwatch":
			c, err := config.NewConfig(os.Getenv("LIBRA_CONFIG_DIR"))
			if err != nil {
				log.Errorf("Failed to read or parse config file: %s", err)
				return nil, err
			}

			conf := c.Backends[name]

			connection, err := NewCloudWatchBackend(CloudWatchConfig{
				Region: conf.Region,
			})
			if err != nil {
				return nil, fmt.Errorf("Bad configuration for %s: %s", name, err)
			}

			configuredBackends[name] = ConfiguredBackend{
				Backend: connection,
				Kind:    backendType,
			}
		case "graphite":
			c, err := config.NewConfig(os.Getenv("LIBRA_CONFIG_DIR"))
			if err != nil {
				log.Errorf("Failed to read or parse config file: %s", err)
				return nil, err
			}

			conf := c.Backends[name]

			password := conf.Password
			if password == "" {
				password = os.Getenv("GRAPHITE_PASSWORD")
			}
			connection, err := NewGraphiteBackend(GraphiteConfig{
				Kind:     conf.Kind,
				Host:     conf.Host,
				Username: conf.Username,
				Password: password,
			})
			if err != nil {
				return nil, fmt.Errorf("Bad configuration for %s: %s", name, err)
			}

			configuredBackends[name] = ConfiguredBackend{
				Backend: connection,
				Kind:    backendType,
			}
		case "prometheus":
			c, err := config.NewConfig(os.Getenv("LIBRA_CONFIG_DIR"))
			if err != nil {
				log.Errorf("Failed to read or parse config file: %s", err)
				return nil, err
			}

			conf := c.Backends[name]

			client, err := NewQueryAPI(conf.Host)
			if err != nil {
				return nil, err
			}

			connection, err := NewPrometheusBackend(PrometheusConfig{}, client)

			if err != nil {
				return nil, fmt.Errorf("Bad configuration for %s: %s", name, err)
			}

			configuredBackends[name] = ConfiguredBackend{
				Backend: connection,
				Kind:    backendType,
			}
		default:
			log.Fatalf("unknown backend type '%s' for backend %s", backendType, name)
			return nil, fmt.Errorf("unknown backend %s", backendType)
		}
	}

	return configuredBackends, nil
}
