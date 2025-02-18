package main

import (
	"github.com/idlab-discover/kebeng/config"
	"github.com/idlab-discover/kebeng/pkg/admind"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(config.LoadConfig)
}

func main() {
	s := &admind.Server{}
	s.Init()
	s.Run()
}
