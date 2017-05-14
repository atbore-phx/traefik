package main

import (
	"bytes"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/containous/traefik/integration/try"
	"github.com/go-check/check"

	checker "github.com/vdemeester/shakers"
)

// HealthCheck test suites (using libcompose)
type HealthCheckSuite struct{ BaseSuite }

func (s *HealthCheckSuite) SetUpSuite(c *check.C) {
	s.createComposeProject(c, "healthcheck")
	s.composeProject.Start(c)

}

func (s *HealthCheckSuite) TestSimpleConfiguration(c *check.C) {

	whoami1Host := s.composeProject.Container(c, "whoami1").NetworkSettings.IPAddress
	whoami2Host := s.composeProject.Container(c, "whoami2").NetworkSettings.IPAddress

	file := s.adaptFile(c, "fixtures/healthcheck/simple.toml", struct {
		Server1 string
		Server2 string
	}{whoami1Host, whoami2Host})
	defer os.Remove(file)
	cmd := exec.Command(traefikBinary, "--configFile="+file)

	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	// wait for traefik
	err = try.GetRequest("http://127.0.0.1:8080/api/providers", 60*time.Second, try.BodyContains("Host:test.localhost"))
	c.Assert(err, checker.IsNil)

	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://127.0.0.1:8000/health", nil)
	c.Assert(err, checker.IsNil)
	req.Host = "test.localhost"

	resp, err := client.Do(req)
	c.Assert(err, checker.IsNil)
	c.Assert(resp.StatusCode, checker.Equals, 200)

	resp, err = client.Do(req)
	c.Assert(err, checker.IsNil)
	c.Assert(resp.StatusCode, checker.Equals, 200)

	// Fix all whoami health to 500
	whoamiHosts := []string{whoami1Host, whoami2Host}
	for _, whoami := range whoamiHosts {
		healthReq, err := http.NewRequest("POST", "http://"+whoami+"/health", bytes.NewBuffer([]byte("500")))
		c.Assert(err, checker.IsNil)
		_, err = client.Do(healthReq)
		c.Assert(err, checker.IsNil)
	}

	try.Sleep(2 * time.Second)

	// Verify frontend health : 500
	err = try.Request(req, 3*time.Second, try.StatusCodeIs(500))
	c.Assert(err, checker.IsNil)

	// Change one whoami health to 200
	healthReq1, err := http.NewRequest("POST", "http://"+whoami1Host+"/health", bytes.NewBuffer([]byte("200")))
	c.Assert(err, checker.IsNil)
	_, err = client.Do(healthReq1)
	c.Assert(err, checker.IsNil)

	// Verify frontend health : before
	err = try.Request(req, 3*time.Second, try.StatusCodeIs(500))
	c.Assert(err, checker.IsNil)

	try.Sleep(2 * time.Second)

	// Verify frontend health : after
	err = try.Request(req, 3*time.Second, try.StatusCodeIs(200))
	c.Assert(err, checker.IsNil)

	// sure?
	err = try.Request(req, 3*time.Second, try.StatusCodeIs(500))
	c.Assert(err, checker.Not(checker.IsNil))

	resp, err = client.Do(req)
	c.Assert(err, checker.IsNil)
	c.Assert(resp.StatusCode, checker.Equals, 200)

	// TODO validate : run on 80
	resp, err = http.Get("http://127.0.0.1:8000/")

	// Expected a 404 as we did not configure anything
	c.Assert(err, checker.IsNil)
	c.Assert(resp.StatusCode, checker.Equals, 404)
}
