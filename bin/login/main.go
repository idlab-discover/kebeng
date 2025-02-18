package main

import (
	"github.com/idlab-discover/kebeng/config"
	"github.com/idlab-discover/kebeng/pkg/login/server"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(config.LoadConfig)
}

func main() {
	s := &server.Server{}
	s.Run()
}
