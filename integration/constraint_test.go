package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/containous/traefik/integration/try"
	"github.com/go-check/check"
	"github.com/hashicorp/consul/api"
	checker "github.com/vdemeester/shakers"
)

// Constraint test suite
type ConstraintSuite struct {
	BaseSuite
	consulIP     string
	consulClient *api.Client
}

func (s *ConstraintSuite) SetUpSuite(c *check.C) {

	s.createComposeProject(c, "constraints")
	s.composeProject.Start(c)

	consul := s.composeProject.Container(c, "consul")

	s.consulIP = consul.NetworkSettings.IPAddress
	config := api.DefaultConfig()
	config.Address = s.consulIP + ":8500"
	consulClient, err := api.NewClient(config)
	if err != nil {
		c.Fatalf("Error creating consul client")
	}
	s.consulClient = consulClient

	// Wait for consul to elect itself leader
	err = try.Do(3*time.Second, func() error {
		leader, err := consulClient.Status().Leader()

		if err != nil || len(leader) == 0 {
			return fmt.Errorf("Leader not find. %v", err)
		}

		return nil
	})
	c.Assert(err, checker.IsNil)
}

func (s *ConstraintSuite) registerService(name string, address string, port int, tags []string) error {
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

func (s *ConstraintSuite) deregisterService(name string, address string) error {
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

func (s *ConstraintSuite) TestMatchConstraintGlobal(c *check.C) {
	cmd := exec.Command(traefikBinary,
		"--consulCatalog",
		"--consulCatalog.endpoint="+s.consulIP+":8500",
		"--consulCatalog.domain=consul.localhost",
		"--configFile=fixtures/consul_catalog/simple.toml",
		"--constraints=tag==api")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	nginx := s.composeProject.Container(c, "nginx")

	err = s.registerService("test", nginx.NetworkSettings.IPAddress, 80, []string{"traefik.tags=api"})
	c.Assert(err, checker.IsNil, check.Commentf("Error registering service"))
	defer s.deregisterService("test", nginx.NetworkSettings.IPAddress)

	req, err := http.NewRequest("GET", "http://127.0.0.1:8000/", nil)
	c.Assert(err, checker.IsNil)
	req.Host = "test.consul.localhost"

	err = try.Request(req, 5*time.Second, try.StatusCodeIs(200))
	c.Assert(err, checker.IsNil)
}

func (s *ConstraintSuite) TestDoesNotMatchConstraintGlobal(c *check.C) {
	cmd := exec.Command(traefikBinary,
		"--consulCatalog",
		"--consulCatalog.endpoint="+s.consulIP+":8500",
		"--consulCatalog.domain=consul.localhost",
		"--configFile=fixtures/consul_catalog/simple.toml",
		"--constraints=tag==api")
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

	err = try.Request(req, 5*time.Second, try.StatusCodeIs(404))
	c.Assert(err, checker.IsNil)
}

func (s *ConstraintSuite) TestMatchConstraintProvider(c *check.C) {
	cmd := exec.Command(traefikBinary,
		"--consulCatalog",
		"--consulCatalog.endpoint="+s.consulIP+":8500",
		"--consulCatalog.domain=consul.localhost",
		"--configFile=fixtures/consul_catalog/simple.toml",
		"--consulCatalog.constraints=tag==api")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	nginx := s.composeProject.Container(c, "nginx")

	err = s.registerService("test", nginx.NetworkSettings.IPAddress, 80, []string{"traefik.tags=api"})
	c.Assert(err, checker.IsNil, check.Commentf("Error registering service"))
	defer s.deregisterService("test", nginx.NetworkSettings.IPAddress)

	req, err := http.NewRequest("GET", "http://127.0.0.1:8000/", nil)
	c.Assert(err, checker.IsNil)
	req.Host = "test.consul.localhost"

	err = try.Request(req, 5*time.Second, try.StatusCodeIs(200))
	c.Assert(err, checker.IsNil)
}

func (s *ConstraintSuite) TestDoesNotMatchConstraintProvider(c *check.C) {
	cmd := exec.Command(traefikBinary,
		"--consulCatalog",
		"--consulCatalog.endpoint="+s.consulIP+":8500",
		"--consulCatalog.domain=consul.localhost",
		"--configFile=fixtures/consul_catalog/simple.toml",
		"--consulCatalog.constraints=tag==api")
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

	err = try.Request(req, 5*time.Second, try.StatusCodeIs(404))
	c.Assert(err, checker.IsNil)
}

func (s *ConstraintSuite) TestMatchMultipleConstraint(c *check.C) {
	cmd := exec.Command(traefikBinary,
		"--consulCatalog",
		"--consulCatalog.endpoint="+s.consulIP+":8500",
		"--consulCatalog.domain=consul.localhost",
		"--configFile=fixtures/consul_catalog/simple.toml",
		"--consulCatalog.constraints=tag==api",
		"--constraints=tag!=us-*")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	nginx := s.composeProject.Container(c, "nginx")

	err = s.registerService("test", nginx.NetworkSettings.IPAddress, 80, []string{"traefik.tags=api", "traefik.tags=eu-1"})
	c.Assert(err, checker.IsNil, check.Commentf("Error registering service"))
	defer s.deregisterService("test", nginx.NetworkSettings.IPAddress)

	req, err := http.NewRequest("GET", "http://127.0.0.1:8000/", nil)
	c.Assert(err, checker.IsNil)
	req.Host = "test.consul.localhost"

	err = try.Request(req, 5*time.Second, try.StatusCodeIs(200))
	c.Assert(err, checker.IsNil)
}

func (s *ConstraintSuite) TestDoesNotMatchMultipleConstraint(c *check.C) {
	cmd := exec.Command(traefikBinary,
		"--consulCatalog",
		"--consulCatalog.endpoint="+s.consulIP+":8500",
		"--consulCatalog.domain=consul.localhost",
		"--configFile=fixtures/consul_catalog/simple.toml",
		"--consulCatalog.constraints=tag==api",
		"--constraints=tag!=us-*")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	nginx := s.composeProject.Container(c, "nginx")

	err = s.registerService("test", nginx.NetworkSettings.IPAddress, 80, []string{"traefik.tags=api", "traefik.tags=us-1"})
	c.Assert(err, checker.IsNil, check.Commentf("Error registering service"))
	defer s.deregisterService("test", nginx.NetworkSettings.IPAddress)

	req, err := http.NewRequest("GET", "http://127.0.0.1:8000/", nil)
	c.Assert(err, checker.IsNil)
	req.Host = "test.consul.localhost"

	err = try.Request(req, 5*time.Second, try.StatusCodeIs(404))
	c.Assert(err, checker.IsNil)
}
