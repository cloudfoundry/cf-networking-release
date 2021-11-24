package viron

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
)

func Parse(env interface{}) error {
	v := reflect.ValueOf(env)
	if v.Kind() != reflect.Ptr {
		return NewInvalidArgumentError(env)
	}

	if v.IsNil() {
		return NewInvalidArgumentError(env)
	}

	t := reflect.TypeOf(env).Elem()
	v = v.Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.FieldByName(field.Name)
		err := loadField(field, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func loadField(field reflect.StructField, value reflect.Value) error {
	name := field.Tag.Get("env")
	actual := os.Getenv(name)
	required, _ := strconv.ParseBool(field.Tag.Get("env-required"))
	if actual == "" {
		actual = field.Tag.Get("env-default")
	}

	if required && actual == "" {
		return NewRequiredFieldError(name)
	}

	var err error
	if value.CanSet() && actual != "" {
		switch value.Kind() {
		case reflect.Bool:
			err = setBool(name, value, actual)
		case reflect.String:
			err = setString(name, value, actual)
		case reflect.Int:
			err = setInt(name, value, actual, 0)
		case reflect.Int8:
			err = setInt(name, value, actual, 8)
		case reflect.Int16:
			err = setInt(name, value, actual, 16)
		case reflect.Int32:
			err = setInt(name, value, actual, 32)
		case reflect.Int64:
			err = setInt(name, value, actual, 64)
		case reflect.Uint, reflect.Uintptr:
			err = setUint(name, value, actual, 0)
		case reflect.Uint8:
			err = setUint(name, value, actual, 8)
		case reflect.Uint16:
			err = setUint(name, value, actual, 16)
		case reflect.Uint32:
			err = setUint(name, value, actual, 32)
		case reflect.Uint64:
			err = setUint(name, value, actual, 64)
		case reflect.Float32:
			err = setFloat(name, value, actual, 32)
		case reflect.Float64:
			err = setFloat(name, value, actual, 64)
		case reflect.Struct:
			err = setStruct(name, value, actual)
		case reflect.Slice:
			err = setSlice(name, value, actual)
		}
	}

	return err
}

func setBool(name string, value reflect.Value, actual string) error {
	x, err := strconv.ParseBool(actual)
	if err != nil {
		return NewParseError(name, actual, value.Kind().String())
	}
	value.SetBool(x)

	return nil
}

func setString(name string, value reflect.Value, actual string) error {
	value.SetString(actual)

	return nil
}

func setInt(name string, value reflect.Value, actual string, bitSize int) error {
	x, err := strconv.ParseInt(actual, 10, bitSize)
	if err != nil {
		return NewParseError(name, actual, value.Kind().String())
	}
	value.SetInt(x)

	return nil
}

func setUint(name string, value reflect.Value, actual string, bitSize int) error {
	x, err := strconv.ParseUint(actual, 10, bitSize)
	if err != nil {
		return NewParseError(name, actual, value.Kind().String())
	}
	value.SetUint(x)

	return nil
}

func setFloat(name string, value reflect.Value, actual string, bitSize int) error {
	x, err := strconv.ParseFloat(actual, bitSize)
	if err != nil {
		return NewParseError(name, actual, value.Kind().String())
	}
	value.SetFloat(x)

	return nil
}

func setStruct(name string, value reflect.Value, actual string) error {
	err := json.Unmarshal([]byte(actual), value.Addr().Interface())
	if err != nil {
		return NewParseError(name, actual, value.Kind().String())
	}

	return nil
}

func setSlice(name string, value reflect.Value, actual string) error {
	var err error

	switch value.Type().Elem().Kind() {
	case reflect.Uint8:
		err = setByteSlice(name, value, actual)
	}

	return err
}

func setByteSlice(name string, value reflect.Value, actual string) error {
	value.SetBytes([]byte(actual))

	return nil
}
