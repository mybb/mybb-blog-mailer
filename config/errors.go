package config

import "fmt"

/// OutOfRangeError is an error returned if a configuration parameter is outside of an allowable range.
type OutOfRangeError struct {
	/// ParameterName is the name of the invalid configuration parameter.
	ParameterName string
}

func (e OutOfRangeError) Error() string {
	return fmt.Sprintf("configuration parameter '%s' has an invalid value", e.ParameterName)
}

/// RequiredConfigMissingError is an error returned if a required configuration parameter is not provided.
type RequiredConfigMissingError struct {
	/// ParameterName is the name of the missing required configuration parameter.
	ParameterName string
}

func (e RequiredConfigMissingError) Error() string {
	return fmt.Sprintf("required configuration parameter '%s' is missing", e.ParameterName)
}
