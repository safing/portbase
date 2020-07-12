package updater

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/google/renameio"

	"github.com/safing/portbase/log"
)

// Common errors.
var (
	ErrUnexpectedStatusCode = errors.New("received unexpected status")
	ErrIncomplete           = errors.New("incomplete download")
)

func (reg *ResourceRegistry) fetchFile(ctx context.Context, client *http.Client, rv *ResourceVersion, tries int) error {
	// backoff when retrying
	if tries > 0 {
		select {
		case <-ctx.Done():
			return nil // module is shutting down
		case <-time.After(time.Duration(tries*tries) * time.Second):
		}
	}

	// check destination dir
	dirPath := filepath.Dir(rv.storagePath())
	err := reg.storageDir.EnsureAbsPath(dirPath)
	if err != nil {
		return fmt.Errorf("could not create updates folder %s: %w", dirPath, err)
	}

	// open file for writing
	atomicFile, err := renameio.TempFile(reg.tmpDir.Path, rv.storagePath())
	if err != nil {
		return fmt.Errorf("could not create temp file for download: %w", err)
	}
	defer atomicFile.Cleanup() //nolint:errcheck // ignore error for now, tmp dir will be cleaned later again anyway

	// start file download
	resp, downloadURL, err := reg.makeRequest(ctx, client, rv.versionedPath(), tries)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// download and write file
	n, err := io.Copy(atomicFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download %q: %w", downloadURL, err)
	}
	if resp.ContentLength != n {
		return fmt.Errorf("failed to finish download of %q: received %d out of %d bytes: %w", downloadURL, n, resp.ContentLength, ErrIncomplete)
	}

	// finalize file
	err = atomicFile.CloseAtomicallyReplace()
	if err != nil {
		return fmt.Errorf("%s: failed to finalize file %s: %w", reg.Name, rv.storagePath(), err)
	}
	// set permissions
	if !onWindows {
		// TODO: only set executable files to 0755, set other to 0644
		err = os.Chmod(rv.storagePath(), 0o755)
		if err != nil {
			log.Warningf("%s: failed to set permissions on downloaded file %s: %s", reg.Name, rv.storagePath(), err)
		}
	}

	log.Infof("%s: fetched %s (stored to %s)", reg.Name, downloadURL, rv.storagePath())
	return nil
}

func (reg *ResourceRegistry) fetchData(ctx context.Context, client *http.Client, downloadPath string, tries int) ([]byte, error) {
	// backoff when retrying
	if tries > 0 {
		select {
		case <-ctx.Done():
			return nil, nil // module is shutting down
		case <-time.After(time.Duration(tries*tries) * time.Second):
		}
	}

	// start file download
	resp, downloadURL, err := reg.makeRequest(ctx, client, downloadPath, tries)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// download and write file
	buf := bytes.NewBuffer(make([]byte, 0, resp.ContentLength))
	n, err := io.Copy(buf, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to download %q: %w", downloadURL, err)
	}
	if resp.ContentLength != n {
		return nil, fmt.Errorf("failed to finish download of %q: received %d out of %d bytes: %w", downloadURL, n, resp.ContentLength, ErrIncomplete)
	}

	return buf.Bytes(), nil
}

func (reg *ResourceRegistry) makeRequest(ctx context.Context, client *http.Client, downloadPath string, tries int) (resp *http.Response, downloadURL string, err error) {
	// parse update URL
	updateBaseURL := reg.UpdateURLs[tries%len(reg.UpdateURLs)]
	u, err := url.Parse(updateBaseURL)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse update URL %q: %w", updateBaseURL, err)
	}
	// add download path
	u.Path = path.Join(u.Path, downloadPath)
	// compile URL
	downloadURL = u.String()

	// create request
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, http.NoBody) //nolint:gosec
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request for %q: %w", downloadURL, err)
	}

	// set user agent
	if reg.UserAgent != "" {
		req.Header.Set("User-Agent", reg.UserAgent)
	}

	// start request
	resp, err = client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to make request to %q: %w", downloadURL, err)
	}

	// check return code
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("failed to fetch %q: %d %s", downloadURL, resp.StatusCode, resp.Status)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch %q: %w %d %s", downloadURL, ErrUnexpectedStatusCode, resp.StatusCode, resp.Status)
	}


	return resp, downloadURL, err
}
