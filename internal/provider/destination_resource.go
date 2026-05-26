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
	_ resource.Resource                = (*destinationResource)(nil)
	_ resource.ResourceWithImportState = (*destinationResource)(nil)
)

// destinationResource manages a Hivehook delivery destination.
type destinationResource struct {
	client *Client
}

// destinationResourceModel is the Terraform state representation of a destination.
type destinationResourceModel struct {
	ID            types.String         `tfsdk:"id"`
	Name          types.String         `tfsdk:"name"`
	URL           types.String         `tfsdk:"url"`
	Status        types.String         `tfsdk:"status"`
	TimeoutMs     types.Int64          `tfsdk:"timeout_ms"`
	RateLimitRPS  types.Int64          `tfsdk:"rate_limit_rps"`
	RetryPolicy   jsontypes.Normalized `tfsdk:"retry_policy"`
	Headers       jsontypes.Normalized `tfsdk:"headers"`
	AuthType      types.String         `tfsdk:"auth_type"`
	DeliveryMode  types.String         `tfsdk:"delivery_mode"`
	Ordered       types.Bool           `tfsdk:"ordered"`
	SigningSecret types.String         `tfsdk:"signing_secret"`
}

func (r *destinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (r *destinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     0,
		Description: "Manages a Hivehook delivery destination.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "Destination UUID.",
			},
			"name":           schema.StringAttribute{Required: true, Description: "Display name."},
			"url":            schema.StringAttribute{Optional: true, Description: "Destination URL."},
			"status":         schema.StringAttribute{Optional: true, Computed: true, Description: "Destination status (ACTIVE or INACTIVE)."},
			"timeout_ms":     schema.Int64Attribute{Optional: true, Computed: true, Description: "HTTP request timeout in milliseconds."},
			"rate_limit_rps": schema.Int64Attribute{Optional: true, Computed: true, Description: "Rate limit in requests per second."},
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
			"auth_type":     schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("NONE"), Description: "Authentication type."},
			"delivery_mode": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("PUSH"), Description: "Delivery mode (PUSH or PULL)."},
			"ordered":       schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Whether deliveries are ordered."},
			"signing_secret": schema.StringAttribute{
				Computed: true, Sensitive: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "HMAC signing secret used to verify outgoing requests.",
			},
		},
	}
}

func (r *destinationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	r.client = client
}

func (r *destinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

const destFields = `id name url signingSecret status timeoutMs rateLimitRps retryPolicy { maxAttempts initialDelay maxDelay backoffFactor } headers authType deliveryMode ordered`

func (r *destinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan destinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{"name": plan.Name.ValueString()}
	setOptionalString(input, "url", plan.URL)
	setOptionalInt64(input, "timeoutMs", plan.TimeoutMs)
	setOptionalInt64(input, "rateLimitRps", plan.RateLimitRPS)
	setOptionalString(input, "authType", plan.AuthType)
	setOptionalString(input, "deliveryMode", plan.DeliveryMode)
	setOptionalBool(input, "ordered", plan.Ordered)
	if err := setOptionalNormalizedJSON(input, "retryPolicy", plan.RetryPolicy); err != nil {
		resp.Diagnostics.AddError("Invalid retry_policy", err.Error())
		return
	}
	if err := setOptionalNormalizedJSON(input, "headers", plan.Headers); err != nil {
		resp.Diagnostics.AddError("Invalid headers", err.Error())
		return
	}

	query := `mutation($input: CreateDestinationInput!) { createDestination(input: $input) { ` + destFields + ` } }`

	var result struct {
		CreateDestination destinationGQL `json:"createDestination"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to create destination", err.Error())
		return
	}

	mapDestToState(&plan, &result.CreateDestination)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *destinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state destinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { destination(id: $id) { ` + destFields + ` } }`

	var result struct {
		Destination *destinationGQL `json:"destination"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read destination", err.Error())
		return
	}
	if result.Destination == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	mapDestToState(&state, result.Destination)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update issues the updateDestination mutation, sending only the fields that
// differ between plan and state. This avoids overwriting server-computed
// fields and minimises wire traffic.
func (r *destinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan destinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state destinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{}
	if !plan.Name.Equal(state.Name) {
		input["name"] = plan.Name.ValueString()
	}
	if !plan.URL.Equal(state.URL) {
		setOptionalString(input, "url", plan.URL)
	}
	if !plan.Status.Equal(state.Status) {
		setOptionalString(input, "status", plan.Status)
	}
	if !plan.TimeoutMs.Equal(state.TimeoutMs) {
		setOptionalInt64(input, "timeoutMs", plan.TimeoutMs)
	}
	if !plan.RateLimitRPS.Equal(state.RateLimitRPS) {
		setOptionalInt64(input, "rateLimitRps", plan.RateLimitRPS)
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

	if len(input) == 0 {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	query := `mutation($id: UUID!, $input: UpdateDestinationInput!) { updateDestination(id: $id, input: $input) { ` + destFields + ` } }`

	var result struct {
		UpdateDestination destinationGQL `json:"updateDestination"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString(), "input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to update destination", err.Error())
		return
	}

	mapDestToState(&plan, &result.UpdateDestination)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *destinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state destinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: UUID!) { deleteDestination(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Failed to delete destination", err.Error())
	}
}

// retryPolicyGQL is the embedded retry policy returned by the server.
type retryPolicyGQL struct {
	MaxAttempts   int     `json:"maxAttempts"`
	InitialDelay  string  `json:"initialDelay"`
	MaxDelay      string  `json:"maxDelay"`
	BackoffFactor float64 `json:"backoffFactor"`
}

// destinationGQL is the GraphQL response shape for a destination.
type destinationGQL struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	URL           string          `json:"url"`
	SigningSecret string          `json:"signingSecret"`
	Status        string          `json:"status"`
	TimeoutMs     int64           `json:"timeoutMs"`
	RateLimitRPS  int64           `json:"rateLimitRps"`
	RetryPolicy   *retryPolicyGQL `json:"retryPolicy"`
	Headers       json.RawMessage `json:"headers"`
	AuthType      string          `json:"authType"`
	DeliveryMode  string          `json:"deliveryMode"`
	Ordered       bool            `json:"ordered"`
}

func mapDestToState(state *destinationResourceModel, d *destinationGQL) {
	state.ID = types.StringValue(d.ID)
	state.Name = types.StringValue(d.Name)
	state.URL = types.StringValue(d.URL)
	// signing_secret is Computed-only; treat empty string as null to avoid
	// perpetual diffs against the server.
	state.SigningSecret = stringValueOrNull(d.SigningSecret)
	state.Status = types.StringValue(d.Status)
	state.TimeoutMs = types.Int64Value(d.TimeoutMs)
	state.RateLimitRPS = types.Int64Value(d.RateLimitRPS)
	state.AuthType = types.StringValue(d.AuthType)
	state.DeliveryMode = types.StringValue(d.DeliveryMode)
	state.Ordered = types.BoolValue(d.Ordered)

	if d.RetryPolicy != nil {
		b, _ := json.Marshal(d.RetryPolicy)
		state.RetryPolicy = jsontypes.NewNormalizedValue(string(b))
	} else {
		state.RetryPolicy = jsontypes.NewNormalizedNull()
	}
	// Pass the raw server bytes straight through so key order is preserved and
	// plan diffs are deterministic.
	state.Headers = normalizedFromRaw(d.Headers)
}
