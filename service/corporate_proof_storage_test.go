package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type corporateProofObjectClientStub struct {
	putInput    *s3.PutObjectInput
	getInput    *s3.GetObjectInput
	deleteInput *s3.DeleteObjectInput
	objectData  []byte
	contentType string
	putErr      error
}

func (client *corporateProofObjectClientStub) PutObject(
	_ context.Context,
	input *s3.PutObjectInput,
	_ ...func(*s3.Options),
) (*s3.PutObjectOutput, error) {
	client.putInput = input
	if client.putErr != nil {
		return nil, client.putErr
	}
	data, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	client.objectData = data
	client.contentType = aws.ToString(input.ContentType)
	return &s3.PutObjectOutput{}, nil
}

func (client *corporateProofObjectClientStub) GetObject(
	_ context.Context,
	input *s3.GetObjectInput,
	_ ...func(*s3.Options),
) (*s3.GetObjectOutput, error) {
	client.getInput = input
	length := int64(len(client.objectData))
	return &s3.GetObjectOutput{
		Body:          io.NopCloser(bytes.NewReader(client.objectData)),
		ContentLength: &length,
		ContentType:   aws.String(client.contentType),
	}, nil
}

func (client *corporateProofObjectClientStub) DeleteObject(
	_ context.Context,
	input *s3.DeleteObjectInput,
	_ ...func(*s3.Options),
) (*s3.DeleteObjectOutput, error) {
	client.deleteInput = input
	return &s3.DeleteObjectOutput{}, nil
}

func TestCorporateProofStorageR2SaveAndRead(t *testing.T) {
	client := &corporateProofObjectClientStub{}
	storage := &CorporateProofStorage{
		mode:       "r2",
		r2Bucket:   "proofs",
		r2Endpoint: "https://account.r2.cloudflarestorage.com",
		r2Prefix:   "corporate-payment-proofs",
		r2Client:   client,
	}
	proof := []byte("\x89PNG\r\n\x1a\nproof")

	location, err := storage.Save(context.Background(), "CORP1.png", "image/png", proof)
	require.NoError(t, err)
	assert.Equal(
		t,
		"https://account.r2.cloudflarestorage.com/proofs/corporate-payment-proofs/CORP1.png",
		location.URL,
	)
	assert.Empty(t, location.StorageName)
	require.NotNil(t, client.putInput)
	assert.Equal(t, "proofs", aws.ToString(client.putInput.Bucket))
	assert.Equal(t, "corporate-payment-proofs/CORP1.png", aws.ToString(client.putInput.Key))
	assert.Equal(t, "*", aws.ToString(client.putInput.IfNoneMatch))
	assert.Equal(t, int64(len(proof)), aws.ToInt64(client.putInput.ContentLength))
	assert.NotEmpty(t, aws.ToString(client.putInput.ChecksumSHA256))

	data, contentType, err := storage.Read(context.Background(), location.URL, "")
	require.NoError(t, err)
	assert.Equal(t, proof, data)
	assert.Equal(t, "image/png", contentType)
	require.NotNil(t, client.getInput)
	assert.Equal(t, "corporate-payment-proofs/CORP1.png", aws.ToString(client.getInput.Key))
}

func TestCorporateProofStorageR2RejectsForeignURL(t *testing.T) {
	storage := &CorporateProofStorage{
		mode:       "r2",
		r2Bucket:   "proofs",
		r2Endpoint: "https://account.r2.cloudflarestorage.com",
		r2Prefix:   "corporate-payment-proofs",
		r2Client:   &corporateProofObjectClientStub{},
	}

	_, _, err := storage.Read(
		context.Background(),
		"https://attacker.example/proofs/corporate-payment-proofs/CORP1.png",
		"",
	)
	require.EqualError(t, err, "corporate proof R2 URL host is invalid")
}

func TestCorporateProofStorageR2UploadFailureReturnsNoLocation(t *testing.T) {
	storage := &CorporateProofStorage{
		mode:       "r2",
		r2Bucket:   "proofs",
		r2Endpoint: "https://account.r2.cloudflarestorage.com",
		r2Client: &corporateProofObjectClientStub{
			putErr: errors.New("denied"),
		},
	}

	location, err := storage.Save(
		context.Background(),
		"CORP1.png",
		"image/png",
		[]byte("proof"),
	)
	require.Error(t, err)
	assert.Empty(t, location.URL)
	assert.Empty(t, location.StorageName)
}

func TestNewCorporateProofStorageRequiresR2(t *testing.T) {
	t.Setenv("CORPORATE_PAYMENT_PROOF_STORAGE", "")

	_, err := NewCorporateProofStorage()
	require.EqualError(t, err, "corporate proof storage must be configured as R2")
}
