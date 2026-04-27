package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func Execute() {
	root := &cobra.Command{Use: "novacore", Short: "NovaCore backend framework CLI"}
	root.AddCommand(makeCommand())
	root.AddCommand(colonMakeCommands()...)
	root.AddCommand(updateCRUDCommand("update:crud"))
	root.AddCommand(relationCommand("make:relation"))
	root.AddCommand(updateCommand())
	root.AddCommand(uninstallCommand())
	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show NovaCore version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "NovaCore %s\n", version)
			fmt.Fprintln(cmd.OutOrStdout(), "Created by Rauf Endro Widagdo aka raufendro")
			fmt.Fprintln(cmd.OutOrStdout(), "Terima kasih sudah menggunakan NovaCore. Semangat coding dan bangun backend yang rapi!")
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
		Short: "Update NovaCore project with git pull",
		RunE: func(cmd *cobra.Command, args []string) error {
			gitCmd := exec.Command("git", "pull")
			gitCmd.Stdout = cmd.OutOrStdout()
			gitCmd.Stderr = cmd.ErrOrStderr()
			gitCmd.Stdin = cmd.InOrStdin()
			return gitCmd.Run()
		},
	}
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
			if err := os.Remove(executable); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "NovaCore CLI removed from %s\n", executable)
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
