package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"

	"github.com/raufendro/novacore/internal/app"
	"github.com/raufendro/novacore/internal/config"
	"github.com/raufendro/novacore/internal/database"
	"github.com/raufendro/novacore/internal/generator"
	"github.com/raufendro/novacore/internal/migration"
	"github.com/raufendro/novacore/internal/seeder"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const version = "0.1.0"
const author = "Rauf Endro Widagdo aka raufendro"
const novaCoreModule = "github.com/raufendro/novacore"

func Execute() {
	root := &cobra.Command{Use: "novacore", Short: "NovaCore backend framework CLI", SilenceUsage: true}
	root.AddCommand(makeCommand())
	root.AddCommand(colonMakeCommands()...)
	root.AddCommand(updateCRUDCommand("update:crud"))
	root.AddCommand(relationCommand("make:relation"))
	root.AddCommand(updateCommand())
	root.AddCommand(uninstallCommand())
	root.AddCommand(setupCommand())
	root.AddCommand(createCommand())
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show NovaCore version",
		Run: func(cmd *cobra.Command, args []string) {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "NovaCore CLI\n")
			fmt.Fprintf(out, "Version     : %s\n", version)
			fmt.Fprintf(out, "Framework   : Production-ready Go REST API framework\n")
			fmt.Fprintf(out, "Author      : %s\n", author)
			fmt.Fprintf(out, "Repository  : %s\n", novaCoreModule)
			fmt.Fprintf(out, "License     : MIT\n")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Terima kasih sudah menggunakan NovaCore.")
			fmt.Fprintln(out, "Semangat coding dan bangun backend yang rapi, aman, dan mudah dikembangkan.")
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "Run NovaCore HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Run()
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "migrate",
		Short: "Run SQL migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			conn, err := database.Connect(cfg)
			if err != nil {
				return err
			}
			return migration.Run(conn.SQL, "migrations")
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "seed",
		Short: "Run seeders",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			conn, err := database.Connect(cfg)
			if err != nil {
				return err
			}
			return seeder.Run(conn.SQL, "seeders")
		},
	})
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func updateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update NovaCore project and reinstall CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := findProjectRoot()
			if err != nil {
				return err
			}
			if err := ensureNovaCoreRoot(root); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "NovaCore project : %s\n", root)
			fmt.Fprintln(out, "Updating source with git pull...")
			if err := runInDir(cmd, root, "git", "pull"); err != nil {
				return err
			}

			fmt.Fprintln(out, "Removing old NovaCore CLI binary...")
			if err := removeInstalledBinary(out); err != nil {
				return err
			}

			fmt.Fprintln(out, "Installing latest NovaCore CLI...")
			if err := runInDir(cmd, root, "go", "install", "./cmd/novacore"); err != nil {
				return err
			}

			fmt.Fprintln(out, "NovaCore update completed.")
			return nil
		},
	}
}

func findProjectRoot() (string, error) {
	if home := strings.TrimSpace(os.Getenv("NOVACORE_HOME")); home != "" {
		abs, err := filepath.Abs(home)
		if err != nil {
			return "", err
		}
		return abs, nil
	}

	gitCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := gitCmd.Output()
	if err == nil {
		root := strings.TrimSpace(string(output))
		if ensureNovaCoreRoot(root) == nil {
			return root, nil
		}
	}

	candidates := discoverProjectRoots()
	if len(candidates) == 0 {
		return "", fmt.Errorf("cannot find NovaCore project root; run this command inside the NovaCore repository or set NOVACORE_HOME")
	}
	sortNovaCoreCandidates(candidates)
	return candidates[0], nil
}

func discoverProjectRoots() []string {
	roots := make([]string, 0)
	seen := map[string]bool{}
	for _, root := range searchRoots() {
		filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				if shouldSkipDiscoveryDir(root, path, entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Name() != "go.mod" {
				return nil
			}
			dir := filepath.Dir(path)
			if seen[dir] || ensureNovaCoreRoot(dir) != nil {
				return nil
			}
			seen[dir] = true
			roots = append(roots, dir)
			return nil
		})
	}
	return roots
}

func searchRoots() []string {
	roots := make([]string, 0)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			roots = append(roots, path)
		}
	}

	home, _ := os.UserHomeDir()
	add(filepath.Join(home, "Developer"))
	add(filepath.Join(home, "Projects"))
	add(filepath.Join(home, "Project"))
	add(filepath.Join(home, "Code"))

	if output, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		add(filepath.Join(strings.TrimSpace(string(output)), "src"))
	}
	return roots
}

func shouldSkipDiscoveryDir(root, path, name string) bool {
	if path == root {
		return false
	}
	if name == ".git" || name == "node_modules" || name == "vendor" || name == ".cache" {
		return true
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return len(strings.Split(rel, string(filepath.Separator))) > 6
}

func sortNovaCoreCandidates(candidates []string) {
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidateScore(candidates[i]) > candidateScore(candidates[j])
	})
}

func candidateScore(path string) int {
	base := strings.ToLower(filepath.Base(path))
	score := 0
	if base == "novacore" {
		score += 100
	}
	if strings.Contains(base, "novacore") {
		score += 50
	}
	if stat, err := os.Stat(filepath.Join(path, ".git")); err == nil && stat.IsDir() {
		score += 10
	}
	return score
}

func ensureNovaCoreRoot(root string) error {
	goMod := filepath.Join(root, "go.mod")
	content, err := os.ReadFile(goMod)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", goMod, err)
	}
	if !strings.Contains(string(content), "module "+novaCoreModule) {
		return fmt.Errorf("%s is not a NovaCore project root", root)
	}
	return nil
}

func runInDir(cmd *cobra.Command, dir, name string, args ...string) error {
	runCmd := exec.Command(name, args...)
	runCmd.Dir = dir
	runCmd.Stdout = cmd.OutOrStdout()
	runCmd.Stderr = cmd.ErrOrStderr()
	runCmd.Stdin = cmd.InOrStdin()
	return runCmd.Run()
}

func removeInstalledBinary(out io.Writer) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	if isGoRunBinary(executable) {
		fmt.Fprintln(out, "Skipped uninstall because NovaCore is running from go run temporary binary.")
		return nil
	}
	if filepath.Base(executable) != "novacore" {
		fmt.Fprintf(out, "Skipped uninstall because current binary is not named novacore: %s\n", executable)
		return nil
	}
	if err := os.Remove(executable); err != nil {
		return err
	}
	fmt.Fprintf(out, "Removed old binary: %s\n", executable)
	return nil
}

func setupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Configure NOVACORE_HOME and PATH for the current user",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err = filepath.Abs(root)
			if err != nil {
				return err
			}
			if err := ensureNovaCoreRoot(root); err != nil {
				return fmt.Errorf("run novacore setup from the NovaCore project root: %w", err)
			}

			goBin, err := goBinPath()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			switch runtime.GOOS {
			case "windows":
				if err := setupWindowsProfile(root, goBin); err != nil {
					return err
				}
			default:
				profile, err := setupUnixProfile(root, goBin)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "Updated shell profile: %s\n", profile)
			}

			fmt.Fprintf(out, "NOVACORE_HOME : %s\n", root)
			fmt.Fprintf(out, "Go bin path   : %s\n", goBin)
			fmt.Fprintln(out, "NovaCore environment configured.")
			fmt.Fprintln(out, "Restart your terminal or reload your shell profile before running novacore from a new directory.")
			return nil
		},
	}
}

func createCommand() *cobra.Command {
	modulePath := ""
	command := &cobra.Command{
		Use:   "create [project-name]",
		Short: "Create a new project from the NovaCore framework template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := strings.TrimSpace(args[0])
			if err := validateProjectName(projectName); err != nil {
				return err
			}
			sourceRoot, err := findProjectRoot()
			if err != nil {
				return fmt.Errorf("cannot find NovaCore template source: %w", err)
			}
			if err := ensureNovaCoreRoot(sourceRoot); err != nil {
				return err
			}
			targetRoot, err := filepath.Abs(projectName)
			if err != nil {
				return err
			}
			if err := ensureCreatableTarget(targetRoot); err != nil {
				return err
			}
			module := strings.TrimSpace(modulePath)
			if module == "" {
				module = sanitizeModulePath(filepath.Base(targetRoot))
			}
			if module == "" {
				return fmt.Errorf("module path cannot be empty")
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Creating NovaCore project: %s\n", targetRoot)
			fmt.Fprintf(out, "Template source         : %s\n", sourceRoot)
			fmt.Fprintf(out, "Module path             : %s\n", module)

			if err := copyProjectTemplate(sourceRoot, targetRoot, module); err != nil {
				return err
			}
			if err := createEnvFile(targetRoot); err != nil {
				return err
			}

			fmt.Fprintln(out, "Project created successfully.")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Next steps:")
			fmt.Fprintf(out, "  cd %s\n", projectName)
			fmt.Fprintln(out, "  go mod tidy")
			fmt.Fprintln(out, "  novacore run")
			return nil
		},
	}
	command.Flags().StringVar(&modulePath, "module", "", "Go module path for the new project")
	return command
}

func validateProjectName(name string) error {
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("project name is required")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("project name must be relative to the current directory")
	}
	clean := filepath.Clean(name)
	if strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("project name cannot point outside the current directory")
	}
	return nil
}

func ensureCreatableTarget(target string) error {
	stat, err := os.Stat(target)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s already exists and is not a directory", target)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s already exists and is not empty", target)
	}
	return nil
}

func sanitizeModulePath(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	var builder strings.Builder
	lastDash := false
	for _, r := range name {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '/' || r == '_' || r == '-' || r == '.'
		if valid {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-./")
}

func copyProjectTemplate(sourceRoot, targetRoot, modulePath string) error {
	return filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(targetRoot, 0o755)
		}
		if shouldSkipTemplatePath(rel, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		targetPath := filepath.Join(targetRoot, rel)
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		return copyTemplateFile(path, targetPath, modulePath)
	})
}

func shouldSkipTemplatePath(rel string, entry os.DirEntry) bool {
	name := entry.Name()
	if name == ".git" || name == ".cache" || name == "bin" || name == "vendor" || name == "node_modules" {
		return true
	}
	if name == ".DS_Store" || name == ".env" || name == ".rencana_update" {
		return true
	}
	if strings.HasPrefix(rel, "database"+string(filepath.Separator)) && strings.HasSuffix(name, ".db") {
		return true
	}
	return false
}

func copyTemplateFile(sourcePath, targetPath, modulePath string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if shouldRewriteTemplateFile(sourcePath) {
		content = []byte(strings.ReplaceAll(string(content), novaCoreModule, modulePath))
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, content, info.Mode().Perm())
}

func shouldRewriteTemplateFile(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".mod", ".md", ".json", ".yml", ".yaml":
		return true
	default:
		return false
	}
}

func createEnvFile(targetRoot string) error {
	example := filepath.Join(targetRoot, ".env.example")
	target := filepath.Join(targetRoot, ".env")
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	content, err := os.ReadFile(example)
	if err != nil {
		return err
	}
	return os.WriteFile(target, content, 0o644)
}

func goBinPath() (string, error) {
	if gobin := strings.TrimSpace(os.Getenv("GOBIN")); gobin != "" {
		return filepath.Abs(gobin)
	}
	output, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return "", fmt.Errorf("cannot read GOPATH with go env: %w", err)
	}
	gopath := strings.TrimSpace(string(output))
	if gopath == "" {
		return "", fmt.Errorf("go env GOPATH returned empty value")
	}
	return filepath.Join(gopath, "bin"), nil
}

func setupUnixProfile(root, goBin string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	shellName := filepath.Base(os.Getenv("SHELL"))
	profile := filepath.Join(home, ".profile")
	switch shellName {
	case "zsh":
		profile = filepath.Join(home, ".zshrc")
	case "bash":
		profile = filepath.Join(home, ".bashrc")
	case "fish":
		profile = filepath.Join(home, ".config", "fish", "config.fish")
	}
	block := unixEnvBlock(root, goBin, shellName == "fish")
	return profile, writeManagedBlock(profile, block)
}

func setupWindowsProfile(root, goBin string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	profiles := []string{
		filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
	}
	block := windowsEnvBlock(root, goBin)
	for _, profile := range profiles {
		if err := writeManagedBlock(profile, block); err != nil {
			return err
		}
	}
	return nil
}

func unixEnvBlock(root, goBin string, fish bool) string {
	if fish {
		return fmt.Sprintf("set -gx NOVACORE_HOME %q\nfish_add_path %q\n", root, goBin)
	}
	return fmt.Sprintf("export NOVACORE_HOME=%q\ncase \":$PATH:\" in\n  *\":%s:\"*) ;;\n  *) export PATH=\"%s:$PATH\" ;;\nesac\n", root, goBin, goBin)
}

func windowsEnvBlock(root, goBin string) string {
	return fmt.Sprintf("$env:NOVACORE_HOME = %q\n[Environment]::SetEnvironmentVariable('NOVACORE_HOME', %q, 'User')\n$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')\nif (($userPath -split ';') -notcontains %q) {\n  [Environment]::SetEnvironmentVariable('Path', ($userPath.TrimEnd(';') + ';' + %q), 'User')\n}\n", root, root, goBin, goBin)
}

func writeManagedBlock(path, block string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	contentBytes, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(contentBytes)
	start := "# >>> NovaCore >>>"
	end := "# <<< NovaCore <<<"
	managed := start + "\n" + block + end + "\n"

	if strings.Contains(content, start) && strings.Contains(content, end) {
		before, rest, _ := strings.Cut(content, start)
		_, after, _ := strings.Cut(rest, end)
		content = strings.TrimRight(before, "\n") + "\n" + managed + strings.TrimLeft(after, "\n")
	} else {
		if strings.TrimSpace(content) != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n" + managed
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func uninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove installed NovaCore CLI binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			executable, err := os.Executable()
			if err != nil {
				return err
			}
			executable, err = filepath.EvalSymlinks(executable)
			if err != nil {
				return err
			}
			if isGoRunBinary(executable) {
				return fmt.Errorf("novacore uninstall must be run from an installed binary, not go run")
			}
			if filepath.Base(executable) != "novacore" {
				return fmt.Errorf("refusing to remove %s because it is not named novacore", executable)
			}
			if err := removeInstalledBinary(cmd.OutOrStdout()); err != nil {
				return err
			}
			return nil
		},
	}
}

func isGoRunBinary(path string) bool {
	clean := filepath.Clean(path)
	return strings.Contains(clean, string(filepath.Separator)+"go-build") ||
		strings.Contains(clean, string(filepath.Separator)+"go-build-") ||
		strings.Contains(clean, string(filepath.Separator)+"go-build"+string(filepath.Separator))
}

func colonMakeCommands() []*cobra.Command {
	kinds := map[string]string{
		"make:module":     "module",
		"make:model":      "model",
		"make:controller": "controller",
		"make:service":    "service",
		"make:repository": "repository",
		"make:endpoint":   "endpoint",
		"make:crud":       "crud",
	}
	commands := make([]*cobra.Command, 0, len(kinds)+1)
	for use, kind := range kinds {
		kind := kind
		public := false
		command := &cobra.Command{
			Use:   use + " [Name]",
			Short: "Generate " + kind,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				reader := bufio.NewReader(cmd.InOrStdin())
				methods, err := promptMethods(reader, cmd, kind)
				if err != nil {
					return err
				}
				fields, err := promptFields(reader, cmd, kind)
				if err != nil {
					return err
				}
				return generator.GenerateWithOptions(kind, args[0], generator.Options{Public: public, Fields: fields, Methods: methods})
			},
		}
		if kind == "crud" || kind == "module" || kind == "endpoint" {
			command.Flags().BoolVar(&public, "public", false, "generate routes without JWT auth middleware")
		}
		commands = append(commands, command)
	}
	commands = append(commands, &cobra.Command{
		Use:   "make:migration [name]",
		Short: "Generate SQL migration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generator.GenerateMigration(args[0])
		},
	})
	commands = append(commands, &cobra.Command{
		Use:   "make:seeder [name]",
		Short: "Generate SQL seeder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generator.GenerateSeeder(args[0])
		},
	})
	return commands
}

func updateCRUDCommand(use string) *cobra.Command {
	public := false
	mode := "reset"
	command := &cobra.Command{
		Use:   use + " [Name]",
		Short: "Update generated CRUD fields, routes, docs, Postman, and reset migration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(cmd.InOrStdin())
			ok, err := confirmDestructiveCRUDUpdate(reader, cmd)
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "CRUD update cancelled.")
				return nil
			}
			old, _ := generator.LoadMetadata(args[0])
			showExistingDefinition(cmd, old)
			methods, err := promptMethodsWithDefault(reader, cmd, "crud", old.Methods)
			if err != nil {
				return err
			}
			fields, err := promptUpdateFields(reader, cmd, old.Fields, mode == "safe")
			if err != nil {
				return err
			}
			return generator.UpdateCRUD(args[0], generator.Options{Public: public, Fields: fields, Methods: methods, Mode: mode})
		},
	}
	command.Flags().BoolVar(&public, "public", false, "generate routes without JWT auth middleware")
	command.Flags().StringVar(&mode, "mode", "reset", "update migration mode: reset or safe")
	command.Flags().Bool("safe", false, "generate safe ALTER TABLE migration")
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		safe, _ := cmd.Flags().GetBool("safe")
		if safe {
			mode = "safe"
		}
		mode = strings.ToLower(strings.TrimSpace(mode))
		if mode != "reset" && mode != "safe" {
			return fmt.Errorf("--mode must be reset or safe")
		}
		return nil
	}
	return command
}

func makeCommand() *cobra.Command {
	make := &cobra.Command{Use: "make", Short: "Generate framework files"}
	add := func(use, kind string) {
		public := false
		command := &cobra.Command{
			Use:   use + " [Name]",
			Short: "Generate " + kind,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				reader := bufio.NewReader(cmd.InOrStdin())
				methods, err := promptMethods(reader, cmd, kind)
				if err != nil {
					return err
				}
				fields, err := promptFields(reader, cmd, kind)
				if err != nil {
					return err
				}
				return generator.GenerateWithOptions(kind, args[0], generator.Options{Public: public, Fields: fields, Methods: methods})
			},
		}
		if kind == "crud" || kind == "module" || kind == "endpoint" {
			command.Flags().BoolVar(&public, "public", false, "generate routes without JWT auth middleware")
		}
		make.AddCommand(command)
	}
	add("module", "module")
	add("model", "model")
	add("controller", "controller")
	add("service", "service")
	add("repository", "repository")
	add("endpoint", "endpoint")
	add("crud", "crud")
	make.AddCommand(updateCRUDCommand("update-crud"))
	make.AddCommand(relationCommand("relation"))
	make.AddCommand(&cobra.Command{
		Use:   "migration [name]",
		Short: "Generate SQL migration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generator.GenerateMigration(args[0])
		},
	})
	make.AddCommand(&cobra.Command{
		Use:   "seeder [name]",
		Short: "Generate SQL seeder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generator.GenerateSeeder(args[0])
		},
	})
	return make
}

func relationCommand(use string) *cobra.Command {
	var relationType string
	var nested bool
	var include bool
	var updatePostman bool
	var updateDocs bool
	var foreignKey string
	command := &cobra.Command{
		Use:   use + " [Source] [Target]",
		Short: "Generate relation between two CRUD models",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || len(args) == 2 {
				return nil
			}
			return fmt.Errorf("use make:relation for interactive mode or make:relation Source Target")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(cmd.InOrStdin())
			source := ""
			target := ""
			nonInteractive := len(args) == 2 && relationType != ""
			if len(args) == 2 {
				source = args[0]
				target = args[1]
			} else {
				var err error
				source, err = ask(reader, cmd, "Source model")
				if err != nil {
					return err
				}
				target, err = ask(reader, cmd, "Target model")
				if err != nil {
					return err
				}
			}
			if relationType == "" {
				selected, err := promptRelationType(reader, cmd)
				if err != nil {
					return err
				}
				relationType = selected
			}
			if foreignKey == "" {
				defaultFK := toSnakeName(target) + "_id"
				if nonInteractive {
					foreignKey = defaultFK
				} else {
					value, err := ask(reader, cmd, "Foreign key ["+defaultFK+"]")
					if err != nil {
						return err
					}
					if strings.TrimSpace(value) == "" {
						foreignKey = defaultFK
					} else {
						foreignKey = value
					}
				}
			}
			if !nonInteractive && !cmd.Flags().Changed("nested") {
				answer, err := ask(reader, cmd, "Generate nested endpoints? [y/N]")
				if err != nil {
					return err
				}
				nested = strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes")
			}
			if !nonInteractive && !cmd.Flags().Changed("include") {
				answer, err := ask(reader, cmd, "Enable include query? [Y/n]")
				if err != nil {
					return err
				}
				include = !strings.EqualFold(answer, "n") && !strings.EqualFold(answer, "no")
			}
			if !nonInteractive && !cmd.Flags().Changed("postman") {
				answer, err := ask(reader, cmd, "Update Postman? [Y/n]")
				if err != nil {
					return err
				}
				updatePostman = !strings.EqualFold(answer, "n") && !strings.EqualFold(answer, "no")
			}
			if !nonInteractive && !cmd.Flags().Changed("docs") {
				answer, err := ask(reader, cmd, "Update docs? [Y/n]")
				if err != nil {
					return err
				}
				updateDocs = !strings.EqualFold(answer, "n") && !strings.EqualFold(answer, "no")
			}
			return generator.ApplyRelation(generator.Relation{Source: source, Target: target, Type: relationType, ForeignKey: foreignKey, Nested: nested, Include: include, UpdatePostman: updatePostman, UpdateDocs: updateDocs})
		},
	}
	command.Flags().StringVar(&relationType, "type", "", "relation type: belongs-to, has-one, has-many, many-to-many")
	command.Flags().BoolVar(&nested, "nested", false, "generate nested endpoint scaffold")
	command.Flags().BoolVar(&include, "include", false, "enable include query for this relation")
	command.Flags().BoolVar(&updatePostman, "postman", true, "update Postman collection")
	command.Flags().BoolVar(&updateDocs, "docs", true, "update docs")
	command.Flags().StringVar(&foreignKey, "foreign-key", "", "foreign key column")
	return command
}

func promptRelationType(reader *bufio.Reader, cmd *cobra.Command) (string, error) {
	fmt.Fprintln(cmd.OutOrStdout(), "Relation type:")
	fmt.Fprintln(cmd.OutOrStdout(), "[1] belongs-to")
	fmt.Fprintln(cmd.OutOrStdout(), "[2] has-one")
	fmt.Fprintln(cmd.OutOrStdout(), "[3] has-many")
	fmt.Fprintln(cmd.OutOrStdout(), "[4] many-to-many")
	for {
		value, err := ask(reader, cmd, "Choose")
		if err != nil {
			return "", err
		}
		switch strings.TrimSpace(value) {
		case "1":
			return "belongs-to", nil
		case "2":
			return "has-one", nil
		case "3":
			return "has-many", nil
		case "4":
			return "many-to-many", nil
		case "belongs-to", "has-one", "has-many", "many-to-many":
			return value, nil
		default:
			fmt.Fprintln(cmd.OutOrStdout(), "Choose 1, 2, 3, or 4.")
		}
	}
}

func confirmDestructiveCRUDUpdate(reader *bufio.Reader, cmd *cobra.Command) (bool, error) {
	fmt.Fprintln(cmd.OutOrStdout(), "Updating CRUD columns will create a reset migration.")
	fmt.Fprintln(cmd.OutOrStdout(), "Existing table data will be deleted and IDs will restart from 0 after the migration is run.")
	for {
		answer, err := ask(reader, cmd, "Continue? [y/N]")
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "y":
			return true, nil
		case "n", "":
			return false, nil
		default:
			fmt.Fprintln(cmd.OutOrStdout(), "Please answer y/Y or n/N.")
		}
	}
}

func promptMethods(reader *bufio.Reader, cmd *cobra.Command, kind string) ([]string, error) {
	return promptMethodsWithDefault(reader, cmd, kind, nil)
}

func promptMethodsWithDefault(reader *bufio.Reader, cmd *cobra.Command, kind string, defaults []string) ([]string, error) {
	if kind != "crud" && kind != "module" {
		return nil, nil
	}
	if len(defaults) == 0 {
		defaults = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	}
	defaultText := strings.Join(defaults, ", ")
	for {
		fmt.Fprintln(cmd.OutOrStdout(), "Choose endpoint methods to generate. Use comma-separated values.")
		fmt.Fprintln(cmd.OutOrStdout(), "Supported methods: GET, POST, PUT, PATCH, DELETE.")
		fmt.Fprintln(cmd.OutOrStdout(), "Example: POST, GET, DELETE")
		value, err := ask(reader, cmd, "Methods ["+defaultText+"]")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(value) == "" {
			value = defaultText
		}
		methods, err := generator.NormalizeMethodNames(strings.Split(value, ","))
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Invalid methods: %v\n", err)
			continue
		}
		return methods, nil
	}
}

func promptFields(reader *bufio.Reader, cmd *cobra.Command, kind string) ([]generator.Field, error) {
	if kind != "crud" && kind != "module" {
		return nil, nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Define fields for this CRUD. Default fields are already included: id, created_at, updated_at, deleted_at.")
	fmt.Fprintln(cmd.OutOrStdout(), "Supported types: string, text, int, uint, float, bool, time.")
	fmt.Fprintln(cmd.OutOrStdout(), "Press Ctrl+D on field name when finished.")

	fields := []generator.Field{}
	reserved := map[string]struct{}{"id": {}, "created_at": {}, "updated_at": {}, "deleted_at": {}}
	for {
		name, finish, err := askFieldName(reader, cmd)
		if err != nil {
			return nil, err
		}
		if finish {
			break
		}
		name = strings.TrimSpace(name)
		if name == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "Field name is required. Press Ctrl+D on field name when finished.")
			continue
		}
		snake := toSnakeName(name)
		if _, ok := reserved[snake]; ok {
			fmt.Fprintf(cmd.OutOrStdout(), "%s is already included by default.\n", snake)
			continue
		}
		dataType, err := ask(reader, cmd, "Type [string]")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(dataType) == "" {
			dataType = "string"
		}
		requiredInput, err := ask(reader, cmd, "Required? [y/N]")
		if err != nil {
			return nil, err
		}
		required := strings.EqualFold(strings.TrimSpace(requiredInput), "y") || strings.EqualFold(strings.TrimSpace(requiredInput), "yes")
		defaultValue, err := ask(reader, cmd, "Default value [none]")
		if err != nil {
			return nil, err
		}
		field, err := generator.NewFieldWithDefault(name, dataType, required, defaultValue)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Invalid field: %v\n", err)
			continue
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func showExistingDefinition(cmd *cobra.Command, meta generator.Metadata) {
	fmt.Fprintf(cmd.OutOrStdout(), "Updating CRUD: %s\n\n", meta.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Existing methods:\n[%s]\n\n", strings.Join(meta.Methods, ", "))
	fmt.Fprintln(cmd.OutOrStdout(), "Existing fields:")
	if len(meta.Fields) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "- none")
		return
	}
	for i, field := range meta.Fields {
		required := "optional"
		if field.Required {
			required = "required"
		}
		defaultValue := field.DefaultValue
		if defaultValue == "" {
			defaultValue = "none"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d. %s %s %s default=%q\n", i+1, field.JSONName, field.GoType, required, defaultValue)
	}
	fmt.Fprintln(cmd.OutOrStdout())
}

func promptUpdateFields(reader *bufio.Reader, cmd *cobra.Command, existing []generator.Field, safe bool) ([]generator.Field, error) {
	fields := append([]generator.Field{}, existing...)
	for {
		fmt.Fprintln(cmd.OutOrStdout(), "Choose action:")
		fmt.Fprintln(cmd.OutOrStdout(), "[1] Add field")
		fmt.Fprintln(cmd.OutOrStdout(), "[2] Edit field")
		fmt.Fprintln(cmd.OutOrStdout(), "[3] Remove field")
		fmt.Fprintln(cmd.OutOrStdout(), "[4] Continue")
		action, err := ask(reader, cmd, "Action")
		if err != nil {
			return nil, err
		}
		switch strings.TrimSpace(action) {
		case "1":
			field, err := promptSingleField(reader, cmd)
			if err != nil {
				return nil, err
			}
			if safe && field.Required && !field.HasDefault {
				fmt.Fprintf(cmd.OutOrStdout(), "Field %q is required. Safe migration needs a default value to fill existing rows.\n", field.JSONName)
				fmt.Fprintln(cmd.OutOrStdout(), "Please provide a default value.")
				continue
			}
			fields = upsertField(fields, field)
			printFields(cmd, fields)
		case "2":
			printFields(cmd, fields)
			index, err := askIndex(reader, cmd, len(fields))
			if err != nil {
				return nil, err
			}
			field, err := promptSingleFieldWithDefault(reader, cmd, fields[index])
			if err != nil {
				return nil, err
			}
			if safe && field.Required && !field.HasDefault && !sameFieldExists(existing, field.SnakeName) {
				fmt.Fprintf(cmd.OutOrStdout(), "Field %q is required. Safe migration needs a default value to fill existing rows.\n", field.JSONName)
				fmt.Fprintln(cmd.OutOrStdout(), "Please provide a default value.")
				continue
			}
			fields[index] = field
			printFields(cmd, fields)
		case "3":
			printFields(cmd, fields)
			index, err := askIndex(reader, cmd, len(fields))
			if err != nil {
				return nil, err
			}
			fields = append(fields[:index], fields[index+1:]...)
			printFields(cmd, fields)
		case "4":
			return fields, nil
		default:
			fmt.Fprintln(cmd.OutOrStdout(), "Please choose 1, 2, 3, or 4.")
		}
	}
}

func promptSingleField(reader *bufio.Reader, cmd *cobra.Command) (generator.Field, error) {
	return promptSingleFieldWithDefault(reader, cmd, generator.Field{})
}

func promptSingleFieldWithDefault(reader *bufio.Reader, cmd *cobra.Command, current generator.Field) (generator.Field, error) {
	for {
		namePrompt := "Field name"
		if current.JSONName != "" {
			namePrompt += " [" + current.JSONName + "]"
		}
		name, err := ask(reader, cmd, namePrompt)
		if err != nil {
			return generator.Field{}, err
		}
		if strings.TrimSpace(name) == "" {
			name = current.JSONName
		}
		if strings.TrimSpace(name) == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "Field name is required.")
			continue
		}
		typeDefault := "string"
		if current.GoType != "" {
			typeDefault = current.GoType
		}
		dataType, err := ask(reader, cmd, "Type ["+typeDefault+"]")
		if err != nil {
			return generator.Field{}, err
		}
		if strings.TrimSpace(dataType) == "" {
			dataType = typeDefault
		}
		requiredDefault := "N"
		if current.Required {
			requiredDefault = "Y"
		}
		requiredInput, err := ask(reader, cmd, "Required? [y/N] current="+requiredDefault)
		if err != nil {
			return generator.Field{}, err
		}
		required := current.Required
		if strings.TrimSpace(requiredInput) != "" {
			required = strings.EqualFold(strings.TrimSpace(requiredInput), "y") || strings.EqualFold(strings.TrimSpace(requiredInput), "yes")
		}
		defaultPrompt := "Default value [none]"
		if current.DefaultValue != "" {
			defaultPrompt = "Default value [" + current.DefaultValue + "]"
		}
		defaultValue, err := ask(reader, cmd, defaultPrompt)
		if err != nil {
			return generator.Field{}, err
		}
		if strings.TrimSpace(defaultValue) == "" {
			defaultValue = current.DefaultValue
		}
		field, err := generator.NewFieldWithDefault(name, dataType, required, defaultValue)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Invalid field: %v\n", err)
			continue
		}
		return field, nil
	}
}

func printFields(cmd *cobra.Command, fields []generator.Field) {
	fmt.Fprintln(cmd.OutOrStdout(), "Current fields:")
	if len(fields) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "- none")
		return
	}
	for i, field := range fields {
		required := "optional"
		if field.Required {
			required = "required"
		}
		defaultValue := field.DefaultValue
		if defaultValue == "" {
			defaultValue = "none"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d. %s %s %s default=%q\n", i+1, field.JSONName, field.GoType, required, defaultValue)
	}
}

func askIndex(reader *bufio.Reader, cmd *cobra.Command, total int) (int, error) {
	for {
		value, err := ask(reader, cmd, "Field number")
		if err != nil {
			return 0, err
		}
		var index int
		if _, err := fmt.Sscanf(value, "%d", &index); err != nil || index < 1 || index > total {
			fmt.Fprintf(cmd.OutOrStdout(), "Choose a number between 1 and %d.\n", total)
			continue
		}
		return index - 1, nil
	}
}

func upsertField(fields []generator.Field, field generator.Field) []generator.Field {
	for i := range fields {
		if fields[i].SnakeName == field.SnakeName {
			fields[i] = field
			return fields
		}
	}
	return append(fields, field)
}

func sameFieldExists(fields []generator.Field, snake string) bool {
	for _, field := range fields {
		if field.SnakeName == snake {
			return true
		}
	}
	return false
}

func askFieldName(reader *bufio.Reader, cmd *cobra.Command) (string, bool, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return askFieldNameRaw(cmd)
	}
	fmt.Fprint(cmd.OutOrStdout(), "Field name: ")
	value, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && strings.TrimSpace(value) == "" {
			return "", true, nil
		}
		if err == io.EOF {
			return strings.TrimSpace(value), false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(value), false, nil
}

func askFieldNameRaw(cmd *cobra.Command) (string, bool, error) {
	fmt.Fprint(cmd.OutOrStdout(), "Field name: ")
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", false, err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var builder strings.Builder
	buffer := make([]byte, 1)
	for {
		_, err := os.Stdin.Read(buffer)
		if err != nil {
			return "", false, err
		}
		ch := buffer[0]
		switch ch {
		case 0x04:
			fmt.Fprintln(cmd.OutOrStdout())
			return "", true, nil
		case '\r', '\n':
			fmt.Fprintln(cmd.OutOrStdout())
			return builder.String(), false, nil
		case 0x7f, '\b':
			if builder.Len() > 0 {
				value := []rune(builder.String())
				builder.Reset()
				builder.WriteString(string(value[:len(value)-1]))
				fmt.Fprint(cmd.OutOrStdout(), "\b \b")
			}
		case 0x1b:
			_ = readEscapeSequence()
		default:
			r := rune(ch)
			if unicode.IsPrint(r) {
				builder.WriteByte(ch)
				fmt.Fprintf(cmd.OutOrStdout(), "%c", ch)
			}
		}
	}
}

func readEscapeSequence() string {
	buffer := make([]byte, 1)
	var builder strings.Builder
	builder.WriteByte(0x1b)
	for i := 0; i < 16; i++ {
		_, err := os.Stdin.Read(buffer)
		if err != nil {
			break
		}
		builder.WriteByte(buffer[0])
		ch := buffer[0]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '~' {
			break
		}
	}
	return builder.String()
}

func ask(reader *bufio.Reader, cmd *cobra.Command, label string) (string, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func toSnakeName(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for i, r := range value {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				builder.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		if r == '-' || r == ' ' {
			builder.WriteByte('_')
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
