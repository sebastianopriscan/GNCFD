package ntptime

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

func GetNTPTime() (time.Time, error) {
	var ts unix.Timespec
	err := unix.ClockGettime(unix.CLOCK_REALTIME, &ts)
	if err != nil {
		return time.Time{}, fmt.Errorf("error getting time from OS, details %s", err)
	}

	// Convert the timespec to a time.Time object
	return time.Unix(int64(ts.Sec), int64(ts.Nsec)), nil
}
