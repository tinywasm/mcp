package orderedmap

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

var (
	_ json.Marshaler   = &OrderedMap[int, any]{}
	_ json.Unmarshaler = &OrderedMap[int, any]{}
)

// MarshalJSON implements the json.Marshaler interface.
func (om *OrderedMap[K, V]) MarshalJSON() ([]byte, error) {
	if om == nil || om.list == nil {
		return []byte("null"), nil
	}

	var buf bytes.Buffer
	buf.WriteByte('{')

	first := true
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		if !first {
			buf.WriteByte(',')
		}
		first = false

		keyBytes, err := marshalKey(pair.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)

		buf.WriteByte(':')

		valBytes, err := json.Marshal(pair.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func marshalKey(key any) ([]byte, error) {
	switch k := key.(type) {
	case string:
		return json.Marshal(k)
	case encoding.TextMarshaler:
		text, err := k.MarshalText()
		if err != nil {
			return nil, err
		}
		return json.Marshal(string(text))
	case int:
		return json.Marshal(strconv.Itoa(k))
	case int8:
		return json.Marshal(strconv.Itoa(int(k)))
	case int16:
		return json.Marshal(strconv.Itoa(int(k)))
	case int32:
		return json.Marshal(strconv.Itoa(int(k)))
	case int64:
		return json.Marshal(strconv.FormatInt(k, 10))
	case uint:
		return json.Marshal(strconv.FormatUint(uint64(k), 10))
	case uint8:
		return json.Marshal(strconv.FormatUint(uint64(k), 10))
	case uint16:
		return json.Marshal(strconv.FormatUint(uint64(k), 10))
	case uint32:
		return json.Marshal(strconv.FormatUint(uint64(k), 10))
	case uint64:
		return json.Marshal(strconv.FormatUint(k, 10))
	default:
		v := reflect.ValueOf(key)
		switch v.Kind() {
		case reflect.String:
			return json.Marshal(v.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return json.Marshal(strconv.FormatInt(v.Int(), 10))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return json.Marshal(strconv.FormatUint(v.Uint(), 10))
		}
		return nil, fmt.Errorf("unsupported key type: %T", key)
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (om *OrderedMap[K, V]) UnmarshalJSON(data []byte) error {
	if om.list == nil {
		om.initialize(0)
	}
    // Clear existing data
    // Assuming internal list methods handle cleanup or re-init logic is handled by Set
    // We iterate and set, effectively rebuilding. If we want to support merge, we keep it.
    // Standard json unmarshal into map replaces content if it's nil, but merges if not nil?
    // Actually `json.Unmarshal` into map clears it? No, it merges.
    // So I should keep it merging.

	dec := json.NewDecoder(bytes.NewReader(data))

	// Check start token
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("expected {, got %v", t)
	}

	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		keyStr, ok := t.(string)
		if !ok {
			return fmt.Errorf("expected string key, got %T", t)
		}

		var key K
		if err := unmarshalKey(keyStr, &key); err != nil {
			return err
		}

		var value V
		if err := dec.Decode(&value); err != nil {
			return err
		}

		om.Set(key, value)
	}

    // Read closing }
    _, err = dec.Token()
    return err
}

func unmarshalKey[K any](keyStr string, out *K) error {
    // Handling generic K is tricky without reflect, but we can do type switches on *K
    // to handle common types efficiently, and fallback to reflect/json unmarshal.

    // Fast path for string
    if ptr, ok := any(out).(*string); ok {
        *ptr = keyStr
        return nil
    }

    // TextUnmarshaler
    if u, ok := any(out).(encoding.TextUnmarshaler); ok {
        return u.UnmarshalText([]byte(keyStr))
    }

	v := reflect.ValueOf(out).Elem()
	switch v.Kind() {
	case reflect.String:
		v.SetString(keyStr)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(keyStr, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(keyStr, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(i)
		return nil
	}

    // Last resort: try to unmarshal "keyStr" as JSON into K
    // This is valid if K is something that unmarshals from string (e.g. enum)
    return json.Unmarshal([]byte(`"` + keyStr + `"`), out)
}
