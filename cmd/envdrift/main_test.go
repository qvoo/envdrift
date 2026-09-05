package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunReportsMissingKeys(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "example.env")
	target := filepath.Join(dir, "local.env")
	if err := os.WriteFile(base, []byte("PORT=8080\nTOKEN=abc\n"), 0o600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(target, []byte("PORT=9090\n"), 0o600); err != nil { t.Fatal(err) }
	var out, errOut bytes.Buffer
	code := run([]string{"--values", base, target}, &out, &errOut)
	if code != 1 { t.Fatalf("exit = %d, want 1; stderr = %s", code, errOut.String()) }
	if !bytes.Contains(out.Bytes(), []byte("missing TOKEN")) { t.Fatalf("output = %s", out.String()) }
}

func TestRunJSONSuccess(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "example.env")
	target := filepath.Join(dir, "local.env")
	if err := os.WriteFile(base, []byte("PORT=8080\n"), 0o600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(target, []byte("PORT=8080\n"), 0o600); err != nil { t.Fatal(err) }
	var out, errOut bytes.Buffer
	if code := run([]string{"--format", "json", base, target}, &out, &errOut); code != 0 { t.Fatalf("exit = %d: %s", code, errOut.String()) }
	if !bytes.Contains(out.Bytes(), []byte("\"findings\": []")) { t.Fatalf("output = %s", out.String()) }
}

