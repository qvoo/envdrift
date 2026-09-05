// Package envdrift parses dotenv files and reports contract drift between them.
package envdrift

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Entry is a dotenv assignment. Value is deliberately omitted from JSON output.
type Entry struct {
	Key   string
	Value string `json:"-"`
	Line  int
}

// File is a parsed dotenv file.
type File struct {
	Path    string
	Entries map[string]Entry
}

// Finding describes one difference between the baseline and a target.
type Finding struct {
	Level  string `json:"level"`
	Kind   string `json:"kind"`
	File   string `json:"file"`
	Key    string `json:"key"`
	Detail string `json:"detail"`
}

// Report is the complete, stable output of a comparison.
type Report struct {
	Baseline string    `json:"baseline"`
	Files    []string  `json:"files"`
	Findings []Finding `json:"findings"`
	Errors   int       `json:"errors"`
	Warnings int       `json:"warnings"`
}

// Options change what counts as drift.
type Options struct {
	CompareValues bool
	IgnoreKeys    []string
}

// Parse reads a conventional dotenv file. It accepts comments, blank lines and
// an optional "export" prefix. Multiline shell expressions are intentionally
// rejected so CI output stays deterministic and easy to diagnose.
func Parse(r io.Reader, path string) (File, error) {
	file := File{Path: path, Entries: make(map[string]Entry)}
	scanner := bufio.NewScanner(r)
	// dotenv values occasionally contain JWTs or certificates. Avoid Scanner's
	// tiny default token limit while still keeping a reasonable upper bound.
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		if strings.HasPrefix(raw, "export ") {
			raw = strings.TrimSpace(strings.TrimPrefix(raw, "export "))
		}
		key, value, ok := strings.Cut(raw, "=")
		if !ok {
			return File{}, fmt.Errorf("%s:%d: expected KEY=value", path, line)
		}
		key = strings.TrimSpace(key)
		if !keyPattern.MatchString(key) {
			return File{}, fmt.Errorf("%s:%d: invalid key %q", path, line, key)
		}
		value = cleanValue(value)
		if previous, exists := file.Entries[key]; exists {
			return File{}, fmt.Errorf("%s:%d: duplicate key %q (first defined on line %d)", path, line, key, previous.Line)
		}
		file.Entries[key] = Entry{Key: key, Value: value, Line: line}
	}
	if err := scanner.Err(); err != nil {
		return File{}, fmt.Errorf("read %s: %w", path, err)
	}
	return file, nil
}

func cleanValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 0 && (value[0] == '\'' || value[0] == '"') {
		quote := value[0]
		if end := strings.IndexByte(value[1:], quote); end >= 0 {
			end++ // compensate for the slice starting after the opening quote
			rest := strings.TrimSpace(value[end+1:])
			if rest == "" || strings.HasPrefix(rest, "#") {
				return value[1:end]
			}
		}
	}
	// A comment is a comment only when separated from a value by whitespace.
	if index := strings.Index(value, " #"); index >= 0 {
		return strings.TrimSpace(value[:index])
	}
	return value
}

// Compare checks every target against baseline. Results are sorted to make
// local use, code review and CI annotations predictable.
func Compare(baseline File, targets []File, options Options) Report {
	ignored := make(map[string]bool, len(options.IgnoreKeys))
	for _, key := range options.IgnoreKeys {
		ignored[key] = true
	}
	report := Report{Baseline: baseline.Path}
	for _, target := range targets {
		report.Files = append(report.Files, target.Path)
		for key, expected := range baseline.Entries {
			if ignored[key] {
				continue
			}
			actual, found := target.Entries[key]
			if !found {
				report.Findings = append(report.Findings, Finding{
					Level: "error", Kind: "missing", File: target.Path, Key: key,
					Detail: fmt.Sprintf("required by %s", baseline.Path),
				})
				continue
			}
			if options.CompareValues && expected.Value != actual.Value {
				report.Findings = append(report.Findings, Finding{
					Level: "warning", Kind: "value", File: target.Path, Key: key,
					Detail: fmt.Sprintf("value fingerprint %s differs from baseline %s", fingerprint(actual.Value), fingerprint(expected.Value)),
				})
			}
		}
		for key := range target.Entries {
			if ignored[key] || baseline.Entries[key].Key != "" {
				continue
			}
			report.Findings = append(report.Findings, Finding{
				Level: "warning", Kind: "extra", File: target.Path, Key: key,
				Detail: fmt.Sprintf("not declared in %s", baseline.Path),
			})
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		left, right := report.Findings[i], report.Findings[j]
		if left.File != right.File { return left.File < right.File }
		if left.Key != right.Key { return left.Key < right.Key }
		return left.Kind < right.Kind
	})
	for _, finding := range report.Findings {
		if finding.Level == "error" { report.Errors++ } else { report.Warnings++ }
	}
	return report
}

func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])[:12]
}

// Fails says whether a report should lead to a non-zero CI exit status.
func (r Report) Fails(level string) bool {
	if r.Errors > 0 { return true }
	return level == "warning" && r.Warnings > 0
}

// WriteText renders human-friendly, secret-safe output.
func WriteText(w io.Writer, report Report) error {
	if len(report.Findings) == 0 {
		_, err := fmt.Fprintf(w, "envdrift: no drift found (%s 鈫?%s)\n", report.Baseline, strings.Join(report.Files, ", "))
		return err
	}
	for _, finding := range report.Findings {
		if _, err := fmt.Fprintf(w, "%s: %s %s in %s 鈥?%s\n", strings.ToUpper(finding.Level), finding.Kind, finding.Key, finding.File, finding.Detail); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "\nSummary: %d error(s), %d warning(s)\n", report.Errors, report.Warnings)
	return err
}

// WriteJSON writes machine-readable output suitable for CI annotations.
func WriteJSON(w io.Writer, report Report) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

