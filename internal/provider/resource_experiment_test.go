package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccExperimentResource(t *testing.T) {
	if os.Getenv("POSTHOG_PROJECT_ID") == "" {
		t.Skip("POSTHOG_PROJECT_ID not set, skipping experiment acceptance test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExperimentResourceConfig("Test Experiment", "test-flag-acc"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("posthog_experiment.test", "id"),
					resource.TestCheckResourceAttr("posthog_experiment.test", "name", "Test Experiment"),
					resource.TestCheckResourceAttr("posthog_experiment.test", "feature_flag_key", "test-flag-acc"),
					resource.TestCheckResourceAttrSet("posthog_experiment.test", "feature_flag_id"),
				),
			},
			{
				ResourceName:      "posthog_experiment.test",
				ImportState:       true,
				ImportStateIdFunc: experimentImportIdFunc,
				ImportStateVerify: true,
			},
			{
				Config: testAccExperimentResourceConfig("Updated Experiment", "test-flag-acc"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("posthog_experiment.test", "id"),
					resource.TestCheckResourceAttr("posthog_experiment.test", "name", "Updated Experiment"),
				),
			},
		},
	})
}

func testAccExperimentResourceConfig(name string, flagKey string) string {
	return fmt.Sprintf(`
resource "posthog_experiment" "test" {
  project_id       = %s
  name             = "%s"
  feature_flag_key = "%s"

  variants = {
    control = {
      percentage = 50
    }
    test = {
      percentage = 50
    }
  }
}
`, testAccProjectId(), name, flagKey)
}

func experimentImportIdFunc(state *terraform.State) (string, error) {
	rawState, ok := state.RootModule().Resources["posthog_experiment.test"]
	if !ok {
		return "", fmt.Errorf("Resource Not found")
	}
	return fmt.Sprintf("%s:%s", rawState.Primary.Attributes["project_id"], rawState.Primary.Attributes["id"]), nil
}
