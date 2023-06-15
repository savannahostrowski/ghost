package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/enescakir/emoji"
	"github.com/savannahostrowski/ghost/ui"

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
		home, _ := os.UserHomeDir()
		path := home + "/.ghost.yaml"
		config, err := loadConfig(path)

		if err != nil {
			return err
		}

		apikey := config.OpenAIAPIKey
		enableGPT4 := config.EnableGPT4

		if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
			return err
		}

		if args[0] == "OPENAI_API_KEY" {
			viper.Set("OPENAI_API_KEY", args[1])
			viper.Set("ENABLE_GPT_4", enableGPT4)
			viper.WriteConfig()
			return nil
		}

		if args[0] == "ENABLE_GPT_4" {
			if args[1] != "true" && args[1] != "false" {
				return fmt.Errorf("invalid value: %s. accepts true or false", args[1])
			}

			viper.Set("OPENAI_API_KEY", apikey)
			viper.Set("ENABLE_GPT_4", args[1])
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
	EnableGPT4   string `mapstructure:"enable_gpt_4"`
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the configuration for Ghost",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		path := home + "/.ghost.yaml"
		config, err := loadConfig(path)

		if err != nil {
			panic(fmt.Errorf("cannot load config file: %w", err))
		}

		var style = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ui.HotPink))

		fmt.Printf("Your Ghost %v configuration file is located at %v\n\n"+
			"OPENAI_API_KEY: %v\n"+
			"ENABLE_GPT_4: %v\n", emoji.Ghost, style.Render(path), style.Render(config.OpenAIAPIKey), style.Render(config.EnableGPT4))
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
