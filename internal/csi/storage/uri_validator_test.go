package storage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURIValidator_Validate_BlockedHosts(t *testing.T) {
	validator := NewURIValidator(nil)

	tests := []struct {
		name string
		uri  string
	}{
		{"AWS metadata IPv4", "http://169.254.169.254/latest/meta-data/"},
		{"AWS metadata IMDSv2", "https://169.254.169.254/latest/api/token"},
		{"AWS metadata with port", "http://169.254.169.254:80/latest/meta-data/"},
		{"AWS metadata non-standard port", "http://169.254.169.254:8080/something"},
		{"AWS metadata IPv6", "http://[fd00:ec2::254]/latest/meta-data/token"},
		{"AWS metadata IPv6 with port", "http://[fd00:ec2::254]:8080/latest/meta-data/"},
		{"AWS metadata in s3 opaque URI", "s3://169.254.169.254/bucket/key"},
		{"GCP metadata", "http://metadata.google.internal/computeMetadata/v1/"},
		{"GCP metadata alt", "http://metadata.goog/computeMetadata/v1/"},
		{"GCP metadata uppercase", "http://METADATA.GOOGLE.INTERNAL/computeMetadata/v1/"},
		{"GCP metadata trailing dot", "http://metadata.google.internal./computeMetadata/v1/"},
		{"GCP metadata.goog trailing dot", "http://metadata.goog./computeMetadata/v1/"},
		{"link-local other", "http://169.254.42.42/something"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.uri)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrArtifactURIBlocked), "expected ErrArtifactURIBlocked, got: %v", err)
		})
	}
}

func TestURIValidator_Validate_AllowedURIs(t *testing.T) {
	validator := NewURIValidator(nil)

	tests := []struct {
		name string
		uri  string
	}{
		{"S3 URI", "s3://my-bucket/model.tar.gz"},
		{"GCS URI", "gs://my-bucket/model/"},
		{"HTTPS URI", "https://example.com/models/v1.tar.gz"},
		{"HTTP URI", "http://internal-host:9000/model.bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.uri)
			assert.NoError(t, err)
		})
	}
}

func TestURIValidator_Validate_AllowlistEnforcement(t *testing.T) {
	validator := NewURIValidator([]string{"s3://trusted-bucket/", "gs://my-gcs-bucket/"})

	tests := []struct {
		name      string
		uri       string
		expectErr bool
		errType   error
	}{
		{"allowed S3 prefix", "s3://trusted-bucket/model.tar.gz", false, nil},
		{"allowed GCS prefix", "gs://my-gcs-bucket/path/model.bin", false, nil},
		{"allowed exact match", "s3://trusted-bucket/", false, nil},
		{"denied S3 different bucket", "s3://untrusted-bucket/model.tar.gz", true, ErrArtifactURINotAllowed},
		{"denied HTTPS", "https://example.com/model.tar.gz", true, ErrArtifactURINotAllowed},
		{"blocked always wins over allowlist", "http://169.254.169.254/latest/", true, ErrArtifactURIBlocked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.uri)
			if tt.expectErr {
				require.Error(t, err)
				assert.True(t, errors.Is(err, tt.errType), "expected %v, got: %v", tt.errType, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestURIValidator_Validate_EmptyAllowlist(t *testing.T) {
	tests := []struct {
		name     string
		prefixes []string
	}{
		{"nil prefixes", nil},
		{"empty slice", []string{}},
		{"only empty strings", []string{"", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewURIValidator(tt.prefixes)
			err := validator.Validate("s3://any-bucket/model.tar.gz")
			assert.NoError(t, err, "empty allowlist should accept all non-blocked URIs")
		})
	}
}

func TestURIValidator_Validate_ProtocolOnlyAllowlist(t *testing.T) {
	validator := NewURIValidator([]string{"s3://"})

	err := validator.Validate("s3://any-bucket/any-path")
	assert.NoError(t, err, "protocol-only prefix should match any bucket")

	err = validator.Validate("gs://any-bucket/any-path")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrArtifactURINotAllowed),
		"protocol-only prefix should reject different protocols")
}

func TestURIValidator_Validate_EmptyPrefixBypass(t *testing.T) {
	validator := NewURIValidator([]string{"", "s3://my-bucket/"})

	err := validator.Validate("gs://evil-bucket/payload")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrArtifactURINotAllowed),
		"empty prefix in allowlist must not match everything")
}

func TestURIValidator_Validate_TrailingSlashSubtlety(t *testing.T) {
	noSlash := NewURIValidator([]string{"s3://my-bucket"})
	err := noSlash.Validate("s3://my-bucket-evil/payload")
	assert.NoError(t, err, "without trailing slash, similar bucket names match (documents the footgun)")

	withSlash := NewURIValidator([]string{"s3://my-bucket/"})
	err = withSlash.Validate("s3://my-bucket-evil/model.tar.gz")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrArtifactURINotAllowed),
		"trailing slash in prefix should prevent partial bucket name matches")

	err = withSlash.Validate("s3://my-bucket/model.tar.gz")
	assert.NoError(t, err)
}

func TestURIValidator_Validate_BlocklistOverridesAllowlist(t *testing.T) {
	validator := NewURIValidator([]string{"http://169.254.169.254/"})

	err := validator.Validate("http://169.254.169.254/latest/meta-data/")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrArtifactURIBlocked),
		"blocklist should take precedence even when URI matches an allowlist entry")
}

func TestNewURIValidator_FiltersEmptyPrefixes(t *testing.T) {
	validator := NewURIValidator([]string{"s3://bucket/", "", "gs://other/", ""})
	assert.Equal(t, []string{"s3://bucket/", "gs://other/"}, validator.AllowedPrefixes)
}

func TestIsBlockedHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		blocked bool
	}{
		{"AWS IPv4", "169.254.169.254", true},
		{"AWS IPv4 with port", "169.254.169.254:80", true},
		{"AWS IPv4 non-standard port", "169.254.169.254:8080", true},
		{"AWS IPv6 bracketed", "[fd00:ec2::254]", true},
		{"AWS IPv6 unbracketed", "fd00:ec2::254", true},
		{"AWS IPv6 with port", "[fd00:ec2::254]:8080", true},
		{"GCP internal", "metadata.google.internal", true},
		{"GCP goog", "metadata.goog", true},
		{"GCP trailing dot", "metadata.google.internal.", true},
		{"GCP goog trailing dot", "metadata.goog.", true},
		{"GCP uppercase", "METADATA.GOOGLE.INTERNAL", true},
		{"GCP mixed case", "Metadata.Google.Internal", true},
		{"link-local other", "169.254.42.42", true},
		{"link-local range", "169.254.1.1", true},
		{"normal host", "example.com", false},
		{"normal S3 host", "my-bucket", false},
		{"normal IP", "10.0.0.1", false},
		{"localhost", "127.0.0.1", false},
		{"empty host", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.blocked, isBlockedHost(tt.host))
		})
	}
}
