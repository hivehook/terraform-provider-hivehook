package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*transformationResource)(nil)
	_ resource.ResourceWithImportState = (*transformationResource)(nil)
)

// transformationResource manages a Hivehook JavaScript transformation.
type transformationResource struct {
	client *Client
}

// transformationResourceModel is the Terraform state representation of a transformation.
type transformationResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Code        types.String `tfsdk:"code"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	FailOpen    types.Bool   `tfsdk:"fail_open"`
	TimeoutMs   types.Int64  `tfsdk:"timeout_ms"`
}

func (r *transformationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transformation"
}

func (r *transformationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     0,
		Description: "Manages a Hivehook JavaScript transformation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "Transformation UUID.",
			},
			"name":        schema.StringAttribute{Required: true, Description: "Display name."},
			"description": schema.StringAttribute{Optional: true, Computed: true, Description: "Optional description."},
			"code":        schema.StringAttribute{Required: true, Description: "JavaScript transformation code."},
			"enabled":     schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the transformation is active."},
			"fail_open":   schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "If true, fall through on errors instead of dropping the event."},
			"timeout_ms":  schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1000), Description: "Per-invocation timeout in milliseconds."},
		},
	}
}

func (r *transformationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	r.client = client
}

func (r *transformationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

const txFields = `id name description code enabled failOpen timeoutMs`

func (r *transformationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan transformationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// enabled has a schema default of true, so the framework resolves it to a
	// concrete bool before Create runs. Always forward it so the server is
	// never left to guess and a user-configured `enabled = false` is honoured
	// on creation.
	input := map[string]any{
		"name":    plan.Name.ValueString(),
		"code":    plan.Code.ValueString(),
		"enabled": plan.Enabled.ValueBool(),
	}
	setOptionalString(input, "description", plan.Description)
	if !plan.FailOpen.IsNull() && !plan.FailOpen.IsUnknown() {
		input["failOpen"] = plan.FailOpen.ValueBool()
	}
	if !plan.TimeoutMs.IsNull() && !plan.TimeoutMs.IsUnknown() {
		input["timeoutMs"] = plan.TimeoutMs.ValueInt64()
	}

	query := `mutation($input: CreateTransformationInput!) { createTransformation(input: $input) { ` + txFields + ` } }`
	var result struct {
		CreateTransformation transformationGQL `json:"createTransformation"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to create transformation", err.Error())
		return
	}

	mapTxToState(&plan, &result.CreateTransformation)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *transformationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state transformationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { transformation(id: $id) { ` + txFields + ` } }`
	var result struct {
		Transformation *transformationGQL `json:"transformation"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read transformation", err.Error())
		return
	}
	if result.Transformation == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapTxToState(&state, result.Transformation)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *transformationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan transformationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state transformationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{
		"name":    plan.Name.ValueString(),
		"code":    plan.Code.ValueString(),
		"enabled": plan.Enabled.ValueBool(),
	}
	setOptionalString(input, "description", plan.Description)
	if !plan.FailOpen.IsNull() && !plan.FailOpen.IsUnknown() {
		input["failOpen"] = plan.FailOpen.ValueBool()
	}
	if !plan.TimeoutMs.IsNull() && !plan.TimeoutMs.IsUnknown() {
		input["timeoutMs"] = plan.TimeoutMs.ValueInt64()
	}

	query := `mutation($id: UUID!, $input: UpdateTransformationInput!) { updateTransformation(id: $id, input: $input) { ` + txFields + ` } }`
	var result struct {
		UpdateTransformation transformationGQL `json:"updateTransformation"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString(), "input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to update transformation", err.Error())
		return
	}

	mapTxToState(&plan, &result.UpdateTransformation)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *transformationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state transformationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: UUID!) { deleteTransformation(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Failed to delete transformation", err.Error())
	}
}

// transformationGQL is the GraphQL response shape for a transformation.
type transformationGQL struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Enabled     bool   `json:"enabled"`
	FailOpen    bool   `json:"failOpen"`
	TimeoutMs   int64  `json:"timeoutMs"`
}

func mapTxToState(state *transformationResourceModel, tx *transformationGQL) {
	state.ID = types.StringValue(tx.ID)
	state.Name = types.StringValue(tx.Name)
	state.Description = types.StringValue(tx.Description)
	state.Code = types.StringValue(tx.Code)
	state.Enabled = types.BoolValue(tx.Enabled)
	state.FailOpen = types.BoolValue(tx.FailOpen)
	state.TimeoutMs = types.Int64Value(tx.TimeoutMs)
}
