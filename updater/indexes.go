package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Index describes an index file pulled by the updater.
type Index struct {
	// Path is the path to the index file
	// on the update server.
	Path string

	// Channel holds the release channel name of the index.
	// It must match the filename without extension.
	Channel string

	// PreRelease signifies that all versions of this index should be marked as
	// pre-releases, no matter if the versions actually have a pre-release tag or
	// not.
	PreRelease bool

	// LastRelease holds the time of the last seen release of this index.
	LastRelease time.Time
}

// IndexFile represents an index file.
type IndexFile struct {
	Channel   string
	Published time.Time

	Releases map[string]string
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
)

// ParseIndexFile parses an index file and checks if it is valid.
func ParseIndexFile(indexData []byte, channel string, currentPublished time.Time) (*IndexFile, error) {
	// Load into struct.
	indexFile := &IndexFile{}
	err := json.Unmarshal(indexData, indexFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signed index data: %w", err)
	}

	// Fallback to old format if there are no releases and no channel is defined.
	// TODO: Remove in v0.10
	if len(indexFile.Releases) == 0 && indexFile.Channel == "" {
		return loadOldIndexFormat(indexData, channel)
	}

	// Check the index metadata.
	switch {
	case !indexFile.Published.IsZero() && time.Now().Before(indexFile.Published):
		return indexFile, ErrIndexFromFuture

	case !indexFile.Published.IsZero() &&
		!currentPublished.IsZero() &&
		currentPublished.After(indexFile.Published):
		return indexFile, ErrIndexIsOlder

	case channel != "" &&
		indexFile.Channel != "" &&
		channel != indexFile.Channel:
		return indexFile, ErrIndexChannelMismatch
	}

	return indexFile, nil
}

func loadOldIndexFormat(indexData []byte, channel string) (*IndexFile, error) {
	releases := make(map[string]string)
	err := json.Unmarshal(indexData, &releases)
	if err != nil {
		return nil, err
	}

	return &IndexFile{
		Channel:   channel,
		Published: time.Now(),
		Releases:  releases,
	}, nil
}
