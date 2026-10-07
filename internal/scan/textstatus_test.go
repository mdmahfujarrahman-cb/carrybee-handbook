package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestParseTextStatuses(t *testing.T) {
	src := StripComments("export const orderStatusSchema = z.enum([\n 'pending',\n // 'old',\n \"sorted\"\n]);\n" +
		"export const hubStatusSchema = z.enum(['active']);\nexport const orderStatusGroupSchema = z.enum(['x']);")
	got := parseTextStatuses(src)
	want := []textEnum{{Name: "orderStatusSchema", Values: []string{"pending", "sorted"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestScanAppTextStatuses(t *testing.T) {
	s := ScanApp(fixtureApp(t))
	want := []string{"pending", "received_at_lmh", "delivered", "paid_return"}
	if !reflect.DeepEqual(s.TextStatuses, want) {
		t.Fatalf("text statuses = %v, want %v", s.TextStatuses, want)
	}
}

// copyTree copies src into a new temp dir and returns it.
func copyTree(t *testing.T, src string) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestScanAppReportsUnreadableFiles(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("file permissions are not enforced here")
	}
	app := fixtureApp(t)
	app.Path = copyTree(t, app.Path)
	locked := filepath.Join(app.Path, "src", "modules", "basket", "view.tsx")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(locked, 0o644); err != nil {
			t.Error(err)
		}
	})
	s := ScanApp(app)
	if s.Error != "" {
		t.Fatal(s.Error)
	}
	if want := []string{"modules/basket/view.tsx"}; !reflect.DeepEqual(s.Skipped, want) {
		t.Fatalf("skipped = %v, want %v", s.Skipped, want)
	}
}
