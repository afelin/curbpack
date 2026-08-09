package demo

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/afelin/cyberready/internal/config"
	"github.com/afelin/cyberready/internal/packs"
	"github.com/afelin/cyberready/internal/release"
	"github.com/afelin/cyberready/internal/tty"
	"github.com/afelin/cyberready/internal/validate"
)

//go:embed data/*
var demoFS embed.FS

const Claim = "Prepares evidence for human review — not a conformity assessment."

// Options for the sandbox demo.
type Options struct {
	KeepDir     bool   // keep temp dir (print path)
	OutDir      string // optional fixed output dir (still never mutates caller product cwd)
	Version     string
	OpenBrowser bool // opt-in: open buyer-onepager in the system browser (default false)
}

// JailOutDir refuses --out that equals or sits under the caller's cwd (product tree).
// Resolves symlinks via EvalSymlinks so a link into the product tree cannot bypass the jail.
func JailOutDir(out, cwd string) error {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	outAbs, err := resolveJailPath(out)
	if err != nil {
		return err
	}
	cwdAbs, err := resolveJailPath(cwd)
	if err != nil {
		return err
	}
	if outAbs == cwdAbs {
		return fmt.Errorf("demo --out refuses product cwd (use a temp path or omit --out)")
	}
	sep := string(os.PathSeparator)
	if strings.HasPrefix(outAbs, cwdAbs+sep) {
		return fmt.Errorf("demo --out refuses paths under product cwd (data-loss jail)")
	}
	return nil
}

func resolveJailPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	// Path may not exist yet — resolve the nearest existing ancestor.
	dir, base := filepath.Dir(abs), filepath.Base(abs)
	for dir != filepath.Dir(dir) {
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			return filepath.Join(resolved, base), nil
		}
		base = filepath.Join(filepath.Base(dir), base)
		dir = filepath.Dir(dir)
	}
	return abs, nil
}

// Run copies the embedded demo-app into a temp git repo, inits house-policy, checks, prepares release.
// It never writes into the caller's product working tree.
func Run(opts Options) error {
	tty.PrintHeader("CYBERREADY DEMO (SANDBOX)")
	fmt.Printf("%s\n", tty.C(tty.Dim, Claim))
	fmt.Printf("%s\n\n", tty.C(tty.Dim, "Isolated temp git repo — your product cwd is not modified."))

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := JailOutDir(opts.OutDir, cwd); err != nil {
		return err
	}

	dir := opts.OutDir
	if dir == "" {
		dir, err = os.MkdirTemp("", "cyberready-demo-*")
		if err != nil {
			return err
		}
		if !opts.KeepDir {
			defer func() {
				_ = os.RemoveAll(dir)
			}()
		}
	} else {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	tty.PrintStatus("sandbox", true, dir)

	if err := copyEmbedded(dir); err != nil {
		return err
	}
	tty.PrintStatus("fixture", true, "testdata/demo-app (house-policy)")

	if err := gitInit(dir); err != nil {
		return err
	}
	tty.PrintStatus("git init", true, "sandbox only")

	// Write config + stubs without touching caller cwd
	cfg := config.File{
		Packs:   []string{"house-policy"},
		Version: opts.Version,
		Claim:   Claim,
	}
	if err := config.Write(dir, cfg); err != nil {
		return err
	}
	paths, err := packs.ScaffoldPaths([]string{"house-policy"})
	if err != nil {
		return err
	}
	for _, rel := range paths {
		p, clean, err := validate.SafeJoin(dir, rel)
		if err != nil {
			return fmt.Errorf("demo scaffold path refused: %s: %w", rel, err)
		}
		if _, err := os.Stat(p); err == nil {
			continue
		}
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(packs.DefaultScaffoldBody(clean)), 0o644); err != nil {
			return err
		}
	}
	_ = os.MkdirAll(filepath.Join(dir, "proof"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "proof", "index.html"), []byte(release.ProofPageHTML()), 0o644)
	_ = os.MkdirAll(filepath.Join(dir, ".github", "cyberready", "cache"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, ".github", "cyberready", "evidence"), 0o755)
	tty.PrintStatus("init", true, "packs=house-policy")

	res, err := validate.Run(validate.Options{RepoRoot: dir})
	if err != nil {
		return err
	}
	if tty.IsTerminal {
		tty.RenderThermometer(res.Score)
	}
	fmt.Println(res.ActionReport)
	if !res.Passed {
		return fmt.Errorf("demo check failed — sandbox at %s (fixture may be stale)", dir)
	}
	tty.PrintStatus("check", true, fmt.Sprintf("score=%d%%", res.Score))

	if err := release.Prepare(release.Options{RepoRoot: dir, PackIDs: []string{"house-policy"}}); err != nil {
		return err
	}
	onepager := filepath.Join(dir, "review-pack", "buyer-onepager.html")
	tty.PrintStatus("buyer-onepager", true, onepager)
	if opts.OpenBrowser {
		openOnePager(onepager)
	} else {
		fmt.Printf("%s\n", tty.C(tty.Dim, "one-pager path printed above — open with: cyberready demo --open (or open the file)"))
	}

	fmt.Printf("\n%s\n", tty.C(tty.Bold+tty.Green, "[+] Demo green in sandbox"))
	if opts.KeepDir || opts.OutDir != "" {
		fmt.Printf("%s\n", tty.C(tty.Dim, "sandbox kept at: "+dir))
	}
	fmt.Printf("%s\n", tty.C(tty.Dim, "next on your repo: cyberready init && cyberready check"))
	fmt.Printf("%s\n", tty.C(tty.Dim, "CI-only: uses: afelin/cyberready@v0.4.2  (heal: true)"))
	fmt.Printf("%s\n", tty.C(tty.Dim, Claim))
	return nil
}

func copyEmbedded(dest string) error {
	return fs.WalkDir(demoFS, "data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("data", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := demoFS.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

func gitInit(dir string) error {
	cmds := [][]string{
		{"git", "init", "-q"},
		{"git", "config", "user.email", "demo@cyberready.local"},
		{"git", "config", "user.name", "CyberReady Demo"},
		{"git", "add", "-A"},
		{"git", "commit", "--allow-empty", "-m", "cyberready demo sandbox", "-q"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %w (%s)", strings.Join(c, " "), err, strings.TrimSpace(string(out)))
		}
	}
	// Re-add after empty commit so working tree files are committed for realism
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = dir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "demo fixture", "-q")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	_ = cmd.Run()
	return nil
}

func openOnePager(path string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return
	}
	_ = cmd.Start()
}
