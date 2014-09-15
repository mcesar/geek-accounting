package util

func Contains(s []string, e string) bool {
	return IndexOf(s, e) != -1
}

func IndexOf(s []string, e string) int {
	for i, a := range s {
		if a == e {
			return i
		}
	}
	return -1
}

func Round(f float64) float64 {
	if f < 0 {
		return float64(int(f - 0.5))
	} else {
		return float64(int(f + 0.5))
	}
}

func Min(i, j int) int {
	if i < j {
		return i
	} else {
		return j
	}
}

func Abs(i int) int {
	if i < 0 {
		return -i
	} else {
		return i
	}
}
