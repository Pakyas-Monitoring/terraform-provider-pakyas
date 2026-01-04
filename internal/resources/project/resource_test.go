package project_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/pakyas/terraform-provider-pakyas/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"pakyas": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("PAKYAS_API_KEY"); v == "" {
		t.Fatal("PAKYAS_API_KEY must be set for acceptance tests")
	}
}

func TestAccProjectResource_basic(t *testing.T) {
	// Generate unique name to avoid conflicts
	uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())
	resourceName := "pakyas_project.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfig(uniqueID, "Test Project", "Test description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Test Project "+uniqueID),
					resource.TestCheckResourceAttr(resourceName, "description", "Test description"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "org_id"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing
			{
				Config: testAccProjectResourceConfig(uniqueID, "Updated Project", "Updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Updated Project "+uniqueID),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated description"),
				),
			},
			// Delete testing happens automatically
		},
	})
}

func TestAccProjectResource_noDescription(t *testing.T) {
	uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())
	resourceName := "pakyas_project.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceConfigNoDescription(uniqueID, "Minimal Project"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Minimal Project "+uniqueID),
					resource.TestCheckNoResourceAttr(resourceName, "description"),
				),
			},
		},
	})
}

func testAccProjectResourceConfig(uniqueID, name, description string) string {
	return fmt.Sprintf(`
resource "pakyas_project" "test" {
  name        = "%s %s"
  description = "%s"
}
`, name, uniqueID, description)
}

func testAccProjectResourceConfigNoDescription(uniqueID, name string) string {
	return fmt.Sprintf(`
resource "pakyas_project" "test" {
  name = "%s %s"
}
`, name, uniqueID)
}
