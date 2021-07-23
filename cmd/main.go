package cmd

import (
	"context"

	"github.com/quangkeu95/binancebot/config"
	"github.com/quangkeu95/binancebot/core"
	"github.com/quangkeu95/binancebot/lib/app"
	"github.com/quangkeu95/binancebot/pkg/exchange"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "binancebot",
	Short: "Start Binance bot",
	Long:  "Start Binance bot",
	RunE:  rootMain,
}

func rootMain(cmd *cobra.Command, args []string) error {
	ex, err := exchange.NewBinance()
	if err != nil {
		return err
	}

	alertOnMAStrategy, err := core.NewAlertOnMAStrategy()
	if err != nil {
		return err
	}

	coreIns, err := core.New(ex, alertOnMAStrategy)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	return coreIns.Run(ctx)
}

func Execute() {
	l, flush, err := app.NewSugaredLogger()
	if err != nil {
		panic(err)
	}

	defer func() {
		flush()
	}()
	zap.ReplaceGlobals(l.Desugar())

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(config.InitConfig)
	// watch for config file changes

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", "", "config file (by default app will try to find config in ./env/mainnet.json)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
