package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*subscriptionResource)(nil)
	_ resource.ResourceWithImportState = (*subscriptionResource)(nil)
)

// subscriptionResource manages a Hivehook subscription that links a source
// to a destination, optionally with filters and a transformation.
type subscriptionResource struct {
	client *Client
}

// subscriptionResourceModel is the Terraform state representation of a subscription.
type subscriptionResourceModel struct {
	ID               types.String         `tfsdk:"id"`
	Name             types.String         `tfsdk:"name"`
	SourceID         types.String         `tfsdk:"source_id"`
	DestinationID    types.String         `tfsdk:"destination_id"`
	FilterConfig     jsontypes.Normalized `tfsdk:"filter_config"`
	TransformationID types.String         `tfsdk:"transformation_id"`
	Enabled          types.Bool           `tfsdk:"enabled"`
}

func (r *subscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subscription"
}

// Note: source_id and destination_id do NOT require replacement. The server's
// UpdateSubscriptionInput accepts both fields, so they can be mutated in place.
func (r *subscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     0,
		Description: "Manages a Hivehook subscription linking a source to a destination.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "Subscription UUID.",
			},
			"name":           schema.StringAttribute{Required: true, Description: "Display name."},
			"source_id":      schema.StringAttribute{Required: true, Description: "Source UUID to subscribe to."},
			"destination_id": schema.StringAttribute{Required: true, Description: "Destination UUID to deliver to."},
			"filter_config": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON-encoded filter configuration.",
			},
			"transformation_id": schema.StringAttribute{Optional: true, Description: "Optional transformation UUID."},
			"enabled":           schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the subscription is enabled."},
		},
	}
}

func (r *subscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	r.client = client
}

func (r *subscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

const subFields = `id name sourceId destinationId filterConfig transformationId enabled`

func (r *subscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan subscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{
		"name":          plan.Name.ValueString(),
		"sourceId":      plan.SourceID.ValueString(),
		"destinationId": plan.DestinationID.ValueString(),
	}
	if err := setOptionalNormalizedJSON(input, "filterConfig", plan.FilterConfig); err != nil {
		resp.Diagnostics.AddError("Invalid filter_config", err.Error())
		return
	}
	setOptionalString(input, "transformationId", plan.TransformationID)
	setOptionalBool(input, "enabled", plan.Enabled)

	query := `mutation($input: CreateSubscriptionInput!) { createSubscription(input: $input) { ` + subFields + ` } }`

	var result struct {
		CreateSubscription subscriptionGQL `json:"createSubscription"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to create subscription", err.Error())
		return
	}

	mapSubToState(&plan, &result.CreateSubscription)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *subscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state subscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { subscription(id: $id) { ` + subFields + ` } }`
	var result struct {
		Subscription *subscriptionGQL `json:"subscription"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read subscription", err.Error())
		return
	}
	if result.Subscription == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapSubToState(&state, result.Subscription)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update issues the updateSubscription mutation, sending only the fields that
// differ between plan and state. source_id and destination_id are included
// when changed because UpdateSubscriptionInput accepts them on the server
// (no RequiresReplace on those attributes).
func (r *subscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan subscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state subscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{}
	if !plan.Name.Equal(state.Name) {
		input["name"] = plan.Name.ValueString()
	}
	if !plan.SourceID.Equal(state.SourceID) {
		input["sourceId"] = plan.SourceID.ValueString()
	}
	if !plan.DestinationID.Equal(state.DestinationID) {
		input["destinationId"] = plan.DestinationID.ValueString()
	}
	if !plan.Enabled.Equal(state.Enabled) {
		setOptionalBool(input, "enabled", plan.Enabled)
	}
	if !plan.FilterConfig.Equal(state.FilterConfig) {
		if err := setOptionalNormalizedJSON(input, "filterConfig", plan.FilterConfig); err != nil {
			resp.Diagnostics.AddError("Invalid filter_config", err.Error())
			return
		}
	}
	if !plan.TransformationID.Equal(state.TransformationID) {
		setOptionalString(input, "transformationId", plan.TransformationID)
	}

	if len(input) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	query := `mutation($id: UUID!, $input: UpdateSubscriptionInput!) { updateSubscription(id: $id, input: $input) { ` + subFields + ` } }`

	var result struct {
		UpdateSubscription subscriptionGQL `json:"updateSubscription"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString(), "input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to update subscription", err.Error())
		return
	}

	mapSubToState(&plan, &result.UpdateSubscription)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *subscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state subscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: UUID!) { deleteSubscription(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Failed to delete subscription", err.Error())
	}
}

// subscriptionGQL is the GraphQL response shape for a subscription.
type subscriptionGQL struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	SourceID         string          `json:"sourceId"`
	DestinationID    string          `json:"destinationId"`
	FilterConfig     json.RawMessage `json:"filterConfig"`
	TransformationID *string         `json:"transformationId"`
	Enabled          bool            `json:"enabled"`
}

func mapSubToState(state *subscriptionResourceModel, s *subscriptionGQL) {
	state.ID = types.StringValue(s.ID)
	state.Name = types.StringValue(s.Name)
	state.SourceID = types.StringValue(s.SourceID)
	state.DestinationID = types.StringValue(s.DestinationID)
	state.Enabled = types.BoolValue(s.Enabled)

	// Pass the raw server bytes straight through so key order is preserved and
	// plan diffs are deterministic.
	state.FilterConfig = normalizedFromRaw(s.FilterConfig)
	if s.TransformationID != nil {
		state.TransformationID = types.StringValue(*s.TransformationID)
	} else {
		state.TransformationID = types.StringNull()
	}
}
