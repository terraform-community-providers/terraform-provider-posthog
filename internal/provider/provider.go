package provider

import (
	"context"
	"net/http"
	"os"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	envVarName          = "POSTHOG_TOKEN"
	errMissingAuthToken = "Required token could not be found. Please set the token using an input variable in the provider configuration block or by using the `" + envVarName + "` environment variable."
	defaultPosthogHost  = "app.posthog.com"
)

func uuidRegex() *regexp.Regexp {
	return regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
}

var _ provider.Provider = &PostHogProvider{}

type PostHogProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type PostHogProviderModel struct {
	Token types.String `tfsdk:"token"`
	Host  types.String `tfsdk:"host"`
}

func (p *PostHogProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "posthog"
	resp.Version = p.version
}

func (p *PostHogProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "The token used to authenticate with PostHog.",
				Optional:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "The host for the PostHog API. **Default** `" + defaultPosthogHost + "`",
				Optional:            true,
			},
		},
	}
}

func (p *PostHogProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data PostHogProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	token := ""
	host := defaultPosthogHost

	if !data.Token.IsNull() {
		token = data.Token.ValueString()
	}

	// If a token wasn't set in the provider configuration block, try and fetch it
	// from the environment variable.
	if token == "" {
		token = os.Getenv(envVarName)
	}

	// If we still don't have a token at this point, we return an error.
	if token == "" {
		resp.Diagnostics.AddError("Missing API token", errMissingAuthToken)
		return
	}

	if !data.Host.IsNull() {
		host = data.Host.ValueString()
	}

	client := http.Client{
		Transport: &authedTransport{
			token:   token,
			host:    host,
			wrapped: http.DefaultTransport,
		},
	}

	resp.DataSourceData = &client
	resp.ResourceData = &client
}

func (p *PostHogProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewExperimentResource,
	}
}

func (p *PostHogProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PostHogProvider{
			version: version,
		}
	}
}
