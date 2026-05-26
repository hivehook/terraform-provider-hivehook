package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure sourceResource implements the full set of optional interfaces.
var (
	_ resource.Resource                = (*sourceResource)(nil)
	_ resource.ResourceWithImportState = (*sourceResource)(nil)
)

// sourceResource manages a Hivehook inbound webhook source.
type sourceResource struct {
	client *Client
}

// sourceResourceModel is the Terraform state representation of a source.
type sourceResourceModel struct {
	ID           types.String         `tfsdk:"id"`
	Name         types.String         `tfsdk:"name"`
	Slug         types.String         `tfsdk:"slug"`
	ProviderType types.String         `tfsdk:"provider_type"`
	VerifyConfig jsontypes.Normalized `tfsdk:"verify_config"`
	Status       types.String         `tfsdk:"status"`
	RateLimitRPS types.Int64          `tfsdk:"rate_limit_rps"`
}

func (r *sourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (r *sourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     0,
		Description: "Manages a Hivehook inbound webhook source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Source UUID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name.",
			},
			"slug": schema.StringAttribute{
				Required:    true,
				Description: "URL slug for ingest endpoint (/ingest/{slug}).",
			},
			"provider_type": schema.StringAttribute{
				Required:      true,
				Description:   "Webhook provider (e.g. generic, stripe, github).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"verify_config": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON-encoded provider verification config.",
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Source status (ACTIVE or INACTIVE).",
			},
			"rate_limit_rps": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Rate limit in requests per second (0 = unlimited).",
			},
		},
	}
}

func (r *sourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	r.client = client
}

func (r *sourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *sourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{
		"name":         plan.Name.ValueString(),
		"slug":         plan.Slug.ValueString(),
		"providerType": plan.ProviderType.ValueString(),
	}
	if err := setOptionalNormalizedJSON(input, "verifyConfig", plan.VerifyConfig); err != nil {
		resp.Diagnostics.AddError("Invalid verify_config", err.Error())
		return
	}
	setOptionalString(input, "status", plan.Status)
	setOptionalInt64(input, "rateLimitRps", plan.RateLimitRPS)

	query := `mutation($input: CreateSourceInput!) {
		createSource(input: $input) {
			id name slug providerType verifyConfig status rateLimitRps
		}
	}`

	var result struct {
		CreateSource sourceGQL `json:"createSource"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to create source", err.Error())
		return
	}

	mapSourceToState(&plan, &result.CreateSource)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) {
		source(id: $id) {
			id name slug providerType verifyConfig status rateLimitRps
		}
	}`

	var result struct {
		Source *sourceGQL `json:"source"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read source", err.Error())
		return
	}
	if result.Source == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapSourceToState(&state, result.Source)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update issues the updateSource mutation, sending only the fields that
// differ between plan and state. This avoids overwriting server-computed
// fields and minimises wire traffic.
func (r *sourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan sourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state sourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// provider_type has RequiresReplace, so it never changes in an Update.
	input := map[string]any{}
	if !plan.Name.Equal(state.Name) {
		input["name"] = plan.Name.ValueString()
	}
	if !plan.Slug.Equal(state.Slug) {
		input["slug"] = plan.Slug.ValueString()
	}
	if !plan.Status.Equal(state.Status) {
		setOptionalString(input, "status", plan.Status)
	}
	if !plan.RateLimitRPS.Equal(state.RateLimitRPS) {
		setOptionalInt64(input, "rateLimitRps", plan.RateLimitRPS)
	}
	if !plan.VerifyConfig.Equal(state.VerifyConfig) {
		if err := setOptionalNormalizedJSON(input, "verifyConfig", plan.VerifyConfig); err != nil {
			resp.Diagnostics.AddError("Invalid verify_config", err.Error())
			return
		}
	}

	if len(input) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	query := `mutation($id: UUID!, $input: UpdateSourceInput!) {
		updateSource(id: $id, input: $input) {
			id name slug providerType verifyConfig status rateLimitRps
		}
	}`

	var result struct {
		UpdateSource sourceGQL `json:"updateSource"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString(), "input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to update source", err.Error())
		return
	}

	mapSourceToState(&plan, &result.UpdateSource)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *sourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: UUID!) { deleteSource(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Failed to delete source", err.Error())
	}
}

// sourceGQL is the GraphQL response shape for a source.
type sourceGQL struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	ProviderType string          `json:"providerType"`
	VerifyConfig json.RawMessage `json:"verifyConfig"`
	Status       string          `json:"status"`
	RateLimitRPS int64           `json:"rateLimitRps"`
}

func mapSourceToState(state *sourceResourceModel, src *sourceGQL) {
	state.ID = types.StringValue(src.ID)
	state.Name = types.StringValue(src.Name)
	state.Slug = types.StringValue(src.Slug)
	state.ProviderType = types.StringValue(src.ProviderType)
	state.Status = types.StringValue(src.Status)
	state.RateLimitRPS = types.Int64Value(src.RateLimitRPS)

	// Pass the raw server bytes straight through so key order is preserved and
	// plan diffs are deterministic.
	state.VerifyConfig = normalizedFromRaw(src.VerifyConfig)
}
