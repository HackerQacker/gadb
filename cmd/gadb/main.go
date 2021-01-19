package main

import (
	"fmt"
	"os"

	"github.com/omerye/gadb"
	"github.com/spf13/cobra"
)

var (
	runAsUser string
	cachePath string
)

var (
	rootCmd = &cobra.Command{
		Use:   "gadb",
		Short: "gadb is like adb but do more",
		Long:  `An extended adb written in Go.`,
	}

	shellCmd = &cobra.Command{
		Use:                "shell COMMAND",
		Short:              "run shell command",
		Long:               `Run remote shell command on device with shell or root command (default: root)`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return gadb.Shell(runAsUser)
			}

			return gadb.UserCommand(runAsUser, args[0], args[1:]...).Run()
		},
	}

	/* TODO:
	 * --sync
	 * many remote files (REMOTE... LOCAL)
	 */
	pullCommand = &cobra.Command{
		Use:   "pull REMOTE [LOCAL]",
		Short: "pull files from device",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 2 {
				return gadb.Pull(args[0], args[1])
			}

			local, err := os.Getwd()
			if err != nil {
				return err
			}

			return gadb.Pull(args[0], local)
		},
	}

	cacheCommand = &cobra.Command{
		Use:   "cache",
		Short: "Save/remove device's files locally",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// TODO
		},
	}

	ppathCommand = &cobra.Command{
		Use:   "ppath PACKAGE",
		Short: "Get APK path of a given package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ppath, err := gadb.PackagePath(args[0])
			if err != nil {
				return err
			}

			fmt.Println(ppath)
			return nil
		},
	}
)

func init() {
	defaultCache := "/tmp/.gadb"
	if c := os.Getenv("GADB_CACHE"); c != "" {
		defaultCache = c
	}

	rootCmd.PersistentFlags().StringVarP(&runAsUser, "user", "u", "root", "Device user to use")
	rootCmd.PersistentFlags().StringVarP(&cachePath, "cache", "", defaultCache, "Root path to cache files")
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(pullCommand)
	rootCmd.AddCommand(ppathCommand)
}

func main() {
	rootCmd.Execute()
}
