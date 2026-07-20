package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractProtocol(t *testing.T) {
	provider := &ModelRegistryProvider{
		URIValidator: NewURIValidator(nil),
	}

	tests := []struct {
		name        string
		uri         string
		expected    string
		expectError error
	}{
		{"s3 protocol", "s3://my-bucket/model.tar.gz", "s3://", nil},
		{"gs protocol", "gs://my-bucket/model/", "gs://", nil},
		{"https protocol", "https://example.com/model.tar.gz", "https://", nil},
		{"http protocol", "http://example.com/model.tar.gz", "http://", nil},
		{"empty URI", "", "", ErrNoStorageURI},
		{"no protocol", "my-bucket/model.tar.gz", "", ErrNoProtocolInSTorageURI},
		{"unsupported protocol", "ftp://example.com/model.tar.gz", "", ErrProtocolNotSupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, err := provider.extractProtocol(tt.uri)
			if tt.expectError != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, string(protocol))
			}
		})
	}
}
