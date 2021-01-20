package main

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/omerye/gadb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	appName           = "gadb"
	defaultDeviceUser = "root"
)

var (
	configFilePath = fmt.Sprintf("$HOME/.config/%s", appName)
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
			runAsUser := viper.GetString("user")
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
		Args:  cobra.MinimumNArgs(1),
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
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cachePath := viper.GetString("cache")
			if cachePath == "" {
				return errors.New("Cache path must be set (either by flag os by config)")
			}

			serial, err := gadb.DeviceSerial()
			if err != nil {
				return err
			}

			model, err := gadb.DeviceModel()
			if err != nil {
				return err
			}

			dirName := fmt.Sprintf("%s-%s", model, serial)
			dirPath := path.Join(cachePath, dirName)
			err = os.MkdirAll(dirPath, 0755)
			if err != nil {
				return err
			}

			rootPathToCopy, err := cmd.Flags().GetStringSlice("root")
			if err != nil {
				return err
			}

			for _, p := range rootPathToCopy {
				local := path.Join(dirPath, p)
				err = gadb.Pull(p, local)
				if err != nil {
					return err
				}
			}
			return nil
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
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configFilePath)
	viper.SetEnvPrefix("GADB")

	viper.SetDefault("user", defaultDeviceUser)
	rootCmd.PersistentFlags().StringP("user", "u", "", "Device user to use")
	viper.BindEnv("USER")

	rootCmd.PersistentFlags().StringP("cache", "", "", "Root path to cache files")
	viper.BindEnv("CACHE")

	cacheCommand.PersistentFlags().StringSliceP("root", "", []string{"/system"}, "Device root paths to cache")

	viper.BindPFlags(rootCmd.Flags())

	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(pullCommand)
	rootCmd.AddCommand(ppathCommand)
	rootCmd.AddCommand(cacheCommand)
}

func main() {
	rootCmd.Execute()
}
