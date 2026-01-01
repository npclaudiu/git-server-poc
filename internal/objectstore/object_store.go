package objectstore

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ObjectStore struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
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

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.Retryer = aws.NopRetryer{}
	})

	return &ObjectStore{
		client:   client,
		uploader: manager.NewUploader(client),
		bucket:   opts.Bucket,
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

		_, err = o.client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(o.bucket),
		})
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	return err
}

func (o *ObjectStore) Head(ctx context.Context, key string) error {
	_, err := o.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (o *ObjectStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := o.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (o *ObjectStore) Put(ctx context.Context, key string, r io.Reader) error {
	_, err := o.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(o.bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	return err
}

func (o *ObjectStore) List(ctx context.Context, prefix string) ([]string, error) {
	out, err := o.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(o.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, obj := range out.Contents {
		keys = append(keys, *obj.Key)
	}
	return keys, nil
}
