package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*applicationResource)(nil)
	_ resource.ResourceWithImportState = (*applicationResource)(nil)
)

// applicationResource manages a Hivehook outbound application.
type applicationResource struct {
	client *Client
}

// applicationResourceModel is the Terraform state representation of an application.
type applicationResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	UID  types.String `tfsdk:"uid"`
}

func (r *applicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *applicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     0,
		Description: "Manages a Hivehook outbound application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "Application UUID.",
			},
			"name": schema.StringAttribute{Required: true, Description: "Display name."},
			"uid": schema.StringAttribute{
				Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "Stable user-facing identifier used in outbound URLs.",
			},
		},
	}
}

func (r *applicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	r.client = client
}

func (r *applicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *applicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan applicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($input: CreateApplicationInput!) { createApplication(input: $input) { id name uid } }`
	var result struct {
		CreateApplication applicationGQL `json:"createApplication"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": map[string]any{"name": plan.Name.ValueString()}}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to create application", err.Error())
		return
	}

	mapAppToState(&plan, &result.CreateApplication)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state applicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { application(id: $id) { id name uid } }`
	var result struct {
		Application *applicationGQL `json:"application"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read application", err.Error())
		return
	}
	if result.Application == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapAppToState(&state, result.Application)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *applicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan applicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state applicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: UUID!, $input: UpdateApplicationInput!) { updateApplication(id: $id, input: $input) { id name uid } }`
	var result struct {
		UpdateApplication applicationGQL `json:"updateApplication"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString(), "input": map[string]any{"name": plan.Name.ValueString()}}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to update application", err.Error())
		return
	}

	mapAppToState(&plan, &result.UpdateApplication)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *applicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state applicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: UUID!) { deleteApplication(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Failed to delete application", err.Error())
	}
}

// applicationGQL is the GraphQL response shape for an application.
type applicationGQL struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	UID  string `json:"uid"`
}

func mapAppToState(state *applicationResourceModel, a *applicationGQL) {
	state.ID = types.StringValue(a.ID)
	state.Name = types.StringValue(a.Name)
	state.UID = types.StringValue(a.UID)
}
