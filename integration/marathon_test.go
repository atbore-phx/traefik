package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/containous/traefik/integration/utils"
	"github.com/gambol99/go-marathon"
	"github.com/go-check/check"

	checker "github.com/vdemeester/shakers"
)

// Marathon test suites (using libcompose)
type MarathonSuite struct{ BaseSuite }

func (s *MarathonSuite) SetUpSuite(c *check.C) {
	s.createComposeProject(c, "marathon")
	s.composeProject.Start(c)

	config := marathon.NewDefaultConfig()

	marathonClient, err := marathon.NewClient(config)
	if err != nil {
		c.Fatalf("Error creating Marathon client. %v", err)
	}

	// Wait for Marathon to elect itself leader
	utils.Try(90*time.Second, func() error {
		leader, err := marathonClient.Leader()

		if err != nil || len(leader) == 0 {
			return fmt.Errorf("Leader not find. %v", err)
		}

		return nil
	})

	c.Assert(err, checker.IsNil)
}

func (s *MarathonSuite) TestSimpleConfiguration(c *check.C) {
	cmd := exec.Command(traefikBinary, "--configFile=fixtures/marathon/simple.toml")
	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	utils.Sleep(500 * time.Millisecond)
	// TODO validate : run on 80
	resp, err := http.Get("http://127.0.0.1:8000/")

	// Expected a 404 as we did not configure anything
	c.Assert(err, checker.IsNil)
	c.Assert(resp.StatusCode, checker.Equals, 404)
}
