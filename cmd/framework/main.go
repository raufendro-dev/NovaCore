package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/raufendro/novacore/internal/config"
	"github.com/raufendro/novacore/internal/database"
	"github.com/raufendro/novacore/internal/generator"
	"github.com/raufendro/novacore/internal/migration"
	"github.com/raufendro/novacore/internal/seeder"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func main() {
	root := &cobra.Command{Use: "novacore", Short: "NovaCore backend framework CLI"}
	root.AddCommand(makeCommand())
	root.AddCommand(colonMakeCommands()...)
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

func promptMethods(reader *bufio.Reader, cmd *cobra.Command, kind string) ([]string, error) {
	if kind != "crud" && kind != "module" {
		return nil, nil
	}
	for {
		fmt.Fprintln(cmd.OutOrStdout(), "Choose endpoint methods to generate. Use comma-separated values.")
		fmt.Fprintln(cmd.OutOrStdout(), "Supported methods: GET, POST, PUT, PATCH, DELETE.")
		fmt.Fprintln(cmd.OutOrStdout(), "Example: POST, GET, DELETE")
		value, err := ask(reader, cmd, "Methods [GET, POST, PUT, PATCH, DELETE]")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(value) == "" {
			value = "GET, POST, PUT, PATCH, DELETE"
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
		field, err := generator.NewField(name, dataType, required)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Invalid field: %v\n", err)
			continue
		}
		fields = append(fields, field)
	}
	return fields, nil
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
