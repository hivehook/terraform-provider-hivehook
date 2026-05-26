package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type sourceDataSource struct {
	client *Client
}

type sourceDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Slug         types.String `tfsdk:"slug"`
	ProviderType types.String `tfsdk:"provider_type"`
	Status       types.String `tfsdk:"status"`
	RateLimitRPS types.Int64  `tfsdk:"rate_limit_rps"`
}

func (d *sourceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (d *sourceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a Hivehook source by ID.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Required: true},
			"name":           schema.StringAttribute{Computed: true},
			"slug":           schema.StringAttribute{Computed: true},
			"provider_type":  schema.StringAttribute{Computed: true},
			"status":         schema.StringAttribute{Computed: true},
			"rate_limit_rps": schema.Int64Attribute{Computed: true},
		},
	}
}

func (d *sourceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	d.client = client
}

func (d *sourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sourceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { source(id: $id) { id name slug providerType status rateLimitRps } }`
	var result struct {
		Source *sourceGQL `json:"source"`
	}
	if err := d.client.Execute(ctx, query, map[string]any{"id": config.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read source", err.Error())
		return
	}
	if result.Source == nil {
		resp.Diagnostics.AddError("Source not found", "No source found with the given ID")
		return
	}

	config.Name = types.StringValue(result.Source.Name)
	config.Slug = types.StringValue(result.Source.Slug)
	config.ProviderType = types.StringValue(result.Source.ProviderType)
	config.Status = types.StringValue(result.Source.Status)
	config.RateLimitRPS = types.Int64Value(result.Source.RateLimitRPS)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

type destinationDataSource struct {
	client *Client
}

type destinationDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	URL          types.String `tfsdk:"url"`
	Status       types.String `tfsdk:"status"`
	AuthType     types.String `tfsdk:"auth_type"`
	DeliveryMode types.String `tfsdk:"delivery_mode"`
	Ordered      types.Bool   `tfsdk:"ordered"`
}

func (d *destinationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (d *destinationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a Hivehook destination by ID.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Required: true},
			"name":          schema.StringAttribute{Computed: true},
			"url":           schema.StringAttribute{Computed: true},
			"status":        schema.StringAttribute{Computed: true},
			"auth_type":     schema.StringAttribute{Computed: true},
			"delivery_mode": schema.StringAttribute{Computed: true},
			"ordered":       schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *destinationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	d.client = client
}

func (d *destinationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config destinationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { destination(id: $id) { id name url status authType deliveryMode ordered } }`
	var result struct {
		Destination *destinationGQL `json:"destination"`
	}
	if err := d.client.Execute(ctx, query, map[string]any{"id": config.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read destination", err.Error())
		return
	}
	if result.Destination == nil {
		resp.Diagnostics.AddError("Destination not found", "No destination found with the given ID")
		return
	}

	config.Name = types.StringValue(result.Destination.Name)
	config.URL = types.StringValue(result.Destination.URL)
	config.Status = types.StringValue(result.Destination.Status)
	config.AuthType = types.StringValue(result.Destination.AuthType)
	config.DeliveryMode = types.StringValue(result.Destination.DeliveryMode)
	config.Ordered = types.BoolValue(result.Destination.Ordered)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

type applicationDataSource struct {
	client *Client
}

type applicationDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	UID  types.String `tfsdk:"uid"`
}

func (d *applicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (d *applicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a Hivehook application by ID.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Required: true},
			"name": schema.StringAttribute{Computed: true},
			"uid":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *applicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	d.client = client
}

func (d *applicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config applicationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { application(id: $id) { id name uid } }`
	var result struct {
		Application *applicationGQL `json:"application"`
	}
	if err := d.client.Execute(ctx, query, map[string]any{"id": config.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read application", err.Error())
		return
	}
	if result.Application == nil {
		resp.Diagnostics.AddError("Application not found", "No application found with the given ID")
		return
	}

	config.Name = types.StringValue(result.Application.Name)
	config.UID = types.StringValue(result.Application.UID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

type endpointDataSource struct {
	client *Client
}

type endpointDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	ApplicationID types.String `tfsdk:"application_id"`
	URL           types.String `tfsdk:"url"`
	Status        types.String `tfsdk:"status"`
	AuthType      types.String `tfsdk:"auth_type"`
	DeliveryMode  types.String `tfsdk:"delivery_mode"`
	Ordered       types.Bool   `tfsdk:"ordered"`
	FilterConfig  types.String `tfsdk:"filter_config"`
}

func (d *endpointDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_endpoint"
}

func (d *endpointDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a Hivehook endpoint by ID.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Required: true},
			"application_id": schema.StringAttribute{Computed: true},
			"url":            schema.StringAttribute{Computed: true},
			"status":         schema.StringAttribute{Computed: true},
			"auth_type":      schema.StringAttribute{Computed: true},
			"delivery_mode":  schema.StringAttribute{Computed: true},
			"ordered":        schema.BoolAttribute{Computed: true},
			"filter_config":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *endpointDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	d.client = client
}

func (d *endpointDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config endpointDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { endpoint(id: $id) { id applicationId url status authType deliveryMode ordered filterConfig } }`
	var result struct {
		Endpoint *endpointGQL `json:"endpoint"`
	}
	if err := d.client.Execute(ctx, query, map[string]any{"id": config.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read endpoint", err.Error())
		return
	}
	if result.Endpoint == nil {
		resp.Diagnostics.AddError("Endpoint not found", "No endpoint found with the given ID")
		return
	}

	config.ApplicationID = types.StringValue(result.Endpoint.ApplicationID)
	config.URL = types.StringValue(result.Endpoint.URL)
	config.Status = types.StringValue(result.Endpoint.Status)
	config.AuthType = types.StringValue(result.Endpoint.AuthType)
	config.DeliveryMode = types.StringValue(result.Endpoint.DeliveryMode)
	config.Ordered = types.BoolValue(result.Endpoint.Ordered)
	if result.Endpoint.FilterConfig != nil {
		b, _ := json.Marshal(result.Endpoint.FilterConfig)
		config.FilterConfig = types.StringValue(string(b))
	} else {
		config.FilterConfig = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
