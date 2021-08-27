package dateutils

import "time"

func Contains(list []time.Time, test time.Time) bool {
	for _, t := range list {
		if t.Year() == test.Year() && t.Month() == test.Month() && t.Day() == test.Day() {
			return true
		}
	}
	return false
}

func OutTomorrow(date time.Time) bool {
	tomorrow := time.Now().AddDate(0, 0, 1)
	if Contains([]time.Time{date}, tomorrow) {
		return true
	}
	return false
}
