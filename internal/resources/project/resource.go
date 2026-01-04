package project

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/pakyas/terraform-provider-pakyas/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ProjectResource{}
	_ resource.ResourceWithImportState = &ProjectResource{}
)

// NewProjectResource creates a new project resource.
func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// ProjectResource defines the resource implementation.
type ProjectResource struct {
	client *client.Client
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a Pakyas project.",
		MarkdownDescription: "Manages a Pakyas project. Projects are containers for organizing health checks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the project (UUID).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the project (1-100 characters).",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the project (max 500 characters).",
				Optional:    true,
			},
			"org_id": schema.StringAttribute{
				Description: "The organization ID this project belongs to.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp when the project was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The timestamp when the project was last updated.",
				Computed:    true,
			},
		},
	}
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating project", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	// Prepare description (nil if not set)
	var description *string
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		desc := data.Description.ValueString()
		description = &desc
	}

	project, err := r.client.CreateProject(ctx, data.Name.ValueString(), description)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Project",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(project.ID)
	data.OrgID = types.StringValue(project.OrgID)
	data.Name = types.StringValue(project.Name)
	if project.Description != nil {
		data.Description = types.StringValue(*project.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.CreatedAt = types.StringValue(project.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	data.UpdatedAt = types.StringValue(project.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	tflog.Debug(ctx, "Created project", map[string]interface{}{
		"id": project.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading project", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	project, err := r.client.GetProject(ctx, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "Project not found, removing from state", map[string]interface{}{
				"id": data.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Project",
			"Could not read project ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(project.ID)
	data.OrgID = types.StringValue(project.OrgID)
	data.Name = types.StringValue(project.Name)
	if project.Description != nil {
		data.Description = types.StringValue(*project.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.CreatedAt = types.StringValue(project.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	data.UpdatedAt = types.StringValue(project.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating project", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Prepare update request with only changed fields
	var name *string
	if !data.Name.Equal(state.Name) {
		n := data.Name.ValueString()
		name = &n
	}

	var description *string
	if !data.Description.Equal(state.Description) {
		if data.Description.IsNull() {
			// Explicitly set to empty string to clear
			empty := ""
			description = &empty
		} else {
			desc := data.Description.ValueString()
			description = &desc
		}
	}

	project, err := r.client.UpdateProject(ctx, state.ID.ValueString(), name, description)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Project",
			"Could not update project, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response to model
	data.ID = types.StringValue(project.ID)
	data.OrgID = types.StringValue(project.OrgID)
	data.Name = types.StringValue(project.Name)
	if project.Description != nil {
		data.Description = types.StringValue(*project.Description)
	} else {
		data.Description = types.StringNull()
	}
	data.CreatedAt = types.StringValue(project.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	data.UpdatedAt = types.StringValue(project.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))

	tflog.Debug(ctx, "Updated project", map[string]interface{}{
		"id": project.ID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting project", map[string]interface{}{
		"id": data.ID.ValueString(),
	})

	err := r.client.DeleteProject(ctx, data.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			// Already deleted, that's fine
			tflog.Debug(ctx, "Project already deleted", map[string]interface{}{
				"id": data.ID.ValueString(),
			})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Project",
			"Could not delete project, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Deleted project", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Importing project", map[string]interface{}{
		"id": req.ID,
	})
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
