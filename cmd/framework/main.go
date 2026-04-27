package main

import (
	"fmt"
	"os"

	"github.com/raufendro/novacore/internal/config"
	"github.com/raufendro/novacore/internal/database"
	"github.com/raufendro/novacore/internal/generator"
	"github.com/raufendro/novacore/internal/migration"
	"github.com/raufendro/novacore/internal/seeder"
	"github.com/spf13/cobra"
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
				return generator.GenerateWithOptions(kind, args[0], generator.Options{Public: public})
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
				return generator.GenerateWithOptions(kind, args[0], generator.Options{Public: public})
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
