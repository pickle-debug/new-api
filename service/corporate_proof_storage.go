package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const MaxCorporateProofBytes int64 = 5 << 20

type CorporateProofLocation struct {
	URL         string
	StorageName string
}

type corporateProofObjectClient interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type CorporateProofStorage struct {
	mode       string
	localDir   string
	r2Bucket   string
	r2Endpoint string
	r2Prefix   string
	r2Client   corporateProofObjectClient
}

func NewCorporateProofStorage() (*CorporateProofStorage, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("CORPORATE_PAYMENT_PROOF_STORAGE")))
	storage := &CorporateProofStorage{
		mode:     mode,
		localDir: corporateProofLocalDirectory(),
	}
	if mode != "r2" {
		return nil, errors.New("corporate proof storage must be configured as R2")
	}
	if err := storage.configureR2(); err != nil {
		return nil, err
	}
	return storage, nil
}

func (storage *CorporateProofStorage) Save(
	ctx context.Context,
	objectName string,
	contentType string,
	data []byte,
) (CorporateProofLocation, error) {
	if int64(len(data)) <= 0 || int64(len(data)) > MaxCorporateProofBytes {
		return CorporateProofLocation{}, errors.New("invalid corporate proof size")
	}

	key := strings.Trim(strings.TrimSpace(objectName), "/")
	if key == "" || strings.Contains(key, "..") {
		return CorporateProofLocation{}, errors.New("invalid corporate proof object key")
	}
	if storage.r2Prefix != "" {
		key = storage.r2Prefix + "/" + key
	}

	checksum := sha256.Sum256(data)
	_, err := storage.r2Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:         aws.String(storage.r2Bucket),
		Key:            aws.String(key),
		Body:           bytes.NewReader(data),
		ContentLength:  aws.Int64(int64(len(data))),
		ContentType:    aws.String(contentType),
		ChecksumSHA256: aws.String(base64.StdEncoding.EncodeToString(checksum[:])),
		IfNoneMatch:    aws.String("*"),
	})
	if err != nil {
		return CorporateProofLocation{}, fmt.Errorf("upload corporate proof to R2: %w", err)
	}

	objectURL, err := url.JoinPath(storage.r2Endpoint, storage.r2Bucket, key)
	if err != nil {
		_ = storage.deleteR2Object(ctx, key)
		return CorporateProofLocation{}, fmt.Errorf("build corporate proof R2 URL: %w", err)
	}
	return CorporateProofLocation{URL: objectURL}, nil
}

func (storage *CorporateProofStorage) Read(
	ctx context.Context,
	proofURL string,
	storageName string,
) ([]byte, string, error) {
	if proofURL != "" {
		if storage.r2Client == nil {
			if err := storage.configureR2(); err != nil {
				return nil, "", err
			}
		}
		key, err := storage.r2KeyFromURL(proofURL)
		if err != nil {
			return nil, "", err
		}
		object, err := storage.r2Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(storage.r2Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return nil, "", fmt.Errorf("read corporate proof from R2: %w", err)
		}
		defer object.Body.Close()
		if object.ContentLength != nil && *object.ContentLength > MaxCorporateProofBytes {
			return nil, "", errors.New("corporate proof in R2 exceeds size limit")
		}
		data, err := io.ReadAll(io.LimitReader(object.Body, MaxCorporateProofBytes+1))
		if err != nil || int64(len(data)) > MaxCorporateProofBytes {
			return nil, "", errors.New("read corporate proof body from R2")
		}
		contentType := "application/octet-stream"
		if object.ContentType != nil && *object.ContentType != "" {
			contentType = *object.ContentType
		}
		return data, contentType, nil
	}

	path := filepath.Join(storage.localDir, filepath.Base(storageName))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > MaxCorporateProofBytes {
		return nil, "", errors.New("local corporate proof exceeds size limit")
	}
	return data, http.DetectContentType(data), nil
}

func (storage *CorporateProofStorage) Delete(
	ctx context.Context,
	proofURL string,
	storageName string,
) error {
	if proofURL != "" {
		if storage.r2Client == nil {
			if err := storage.configureR2(); err != nil {
				return err
			}
		}
		key, err := storage.r2KeyFromURL(proofURL)
		if err != nil {
			return err
		}
		return storage.deleteR2Object(ctx, key)
	}
	if storageName == "" {
		return nil
	}
	return os.Remove(filepath.Join(storage.localDir, filepath.Base(storageName)))
}

func (storage *CorporateProofStorage) configureR2() error {
	accountID := strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID"))
	bucket := strings.TrimSpace(os.Getenv("R2_BUCKET_NAME"))
	accessKeyID := strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID"))
	secretAccessKey := strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY"))
	endpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("R2_ENDPOINT")), "/")
	region := strings.TrimSpace(os.Getenv("R2_REGION"))
	if region == "" {
		region = "auto"
	}
	if endpoint == "" && accountID != "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	}
	if bucket == "" || accessKeyID == "" || secretAccessKey == "" || endpoint == "" {
		return errors.New("R2 corporate proof storage is not fully configured")
	}
	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil || parsedEndpoint.Scheme != "https" || parsedEndpoint.Host == "" || parsedEndpoint.Path != "" {
		return errors.New("R2 endpoint is invalid")
	}

	storage.r2Bucket = bucket
	storage.r2Endpoint = endpoint
	storage.r2Prefix = strings.Trim(strings.TrimSpace(os.Getenv("R2_PREFIX")), "/")
	storage.r2Client = s3.NewFromConfig(aws.Config{
		Region: region,
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		),
	}, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})
	return nil
}

func (storage *CorporateProofStorage) deleteR2Object(ctx context.Context, key string) error {
	_, err := storage.r2Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(storage.r2Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete corporate proof from R2: %w", err)
	}
	return nil
}

func (storage *CorporateProofStorage) r2KeyFromURL(proofURL string) (string, error) {
	objectURL, err := url.Parse(proofURL)
	if err != nil || objectURL.Scheme != "https" || objectURL.Host == "" {
		return "", errors.New("corporate proof R2 URL is invalid")
	}
	endpointURL, err := url.Parse(storage.r2Endpoint)
	if err != nil || !strings.EqualFold(objectURL.Host, endpointURL.Host) {
		return "", errors.New("corporate proof R2 URL host is invalid")
	}
	path := strings.TrimPrefix(objectURL.Path, "/")
	bucketPrefix := storage.r2Bucket + "/"
	if !strings.HasPrefix(path, bucketPrefix) {
		return "", errors.New("corporate proof R2 URL bucket is invalid")
	}
	key := strings.TrimPrefix(path, bucketPrefix)
	if key == "" || strings.Contains(key, "..") {
		return "", errors.New("corporate proof R2 URL key is invalid")
	}
	if storage.r2Prefix != "" && !strings.HasPrefix(key, storage.r2Prefix+"/") {
		return "", errors.New("corporate proof R2 URL prefix is invalid")
	}
	return key, nil
}

func corporateProofLocalDirectory() string {
	if directory := strings.TrimSpace(os.Getenv("CORPORATE_PAYMENT_PROOF_DIR")); directory != "" {
		return directory
	}
	return filepath.Join("data", "corporate-payment-proofs")
}
