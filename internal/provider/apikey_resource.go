package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*apiKeyResource)(nil)
	_ resource.ResourceWithImportState = (*apiKeyResource)(nil)
)

// apiKeyResource manages a Hivehook API key. API keys are immutable on the
// server, so any configured-attribute change triggers replacement instead of
// an in-place update.
type apiKeyResource struct {
	client *Client
}

// apiKeyResourceModel is the Terraform state representation of an API key.
type apiKeyResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Scopes    types.List   `tfsdk:"scopes"`
	SourceIDs types.List   `tfsdk:"source_ids"`
	ExpiresAt types.String `tfsdk:"expires_at"`
	RawKey    types.String `tfsdk:"raw_key"`
	KeyPrefix types.String `tfsdk:"key_prefix"`
}

func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

// Every user-configurable attribute forces replacement, so rotating a key is
// the only allowed mutation.
func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     0,
		Description: "Manages a Hivehook API key. API keys are immutable; any change forces replacement (rotation).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "API key UUID.",
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Display name.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"scopes": schema.ListAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				Description:   "Scopes granted to the key.",
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplaceIfConfigured()},
			},
			"source_ids": schema.ListAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				Description:   "Optional list of source UUIDs the key is scoped to.",
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplaceIfConfigured()},
			},
			"expires_at": schema.StringAttribute{
				Optional:      true,
				Description:   "Optional RFC3339 expiration timestamp.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"raw_key": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "Full API key (only available at creation).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key_prefix": schema.StringAttribute{
				Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "Public prefix of the API key (safe to display).",
			},
		},
	}
}

func (r *apiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	client, err := configureClient(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected ProviderData type", err.Error())
		return
	}
	r.client = client
}

// Note: the raw_key attribute will be unknown after import because the server
// only returns it at creation time.
func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := map[string]any{"name": plan.Name.ValueString()}

	if !plan.Scopes.IsNull() {
		var scopes []string
		resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &scopes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		input["scopes"] = scopes
	}

	if !plan.SourceIDs.IsNull() {
		var sourceIDs []string
		resp.Diagnostics.Append(plan.SourceIDs.ElementsAs(ctx, &sourceIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		input["sourceIds"] = sourceIDs
	}

	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		input["expiresAt"] = plan.ExpiresAt.ValueString()
	}

	query := `mutation($input: CreateAPIKeyInput!) {
		createAPIKey(input: $input) {
			apiKey { id name keyPrefix scopes sourceIds expiresAt }
			rawKey
		}
	}`

	var result struct {
		CreateAPIKey struct {
			APIKey apiKeyGQL `json:"apiKey"`
			RawKey string    `json:"rawKey"`
		} `json:"createAPIKey"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"input": input}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to create API key", err.Error())
		return
	}

	plan.ID = types.StringValue(result.CreateAPIKey.APIKey.ID)
	plan.Name = types.StringValue(result.CreateAPIKey.APIKey.Name)
	// raw_key and key_prefix are Computed-only; represent an empty server
	// response as null state so subsequent plans don't show drift.
	plan.KeyPrefix = stringValueOrNull(result.CreateAPIKey.APIKey.KeyPrefix)
	plan.RawKey = stringValueOrNull(result.CreateAPIKey.RawKey)

	scopeVals, diags := types.ListValueFrom(ctx, types.StringType, result.CreateAPIKey.APIKey.Scopes)
	resp.Diagnostics.Append(diags...)
	plan.Scopes = scopeVals

	sourceIDVals, diags := types.ListValueFrom(ctx, types.StringType, result.CreateAPIKey.APIKey.SourceIDs)
	resp.Diagnostics.Append(diags...)
	plan.SourceIDs = sourceIDVals

	if result.CreateAPIKey.APIKey.ExpiresAt != nil {
		plan.ExpiresAt = types.StringValue(*result.CreateAPIKey.APIKey.ExpiresAt)
	} else {
		plan.ExpiresAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `query($id: UUID!) { apiKey(id: $id) { id name keyPrefix scopes sourceIds expiresAt revokedAt } }`
	var result struct {
		APIKey *apiKeyGQL `json:"apiKey"`
	}
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, &result); err != nil {
		resp.Diagnostics.AddError("Failed to read API key", err.Error())
		return
	}
	if result.APIKey == nil || result.APIKey.RevokedAt != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(result.APIKey.Name)
	state.KeyPrefix = stringValueOrNull(result.APIKey.KeyPrefix)

	scopeVals, diags := types.ListValueFrom(ctx, types.StringType, result.APIKey.Scopes)
	resp.Diagnostics.Append(diags...)
	state.Scopes = scopeVals

	sourceIDVals, diags := types.ListValueFrom(ctx, types.StringType, result.APIKey.SourceIDs)
	resp.Diagnostics.Append(diags...)
	state.SourceIDs = sourceIDVals

	if result.APIKey.ExpiresAt != nil {
		state.ExpiresAt = types.StringValue(*result.APIKey.ExpiresAt)
	} else {
		state.ExpiresAt = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op: every user-configurable attribute forces replacement,
// so the Terraform runtime never invokes Update with substantive changes.
func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query := `mutation($id: UUID!) { revokeAPIKey(id: $id) }`
	if err := r.client.Execute(ctx, query, map[string]any{"id": state.ID.ValueString()}, nil); err != nil {
		resp.Diagnostics.AddError("Failed to revoke API key", err.Error())
	}
}

// apiKeyGQL is the GraphQL response shape for an API key.
type apiKeyGQL struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	KeyPrefix string   `json:"keyPrefix"`
	Scopes    []string `json:"scopes"`
	SourceIDs []string `json:"sourceIds"`
	ExpiresAt *string  `json:"expiresAt"`
	RevokedAt *string  `json:"revokedAt"`
}
