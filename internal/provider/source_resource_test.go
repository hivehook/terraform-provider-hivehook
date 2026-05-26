package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccProtoV6ProviderFactories registers the provider for acceptance tests
// under its canonical "hivehook" address. Acceptance tests are gated on TF_ACC.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"hivehook": providerserver.NewProtocol6WithError(New()),
}

// testAccPreCheck validates the prerequisites for running acceptance tests
// against a real Hivehook server.
func testAccPreCheck(t *testing.T) {
	if os.Getenv("HIVEHOOK_URL") == "" {
		t.Fatal("HIVEHOOK_URL must be set for acceptance tests")
	}
	if os.Getenv("HIVEHOOK_API_KEY") == "" {
		t.Fatal("HIVEHOOK_API_KEY must be set for acceptance tests")
	}
}

// TestAccSourceResource_basic verifies the resource lifecycle (create, read,
// destroy) against a live Hivehook server. The test is skipped unless TF_ACC=1
// is set in the environment.
func TestAccSourceResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests skipped; set TF_ACC=1 to run")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "hivehook_source" "test" {
  name          = "tf-acc-test"
  slug          = "tf-acc-test"
  provider_type = "generic"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hivehook_source.test", "id"),
					resource.TestCheckResourceAttr("hivehook_source.test", "name", "tf-acc-test"),
					resource.TestCheckResourceAttr("hivehook_source.test", "slug", "tf-acc-test"),
					resource.TestCheckResourceAttr("hivehook_source.test", "provider_type", "generic"),
				),
			},
			{
				ResourceName:      "hivehook_source.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Compile-time guard: confirm the helper functions are referenced.
var _ = context.Background
