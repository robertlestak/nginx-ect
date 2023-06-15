package utils

func Contains(slice []string, search string) bool {
	for _, item := range slice {
		if item == search {
			return true
		}
	}
	return false
}

func CleanSlice(slice []string) []string {
	// ensure no duplicates, and that no values are empty
	cleanSlice := []string{}
	for _, item := range slice {
		if item == "" {
			continue
		}
		if !Contains(cleanSlice, item) {
			cleanSlice = append(cleanSlice, item)
		}
	}
	return cleanSlice
}
