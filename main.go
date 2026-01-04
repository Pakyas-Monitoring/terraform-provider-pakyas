// Pakyas Terraform Provider
//
// This provider enables management of Pakyas resources via Terraform.
// Supports: pakyas_project, pakyas_check
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/pakyas/terraform-provider-pakyas/internal/provider"
)

// These will be set by GoReleaser during build
var (
	version = "dev"
	commit  = "none"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/pakyas/pakyas",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
