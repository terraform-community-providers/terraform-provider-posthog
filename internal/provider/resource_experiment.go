package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ExperimentResource{}
var _ resource.ResourceWithImportState = &ExperimentResource{}

func NewExperimentResource() resource.Resource {
	return &ExperimentResource{}
}

type ExperimentResource struct {
	client *http.Client
}

type ExperimentResourceModel struct {
	Id             types.Int64  `tfsdk:"id"`
	ProjectId      types.Int64  `tfsdk:"project_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	FeatureFlagKey types.String `tfsdk:"feature_flag_key"`
	FeatureFlagId  types.Int64  `tfsdk:"feature_flag_id"`
	Variants       types.Map    `tfsdk:"variants"`
	StartDate      types.String `tfsdk:"start_date"`
	EndDate        types.String `tfsdk:"end_date"`
}

type VariantModel struct {
	Percentage types.Int64 `tfsdk:"percentage"`
}

func (r *ExperimentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_experiment"
}

func (r *ExperimentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "PostHog experiment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Identifier of the experiment.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.Int64Attribute{
				MarkdownDescription: "Identifier of the project the experiment belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the experiment.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the experiment.",
				Optional:            true,
			},
			"feature_flag_key": schema.StringAttribute{
				MarkdownDescription: "Key for the feature flag that controls this experiment.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.UTF8LengthAtLeast(1),
				},
			},
			"feature_flag_id": schema.Int64Attribute{
				MarkdownDescription: "Identifier of the feature flag created for this experiment.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"variants": schema.MapNestedAttribute{
				MarkdownDescription: "Experiment variants. Must include 'control' as the first variant. Percentages must sum to 100.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"percentage": schema.Int64Attribute{
							MarkdownDescription: "Rollout percentage for this variant.",
							Required:            true,
						},
					},
				},
			},
			"start_date": schema.StringAttribute{
				MarkdownDescription: "Start date of the experiment (ISO 8601 format).",
				Optional:            true,
			},
			"end_date": schema.StringAttribute{
				MarkdownDescription: "End date of the experiment (ISO 8601 format).",
				Optional:            true,
			},
		},
	}
}

func (r *ExperimentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*http.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ExperimentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ExperimentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	variants, diags := r.variantsFromModel(ctx, data.Variants)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := ExperimentCreateInput{
		Name:           data.Name.ValueString(),
		Description:    data.Description.ValueString(),
		FeatureFlagKey: data.FeatureFlagKey.ValueString(),
		Parameters: ExperimentParameters{
			FeatureFlagVariants: variants,
		},
	}

	if !data.StartDate.IsNull() {
		input.StartDate = data.StartDate.ValueStringPointer()
	}

	if !data.EndDate.IsNull() {
		input.EndDate = data.EndDate.ValueStringPointer()
	}

	var experiment Experiment

	err := call(r.client, http.MethodPost, fmt.Sprintf("/projects/%d/experiments", data.ProjectId.ValueInt64()), input, &experiment)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create experiment, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created an experiment")

	r.populateModel(ctx, data, &experiment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExperimentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ExperimentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var experiment Experiment

	err := get(r.client, fmt.Sprintf("/projects/%d/experiments/%d", data.ProjectId.ValueInt64(), data.Id.ValueInt64()), &experiment)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read experiment, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "read an experiment")

	r.populateModel(ctx, data, &experiment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExperimentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ExperimentResourceModel
	var state *ExperimentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	variants, diags := r.variantsFromModel(ctx, data.Variants)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := ExperimentUpdateInput{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Parameters: ExperimentParameters{
			FeatureFlagVariants: variants,
		},
	}

	if !data.StartDate.IsNull() {
		input.StartDate = data.StartDate.ValueStringPointer()
	}

	if !data.EndDate.IsNull() {
		input.EndDate = data.EndDate.ValueStringPointer()
	}

	var experiment Experiment

	err := call(r.client, http.MethodPatch, fmt.Sprintf("/projects/%d/experiments/%d", data.ProjectId.ValueInt64(), data.Id.ValueInt64()), input, &experiment)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update experiment, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated an experiment")

	r.populateModel(ctx, data, &experiment)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExperimentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ExperimentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	input := ExperimentDeleteInput{Deleted: true}

	err := call(r.client, http.MethodPatch, fmt.Sprintf("/projects/%d/experiments/%d", data.ProjectId.ValueInt64(), data.Id.ValueInt64()), input, &Experiment{})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete experiment, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted an experiment")
}

func (r *ExperimentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectId, experimentId, err := parseExperimentImportId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Import Identifier", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), experimentId)...)
}

func parseExperimentImportId(id string) (int64, int64, error) {
	parts := strings.Split(id, ":")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, 0, fmt.Errorf("expected import identifier with format: project_id:experiment_id. Got: %q", id)
	}

	projectId, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("expected project_id to be a number. Got: %q", parts[0])
	}

	experimentId, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("expected experiment_id to be a number. Got: %q", parts[1])
	}

	return projectId, experimentId, nil
}

func (r *ExperimentResource) variantsFromModel(ctx context.Context, variantsMap types.Map) ([]FeatureFlagVariant, diag.Diagnostics) {
	var diags diag.Diagnostics
	variants := make([]FeatureFlagVariant, 0)

	elements := variantsMap.Elements()

	hasControl := false
	for key := range elements {
		if key == "control" {
			hasControl = true
			break
		}
	}

	if !hasControl {
		diags.AddError("Invalid Variants", "Variants must include 'control'.")
		return nil, diags
	}

	if controlVal, ok := elements["control"]; ok {
		var variant VariantModel
		diags.Append(controlVal.(types.Object).As(ctx, &variant, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		variants = append(variants, FeatureFlagVariant{
			Key:               "control",
			RolloutPercentage: int(variant.Percentage.ValueInt64()),
		})
	}

	for key, val := range elements {
		if key == "control" {
			continue
		}
		var variant VariantModel
		diags.Append(val.(types.Object).As(ctx, &variant, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return nil, diags
		}
		variants = append(variants, FeatureFlagVariant{
			Key:               key,
			RolloutPercentage: int(variant.Percentage.ValueInt64()),
		})
	}

	return variants, diags
}

func (r *ExperimentResource) populateModel(ctx context.Context, data *ExperimentResourceModel, experiment *Experiment) {
	data.Id = types.Int64Value(experiment.Id)
	data.Name = types.StringValue(experiment.Name)

	if experiment.Description != "" {
		data.Description = types.StringValue(experiment.Description)
	}

	data.FeatureFlagKey = types.StringValue(experiment.FeatureFlag.Key)
	data.FeatureFlagId = types.Int64Value(experiment.FeatureFlag.Id)

	if experiment.StartDate != nil {
		data.StartDate = types.StringValue(*experiment.StartDate)
	}

	if experiment.EndDate != nil {
		data.EndDate = types.StringValue(*experiment.EndDate)
	}

	variantElements := make(map[string]attr.Value)
	for _, v := range experiment.Parameters.FeatureFlagVariants {
		variantElements[v.Key] = types.ObjectValueMust(
			map[string]attr.Type{"percentage": types.Int64Type},
			map[string]attr.Value{"percentage": types.Int64Value(int64(v.RolloutPercentage))},
		)
	}

	data.Variants = types.MapValueMust(
		types.ObjectType{AttrTypes: map[string]attr.Type{"percentage": types.Int64Type}},
		variantElements,
	)
}
