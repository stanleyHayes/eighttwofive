package media_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hayfordstanley/eightfivetwo/services/api/internal/adapter/media"
)

func TestCloudinarySigner_DeterministicSignature(t *testing.T) {
	t.Parallel()

	fixed := time.Unix(1710000000, 0)
	makeSigner := func() *media.CloudinarySigner {
		s := media.NewCloudinarySigner("test-cloud", "api-key-123", "super-secret")
		s.SetNow(func() time.Time { return fixed })

		return s
	}

	signer := makeSigner()
	sig, err := signer.SignUpload("eightfivetwo")
	require.NoError(t, err)

	assert.Equal(t, "test-cloud", sig.CloudName)
	assert.Equal(t, "api-key-123", sig.APIKey)
	assert.Equal(t, fixed.Unix(), sig.Timestamp)
	assert.Equal(t, "eightfivetwo", sig.Folder)
	assert.NotEmpty(t, sig.Signature)

	// Same parameters must produce the same signature.
	second, err := makeSigner().SignUpload("eightfivetwo")
	require.NoError(t, err)
	assert.Equal(t, sig.Signature, second.Signature)

	// A different folder changes the signature.
	third, err := makeSigner().SignUpload("other-folder")
	require.NoError(t, err)
	assert.NotEqual(t, sig.Signature, third.Signature)

	// A different timestamp changes the signature.
	later := fixed.Add(time.Hour)
	laterSigner := media.NewCloudinarySigner("test-cloud", "api-key-123", "super-secret")
	laterSigner.SetNow(func() time.Time { return later })

	fourth, err := laterSigner.SignUpload("eightfivetwo")
	require.NoError(t, err)
	assert.NotEqual(t, sig.Signature, fourth.Signature)
}
