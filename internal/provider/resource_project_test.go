package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func tokenRegex() *regexp.Regexp {
	return regexp.MustCompile("^phc_")
}

func TestAccProjectResourceDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfigDefault("Todo"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("posthog_project.test", "id"),
					resource.TestCheckResourceAttr("posthog_project.test", "name", "Todo"),
					resource.TestCheckResourceAttr("posthog_project.test", "organization_id", "0190127c-7e34-0000-ca15-525976388b8d"),
					resource.TestMatchResourceAttr("posthog_project.test", "token", tokenRegex()),
				),
			},
			// ImportState testing
			{
				ResourceName:      "posthog_project.test",
				ImportState:       true,
				ImportStateIdFunc: projectImportIdFunc,
				ImportStateVerify: true,
			},
			// Update with default values
			{
				Config: testAccProjectResourceConfigDefault("Todo"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("posthog_project.test", "id"),
					resource.TestCheckResourceAttr("posthog_project.test", "name", "Todo"),
					resource.TestCheckResourceAttr("posthog_project.test", "organization_id", "0190127c-7e34-0000-ca15-525976388b8d"),
					resource.TestMatchResourceAttr("posthog_project.test", "token", tokenRegex()),
				),
			},
			// Update and Read testing
			{
				Config: testAccProjectResourceConfigNonDefault("Todo app"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("posthog_project.test", "id"),
					resource.TestCheckResourceAttr("posthog_project.test", "name", "Todo app"),
					resource.TestCheckResourceAttr("posthog_project.test", "organization_id", "0190127c-7e34-0000-ca15-525976388b8d"),
					resource.TestMatchResourceAttr("posthog_project.test", "token", tokenRegex()),
				),
			},
			// ImportState testing
			{
				ResourceName:      "posthog_project.test",
				ImportState:       true,
				ImportStateIdFunc: projectImportIdFunc,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectResourceConfigDefault(title string) string {
	return fmt.Sprintf(`
resource "posthog_project" "test" {
  name = "%s"

  organization_id = "0190127c-7e34-0000-ca15-525976388b8d"
}
`, title)
}

func testAccProjectResourceConfigNonDefault(title string) string {
	return fmt.Sprintf(`
resource "posthog_project" "test" {
  name = "%s"

  organization_id = "0190127c-7e34-0000-ca15-525976388b8d"
}
`, title)
}

func projectImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["posthog_project.test"]

	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}

	return fmt.Sprintf("%s:%s", rawState.Primary.Attributes["organization_id"], rawState.Primary.Attributes["id"]), nil
}
