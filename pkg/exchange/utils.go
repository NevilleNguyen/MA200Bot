package exchange

func ParseTimeframeToSeconds(timeframe string) int64 {
	switch timeframe {
	case "1h":
		return 3600
	case "4h":
		return 14400
	case "1d":
		return 86400
	}
	return 0
}
