package util

/*
cleans a map from emty strng or string slices with length 0
*/
func ClearMap(in map[string]interface{}) map[string]interface{} {
	for key, elem := range in {
		if str, ok := elem.(string); ok {
			if str == "" {
				delete(in, key)
			}
		}
		if strSlc, ok := elem.([]string); ok {
			if len(strSlc) == 0 {
				delete(in, key)
			}
		}
		if anySlc, ok := elem.([]interface{}); ok {
			if len(anySlc) == 0 {
				delete(in, key)
			}
		}
	}
	return in
}
