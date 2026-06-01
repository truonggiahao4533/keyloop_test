package postgrerepository

import "time"

func durationToTime(d time.Duration) time.Time {
	return time.Date(0, 1, 1, int(d.Hours()), int(d.Minutes())%60, 0, 0, time.UTC)
}

func timeToDuration(t time.Time) time.Duration {
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute
}
