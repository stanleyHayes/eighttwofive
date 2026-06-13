// Package media contains UploadSigner adapters.
package media

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/cloudinary/cloudinary-go/v2/api"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/domain"
)

// CloudinarySigner produces signatures for direct browser-to-Cloudinary uploads,
// so file bytes never pass through this API.
type CloudinarySigner struct {
	cloudName string
	apiKey    string
	apiSecret string
	now       func() time.Time
}

// NewCloudinarySigner builds a signer from Cloudinary credentials.
func NewCloudinarySigner(cloudName, apiKey, apiSecret string) *CloudinarySigner {
	return &CloudinarySigner{
		cloudName: cloudName,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		now:       time.Now,
	}
}

// SetNow overrides the clock used for signing timestamps. Exposed for tests.
func (s *CloudinarySigner) SetNow(now func() time.Time) {
	s.now = now
}

// SignUpload signs the upload parameters for the given folder.
func (s *CloudinarySigner) SignUpload(folder string) (domain.UploadSignature, error) {
	timestamp := s.now().Unix()
	params := url.Values{
		"timestamp": []string{strconv.FormatInt(timestamp, 10)},
		"folder":    []string{folder},
	}

	signature, err := api.SignParameters(params, s.apiSecret)
	if err != nil {
		return domain.UploadSignature{}, fmt.Errorf("sign upload params: %w", err)
	}

	return domain.UploadSignature{
		CloudName: s.cloudName,
		APIKey:    s.apiKey,
		Timestamp: timestamp,
		Folder:    folder,
		Signature: signature,
	}, nil
}
