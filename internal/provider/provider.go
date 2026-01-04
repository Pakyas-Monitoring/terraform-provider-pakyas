package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/pakyas/terraform-provider-pakyas/internal/client"
	projectResource "github.com/pakyas/terraform-provider-pakyas/internal/resources/project"
	checkResource "github.com/pakyas/terraform-provider-pakyas/internal/resources/check"
)

// Ensure PakyasProvider satisfies various provider interfaces.
var _ provider.Provider = &PakyasProvider{}

// PakyasProvider defines the provider implementation.
type PakyasProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and run locally, and "test" when running acceptance
	// testing.
	version string
}

// PakyasProviderModel describes the provider data model.
type PakyasProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
	APIURL types.String `tfsdk:"api_url"`
}

func (p *PakyasProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pakyas"
	resp.Version = p.version
}

func (p *PakyasProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Pakyas cron job monitoring service.",
		MarkdownDescription: `
The Pakyas provider is used to manage [Pakyas](https://pakyas.com) cron job monitoring resources.

## Authentication

The provider requires an API key to authenticate. You can provide it via:

1. The ` + "`api_key`" + ` provider attribute
2. The ` + "`PAKYAS_API_KEY`" + ` environment variable

API keys can be created in the Pakyas dashboard under Settings > API Keys.

## Example Usage

` + "```hcl" + `
provider "pakyas" {
  api_key = var.pakyas_api_key
  # api_url = "https://api.pakyas.com"  # Optional, defaults to production
}

resource "pakyas_project" "prod" {
  name        = "Production"
  description = "Production cron jobs"
}

resource "pakyas_check" "daily_backup" {
  project_id     = pakyas_project.prod.id
  name           = "Daily Backup"
  slug           = "daily-backup"
  period_seconds = 86400
  grace_seconds  = 3600
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description:         "API key for Pakyas authentication. Can also be set via PAKYAS_API_KEY environment variable.",
				MarkdownDescription: "API key for Pakyas authentication. Can also be set via `PAKYAS_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				Description:         "Base URL for the Pakyas API. Defaults to https://api.pakyas.com. Can also be set via PAKYAS_API_URL environment variable.",
				MarkdownDescription: "Base URL for the Pakyas API. Defaults to `https://api.pakyas.com`. Can also be set via `PAKYAS_API_URL` environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *PakyasProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Pakyas provider")

	var config PakyasProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Determine API key
	apiKey := os.Getenv("PAKYAS_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Pakyas API Key",
			"The provider cannot create the Pakyas API client as there is a missing or empty value for the Pakyas API key. "+
				"Set the api_key value in the configuration or use the PAKYAS_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
		return
	}

	// Determine API URL
	apiURL := os.Getenv("PAKYAS_API_URL")
	if !config.APIURL.IsNull() {
		apiURL = config.APIURL.ValueString()
	}
	if apiURL == "" {
		apiURL = client.DefaultBaseURL
	}

	tflog.Debug(ctx, "Creating Pakyas client", map[string]interface{}{
		"api_url": apiURL,
	})

	// Create client
	c, err := client.New(ctx, client.ClientConfig{
		APIKey:    apiKey,
		BaseURL:   apiURL,
		UserAgent: "terraform-provider-pakyas/" + p.version,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Pakyas API Client",
			"An unexpected error occurred when creating the Pakyas API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Pakyas Client Error: "+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Pakyas provider configured", map[string]interface{}{
		"org_id":        c.OrgID(),
		"ping_url_base": c.PingURLBase(),
	})

	// Make the client available to resources and data sources
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *PakyasProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		projectResource.NewProjectResource,
		checkResource.NewCheckResource,
	}
}

func (p *PakyasProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Data sources are post-MVP
	}
}

// New returns a new provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PakyasProvider{
			version: version,
		}
	}
}
