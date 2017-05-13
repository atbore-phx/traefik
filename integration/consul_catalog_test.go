package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/containous/traefik/integration/utils"
	"github.com/go-check/check"
	"github.com/hashicorp/consul/api"
	checker "github.com/vdemeester/shakers"
)

// Consul catalog test suites
type ConsulCatalogSuite struct {
	BaseSuite
	consulIP     string
	consulClient *api.Client
}

func (s *ConsulCatalogSuite) SetUpSuite(c *check.C) {

	s.createComposeProject(c, "consul_catalog")
	s.composeProject.Start(c)

	consul := s.composeProject.Container(c, "consul")

	s.consulIP = consul.NetworkSettings.IPAddress
	config := api.DefaultConfig()
	config.Address = s.consulIP + ":8500"
	consulClient, err := api.NewClient(config)
	if err != nil {
		c.Fatalf("Error creating consul client. %v", err)
	}
	s.consulClient = consulClient

	// Wait for consul to elect itself leader
	err = utils.Try(2*time.Second, func() error {
		leader, err := consulClient.Status().Leader()

		if err != nil || len(leader) == 0 {
			return fmt.Errorf("Leader not find. %v", err)
		}

		return nil
	})
	c.Assert(err, checker.IsNil)
}

func (s *ConsulCatalogSuite) registerService(name string, address string, port int, tags []string) error {
	catalog := s.consulClient.Catalog()
	_, err := catalog.Register(
		&api.CatalogRegistration{
			Node:    address,
			Address: address,
			Service: &api.AgentService{
				ID:      name,
				Service: name,
				Address: address,
				Port:    port,
				Tags:    tags,
			},
		},
		&api.WriteOptions{},
	)
	return err
}

func (s *ConsulCatalogSuite) deregisterService(name string, address string) error {
	catalog := s.consulClient.Catalog()
	_, err := catalog.Deregister(
		&api.CatalogDeregistration{
			Node:      address,
			Address:   address,
			ServiceID: name,
		},
		&api.WriteOptions{},
	)
	return err
}

func (s *ConsulCatalogSuite) TestSimpleConfiguration(c *check.C) {
	cmd := exec.Command(traefikBinary,
		"--consulCatalog",
		"--consulCatalog.endpoint="+s.consulIP+":8500",
		"--configFile=fixtures/consul_catalog/simple.toml")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	// FIXME replace by a Try
	utils.Sleep(500 * time.Millisecond)
	// TODO validate : run on 80
	resp, err := http.Get("http://127.0.0.1:8000/")

	// Expected a 404 as we did not configure anything
	c.Assert(err, checker.IsNil)
	c.Assert(resp.StatusCode, checker.Equals, 404)
}

func (s *ConsulCatalogSuite) TestSingleService(c *check.C) {
	cmd := exec.Command(traefikBinary,
		"--consulCatalog",
		"--consulCatalog.endpoint="+s.consulIP+":8500",
		"--consulCatalog.domain=consul.localhost",
		"--configFile=fixtures/consul_catalog/simple.toml")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	nginx := s.composeProject.Container(c, "nginx")

	err = s.registerService("test", nginx.NetworkSettings.IPAddress, 80, []string{})
	c.Assert(err, checker.IsNil, check.Commentf("Error registering service"))
	defer s.deregisterService("test", nginx.NetworkSettings.IPAddress)

	req, err := http.NewRequest("GET", "http://127.0.0.1:8000/", nil)
	c.Assert(err, checker.IsNil)
	req.Host = "test.consul.localhost"

	cond := utils.ComposeCondition(utils.UntilStatusCodeIs(200), utils.BodyContains("Welcome to nginx!"))
	err = utils.TryRequest(req, 5*time.Second, cond)
	c.Assert(err, checker.IsNil)
}
