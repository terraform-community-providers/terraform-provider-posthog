---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "posthog_project Resource - terraform-provider-posthog"
subcategory: ""
description: |-
  PostHog project.
---

# posthog_project (Resource)

PostHog project.

## Example Usage

```terraform
resource "posthog_project" "platform" {
  name            = "example"
  organization_id = "example"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the project.
- `organization_id` (String) Identifier of the organization the project belongs to.

### Read-Only

- `id` (Number) Identifier of the project.
- `token` (String) API token of the project.

## Import

Import is supported using the following syntax:

```shell
terraform import posthog_project.platform platform
```