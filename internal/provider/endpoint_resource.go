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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*endpointResource)(nil)
	_ resource.ResourceWithImportState = (*endpointResource)(nil)
)

// endpointResource manages a Hivehook outbound endpoint.
type endpointResource struct {
	client *Client
}

// endpointResourceModel is the Terraform state representation of an endpoint.
type endpointResourceModel struct {
	ID               types.String         `tfsdk:"id"`
	ApplicationID    types.String         `tfsdk:"application_id"`
	URL              types.String         `tfsdk:"url"`
	Status           types.String         `tfsdk:"status"`
	RateLimitRPS     types.Int64          `tfsdk:"rate_limit_rps"`
	TimeoutMs        types.Int64          `tfsdk:"timeout_ms"`
	RetryPolicy      jsontypes.Normalized `tfsdk:"retry_policy"`
	Headers          jsontypes.Normalized `tfsdk:"headers"`
	FilterConfig     jsontypes.Normalized `tfsdk:"filter_config"`
	TransformationID types.String         `tfsdk:"transformation_id"`
	AuthType         types.String         `tfsdk:"auth_type"`
	DeliveryMode     types.String         `tfsdk:"delivery_mode"`
	Ordered          types.Bool           `tfsdk:"ordered"`
	SigningSecret    types.String         `tfsdk:"signing_secret"`
}

func (r *endpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_endpoint"
}

func (r *endpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     0,
		Description: "Manages a Hivehook outbound endpoint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "Endpoint UUID.",
			},
			"application_id": schema.StringAttribute{Required: true, Description: "Owning application UUID."},
			"url":            schema.StringAttribute{Optional: true, Description: "Endpoint URL."},
			"status":         schema.StringAttribute{Optional: true, Computed: true, Description: "Endpoint status."},
			"rate_limit_rps": schema.Int64Attribute{Optional: true, Computed: true, Description: "Rate limit in requests per second."},
			"timeout_ms":     schema.Int64Attribute{Optional: true, Computed: true, Description: "HTTP timeout in milliseconds."},
			"retry_policy": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON-encoded retry policy.",
			},
			"headers": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON-encoded custom headers.",
			},
			"filter_config": schema.StringAttribute{
				CustomType:  jsontypes.NormalizedType{},
				Optional:    true,
				Description: "JSON-encoded filter configuration.",
			},
			"transformation_id": schema.StringAttribute{Optional: true, Description: "Optional transformation UUID."},
			"auth_type":         schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("NONE"), Description: "Authentication type."},
			"delivery_mode":     schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("PUSH"), Description: "Delivery mode (PUSH or PULL)."},
			"ordered":           schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether deliveries are ordered."},
			"signing_secret": schema.StringAttribute{
				Computed: true, Sensitive: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "HMAC signing secret used to verify outgoing requests.",
			},
		},
	}
}

func (r *endpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	r.client = client
}

func (r *endpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

const epFields = `id applicationId url signingSecret status rateLimitRps timeoutMs retryPolicy { maxAttempts initialDelay maxDelay backoffFactor } headers filterConfig transformationId authType deliveryMode ordered`

func (r *endpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan endpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{"applicationId": plan.ApplicationID.ValueString()}
	setOptionalString(input, "url", plan.URL)
	setOptionalInt64(input, "rateLimitRps", plan.RateLimitRPS)
	setOptionalInt64(input, "timeoutMs", plan.TimeoutMs)
	setOptionalString(input, "authType", plan.AuthType)
	setOptionalString(input, "deliveryMode", plan.DeliveryMode)
	setOptionalBool(input, "ordered", plan.Ordered)
	setOptionalString(input, "transformationId", plan.TransformationID)
	if err := setOptionalNormalizedJSON(input, "retryPolicy", plan.RetryPolicy); err != nil {
		resp.Diagnostics.AddError("Invalid retry_policy", err.Error())
		return
	}
	if err := setOptionalNormalizedJSON(input, "headers", plan.Headers); err != nil {
		resp.Diagnostics.AddError("Invalid headers", err.Error())
		return
	}
	if err := setOptionalNormalizedJSON(input, "filterConfig", plan.FilterConfig); err != nil {
		resp.Diagnostics.AddError("Invalid filter_config", err.Error())
		return
	}

	query := `mutation($input: CreateEndpointInput!) { createEndpoint(input: $input) { ` + epFields + ` } }`
	var result struct {
		CreateEndpoint endpointGQL `json:"createEndpoint"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to create endpoint", err.Error())
		return
	}

	mapEndpointToState(&plan, &result.CreateEndpoint)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *endpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state endpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { endpoint(id: $id) { ` + epFields + ` } }`
	var result struct {
		Endpoint *endpointGQL `json:"endpoint"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read endpoint", err.Error())
		return
	}
	if result.Endpoint == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapEndpointToState(&state, result.Endpoint)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update issues the updateEndpoint mutation, sending only the fields that
// differ between plan and state. This avoids overwriting server-computed
// fields and minimises wire traffic.
func (r *endpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan endpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state endpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{}
	if !plan.URL.Equal(state.URL) {
		setOptionalString(input, "url", plan.URL)
	}
	if !plan.Status.Equal(state.Status) {
		setOptionalString(input, "status", plan.Status)
	}
	if !plan.RateLimitRPS.Equal(state.RateLimitRPS) {
		setOptionalInt64(input, "rateLimitRps", plan.RateLimitRPS)
	}
	if !plan.TimeoutMs.Equal(state.TimeoutMs) {
		setOptionalInt64(input, "timeoutMs", plan.TimeoutMs)
	}
	if !plan.AuthType.Equal(state.AuthType) {
		setOptionalString(input, "authType", plan.AuthType)
	}
	if !plan.DeliveryMode.Equal(state.DeliveryMode) {
		setOptionalString(input, "deliveryMode", plan.DeliveryMode)
	}
	if !plan.Ordered.Equal(state.Ordered) {
		setOptionalBool(input, "ordered", plan.Ordered)
	}
	if !plan.TransformationID.Equal(state.TransformationID) {
		setOptionalString(input, "transformationId", plan.TransformationID)
	}
	if !plan.RetryPolicy.Equal(state.RetryPolicy) {
		if err := setOptionalNormalizedJSON(input, "retryPolicy", plan.RetryPolicy); err != nil {
			resp.Diagnostics.AddError("Invalid retry_policy", err.Error())
			return
		}
	}
	if !plan.Headers.Equal(state.Headers) {
		if err := setOptionalNormalizedJSON(input, "headers", plan.Headers); err != nil {
			resp.Diagnostics.AddError("Invalid headers", err.Error())
			return
		}
	}
	if !plan.FilterConfig.Equal(state.FilterConfig) {
		if err := setOptionalNormalizedJSON(input, "filterConfig", plan.FilterConfig); err != nil {
			resp.Diagnostics.AddError("Invalid filter_config", err.Error())
			return
		}
	}

	if len(input) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	query := `mutation($id: UUID!, $input: UpdateEndpointInput!) { updateEndpoint(id: $id, input: $input) { ` + epFields + ` } }`
	var result struct {
		UpdateEndpoint endpointGQL `json:"updateEndpoint"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString(), "input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to update endpoint", err.Error())
		return
	}

	mapEndpointToState(&plan, &result.UpdateEndpoint)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *endpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state endpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: UUID!) { deleteEndpoint(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Failed to delete endpoint", err.Error())
	}
}

// endpointGQL is the GraphQL response shape for an endpoint.
type endpointGQL struct {
	ID               string          `json:"id"`
	ApplicationID    string          `json:"applicationId"`
	URL              string          `json:"url"`
	SigningSecret    string          `json:"signingSecret"`
	Status           string          `json:"status"`
	RateLimitRPS     int64           `json:"rateLimitRps"`
	TimeoutMs        int64           `json:"timeoutMs"`
	RetryPolicy      *retryPolicyGQL `json:"retryPolicy"`
	Headers          json.RawMessage `json:"headers"`
	FilterConfig     json.RawMessage `json:"filterConfig"`
	TransformationID *string         `json:"transformationId"`
	AuthType         string          `json:"authType"`
	DeliveryMode     string          `json:"deliveryMode"`
	Ordered          bool            `json:"ordered"`
}

func mapEndpointToState(state *endpointResourceModel, e *endpointGQL) {
	state.ID = types.StringValue(e.ID)
	state.ApplicationID = types.StringValue(e.ApplicationID)
	state.URL = types.StringValue(e.URL)
	// signing_secret is Computed-only; treat empty string as null to avoid
	// perpetual diffs against the server.
	state.SigningSecret = stringValueOrNull(e.SigningSecret)
	state.Status = types.StringValue(e.Status)
	state.RateLimitRPS = types.Int64Value(e.RateLimitRPS)
	state.TimeoutMs = types.Int64Value(e.TimeoutMs)
	state.AuthType = types.StringValue(e.AuthType)
	state.DeliveryMode = types.StringValue(e.DeliveryMode)
	state.Ordered = types.BoolValue(e.Ordered)

	if e.RetryPolicy != nil {
		b, _ := json.Marshal(e.RetryPolicy)
		state.RetryPolicy = jsontypes.NewNormalizedValue(string(b))
	} else {
		state.RetryPolicy = jsontypes.NewNormalizedNull()
	}
	// Pass the raw server bytes straight through so key order is preserved and
	// plan diffs are deterministic.
	state.Headers = normalizedFromRaw(e.Headers)
	state.FilterConfig = normalizedFromRaw(e.FilterConfig)
	if e.TransformationID != nil {
		state.TransformationID = types.StringValue(*e.TransformationID)
	} else {
		state.TransformationID = types.StringNull()
	}
}
