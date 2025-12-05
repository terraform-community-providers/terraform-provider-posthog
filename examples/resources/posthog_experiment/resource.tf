resource "posthog_experiment" "checkout" {
  project_id       = 12345
  name             = "Checkout flow test"
  feature_flag_key = "checkout-experiment"

  variants = {
    control = {
      percentage = 50
    }
    test = {
      percentage = 50
    }
  }
}
