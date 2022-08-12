package updater

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/safing/portbase/formats/dsd"

	"github.com/safing/jess"
)

// VerificationOptions holds options for verification of files.
type VerificationOptions struct {
	TrustStore     jess.TrustStore
	DownloadPolicy SignaturePolicy
	DiskLoadPolicy SignaturePolicy
}

// GetVerificationOptions returns the verification options for the given identifier.
func (reg *ResourceRegistry) GetVerificationOptions(identifier string) *VerificationOptions {
	if reg.Verification == nil {
		return nil
	}

	var (
		longestPrefix = -1
		bestMatch     *VerificationOptions
	)
	for prefix, opts := range reg.Verification {
		if len(prefix) > longestPrefix && strings.HasPrefix(identifier, prefix) {
			longestPrefix = len(prefix)
			bestMatch = opts
		}
	}

	return bestMatch
}

// SignaturePolicy defines behavior in case of errors.
type SignaturePolicy uint8

// Signature Policies.
const (
	// SignaturePolicyRequire fails on any error.
	SignaturePolicyRequire = iota

	// SignaturePolicyWarn only warns on errors.
	SignaturePolicyWarn

	// SignaturePolicyDisable only downloads signatures, but does not verify them.
	SignaturePolicyDisable
)

// IndexFile represents an index file.
type IndexFile struct {
	Channel   string
	Published time.Time
	Expires   time.Time

	Versions map[string]string
}

var (
	// ErrIndexFromFuture is returned when a signed index is parsed with a
	// Published timestamp that lies in the future.
	ErrIndexFromFuture = errors.New("index is from the future")

	// ErrIndexIsOlder is returned when a signed index is parsed with an older
	// Published timestamp than the current Published timestamp.
	ErrIndexIsOlder = errors.New("index is older than the current one")

	// ErrIndexChannelMismatch is returned when a signed index is parsed with a
	// different channel that the expected one.
	ErrIndexChannelMismatch = errors.New("index does not match the expected channel")

	// ErrIndexExpired is returned when a signed index is parsed with a Expires
	// timestamp in the past.
	ErrIndexExpired = errors.New("index has expired")

	verificationRequirements = jess.NewRequirements().
					Remove(jess.Confidentiality).
					Remove(jess.RecipientAuthentication)
)

// ParseIndex parses the signed index and checks if it is valid.
func ParseIndex(indexData []byte, verifOpts *VerificationOptions, channel string, currentPublished time.Time) (*IndexFile, error) {
	// FIXME: fall back to the old index format.
	// FIXME: use this function for index parsing.

	// Parse data.
	letter, err := jess.LetterFromDSD(indexData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signed index: %w", err)
	}

	// Verify signatures.
	signedIndexData, err := letter.Open(verificationRequirements, verifOpts.TrustStore)
	if err != nil {
		return nil, fmt.Errorf("failed to verify signature: %w", err)
	}

	// Load into struct.
	signedIndex := &IndexFile{}
	_, err = dsd.Load(signedIndexData, signedIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signed index data: %w", err)
	}

	// Check the index metadata.
	switch {
	case time.Now().Before(signedIndex.Published):
		return signedIndex, ErrIndexFromFuture

	case time.Now().After(signedIndex.Expires):
		return signedIndex, ErrIndexExpired

	case currentPublished.After(signedIndex.Published):
		return signedIndex, ErrIndexIsOlder

	case channel != "" && channel != signedIndex.Channel:
		return signedIndex, ErrIndexChannelMismatch
	}

	return signedIndex, nil
}
