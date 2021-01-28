package gcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/turbot/go-kit/types"
	"github.com/turbot/steampipe-plugin-sdk/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/plugin"
	"github.com/turbot/steampipe-plugin-sdk/plugin/transform"

	"google.golang.org/api/cloudfunctions/v1"
)

func tableGcpCloudfunctionFunction(ctx context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "gcp_cloudfunctions_function",
		Description: "GCP Cloud Function",
		Get: &plugin.GetConfig{
			KeyColumns: plugin.SingleColumn("name"),
			Hydrate:    getCloudFunction,
		},
		List: &plugin.ListConfig{
			Hydrate: listCloudFunctions,
		},
		Columns: []*plugin.Column{
			// commonly used columns
			{
				Name:        "name",
				Description: "The name of the function.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "status",
				Description: "Status of the function deployment (ACTIVE, OFFLINE, CLOUD_FUNCTION_STATUS_UNSPECIFIED,DEPLOY_IN_PROGRESS, DELETE_IN_PROGRESS, UNKNOWN).",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "description",
				Description: "User-provided description of a function.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "runtime",
				Description: "The runtime in which to run the function.",
				Type:        proto.ColumnType_STRING,
			},

			// other columns
			{
				Name:        "available_memory_mb",
				Description: "The amount of memory in MB available for the function.",
				Type:        proto.ColumnType_INT,
			},
			{
				Name:        "build_environment_variables",
				Description: "Environment variables that shall be available during build time",
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "build_id",
				Description: "The Cloud Build ID of the latest successful deployment of the function.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "entry_point",
				Description: "The name of the function (as defined in source code) that will be executed.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "environment_variables",
				Description: "Environment variables that shall be available during function execution.",
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "event_trigger",
				Description: "A source that fires events in response to a condition in another service.",
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "https_trigger",
				Description: "An HTTPS endpoint type of source that can be triggered via URL.",
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "iam_policy",
				Description: "The IAM policy for the function.", Transform: transform.FromValue(), Hydrate: getGcpCloudFunctionIamPolicy,
				Type: proto.ColumnType_JSON,
			},
			{
				Name:        "ingress_settings",
				Description: "The ingress settings for the function, controlling what traffic can reach it (INGRESS_SETTINGS_UNSPECIFIED, ALLOW_ALL, ALLOW_INTERNAL_ONLY, ALLOW_INTERNAL_AND_GCLB).",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "labels",
				Description: "Labels that apply to this function.",
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "max_instances",
				Description: "The limit on the maximum number of function instances that may coexist at a given time. In some cases, such as rapid traffic surges, Cloud Functions may, for a short period of time, create more instances than the specified max instances limit.",
				Type:        proto.ColumnType_INT,
			},
			{
				Name:        "network",
				Description: "The VPC Network that this cloud function can connect to.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "service_account_email",
				Description: "The email of the function's service account.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "source_archive_url",
				Description: "The Google Cloud Storage URL, starting with gs://, pointing to the zip archive which contains the function.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "source_repository",
				Description: "**Beta Feature** The source repository where a function is hosted.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "source_upload_url",
				Description: "The Google Cloud Storage signed URL used for source uploading, generated by google.cloud.functions.v1.GenerateUploadUrl",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "timeout",
				Description: "The function execution timeout. Execution is consideredfailed and can be terminated if the function is not completed at the end of the timeout period. Defaults to 60 seconds.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "update_time",
				Description: "The last update timestamp of the Cloud Function.",
				Type:        proto.ColumnType_TIMESTAMP,
			},
			{
				Name:        "version_id",
				Description: "The version identifier of the Cloud Function. Each deployment attempt results in a new version of a function being created.",
				Type:        proto.ColumnType_INT,
			},
			{
				Name:        "vpc_connector",
				Description: "The VPC Network Connector that this cloud function can  connect to. This field is mutually exclusive with `network` field and will eventually replace it.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "vpc_connector_egress_settings",
				Description: "The egress settings for the connector, controlling what traffic is diverted through it (VPC_CONNECTOR_EGRESS_SETTINGS_UNSPECIFIED, PRIVATE_RANGES_ONLY, ALL_TRAFFIC).",
				Type:        proto.ColumnType_STRING,
			},

			// standard steampipe columns
			{
				Name:        "title",
				Description: ColumnDescriptionTitle,
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Name"),
			},
			{
				Name:        "tags",
				Description: ColumnDescriptionTags,
				Type:        proto.ColumnType_JSON,
				Transform:   transform.FromField("Labels"),
			},
			{
				Name:        "akas",
				Description: ColumnDescriptionAkas,
				Type:        proto.ColumnType_JSON,
				Transform:   transform.From(functionAka),
			},

			// standard gcp columns
			{
				Name:        "project",
				Description: ColumnDescriptionProject,
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromConstant(activeProject()),
			},
			{
				Name:        "location",
				Description: ColumnDescriptionLocation,
				Type:        proto.ColumnType_STRING,
				Transform:   transform.FromField("Name").Transform(locationFromFunctionName),
			},
		},
	}
}

//// HYDRATE FUNCTIONS

func listCloudFunctions(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	logger.Trace("listCloudFunctions")

	service, err := cloudfunctions.NewService(ctx)
	if err != nil {
		return nil, err
	}

	project := activeProject()
	data := "projects/" + project + "/locations/-" // '-' for all locations...

	resp := service.Projects.Locations.Functions.List(data)
	if err := resp.Pages(
		ctx,
		func(page *cloudfunctions.ListFunctionsResponse) error {
			for _, item := range page.Functions {
				d.StreamListItem(ctx, item)
			}
			return nil
		},
	); err != nil {
		return nil, err
	}

	return nil, nil
}

func getCloudFunction(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	logger.Trace("GetCloudFunction")

	service, err := cloudfunctions.NewService(ctx)
	if err != nil {
		return nil, err
	}

	name := d.KeyColumnQuals["name"].GetStringValue()

	cloudFunction, err := service.Projects.Locations.Functions.Get(name).Do()
	if err != nil {
		return nil, err
	}
	return cloudFunction, nil
}

func getGcpCloudFunctionIamPolicy(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	logger := plugin.Logger(ctx)
	logger.Trace("getGcpCloudFunctionIamPolicy")

	service, err := cloudfunctions.NewService(ctx)
	if err != nil {
		return nil, err
	}

	function := h.Item.(*cloudfunctions.CloudFunction)

	resp, err := service.Projects.Locations.Functions.GetIamPolicy(function.Name).Do()
	if err != nil {
		return nil, err
	}

	if resp != nil {
		return resp, nil
	}

	return cloudfunctions.Policy{}, nil
}

//// TRANSFORM FUNCTIONS

func functionAka(_ context.Context, d *transform.TransformData) (interface{}, error) {
	i := d.HydrateItem.(*cloudfunctions.CloudFunction)

	functionNamePath := types.SafeString(i.Name)

	//ex: gcp://cloudfunctions.googleapis.com/projects/project-aaa/locations/us-central1/functions/hello-world
	akas := []string{"gcp://cloudfunctions.googleapis.com/" + functionNamePath}

	return akas, nil

}

func locationFromFunctionName(_ context.Context, d *transform.TransformData) (interface{}, error) {
	functionName := types.SafeString(d.Value)
	parts := strings.Split(functionName, "/")
	if len(parts) != 6 {
		return nil, fmt.Errorf("Transform locationFromFunctionName failed - unexpected name format: %s", functionName)
	}
	return parts[3], nil
}