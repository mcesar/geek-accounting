package collections

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
