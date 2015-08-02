package math

func Round(f float64) float64 {
	if f < 0 {
		return float64(int(f - 0.5))
	} else {
		return float64(int(f + 0.5))
	}
}

func Max(i, j int) int {
	if i > j {
		return i
	} else {
		return j
	}
}

func MaxU64(i, j uint64) uint64 {
	if i > j {
		return i
	} else {
		return j
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
