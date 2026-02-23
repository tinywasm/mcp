package cast

import (
	"fmt"
	"strconv"
)

func ToBool(i any) bool {
	v, _ := ToBoolE(i)
	return v
}

func ToBoolE(i any) (bool, error) {
	switch b := i.(type) {
	case bool:
		return b, nil
	case nil:
		return false, nil
	case int:
		return b != 0, nil
	case string:
		return strconv.ParseBool(b)
	}
	return false, fmt.Errorf("unable to cast %#v of type %T to bool", i, i)
}

func ToFloat64(i any) float64 {
	v, _ := ToFloat64E(i)
	return v
}

func ToFloat64E(i any) (float64, error) {
	switch f := i.(type) {
	case float64:
		return f, nil
	case float32:
		return float64(f), nil
	case int:
		return float64(f), nil
	case int64:
		return float64(f), nil
	case int32:
		return float64(f), nil
	case int16:
		return float64(f), nil
	case int8:
		return float64(f), nil
	case uint:
		return float64(f), nil
	case uint64:
		return float64(f), nil
	case uint32:
		return float64(f), nil
	case uint16:
		return float64(f), nil
	case uint8:
		return float64(f), nil
	case string:
		return strconv.ParseFloat(f, 64)
	}
	return 0, fmt.Errorf("unable to cast %#v of type %T to float64", i, i)
}

func ToFloat32(i any) float32 {
	v, _ := ToFloat64E(i)
	return float32(v)
}

func ToInt64(i any) int64 {
	v, _ := ToInt64E(i)
	return v
}

func ToInt64E(i any) (int64, error) {
	switch v := i.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case int32:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case uint:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case string:
		// Try to parse as float first if it contains a dot, because cast does that
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return int64(f), nil
		}
		return strconv.ParseInt(v, 0, 64)
	case nil:
		return 0, nil
	}
	return 0, fmt.Errorf("unable to cast %#v of type %T to int64", i, i)
}

func ToInt(i any) int {
	v, _ := ToInt64E(i)
	return int(v)
}

func ToInt32(i any) int32 {
	v, _ := ToInt64E(i)
	return int32(v)
}

func ToInt16(i any) int16 {
	v, _ := ToInt64E(i)
	return int16(v)
}

func ToInt8(i any) int8 {
	v, _ := ToInt64E(i)
	return int8(v)
}

func ToUint64(i any) uint64 {
	v, _ := ToUint64E(i)
	return v
}

func ToUint64E(i any) (uint64, error) {
	switch v := i.(type) {
	case uint:
		return uint64(v), nil
	case uint64:
		return v, nil
	case uint32:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint8:
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("unable to cast negative value")
		}
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("unable to cast negative value")
		}
		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("unable to cast negative value")
		}
		return uint64(v), nil
	case int16:
		if v < 0 {
			return 0, fmt.Errorf("unable to cast negative value")
		}
		return uint64(v), nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("unable to cast negative value")
		}
		return uint64(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("unable to cast negative value")
		}
		return uint64(v), nil
	case float32:
		if v < 0 {
			return 0, fmt.Errorf("unable to cast negative value")
		}
		return uint64(v), nil
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			if f < 0 {
				return 0, fmt.Errorf("unable to cast negative value")
			}
			return uint64(f), nil
		}
		return strconv.ParseUint(v, 0, 64)
	case nil:
		return 0, nil
	}
	return 0, fmt.Errorf("unable to cast %#v of type %T to uint64", i, i)
}

func ToUint(i any) uint {
	v, _ := ToUint64E(i)
	return uint(v)
}

func ToUint32(i any) uint32 {
	v, _ := ToUint64E(i)
	return uint32(v)
}

func ToUint16(i any) uint16 {
	v, _ := ToUint64E(i)
	return uint16(v)
}

func ToUint8(i any) uint8 {
	v, _ := ToUint64E(i)
	return uint8(v)
}

func ToString(i any) string {
	switch s := i.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	case fmt.Stringer:
		return s.String()
	case nil:
		return ""
	case error:
		return s.Error()
	}
	return fmt.Sprintf("%v", i)
}

func ToStringSlice(i any) []string {
	switch v := i.(type) {
	case []string:
		return v
	case []any:
		var res []string
		for _, e := range v {
			res = append(res, ToString(e))
		}
		return res
	}
	return []string{ToString(i)}
}

func ToStringMap(i any) map[string]any {
	switch v := i.(type) {
	case map[string]any:
		return v
	case map[any]any:
		m := make(map[string]any)
		for k, val := range v {
			m[ToString(k)] = val
		}
		return m
	}
	return make(map[string]any)
}
