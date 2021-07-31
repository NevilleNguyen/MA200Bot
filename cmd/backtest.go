package cmd

import (
	"github.com/quangkeu95/binancebot/core"
	"github.com/quangkeu95/binancebot/pkg/exchange"
	"github.com/quangkeu95/binancebot/pkg/model"
	"github.com/quangkeu95/binancebot/pkg/notification"
	"github.com/spf13/cobra"
)

var backtestCmd = &cobra.Command{
	Use:   "backtest",
	Short: "Do backtesting with data from csv file",
	Long:  "Do backtesting with data from csv file",
	RunE:  backtestMain,
}

func backtestMain(cmd *cobra.Command, args []string) error {
	csvFeed, err := exchange.NewCSVFeed(
		exchange.SymbolFeed{
			SymbolInfo: model.SymbolInfo{
				Symbol:     "SXPUSDT",
				BaseAsset:  "SXP",
				QuoteAsset: "USDT",
				Status:     model.SymbolStatusTrading.String(),
			},
			Timeframe: "4h",
			File:      "testdata/sxpusdt-4h-test1.csv",
		},
		exchange.SymbolFeed{
			SymbolInfo: model.SymbolInfo{
				Symbol:     "KNCUSDT",
				BaseAsset:  "KNC",
				QuoteAsset: "USDT",
				Status:     model.SymbolStatusTrading.String(),
			},
			Timeframe: "4h",
			File:      "testdata/kncusdt-4h-test1.csv",
		},
	)

	if err != nil {
		return err
	}

	notifier := notification.NewMocNotifier()

	strategy, err := core.NewAlertOnMAStrategy(notifier)
	if err != nil {
		return err
	}
	listTimeframes := []string{"4h"}

	coreIns, err := core.New(csvFeed, strategy)
	if err != nil {
		return err
	}
	return coreIns.Run(cmd.Context(), listTimeframes)
}

func init() {
	rootCmd.AddCommand(backtestCmd)
}
