package internal

import (
	"context"

	"github.com/moby/moby/client"
)

// ImageClient is a subset of Docker's ImageAPIClient interface with only what we are using for cnab-to-oci.
type ImageClient interface {
	ImagePush(ctx context.Context, ref string, options client.ImagePushOptions) (client.ImagePushResponse, error)
	ImageTag(ctx context.Context, image, ref string) error
}

// ClientWrapper wraps the moby/moby/client.Client to provide a simpler ImageTag interface
type ClientWrapper struct {
	*client.Client
}

// ImageTag wraps the new moby client ImageTag method to provide the old interface
func (c *ClientWrapper) ImageTag(ctx context.Context, image, ref string) error {
	_, err := c.Client.ImageTag(ctx, client.ImageTagOptions{
		Source: image,
		Target: ref,
	})
	return err
}
