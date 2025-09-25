/*
Copyright © 2025 elsejj
*/
package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/elsejj/gpt/internal/llm"
	"github.com/elsejj/gpt/internal/mcps"
	"github.com/elsejj/gpt/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

//go:embed version.txt
var appVersion string

// rootCmd represents the base command when called without any subcommands
// It is the entry point for the CLI application.
var rootCmd = &cobra.Command{
	Use:   "gpt",
	Short: "a cli tool for gpt.\n version: " + appVersion,
	Long: `gpt is a cli tool for gpt. 
It send prompt from user input, file to gpt compatible api and get response back.

Version: ` + appVersion,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

		verbose := viper.GetInt("verbose")
		if verbose >= 2 {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		} else if verbose >= 1 {
			slog.SetLogLoggerLevel(slog.LevelInfo)
		} else {
			slog.SetLogLoggerLevel(slog.LevelWarn)
		}

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

		mcpServers, err := mcps.New(viper.GetStringSlice("mcp")...)
		if err != nil {
			slog.Error("Error creating mcp client", "err", err)
			os.Exit(1)
		}
		defer mcpServers.Shutdown()

		appConf.Prompt = &utils.Prompt{
			System:        utils.UserPrompt(viper.GetStringSlice("system")),
			Images:        viper.GetStringSlice("images"),
			User:          utils.UserPrompt(args),
			WithUsage:     viper.GetBool("usage"),
			JsonMode:      viper.GetBool("json"),
			OverrideModel: viper.GetString("model"),
			OnlyCodeBlock: viper.GetBool("code"),
			Temperature:   viper.GetFloat64("temperature"),
			MCPServers:    mcpServers,
		}

		appConf.PickupModel()
		var w io.Writer
		if appConf.Prompt.OnlyCodeBlock || appConf.Prompt.JsonMode {
			w = bytes.NewBuffer(nil)
		} else {
			w = os.Stdout
		}
		err = llm.Chat(appConf, w)
		if err != nil {
			slog.Error("Error sending prompt", "err", err)
			os.Exit(1)
		}

		if appConf.Prompt.OnlyCodeBlock || appConf.Prompt.JsonMode {
			buf := w.(*bytes.Buffer)
			os.Stdout.Write(llm.ExtractCodeBlock(buf.Bytes()))
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
// The function will exit the application if any error occurs.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// cobra.OnInitialize is a callback function that is called after the command is initialized.
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path, default is "+utils.ConfigPath("config.yaml"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().Float64P("temperature", "t", 1.0, "the temperature of the model")
	rootCmd.Flags().StringArrayP("system", "s", []string{}, "System prompt")
	rootCmd.Flags().StringArrayP("images", "i", []string{}, "Images to be used as prompt")
	rootCmd.Flags().BoolP("usage", "u", false, "Show usage")
	rootCmd.Flags().BoolP("json", "j", false, "force output in json format")
	rootCmd.Flags().BoolP("version", "V", false, "Show version")
	rootCmd.Flags().IntP("verbose", "v", 0, "Verbose level, 0-3, default 0, 0 is no verbose")
	rootCmd.Flags().StringP("model", "m", "", "Model override default model, with format 'model[:provider]'")
	rootCmd.Flags().BoolP("code", "c", false, "extract first code block if exists, useful for pipe code generation to next command")
	rootCmd.Flags().StringArrayP("mcp", "M", []string{}, "model context provider to be used, can be a file path(stdio) or a url(sse)")

	viper.BindPFlags(rootCmd.Flags())
}

// initConfig reads in config file and ENV variables if set.
// It also sets up the viper configuration.
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
