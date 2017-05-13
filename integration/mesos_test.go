package main

import (
	"net/http"
	"os/exec"
	"time"

	"github.com/go-check/check"

	"github.com/containous/traefik/integration/utils"
	checker "github.com/vdemeester/shakers"
)

// Mesos test suites (using libcompose)
type MesosSuite struct{ BaseSuite }

func (s *MesosSuite) SetUpSuite(c *check.C) {
	s.createComposeProject(c, "mesos")
}

func (s *MesosSuite) TestSimpleConfiguration(c *check.C) {
	cmd := exec.Command(traefikBinary, "--configFile=fixtures/mesos/simple.toml")
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
