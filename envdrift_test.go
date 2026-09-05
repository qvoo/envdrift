package envdrift

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, source, path string) File {
	t.Helper()
	file, err := Parse(strings.NewReader(source), path)
	if err != nil { t.Fatal(err) }
	return file
}

func TestParseAcceptsCommonDotenvSyntax(t *testing.T) {
	file := mustParse(t, "# app config\nexport PORT = 8080\nNAME=\"Ada Lovelace\" # comment\nTOKEN=a#b\n", "sample.env")
	if got, want := file.Entries["PORT"].Value, "8080"; got != want { t.Fatalf("PORT = %q, want %q", got, want) }
	if got, want := file.Entries["NAME"].Value, "Ada Lovelace"; got != want { t.Fatalf("NAME = %q, want %q", got, want) }
	if got, want := file.Entries["TOKEN"].Value, "a#b"; got != want { t.Fatalf("TOKEN = %q, want %q", got, want) }
}

func TestParseRejectsDuplicateKeys(t *testing.T) {
	_, err := Parse(strings.NewReader("PORT=1\nPORT=2\n"), "duplicate.env")
	if err == nil || !strings.Contains(err.Error(), "duplicate key") { t.Fatalf("got %v, want duplicate-key error", err) }
}

func TestCompareFindsShapeAndValueDrift(t *testing.T) {
	base := mustParse(t, "PORT=8080\nAPI_KEY=abc\n", "example.env")
	target := mustParse(t, "PORT=9090\nDEBUG=true\n", "local.env")
	report := Compare(base, []File{target}, Options{CompareValues: true})
	if report.Errors != 1 || report.Warnings != 2 { t.Fatalf("errors/warnings = %d/%d, want 1/2", report.Errors, report.Warnings) }
	if !report.Fails("error") || !report.Fails("warning") { t.Fatal("expected report to fail") }
	for _, finding := range report.Findings {
		if strings.Contains(finding.Detail, "9090") || strings.Contains(finding.Detail, "8080") { t.Fatalf("finding leaked a value: %#v", finding) }
	}
}

func TestCompareCanIgnoreKeys(t *testing.T) {
	base := mustParse(t, "PORT=8080\n", "example.env")
	target := mustParse(t, "PORT=9090\n", "local.env")
	report := Compare(base, []File{target}, Options{CompareValues: true, IgnoreKeys: []string{"PORT"}})
	if len(report.Findings) != 0 { t.Fatalf("findings = %#v, want none", report.Findings) }
}

