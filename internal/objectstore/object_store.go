package objectstore

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ObjectStore struct {
	client *s3.Client
	bucket string
}

type Options struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
}

func New(ctx context.Context, opts Options) (*ObjectStore, error) {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               opts.Endpoint,
			SigningRegion:     region,
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(opts.Region),
		config.WithEndpointResolverWithOptions(resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretKey, "")),
	)

	if err != nil {
		return nil, err
	}

	return &ObjectStore{
		client: s3.NewFromConfig(cfg),
		bucket: opts.Bucket,
	}, nil
}

func (o *ObjectStore) Ping(ctx context.Context) error {
	_, err := o.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(o.bucket),
	})
	return err
}
