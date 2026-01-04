package check

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CheckResourceModel describes the resource data model.
type CheckResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ProjectID     types.String `tfsdk:"project_id"`
	Name          types.String `tfsdk:"name"`
	Slug          types.String `tfsdk:"slug"`
	PeriodSeconds types.Int64  `tfsdk:"period_seconds"`
	GraceSeconds  types.Int64  `tfsdk:"grace_seconds"`
	Description   types.String `tfsdk:"description"`
	Tags          types.Set    `tfsdk:"tags"`
	Paused        types.Bool   `tfsdk:"paused"`
	PublicID      types.String `tfsdk:"public_id"`
	PingURL       types.String `tfsdk:"ping_url"`
	Status        types.String `tfsdk:"status"`
	CreatedAt     types.String `tfsdk:"created_at"`
}
