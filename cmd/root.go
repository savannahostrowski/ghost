package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/enescakir/emoji"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "ghost",
		Short: fmt.Sprintf("\n%v Ghost is an experimental CLI that intelligently scaffolds a GitHub Action workflow based on your local application stack and natural language, using OpenAI.", emoji.Ghost),
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(InitConfig)

	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.AddCommand(runCmd, configCmd)
}

func InitConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configHome, err := os.UserHomeDir()
		configName := ".ghost"
		configType := "yaml"

		cobra.CheckErr(err)
		viper.AddConfigPath(configHome)
		viper.SetConfigType(configType)
		viper.SetConfigName(configName)
		viper.SetDefault("enable_gpt_4", "false")
		configPath := filepath.Join(configHome, configName+"."+configType)

		if _, err := os.Stat(configPath); err == nil {
			viper.AutomaticEnv()
		} else if errors.Is(err, os.ErrNotExist) {
			if err := viper.SafeWriteConfig(); err != nil {
				if err != nil {
					panic(fmt.Sprintf("could not write config file: %v", err))
				}
			}
		}
	}
}
