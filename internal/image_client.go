package internal

import (
	"context"

	"github.com/moby/moby/client"
)

// ImageClient is a subset of Docker's ImageAPIClient interface with only what we are using for cnab-to-oci.
type ImageClient interface {
	ImagePush(ctx context.Context, ref string, options client.ImagePushOptions) (client.ImagePushResponse, error)
	ImageTag(ctx context.Context, options client.ImageTagOptions) (client.ImageTagResult, error)
}
