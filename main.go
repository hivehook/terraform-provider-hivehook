// Package main is the entry point for the Hivehook Terraform provider plugin.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hivehook/terraform-provider-hivehook/internal/provider"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	if err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/hivehook/hivehook",
		Debug:   debug,
	}); err != nil {
		log.Fatal(err)
	}
}
