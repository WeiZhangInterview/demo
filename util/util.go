package util

func Contains(s []any, e any) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
