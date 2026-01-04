package check

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/pakyas/terraform-provider-pakyas/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &CheckResource{}
	_ resource.ResourceWithImportState = &CheckResource{}
)

// Slug validation regex: lowercase alphanumeric with optional hyphens
var slugRegex = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// NewCheckResource creates a new check resource.
func NewCheckResource() resource.Resource {
	return &CheckResource{}
}

// CheckResource defines the resource implementation.
type CheckResource struct {
	client *client.Client
}

func (r *CheckResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_check"
}

func (r *CheckResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a Pakyas health check.",
		MarkdownDescription: "Manages a Pakyas health check. Checks monitor periodic jobs like cron tasks, backups, and scheduled processes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the check (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID this check belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the check (1-100 characters).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the check (unique within project, lowercase alphanumeric with hyphens).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(slugRegex, "must be lowercase alphanumeric with optional hyphens"),
				},
			},
			"period_seconds": schema.Int64Attribute{
				Description: "Expected interval between pings in seconds (60-2,592,000).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(60, 2592000),
				},
			},
			"grace_seconds": schema.Int64Attribute{
				Description: "Grace period in seconds before alerting (0-86,400). Default: 0.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.Between(0, 86400),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the check (max 500 characters).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(500),
				},
			},
			"tags": schema.SetAttribute{
				Description: "Tags for organizing and filtering checks.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"paused": schema.BoolAttribute{
				Description: "Whether the check is paused. Default: false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"public_id": schema.StringAttribute{
				Description: "The public ID used in the ping URL.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ping_url": schema.StringAttribute{
				Description: "The full URL to ping this check.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Current status of the check (new, up, down, late, paused).",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the check was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *CheckResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *CheckResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CheckResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating check", map[string]interface{}{
		"name":       data.Name.ValueString(),
		"project_id": data.ProjectID.ValueString(),
	})

	// Build create request
	createReq := client.CreateCheckRequest{
		ProjectID:     data.ProjectID.ValueString(),
		Name:          data.Name.ValueString(),
		Slug:          data.Slug.ValueString(),
		PeriodSeconds: data.PeriodSeconds.ValueInt64(),
		GraceSeconds:  data.GraceSeconds.ValueInt64(),
		Paused:        data.Paused.ValueBool(),
	}

	// Description
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		createReq.Description = &desc
	}

	// Tags
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		var tags []string
		resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Tags = tags
	}

	check, err := r.client.CreateCheck(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Check",
			"Could not create check, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to model
	r.mapCheckToModel(ctx, check, &data)

	tflog.Debug(ctx, "Created check", map[string]interface{}{
		"id": check.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading check", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	check, err := r.client.GetCheck(ctx, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "Check not found, removing from state", map[string]interface{}{
				"id": data.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Check",
			"Could not read check ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to model
	r.mapCheckToModel(ctx, check, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CheckResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating check", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Build update request with only changed fields
	updateReq := client.UpdateCheckRequest{}

	if !data.Name.Equal(state.Name) {
		n := data.Name.ValueString()
		updateReq.Name = &n
	}

	if !data.PeriodSeconds.Equal(state.PeriodSeconds) {
		p := data.PeriodSeconds.ValueInt64()
		updateReq.PeriodSeconds = &p
	}

	if !data.GraceSeconds.Equal(state.GraceSeconds) {
		g := data.GraceSeconds.ValueInt64()
		updateReq.GraceSeconds = &g
	}

	if !data.Description.Equal(state.Description) {
		if data.Description.IsNull() {
			empty := ""
			updateReq.Description = &empty
		} else {
			desc := data.Description.ValueString()
			updateReq.Description = &desc
		}
	}

	if !data.Tags.Equal(state.Tags) {
		var tags []string
		if !data.Tags.IsNull() {
			resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		updateReq.Tags = tags
	}

	if !data.Paused.Equal(state.Paused) {
		p := data.Paused.ValueBool()
		updateReq.Paused = &p
	}

	check, err := r.client.UpdateCheck(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Check",
			"Could not update check, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to model
	r.mapCheckToModel(ctx, check, &data)

	tflog.Debug(ctx, "Updated check", map[string]interface{}{
		"id": check.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CheckResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CheckResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting check", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.DeleteCheck(ctx, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "Check already deleted", map[string]interface{}{
				"id": data.ID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Check",
			"Could not delete check, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Deleted check", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *CheckResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing check", map[string]interface{}{
		"id": req.ID,
	})
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapCheckToModel maps an API Check to the Terraform model.
func (r *CheckResource) mapCheckToModel(ctx context.Context, check *client.Check, data *CheckResourceModel) {
	data.ID = types.StringValue(check.ID)
	data.ProjectID = types.StringValue(check.ProjectID)
	data.Name = types.StringValue(check.Name)
	data.Slug = types.StringValue(check.Slug)
	data.PeriodSeconds = types.Int64Value(check.PeriodSeconds)
	data.GraceSeconds = types.Int64Value(check.GraceSeconds)
	data.Paused = types.BoolValue(check.Paused)
	data.PublicID = types.StringValue(check.PublicID)
	data.Status = types.StringValue(check.Status)
	data.CreatedAt = types.StringValue(check.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))

	// Compute ping_url from ping_url_base + public_id
	data.PingURL = types.StringValue(r.client.PingURLBase() + "/" + check.PublicID)

	// Description
	if check.Description != nil {
		data.Description = types.StringValue(*check.Description)
	} else {
		data.Description = types.StringNull()
	}

	// Tags (as Set)
	if len(check.Tags) > 0 {
		tagValues := make([]attr.Value, len(check.Tags))
		for i, tag := range check.Tags {
			tagValues[i] = types.StringValue(tag)
		}
		data.Tags = types.SetValueMust(types.StringType, tagValues)
	} else {
		data.Tags = types.SetNull(types.StringType)
	}
}
