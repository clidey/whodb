package clickhouse

import (
	"fmt"
	"strconv"
)

func convertStringValue(value, columnType string) (interface{}, error) {
	switch columnType {
	case "Int8", "Int16", "Int32", "Int64", "UInt8", "UInt16", "UInt32", "UInt64":
		return strconv.ParseInt(value, 10, 64)
	case "Float32", "Float64":
		return strconv.ParseFloat(value, 64)
	case "Bool":
		return strconv.ParseBool(value)
	case "String", "FixedString":
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported column type: %s", columnType)
	}
}
