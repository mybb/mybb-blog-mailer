package helpers

import (
	"os"
	"strconv"
)

/// GetIntEnv gets an integer value from the environment, returning a default value if the value doesn't exist or is not numeric.
func GetIntEnv(name string, defaultValue int) int {
	if strVal, ok := os.LookupEnv(name); ok && len(strVal) > 0 {
		if intVal, err := strconv.Atoi(strVal); err == nil {
			return intVal
		}
	}

	return defaultValue
}

/// GetEnv gets a string value from the environment, returning a default value if the value doesn't exist.
func GetEnv(name string, defaultValue string) string {
	if strVal, ok := os.LookupEnv(name); ok && len(strVal) > 0 {
		return strVal
	}

	return defaultValue
}