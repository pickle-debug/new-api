package controller

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadCorporateProof(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		content   string
		wantMime  string
		wantError string
	}{
		{
			name:     "accepts png by content",
			filename: "receipt.txt",
			content:  "\x89PNG\r\n\x1a\n" + strings.Repeat("\x00", 16),
			wantMime: "image/png",
		},
		{
			name:      "rejects executable content with image extension",
			filename:  "receipt.png",
			content:   "MZ" + strings.Repeat("\x00", 32),
			wantError: "仅支持 JPG、PNG 或 PDF 格式的转账凭证",
		},
		{
			name:      "rejects empty file",
			filename:  "receipt.pdf",
			wantError: "转账凭证大小不能超过 5MB",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header := corporateProofTestHeader(test.filename, test.content)
			data, mimeType, extension, err := readCorporateProof(header)
			if test.wantError != "" {
				require.EqualError(t, err, test.wantError)
				assert.Empty(t, data)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantMime, mimeType)
			assert.NotEmpty(t, extension)
			assert.Equal(t, test.content, string(data))
		})
	}
}

func TestReadCorporateProofRejectsOversizedFile(t *testing.T) {
	header := corporateProofTestHeader("receipt.png", "\x89PNG\r\n\x1a\n")
	header.Size = service.MaxCorporateProofBytes + 1

	_, _, _, err := readCorporateProof(header)
	require.EqualError(t, err, "转账凭证大小不能超过 5MB")
}

func corporateProofTestHeader(filename string, content string) *multipart.FileHeader {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("proof", filename)
	if err != nil {
		panic(err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	form, err := multipart.NewReader(&body, writer.Boundary()).ReadForm(service.MaxCorporateProofBytes + 1024)
	if err != nil {
		panic(err)
	}
	return form.File["proof"][0]
}
