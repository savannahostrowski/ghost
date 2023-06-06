package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure Ghost CLI",
}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a configuration for Ghost",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
			return err
		}
		if args[0] == "OPENAI_API_KEY" {
			viper.Set(args[0], args[1])
			viper.WriteConfig()
			return nil
		}

		return fmt.Errorf("invalid key: %s", args[0])
	},
	Run: func(cmd *cobra.Command, args []string) {

	},
}

type Config struct {
	OpenAIAPIKey string `mapstructure:"openai_api_key"`
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the configuration for Ghost",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		path := home + "/.ghost.yaml"
		config, err := loadConfig(path)

		if err != nil {
			log.Fatal("cannot load config:", err)
		}
		fmt.Println("OPENAI_API_KEY=",config.OpenAIAPIKey)

	},
}

func init() {
	configCmd.AddCommand(setCmd, getCmd)
}

func loadConfig(path string) (config Config, err error) {
    viper.AddConfigPath(path)
    viper.SetConfigName(".ghost")
    viper.SetConfigType("yaml")

    viper.AutomaticEnv()

    err = viper.ReadInConfig()
    if err != nil {
        return
    }

    err = viper.Unmarshal(&config)
    return
}