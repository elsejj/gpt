/*
Copyright © 2025 elsejj
*/
package cmd

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/elsejj/gpt/internal/llm"
	"github.com/elsejj/gpt/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

//go:embed version.txt
var appVersion string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gpt",
	Short: "a cli tool for gpt.\n version: " + appVersion,
	Long: `gpt is a cli tool for gpt. 
It send prompt from user input, file to gpt compatible api and get response back.

Version: ` + appVersion,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if len(cfgFile) == 0 {
			cfgFile = utils.ConfigPath("config.yaml")
		}
		err := utils.InitConfig(cfgFile)
		if err != nil {
			os.Exit(1)
		}
		appConf, err := utils.LoadConfig(cfgFile)
		if err != nil {
			os.Exit(1)
		}
		if viper.GetBool("version") || len(args) == 0 {
			fmt.Println("Version:    ", appVersion)
			fmt.Println("ConfigFile: ", cfgFile)
			fmt.Println("Gateway:    ", appConf.LLM.Gateway)
			fmt.Println("Provider:   ", appConf.LLM.Provider)
			fmt.Println("Model:      ", appConf.LLM.Model)
			os.Exit(0)
		}

		appConf.Prompt = &utils.Prompt{
			System:    utils.UserPrompt(viper.GetStringSlice("system")),
			Images:    viper.GetStringSlice("images"),
			User:      utils.UserPrompt(args),
			WithUsage: viper.GetBool("usage"),
			JsonMode:  viper.GetBool("json"),
			Verbose:   viper.GetBool("verbose"),
		}
		w := os.Stdout
		err = llm.Chat(appConf, w)
		if err != nil {
			slog.Error("Error sending prompt", "err", err)
			os.Exit(1)
		}
		os.Stdout.WriteString("\n")
		os.Stdout.Sync()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path, default is "+utils.ConfigPath("config.yaml"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().StringArrayP("system", "s", []string{}, "System prompt")
	rootCmd.Flags().StringArrayP("images", "i", []string{}, "Images to be used as prompt")
	rootCmd.Flags().BoolP("usage", "u", false, "Show usage")
	rootCmd.Flags().BoolP("json", "j", false, "force output in json format")
	rootCmd.Flags().BoolP("version", "v", false, "Show version")
	rootCmd.Flags().BoolP("verbose", "V", false, "Verbose output")

	viper.BindPFlags(rootCmd.Flags())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".gpt" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gpt")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
