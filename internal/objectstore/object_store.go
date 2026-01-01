package objectstore

import (
	"context"
	"time"

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
		client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
			o.Retryer = aws.NopRetryer{}
		}),
		bucket: opts.Bucket,
	}, nil
}

func (o *ObjectStore) Ping(ctx context.Context) error {
	_, err := o.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(o.bucket),
	})
	return err
}

func (o *ObjectStore) EnsureBucket(ctx context.Context) error {
	var err error
	for i := 0; i < 30; i++ {
		_, err = o.client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(o.bucket),
		})
		if err == nil {
			return nil
		}

		// We assume any error means the bucket does not exist or we can't access it.
		// In a robust implementation we should check specifically for NotFound.
		// For this PoC, we'll attempt creation.
		_, err = o.client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(o.bucket),
		})
		if err == nil {
			return nil
		}

		// If error is not nil, wait and retry.
		// We could log the error here if we had a logger, but for now we just retry.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	return err
}
