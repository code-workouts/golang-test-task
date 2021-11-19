package envconfig

import "time"

var (
	LogEventsCount   = 3
	LogBatchDuration = time.Millisecond * 1000 * 5 //5 Seconds
)
