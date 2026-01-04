package check_test

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

func TestAccCheckResource_basic(t *testing.T) {
	uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())
	resourceName := "pakyas_check.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccCheckResourceConfig(uniqueID, "Test Check", 3600, 300, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Test Check"),
					resource.TestCheckResourceAttr(resourceName, "slug", "test-check-"+uniqueID),
					resource.TestCheckResourceAttr(resourceName, "period_seconds", "3600"),
					resource.TestCheckResourceAttr(resourceName, "grace_seconds", "300"),
					resource.TestCheckResourceAttr(resourceName, "paused", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "project_id"),
					resource.TestCheckResourceAttrSet(resourceName, "public_id"),
					resource.TestCheckResourceAttrSet(resourceName, "ping_url"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing - change name and period
			{
				Config: testAccCheckResourceConfig(uniqueID, "Updated Check", 7200, 600, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Updated Check"),
					resource.TestCheckResourceAttr(resourceName, "period_seconds", "7200"),
					resource.TestCheckResourceAttr(resourceName, "grace_seconds", "600"),
				),
			},
			// Update testing - pause the check
			{
				Config: testAccCheckResourceConfig(uniqueID, "Updated Check", 7200, 600, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "paused", "true"),
				),
			},
			// Update testing - resume the check
			{
				Config: testAccCheckResourceConfig(uniqueID, "Updated Check", 7200, 600, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "paused", "false"),
				),
			},
			// Delete testing happens automatically
		},
	})
}

func TestAccCheckResource_withTags(t *testing.T) {
	uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())
	resourceName := "pakyas_check.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceConfigWithTags(uniqueID, []string{"backup", "database"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
			// Update tags
			{
				Config: testAccCheckResourceConfigWithTags(uniqueID, []string{"production", "critical"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				),
			},
		},
	})
}

func testAccCheckResourceConfig(uniqueID, name string, periodSeconds, graceSeconds int, paused bool) string {
	return fmt.Sprintf(`
resource "pakyas_project" "test" {
  name = "Test Project %[1]s"
}

resource "pakyas_check" "test" {
  project_id     = pakyas_project.test.id
  name           = "%[2]s"
  slug           = "test-check-%[1]s"
  period_seconds = %[3]d
  grace_seconds  = %[4]d
  paused         = %[5]t
}
`, uniqueID, name, periodSeconds, graceSeconds, paused)
}

func testAccCheckResourceConfigWithTags(uniqueID string, tags []string) string {
	tagList := ""
	for i, tag := range tags {
		if i > 0 {
			tagList += ", "
		}
		tagList += fmt.Sprintf(`"%s"`, tag)
	}

	return fmt.Sprintf(`
resource "pakyas_project" "test" {
  name = "Test Project %[1]s"
}

resource "pakyas_check" "test" {
  project_id     = pakyas_project.test.id
  name           = "Tagged Check"
  slug           = "tagged-check-%[1]s"
  period_seconds = 3600
  tags           = [%[2]s]
}
`, uniqueID, tagList)
}
