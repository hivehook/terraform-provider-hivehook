// Package provider implements the Terraform provider for Hivehook.
package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// hivehookProvider is the Hivehook Terraform provider implementation.
type hivehookProvider struct {
	client *Client
}

// hivehookProviderModel maps provider schema attributes to a Go type.
type hivehookProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
}

// Version is the provider version, set at build time via -ldflags.
var Version = "dev"

// New returns a new Hivehook provider instance.
func New() provider.Provider {
	return &hivehookProvider{}
}

// Metadata returns the provider type name.
func (p *hivehookProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hivehook"
	resp.Version = Version
}

// Schema defines the provider-level configuration schema.
func (p *hivehookProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Hivehook webhook gateway resources.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "Hivehook endpoint. Defaults to https://app.hivehook.com. Override for enterprise self-host deployments. Can also be set via HIVEHOOK_URL env var.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "API key for authentication. Can also be set via HIVEHOOK_API_KEY env var.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// ValidateConfig emits a warning when no API key is configured via either
// provider configuration or the HIVEHOOK_API_KEY environment variable.
func (p *hivehookProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var config hivehookProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	configKeyEmpty := config.APIKey.IsNull() || config.APIKey.IsUnknown() || config.APIKey.ValueString() == ""
	envKeyEmpty := os.Getenv("HIVEHOOK_API_KEY") == ""
	if configKeyEmpty && envKeyEmpty {
		resp.Diagnostics.AddWarning(
			"Hivehook API key not configured",
			"No API key was provided via the api_key argument or the HIVEHOOK_API_KEY environment variable. "+
				"Requests to the Hivehook API will be unauthenticated and likely rejected.",
		)
	}

	// Validate the endpoint URL when supplied via provider configuration so
	// we surface obvious typos at plan time rather than at the first request.
	if !config.Endpoint.IsNull() && !config.Endpoint.IsUnknown() {
		endpoint := config.Endpoint.ValueString()
		if endpoint != "" {
			u, err := url.Parse(endpoint)
			if err != nil {
				resp.Diagnostics.AddAttributeError(
					path.Root("endpoint"),
					"Invalid Hivehook endpoint URL",
					fmt.Sprintf("Failed to parse %q: %s", endpoint, err),
				)
				return
			}
			scheme := strings.ToLower(u.Scheme)
			if scheme != "http" && scheme != "https" {
				resp.Diagnostics.AddAttributeError(
					path.Root("endpoint"),
					"Invalid Hivehook endpoint scheme",
					fmt.Sprintf("Endpoint %q must use the http or https scheme, got %q.", endpoint, u.Scheme),
				)
				return
			}
			if u.Host == "" {
				resp.Diagnostics.AddAttributeError(
					path.Root("endpoint"),
					"Invalid Hivehook endpoint",
					fmt.Sprintf("Endpoint %q is missing a host component.", endpoint),
				)
				return
			}
		}
	}
}

// Configure establishes a shared client used by every resource and data source.
func (p *hivehookProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config hivehookProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := config.Endpoint.ValueString()
	if endpoint == "" {
		endpoint = os.Getenv("HIVEHOOK_URL")
	}
	if endpoint == "" {
		endpoint = "https://app.hivehook.com"
	}

	apiKey := config.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("HIVEHOOK_API_KEY")
	}

	tflog.Info(ctx, "configuring Hivehook client", map[string]any{"endpoint": endpoint})

	p.client = NewClient(endpoint, apiKey)
	resp.ResourceData = p.client
	resp.DataSourceData = p.client
}

// Resources returns the set of managed resources offered by the provider.
func (p *hivehookProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		func() resource.Resource { return &sourceResource{} },
		func() resource.Resource { return &destinationResource{} },
		func() resource.Resource { return &subscriptionResource{} },
		func() resource.Resource { return &applicationResource{} },
		func() resource.Resource { return &endpointResource{} },
		func() resource.Resource { return &apiKeyResource{} },
		func() resource.Resource { return &alertRuleResource{} },
		func() resource.Resource { return &transformationResource{} },
	}
}

// DataSources returns the set of data sources offered by the provider.
func (p *hivehookProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return &sourceDataSource{} },
		func() datasource.DataSource { return &destinationDataSource{} },
		func() datasource.DataSource { return &applicationDataSource{} },
		func() datasource.DataSource { return &endpointDataSource{} },
	}
}

// configureClient safely extracts the shared *Client from ProviderData.
// It is used by every resource and data source Configure method.
func configureClient(providerData any) (*Client, error) {
	if providerData == nil {
		return nil, nil
	}
	client, ok := providerData.(*Client)
	if !ok {
		return nil, fmt.Errorf("expected *Client, got %T", providerData)
	}
	return client, nil
}
