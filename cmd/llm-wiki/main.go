package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m16khb/llm-wiki/internal/graph"
	"github.com/m16khb/llm-wiki/internal/hooks"
	"github.com/m16khb/llm-wiki/internal/importexport"
	indexer "github.com/m16khb/llm-wiki/internal/index"
	"github.com/m16khb/llm-wiki/internal/lint"
	"github.com/m16khb/llm-wiki/internal/logstore"
	"github.com/m16khb/llm-wiki/internal/mcp"
	"github.com/m16khb/llm-wiki/internal/okf"
	"github.com/m16khb/llm-wiki/internal/querypack"
	"github.com/m16khb/llm-wiki/internal/validate"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

func main() {
	if err := rootCmd().Execute(); err != nil {
		if exit, ok := err.(silentExit); ok {
			os.Exit(exit.code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "llm-wiki",
		Short:         "Local-first OKF-native LLM Wiki toolkit",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetVersionTemplate("llm-wiki {{.Version}}\n")
	cmd.AddCommand(initCmd(), validateCmd(), lintCmd(), indexCmd(), logCmd(), graphCmd(), queryPackCmd(), importCmd(), exportCmd(), hookCmd(), mcpCmd())
	return cmd
}

func initCmd() *cobra.Command {
	var profile string
	var okfVersion string
	cmd := &cobra.Command{
		Use:  "init <path>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if okfVersion == "" {
				okfVersion = okf.Version
			}
			if okfVersion != okf.Version {
				return fmt.Errorf("unsupported okf version %q", okfVersion)
			}
			root, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			if err := os.MkdirAll(root, 0o755); err != nil {
				return err
			}
			if profile == "obsidian" {
				if err := os.MkdirAll(filepath.Join(root, ".obsidian"), 0o755); err != nil {
					return err
				}
			}
			concept := []byte("---\ntype: concept\ntitle: Welcome\n---\n\n# Welcome\n")
			if err := os.WriteFile(filepath.Join(root, "welcome.md"), concept, 0o644); err != nil {
				return err
			}
			if _, err := indexer.Write(root); err != nil {
				return err
			}
			if _, err := logstore.Append(root, "init", "initialized OKF bundle"); err != nil {
				return err
			}
			result, err := validate.Bundle(root)
			if err != nil {
				return err
			}
			return writeJSON(cmd, result)
		},
	}
	cmd.Flags().StringVar(&profile, "profile", "obsidian", "bundle profile")
	cmd.Flags().StringVar(&okfVersion, "okf-version", okf.Version, "OKF version")
	return cmd
}

func validateCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:  "validate <path>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := validate.Bundle(args[0])
			if err != nil {
				return err
			}
			if jsonOut {
				if err := writeJSON(cmd, result); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "ok=%v concepts=%d\n", result.OK, result.ConceptCount)
			}
			if !result.OK {
				return silentExit{code: 1}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func lintCmd() *cobra.Command {
	var jsonOut bool
	var fix bool
	cmd := &cobra.Command{
		Use:  "lint <path>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := lint.Bundle(args[0], fix)
			if err != nil {
				return err
			}
			if jsonOut {
				if err := writeJSON(cmd, result); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "ok=%v warnings=%d\n", result.OK, len(result.Warnings))
			}
			if !result.OK {
				return silentExit{code: 1}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	cmd.Flags().BoolVar(&fix, "fix", false, "apply safe fixes")
	return cmd
}

func indexCmd() *cobra.Command {
	var write bool
	cmd := &cobra.Command{
		Use:  "index <path>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !write {
				return fmt.Errorf("index currently requires --write")
			}
			result, err := indexer.Write(args[0])
			if err != nil {
				return err
			}
			return writeJSON(cmd, result)
		},
	}
	cmd.Flags().BoolVar(&write, "write", false, "write index.md")
	return cmd
}

func logCmd() *cobra.Command {
	var op string
	var message string
	cmd := &cobra.Command{
		Use:  "log <path> append",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[1] != "append" {
				return fmt.Errorf("unsupported log action %q", args[1])
			}
			result, err := logstore.Append(args[0], op, message)
			if err != nil {
				return err
			}
			return writeJSON(cmd, result)
		},
	}
	cmd.Flags().StringVar(&op, "op", "", "operation name")
	cmd.Flags().StringVar(&message, "message", "", "log message")
	return cmd
}

func graphCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:  "graph <path>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := graph.Build(args[0])
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(cmd, result)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "nodes=%d edges=%d\n", len(result.Nodes), len(result.Edges))
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func queryPackCmd() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:  "query-pack <path> <question>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := querypack.Build(args[0], args[1])
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(cmd, result)
			}
			for _, ctx := range result.Contexts {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", ctx.Path, ctx.Title)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func importCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "import"}
	cmd.AddCommand(nvkCmd("import"))
	return cmd
}

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "export"}
	cmd.AddCommand(nvkCmd("export"))
	return cmd
}

func nvkCmd(action string) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:  "nvk <source> <dest>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := importexport.NVK(action, args[0], args[1], dryRun)
			if err != nil {
				return err
			}
			return writeJSON(cmd, result)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "plan without writing")
	return cmd
}

func hookCmd() *cobra.Command {
	var host string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:  "hook <event>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return err
			}
			result, err := hooks.AppendEvent(root, hooks.Event{Event: args[0], Host: host})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeJSON(cmd, result)
			}
			return writeJSON(cmd, hooks.OutputForHost(host, args[0], "noop"))
		},
	}
	cmd.Flags().StringVar(&host, "host", "codex", "hook host: claude, codex, reasonix")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit diagnostic JSON")
	return cmd
}

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run the llm-wiki MCP stdio adapter",
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcp.RunStdio(cmd.Context())
		},
	}
}

type silentExit struct {
	code int
}

func (s silentExit) Error() string {
	return "command failed"
}

func writeJSON(cmd *cobra.Command, value any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
