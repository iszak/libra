package command

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"flag"

	"time"

	"github.com/ant0ine/go-json-rest/rest"
	napi "github.com/hashicorp/nomad/api"
	"github.com/mitchellh/cli"
	"github.com/sirupsen/logrus"
	"github.com/underarmour/libra/api"
	"github.com/underarmour/libra/backend"
	"github.com/underarmour/libra/config"
	"github.com/underarmour/libra/nomad"
	"github.com/underarmour/libra/structs"
	"gopkg.in/robfig/cron.v2"
)

// ServerCommand is a Command implementation prints the version.
type ServerCommand struct {
	ConfDir string
	Ui      cli.Ui
}

func (c *ServerCommand) Help() string {
	helpText := `
Usage: libra server [options]
  Run a Libra server. The other commands require a server to be configured.
`
	return strings.TrimSpace(helpText)
}

func (c *ServerCommand) Run(args []string) int {
	serverFlags := flag.NewFlagSet("server", flag.ContinueOnError)
	serverFlags.StringVar(&c.ConfDir, "conf", "/etc/libra", "Config directory for Libra")
	if err := serverFlags.Parse(args); err != nil {
		return 1
	}

	os.Setenv("LIBRA_CONFIG_DIR", c.ConfDir)
	s := rest.NewApi()
	logger := logrus.New()
	w := logger.Writer()
	defer w.Close()

	loggingMw := &rest.AccessLogApacheMiddleware{
		Logger: log.New(w, "[access] ", 0),
	}

	mw := []rest.Middleware{
		loggingMw,
		&rest.ContentTypeCheckerMiddleware{},
		&rest.GzipMiddleware{},
		&rest.JsonIndentMiddleware{},
		&rest.PoweredByMiddleware{},
		&rest.RecorderMiddleware{},
		&rest.RecoverMiddleware{
			EnableResponseStackTrace: true,
		},
		&rest.TimerMiddleware{},
	}

	cfg := newRootConfig()
	backends := newBackends(cfg)
	client := newNomadClient(cfg.Nomad)

	s.Use(mw...)
	router, err := rest.MakeRouter(
		rest.Post("/scale", api.ScaleHandler(cfg, client)),
		rest.Post("/capacity", api.CapacityHandler(cfg, client)),
		rest.Post("/grafana", api.GrafanaHandler(client)),
		rest.Get("/backends", api.BackendsHandler(backends)),
		rest.Get("/ping", api.PingHandler),
		rest.Get("/", api.HomeHandler),
		rest.Post("/restart", api.RestartHandler(client)),
	)
	if err != nil {
		logrus.Fatal(err)
	}

	s.SetApp(router)

	cr, _, err := loadRules(cfg, backends, client)
	cr.Start()
	if err != nil {
		logrus.Errorf("Problem with the Libra server: %s", err)
		return 1
	}

	err = http.ListenAndServe(":8646", s.MakeHandler())
	if err != nil {
		logrus.Errorf("Problem with the Libra server: %s", err)
		return 1
	}
	return 0
}

func (c *ServerCommand) Synopsis() string {
	return "Run a Libra server"
}

func newNomadClient(cfg nomad.Config) *napi.Client {
	n, err := nomad.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Nomad Client: %s", err)
	}
	logrus.Info("Successfully created Nomad Client")
	dc, err := n.Agent().Datacenter()
	if err != nil {
		logrus.Fatalf("  Failed to get Nomad DC: %s", err)
	}
	logrus.Infof("  -> DC: %s", dc)
	return n
}

func newRootConfig() *config.RootConfig {
	cfg, err := config.NewConfig(os.Getenv("LIBRA_CONFIG_DIR"))
	if err != nil {
		logrus.Fatalf("Failed to read or parse config file: %s", err)
	}
	logrus.Info("Loaded and parsed configuration file")
	return cfg
}

func newBackends(cfg *config.RootConfig) backend.ConfiguredBackends {
	backends, err := backend.InitializeBackends(cfg.Backends)
	if err != nil {
		logrus.Fatalf("%s", err)
	}
	return backends
}

func loadRules(config *config.RootConfig, backends backend.ConfiguredBackends, client *napi.Client) (*cron.Cron, []cron.EntryID, error) {
	logrus.Info("")
	logrus.Infof("Found %d backends", len(backends))
	for name, cb := range backends {
		logrus.Infof("  -> %s (%s)", name, cb.Kind)
	}
	logrus.Info("")
	logrus.Infof("Found %d jobs", len(config.Jobs))

	cr := cron.New()
	ids := []cron.EntryID{}

	for _, job := range config.Jobs {
		logrus.Infof("  -> Job: %s", job.Name)

		for _, group := range job.Groups {
			logrus.Infof("  --> Group: %s", group.Name)
			logrus.Infof("      min_count = %d", group.MinCount)
			logrus.Infof("      max_count = %d", group.MaxCount)

			for name, rule := range group.Rules {
				cfID, err := cr.AddFunc(rule.Period, createCronFunc(rule, client, job.Name, group.Name, group.MinCount, group.MaxCount))
				if err != nil {
					logrus.Errorf("Problem adding autoscaling rule to cron: %s", err)
					return cr, ids, err
				}
				ids = append(ids, cfID)
				logrus.Infof("  ----> Rule: %s", rule.Name)
				if be, ok := backends[rule.Backend]; ok {
					rule.BackendInstance = be.Backend
					continue
				}

				return cr, ids, fmt.Errorf("Unknown backend: %s (%s)", rule.Backend, name)
			}
		}
	}
	return cr, ids, nil
}

func createCronFunc(rule *structs.Rule, client *napi.Client, job, group string, min, max int) func() {
	return func() {
		n := rand.Intn(10) // offset cron jobs slightly so they don't collide
		time.Sleep(time.Duration(n) * time.Second)
		err := backend.Work(rule, client, job, group, min, max)

		if err != nil {
			logrus.Errorf("%s", err)
		}
	}
}
