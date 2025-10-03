package queryparser

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// ParseQueryParams parses URL query parameters into a struct using reflection
// The struct fields should have `query:"param_name"` tags to specify the parameter names
func ParseQueryParams(values url.Values, target interface{}) error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer to struct")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Get the query tag
		queryTag := fieldType.Tag.Get("query")
		if queryTag == "" {
			continue
		}

		// Get the value from query parameters
		queryValue := values.Get(queryTag)
		queryValues := values[queryTag]

		if err := setFieldValue(field, fieldType.Type, queryValue, queryValues); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// setFieldValue sets the value of a struct field based on its type
func setFieldValue(field reflect.Value, fieldType reflect.Type, singleValue string, multipleValues []string) error {
	switch fieldType.Kind() {
	case reflect.String:
		if singleValue != "" {
			field.SetString(singleValue)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if singleValue != "" {
			val, err := strconv.ParseInt(singleValue, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer value: %s", singleValue)
			}
			field.SetInt(val)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if singleValue != "" {
			val, err := strconv.ParseUint(singleValue, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid unsigned integer value: %s", singleValue)
			}
			field.SetUint(val)
		}

	case reflect.Float32, reflect.Float64:
		if singleValue != "" {
			val, err := strconv.ParseFloat(singleValue, 64)
			if err != nil {
				return fmt.Errorf("invalid float value: %s", singleValue)
			}
			field.SetFloat(val)
		}

	case reflect.Bool:
		if singleValue != "" {
			val, err := strconv.ParseBool(singleValue)
			if err != nil {
				return fmt.Errorf("invalid boolean value: %s", singleValue)
			}
			field.SetBool(val)
		}

	case reflect.Slice:
		if len(multipleValues) > 0 {
			if err := setSliceValue(field, fieldType.Elem(), multipleValues); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unsupported field type: %s", fieldType.Kind())
	}

	return nil
}

// setSliceValue sets slice values based on the element type
func setSliceValue(field reflect.Value, elemType reflect.Type, values []string) error {
	slice := reflect.MakeSlice(field.Type(), 0, len(values))

	for _, value := range values {
		// Handle comma-separated values in a single parameter
		if strings.Contains(value, ",") {
			subValues := strings.Split(value, ",")
			for _, subValue := range subValues {
				subValue = strings.TrimSpace(subValue)
				if subValue != "" {
					elem, err := parseSliceElement(elemType, subValue)
					if err != nil {
						return err
					}
					slice = reflect.Append(slice, elem)
				}
			}
		} else {
			value = strings.TrimSpace(value)
			if value != "" {
				elem, err := parseSliceElement(elemType, value)
				if err != nil {
					return err
				}
				slice = reflect.Append(slice, elem)
			}
		}
	}

	field.Set(slice)
	return nil
}

// parseSliceElement parses a single element for a slice
func parseSliceElement(elemType reflect.Type, value string) (reflect.Value, error) {
	switch elemType.Kind() {
	case reflect.String:
		return reflect.ValueOf(value), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid integer value in slice: %s", value)
		}
		return reflect.ValueOf(val).Convert(elemType), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid unsigned integer value in slice: %s", value)
		}
		return reflect.ValueOf(val).Convert(elemType), nil

	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid float value in slice: %s", value)
		}
		return reflect.ValueOf(val).Convert(elemType), nil

	case reflect.Bool:
		val, err := strconv.ParseBool(value)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid boolean value in slice: %s", value)
		}
		return reflect.ValueOf(val), nil

	default:
		return reflect.Value{}, fmt.Errorf("unsupported slice element type: %s", elemType.Kind())
	}
}

// ParseQueryParamsWithDefaults parses query parameters and applies default values
func ParseQueryParamsWithDefaults(values url.Values, target interface{}, defaults map[string]interface{}) error {
	if err := ParseQueryParams(values, target); err != nil {
		return err
	}

	return applyDefaults(target, defaults)
}

// applyDefaults applies default values to struct fields that are zero values
func applyDefaults(target interface{}, defaults map[string]interface{}) error {
	rv := reflect.ValueOf(target).Elem()
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		if !field.CanSet() {
			continue
		}

		queryTag := fieldType.Tag.Get("query")
		if queryTag == "" {
			continue
		}

		defaultValue, exists := defaults[queryTag]
		if !exists {
			continue
		}

		// Only apply default if field is zero value
		if field.IsZero() {
			defaultVal := reflect.ValueOf(defaultValue)
			if defaultVal.Type().ConvertibleTo(field.Type()) {
				field.Set(defaultVal.Convert(field.Type()))
			}
		}
	}

	return nil
}
