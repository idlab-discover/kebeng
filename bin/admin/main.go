package main

import (
	"fmt"
	"os"

	"github.com/idlab-discover/kebeng/cmd/admin"
	"github.com/idlab-discover/kebeng/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	cobra.OnInitialize(config.LoadConfig)
}

func main() {
	logrus.SetLevel(logrus.TraceLevel)

	if err := admin.Admin.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
