package sbom

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/afelin/cyberready/internal/buildinfo"
)

// Summary is a lightweight SBOM digest kept for backward compatibility.
type Summary struct {
	Status       string   `json:"status"`
	Format       string   `json:"format"`
	GeneratedAt  string   `json:"generated_at"`
	Source       string   `json:"source,omitempty"`
	PackageCount int      `json:"package_count"`
	Packages     []string `json:"packages,omitempty"`
	Note         string   `json:"note"`
	CycloneDXPath string  `json:"cyclonedx_path,omitempty"`
}

// Component is one CycloneDX component.
type Component struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	PURL    string `json:"purl,omitempty"`
	BomRef  string `json:"bom-ref,omitempty"`
}

// Document is a CycloneDX 1.5 BOM (stdlib JSON subset).
type Document struct {
	BomFormat    string `json:"bomFormat"`
	SpecVersion  string `json:"specVersion"`
	SerialNumber string `json:"serialNumber,omitempty"`
	Version      int    `json:"version"`
	Metadata     struct {
		Timestamp string `json:"timestamp"`
		Tools     struct {
			Components []Component `json:"components"`
		} `json:"tools"`
		Component Component `json:"component"`
	} `json:"metadata"`
	Components []Component `json:"components"`
}

// Package is a normalized name@version from a lockfile.
type Package struct {
	Name      string
	Version   string
	Ecosystem string // npm|golang
}

// FromLockfiles scans npm/pnpm lockfiles if present (summary view).
func FromLockfiles(root string) (Summary, error) {
	pkgs, source, err := CollectPackages(root)
	now := time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		return Summary{}, err
	}
	names := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		names = append(names, p.Name+"@"+p.Version)
	}
	sort.Strings(names)
	return Summary{
		Status:       "ok",
		Format:       "cyclonedx-1.5-compatible-summary",
		GeneratedAt:  now,
		Source:       source,
		PackageCount: len(pkgs),
		Packages:     truncate(names, 40),
		Note:         "Draft inventory for human review. Not a signed SBOM attestation. See CycloneDX JSON for full BOM.",
	}, nil
}

// FileDigest returns sha256 hex of path contents, or "" if unreadable.
func FileDigest(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// WriteCycloneDX writes a CycloneDX 1.5 JSON BOM under outPath (or default evidence path).
func WriteCycloneDX(root, outPath string) (Document, string, error) {
	pkgs, source, err := CollectPackages(root)
	if err != nil {
		return Document{}, "", err
	}
	doc := BuildCycloneDX(root, pkgs, source)
	if outPath == "" {
		outPath = filepath.Join(root, ".github", "cyberready", "evidence", "sbom.cdx.json")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return Document{}, "", err
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return Document{}, "", err
	}
	if err := os.WriteFile(outPath, append(b, '\n'), 0o644); err != nil {
		return Document{}, "", err
	}
	return doc, outPath, nil
}

// BuildCycloneDX constructs a CycloneDX 1.5 document from packages.
// Metadata timestamp/serial are derived from package content (not wall clock) so digests are reproducible.
func BuildCycloneDX(root string, pkgs []Package, source string) Document {
	product := filepath.Base(root)
	comps := make([]Component, 0, len(pkgs))
	for _, p := range pkgs {
		eco := p.Ecosystem
		if eco == "" {
			eco = "npm"
		}
		var ref string
		switch eco {
		case "golang", "go":
			ref = "pkg:golang/" + p.Name + "@" + p.Version
		default:
			ref = "pkg:npm/" + p.Name + "@" + p.Version
		}
		comps = append(comps, Component{
			Type:    "library",
			Name:    p.Name,
			Version: p.Version,
			PURL:    ref,
			BomRef:  ref,
		})
	}
	sort.Slice(comps, func(i, j int) bool { return comps[i].BomRef < comps[j].BomRef })
	var seed strings.Builder
	seed.WriteString(product)
	seed.WriteByte('|')
	seed.WriteString(source)
	for _, c := range comps {
		seed.WriteByte('|')
		seed.WriteString(c.BomRef)
	}
	sum := sha256.Sum256([]byte(seed.String()))
	serial := fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
	var doc Document
	doc.BomFormat = "CycloneDX"
	doc.SpecVersion = "1.5"
	doc.SerialNumber = serial
	doc.Version = 1
	doc.Metadata.Timestamp = "2024-01-01T00:00:00Z"
	doc.Metadata.Tools.Components = []Component{{
		Type:    "application",
		Name:    "cyberready",
		Version: buildinfo.Version,
	}}
	doc.Metadata.Component = Component{
		Type:   "application",
		Name:   product,
		BomRef: "app:" + product,
	}
	doc.Components = comps
	return doc
}

// CollectPackages returns normalized packages and the source lockfile name.
// Prefers npm/pnpm lockfiles; also merges go.mod require lines when present.
func CollectPackages(root string) ([]Package, string, error) {
	var out []Package
	sources := make([]string, 0, 2)
	seen := map[string]struct{}{}
	add := func(pkgs []Package) {
		for _, p := range pkgs {
			key := p.Ecosystem + "|" + p.Name + "@" + p.Version
			if p.Ecosystem == "" {
				key = "npm|" + p.Name + "@" + p.Version
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, p)
		}
	}

	if p := filepath.Join(root, "pnpm-lock.yaml"); fileExists(p) {
		pkgs, err := parsePnpmLockPackages(p)
		if err != nil {
			return nil, "", err
		}
		for i := range pkgs {
			pkgs[i].Ecosystem = "npm"
		}
		add(pkgs)
		sources = append(sources, "pnpm-lock.yaml")
	} else if p := filepath.Join(root, "package-lock.json"); fileExists(p) {
		pkgs, err := parseNPMLockPackages(p)
		if err != nil {
			return nil, "", err
		}
		for i := range pkgs {
			pkgs[i].Ecosystem = "npm"
		}
		add(pkgs)
		sources = append(sources, "package-lock.json")
	} else if p := filepath.Join(root, "package.json"); fileExists(p) {
		pkgs, err := parsePackageJSONPackages(p)
		if err != nil {
			return nil, "", err
		}
		for i := range pkgs {
			pkgs[i].Ecosystem = "npm"
		}
		add(pkgs)
		sources = append(sources, "package.json")
	}

	if p := filepath.Join(root, "go.mod"); fileExists(p) {
		pkgs, err := parseGoModPackages(p)
		if err != nil {
			return nil, "", err
		}
		add(pkgs)
		sources = append(sources, "go.mod")
	}

	if len(out) == 0 && len(sources) == 0 {
		return nil, "", fmt.Errorf("no package-lock.json, pnpm-lock.yaml, package.json, or go.mod")
	}
	return out, strings.Join(sources, "+"), nil
}

// IsUnavailable reports whether err means no supported lockfile/manifest (not an I/O failure).
func IsUnavailable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no package-lock.json, pnpm-lock.yaml, package.json, or go.mod")
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func truncate(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	return append(in[:n], fmt.Sprintf("…+%d more", len(in)-n))
}

func parsePackageJSONPackages(path string) ([]Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	var out []Package
	for k, v := range m.Dependencies {
		out = append(out, Package{Name: k, Version: v})
	}
	for k, v := range m.DevDependencies {
		out = append(out, Package{Name: k, Version: v})
	}
	return out, nil
}

func parseNPMLockPackages(path string) ([]Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lock struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}
	var out []Package
	seen := map[string]struct{}{}
	if len(lock.Packages) > 0 {
		for name, meta := range lock.Packages {
			if name == "" {
				continue
			}
			n := strings.TrimPrefix(name, "node_modules/")
			if strings.Contains(n, "node_modules/") {
				continue
			}
			key := n + "@" + meta.Version
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, Package{Name: n, Version: meta.Version})
		}
		return out, nil
	}
	for name, meta := range lock.Dependencies {
		out = append(out, Package{Name: name, Version: meta.Version})
	}
	return out, nil
}

func parsePnpmLockPackages(path string) ([]Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Package
	seen := map[string]struct{}{}
	inPackages := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "packages:") {
			inPackages = true
			continue
		}
		if !inPackages {
			continue
		}
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(line, "packages") {
			break
		}
		trim := strings.TrimSpace(line)
		if !strings.HasPrefix(trim, "/") || !strings.HasSuffix(trim, ":") {
			continue
		}
		pkg := strings.TrimSuffix(trim, ":")
		name, ver := splitPnpmKey(pkg)
		if name == "" {
			continue
		}
		key := name + "@" + ver
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Package{Name: name, Version: ver})
	}
	return out, sc.Err()
}

func splitPnpmKey(key string) (name, version string) {
	key = strings.TrimPrefix(key, "/")
	// /axios@1.6.0 or /@scope/pkg@1.0.0
	if strings.HasPrefix(key, "@") {
		idx := strings.LastIndex(key, "@")
		if idx <= 0 {
			return key, ""
		}
		return key[:idx], key[idx+1:]
	}
	idx := strings.LastIndex(key, "@")
	if idx < 0 {
		return key, ""
	}
	return key[:idx], key[idx+1:]
}

func parseGoModPackages(path string) ([]Package, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Package
	inBlock := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "require (") {
			inBlock = true
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			line = strings.TrimSuffix(line, "// indirect")
			line = strings.TrimSpace(line)
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				out = append(out, Package{Name: fields[0], Version: fields[1], Ecosystem: "golang"})
			}
			continue
		}
		if strings.HasPrefix(line, "require ") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				out = append(out, Package{Name: fields[1], Version: fields[2], Ecosystem: "golang"})
			}
		}
	}
	return out, sc.Err()
}
