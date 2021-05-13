// +build linux

package renameio

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestTempDir(t *testing.T) {
	if tmpdir, ok := os.LookupEnv("TMPDIR"); ok {
		defer os.Setenv("TMPDIR", tmpdir) // restore
	} else {
		defer os.Unsetenv("TMPDIR") // restore
	}

	mount1, err := ioutil.TempDir("", "tempdirtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(mount1)

	mount2, err := ioutil.TempDir("", "tempdirtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(mount2)

	if err := syscall.Mount("tmpfs", mount1, "tmpfs", 0, ""); err != nil {
		t.Skipf("cannot mount tmpfs on %s: %v", mount1, err)
	}
	defer syscall.Unmount(mount1, 0)

	if err := syscall.Mount("tmpfs", mount2, "tmpfs", 0, ""); err != nil {
		t.Skipf("cannot mount tmpfs on %s: %v", mount2, err)
	}
	defer syscall.Unmount(mount2, 0)

	tests := []struct {
		name   string
		dir    string
		path   string
		TMPDIR string
		want   string
	}{
		{
			name: "implicit TMPDIR",
			path: filepath.Join(os.TempDir(), "foo.txt"),
			want: os.TempDir(),
		},

		{
			name:   "explicit TMPDIR",
			path:   filepath.Join(mount1, "foo.txt"),
			TMPDIR: mount1,
			want:   mount1,
		},

		{
			name:   "explicit unsuitable TMPDIR",
			path:   filepath.Join(mount1, "foo.txt"),
			TMPDIR: mount2,
			want:   mount1,
		},

		{
			name:   "nonexistant TMPDIR",
			path:   filepath.Join(mount1, "foo.txt"),
			TMPDIR: "/nonexistant",
			want:   mount1,
		},

		{
			name:   "caller-specified",
			dir:    "/overridden",
			path:   filepath.Join(mount1, "foo.txt"),
			TMPDIR: "/nonexistant",
			want:   "/overridden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.TMPDIR == "" {
				os.Unsetenv("TMPDIR")
			} else {
				os.Setenv("TMPDIR", tt.TMPDIR)
			}
			if got := tempDir(tt.dir, tt.path); got != tt.want {
				t.Fatalf("tempDir(%q, %q): got %q, want %q", tt.dir, tt.path, got, tt.want)
			}
		})
	}
}
