package series

func MA(data []float64, periods int) float64 {
	if len(data) == 0 || len(data) < periods {
		return 0
	}

	var sum = float64(0)
	for i := len(data) - 1; i >= len(data)-periods; i-- {
		sum += data[i]
	}
	return sum / float64(periods)
}
