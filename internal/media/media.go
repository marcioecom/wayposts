package media

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

var ErrNotFound = errors.New("media: objeto não encontrado")

type ObjectInfo struct {
	Size        int64
	ContentType string
}

type Store interface {
	PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error)
	Head(ctx context.Context, key string) (ObjectInfo, error)
}

type Config struct {
	Bucket     string
	Region     string
	Endpoint   string
	AccessKey  string
	SecretKey  string
	PathStyle  bool
	PublicBase string
}

func (c Config) Enabled() bool {
	return c.Bucket != ""
}

type S3 struct {
	presign *s3.PresignClient
	client  *s3.Client
	bucket  string
}

// NewS3 monta o cliente. BaseEndpoint permite apontar para R2 ou MinIO sem
// mudar o código.
func NewS3(ctx context.Context, cfg Config) (*S3, error) {
	if !cfg.Enabled() {
		return nil, errors.New("media: bucket não configurado")
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.AccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("media: config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.PathStyle
	})

	return &S3{
		presign: s3.NewPresignClient(client),
		client:  client,
		bucket:  cfg.Bucket,
	}, nil
}

// PresignPut devolve uma URL de PUT direto no bucket. O Content-Type entra na
// assinatura, então o cliente precisa enviar exatamente o mesmo header.
func (s *S3) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	req, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("media: presign: %w", err)
	}
	return req.URL, nil
}

func (s *S3) Head(ctx context.Context, key string) (ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "NotFound", "NoSuchKey", "404":
				return ObjectInfo{}, ErrNotFound
			}
		}
		return ObjectInfo{}, fmt.Errorf("media: head: %w", err)
	}

	info := ObjectInfo{ContentType: aws.ToString(out.ContentType)}
	if out.ContentLength != nil {
		info.Size = *out.ContentLength
	}
	return info, nil
}

func (s *S3) URL(base, key string) string {
	if base == "" {
		return key
	}
	return strings.TrimSuffix(base, "/") + "/" + key
}

var extensions = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

func AllowedContentType(ct string) (string, bool) {
	ext, ok := extensions[strings.ToLower(strings.TrimSpace(ct))]
	return ext, ok
}

func Key(authorID, objectID int64, contentType string) (string, error) {
	ext, ok := AllowedContentType(contentType)
	if !ok {
		return "", fmt.Errorf("media: tipo não suportado: %q", contentType)
	}
	return "media/" + strconv.FormatInt(authorID, 10) + "/" +
		strconv.FormatInt(objectID, 10) + "." + ext, nil
}

func OwnedBy(key string, authorID int64) bool {
	return strings.HasPrefix(key, "media/"+strconv.FormatInt(authorID, 10)+"/")
}
