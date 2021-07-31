package download

import (
	"context"
	"encoding/csv"
	"os"
	"time"

	"github.com/quangkeu95/binancebot/pkg/exchange"
	"github.com/xhit/go-str2duration/v2"
	"go.uber.org/zap"
)

const batchSize = 500

type Downloader struct {
	l        *zap.SugaredLogger
	exchange exchange.Exchange
}

func NewDownloader(exchange exchange.Exchange) *Downloader {
	return &Downloader{
		l:        zap.S(),
		exchange: exchange,
	}
}

type Parameters struct {
	Start time.Time
	End   time.Time
}

type Option func(*Parameters)

func WithInterval(start, end int64) Option {
	return func(parameters *Parameters) {
		parameters.Start = time.Unix(0, start*int64(time.Millisecond))
		parameters.End = time.Unix(0, end*int64(time.Millisecond))
	}
}

func WithDays(days int) Option {
	return func(parameters *Parameters) {
		parameters.Start = time.Now().AddDate(0, 0, -days)
		parameters.End = time.Now()
	}
}

func candlesCount(start, end time.Time, timeframe string) (int, time.Duration, error) {
	totalDuration := end.Sub(start)
	interval, err := str2duration.ParseDuration(timeframe)
	if err != nil {
		return 0, 0, err
	}
	return int(totalDuration / interval), interval, nil
}

func (d *Downloader) Download(ctx context.Context, symbol, timeframe string, output string, options ...Option) error {
	recordFile, err := os.Create(output)
	if err != nil {
		return err
	}

	now := time.Now()
	parameters := &Parameters{
		Start: now.AddDate(0, -1, 0),
		End:   now,
	}

	for _, option := range options {
		option(parameters)
	}

	parameters.Start = time.Date(parameters.Start.Year(), parameters.Start.Month(), parameters.Start.Day(),
		0, 0, 0, 0, time.UTC)
	parameters.End = time.Date(parameters.End.Year(), parameters.End.Month(), parameters.End.Day(),
		0, 0, 0, 0, time.UTC)

	candlesCount, interval, err := candlesCount(parameters.Start, parameters.End, timeframe)
	if err != nil {
		return err
	}

	d.l.Infow("Downloading candle..", "symbol", symbol, "timeframe", timeframe, "candle_count", candlesCount)
	writer := csv.NewWriter(recordFile)
	for begin := parameters.Start; begin.Before(parameters.End); begin = begin.Add(interval * batchSize) {
		end := begin.Add(interval * batchSize)
		if end.After(parameters.End) {
			end = parameters.End
		}

		candles, err := d.exchange.CandlesByPeriod(ctx, symbol, timeframe, begin, end)
		if err != nil {
			return err
		}

		for _, candle := range candles {
			err := writer.Write(candle.ToSlice())
			if err != nil {
				return err
			}
		}
	}
	writer.Flush()
	d.l.Infow("Downloading done")
	return writer.Error()
}
