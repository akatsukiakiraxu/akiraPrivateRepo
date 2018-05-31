package core

func ToStringSlice(jsonArray []interface{}) []string {
	slice := make([]string, len(jsonArray))
	for i, value := range jsonArray {
		slice[i] = value.(string)
	}
	return slice
}
