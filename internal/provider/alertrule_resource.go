package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*alertRuleResource)(nil)
	_ resource.ResourceWithImportState = (*alertRuleResource)(nil)
)

// alertRuleResource manages a Hivehook alert rule.
type alertRuleResource struct {
	client *Client
}

// alertRuleResourceModel is the Terraform state representation of an alert rule.
type alertRuleResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	ConditionType types.String `tfsdk:"condition_type"`
	Threshold     types.Int64  `tfsdk:"threshold"`
	WebhookURL    types.String `tfsdk:"webhook_url"`
	Cooldown      types.String `tfsdk:"cooldown"`
	Enabled       types.Bool   `tfsdk:"enabled"`
}

func (r *alertRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (r *alertRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     0,
		Description: "Manages a Hivehook alert rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "Alert rule UUID.",
			},
			"name":           schema.StringAttribute{Required: true, Description: "Display name."},
			"condition_type": schema.StringAttribute{Required: true, Description: "Alert condition (e.g. dlq_size_exceeded)."},
			"threshold":      schema.Int64Attribute{Required: true, Description: "Numeric threshold for the condition."},
			"webhook_url":    schema.StringAttribute{Required: true, Description: "Webhook URL to notify on trigger."},
			"cooldown":       schema.StringAttribute{Optional: true, Computed: true, Description: "Duration between repeat alerts (e.g. 1h)."},
			"enabled":        schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the rule is active."},
		},
	}
}

func (r *alertRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	r.client = client
}

func (r *alertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

const alertFields = `id name conditionType threshold webhookUrl cooldown enabled`

func (r *alertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{
		"name":          plan.Name.ValueString(),
		"conditionType": plan.ConditionType.ValueString(),
		"threshold":     plan.Threshold.ValueInt64(),
		"webhookUrl":    plan.WebhookURL.ValueString(),
	}
	setOptionalString(input, "cooldown", plan.Cooldown)
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		input["enabled"] = plan.Enabled.ValueBool()
	}

	query := `mutation($input: CreateAlertRuleInput!) { createAlertRule(input: $input) { ` + alertFields + ` } }`
	var result struct {
		CreateAlertRule alertRuleGQL `json:"createAlertRule"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to create alert rule", err.Error())
		return
	}

	mapAlertToState(&plan, &result.CreateAlertRule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { alertRule(id: $id) { ` + alertFields + ` } }`
	var result struct {
		AlertRule *alertRuleGQL `json:"alertRule"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read alert rule", err.Error())
		return
	}
	if result.AlertRule == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapAlertToState(&state, result.AlertRule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update issues the updateAlertRule mutation, sending only the fields that
// differ between plan and state. This avoids overwriting server-computed
// fields and minimises wire traffic.
func (r *alertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state alertRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{}
	if !plan.Name.Equal(state.Name) {
		input["name"] = plan.Name.ValueString()
	}
	if !plan.ConditionType.Equal(state.ConditionType) {
		input["conditionType"] = plan.ConditionType.ValueString()
	}
	if !plan.Threshold.Equal(state.Threshold) {
		input["threshold"] = plan.Threshold.ValueInt64()
	}
	if !plan.WebhookURL.Equal(state.WebhookURL) {
		input["webhookUrl"] = plan.WebhookURL.ValueString()
	}
	if !plan.Enabled.Equal(state.Enabled) {
		input["enabled"] = plan.Enabled.ValueBool()
	}
	if !plan.Cooldown.Equal(state.Cooldown) {
		setOptionalString(input, "cooldown", plan.Cooldown)
	}

	if len(input) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	query := `mutation($id: UUID!, $input: UpdateAlertRuleInput!) { updateAlertRule(id: $id, input: $input) { ` + alertFields + ` } }`
	var result struct {
		UpdateAlertRule alertRuleGQL `json:"updateAlertRule"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString(), "input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to update alert rule", err.Error())
		return
	}

	mapAlertToState(&plan, &result.UpdateAlertRule)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: UUID!) { deleteAlertRule(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Failed to delete alert rule", err.Error())
	}
}

// alertRuleGQL is the GraphQL response shape for an alert rule.
type alertRuleGQL struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ConditionType string `json:"conditionType"`
	Threshold     int64  `json:"threshold"`
	WebhookURL    string `json:"webhookUrl"`
	Cooldown      string `json:"cooldown"`
	Enabled       bool   `json:"enabled"`
}

func mapAlertToState(state *alertRuleResourceModel, a *alertRuleGQL) {
	state.ID = types.StringValue(a.ID)
	state.Name = types.StringValue(a.Name)
	state.ConditionType = types.StringValue(a.ConditionType)
	state.Threshold = types.Int64Value(a.Threshold)
	state.WebhookURL = types.StringValue(a.WebhookURL)
	state.Cooldown = types.StringValue(a.Cooldown)
	state.Enabled = types.BoolValue(a.Enabled)
}
