package storage

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func newAwsConfig(ctx context.Context) (aws.Config, error) {
	endpoint := os.Getenv("AWS_ENDPOINT")
	customResolver := aws.EndpointResolverWithOptions(
		aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if endpoint != "" {
					return aws.Endpoint{
						URL:           endpoint,
						SigningRegion: region,
						Source:        aws.EndpointSourceCustom,
					}, nil
				}
				// returning EndpointNotFoundError will allow the service to fallback to its default resolution
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			},
		),
	)
	return config.LoadDefaultConfig(ctx, config.WithEndpointResolverWithOptions(customResolver))
}
