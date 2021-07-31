package cmd

import (
	"fmt"

	"github.com/quangkeu95/binancebot/pkg/download"
	"github.com/quangkeu95/binancebot/pkg/exchange"
	"github.com/spf13/cobra"
)

var (
	downloadCmdSymbol, downloadCmdTimeframe, downloadCmdOutput string
	downdloadCmdDays                                           int
	downloadCmdStart, downloadCmdEnd                           int64
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download candle within time range",
	Long:  "Download candle within time range",
	RunE:  downloadMain,
}

func downloadMain(cmd *cobra.Command, args []string) error {
	exc, err := exchange.NewBinance()
	if err != nil {
		return err
	}

	var options []download.Option

	if downdloadCmdDays > 0 {
		options = append(options, download.WithDays(downdloadCmdDays))
	} else {
		if downloadCmdStart > 0 && downloadCmdEnd > 0 && downloadCmdStart > downloadCmdEnd {
			options = append(options, download.WithInterval(downloadCmdStart, downloadCmdEnd))
		} else {
			return fmt.Errorf("START and END must be informed together")
		}
	}

	// return data.NewDownloader(exc).Download(c.Context, c.String("pair"),
	// 	c.String("timeframe"), c.String("output"), options...)
	return download.NewDownloader(exc).Download(cmd.Context(),
		downloadCmdSymbol, downloadCmdTimeframe, downloadCmdOutput, options...)
}

func init() {
	downloadCmd.Flags().StringVarP(&downloadCmdSymbol, "symbol", "S", "BTCUSDT", "Symbol config")
	downloadCmd.Flags().StringVarP(&downloadCmdTimeframe, "timeframe", "t", "1h", "Timeframe config")
	downloadCmd.Flags().StringVarP(&downloadCmdOutput, "output", "o", "./data/out.csv", "Output file")
	downloadCmd.Flags().IntVarP(&downdloadCmdDays, "days", "d", 0, "Number of days")
	downloadCmd.Flags().Int64VarP(&downloadCmdStart, "start", "s", 0, "Start time in milliseconds")
	downloadCmd.Flags().Int64VarP(&downloadCmdEnd, "end", "e", 0, "End time in milliseconds")

	rootCmd.AddCommand(downloadCmd)
}