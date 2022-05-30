package seslog2

import (
	"strings"
	"time"
)
import "github.com/cstockton/go-conv"

type VarType int

const (
	VarString VarType = iota
	VarDateTime
	VarInt
	VarFloat
	VarBool
)

var varTypeToClickHouse = map[VarType]string{
	VarString:   "String",
	VarDateTime: "DateTime",
	VarInt:      "Int32",
	VarFloat:    "Float64",
	VarBool:     "Bool",
}

//var strToVarType = map[string]VarType{
//	"String":      VarString,
//	"VarDateTime": VarDateTime,
//	"VarInt":      VarInt,
//	"VarFloat":    VarFloat,
//	"VarBool":     VarBool,
//}

var nameToVarType = map[string]VarType{
	"status": VarInt,
	"time":   VarDateTime,
}

func fieldNameToVarType(f string) (VarType, string) {
	if varType, ok := nameToVarType[f]; ok {
		return varType, f
	}
	switch true {
	case !strings.Contains(f, "_"):
		return VarString, f
	case strings.HasSuffix(f, "_str"):
		return VarString, strings.TrimSuffix(f, "_str")
	case strings.HasSuffix(f, "_dtime"):
		return VarDateTime, strings.TrimSuffix(f, "_dtime")
	case strings.HasSuffix(f, "_time"):
		return VarFloat, f
	case strings.HasSuffix(f, "_int"):
		return VarInt, strings.TrimSuffix(f, "_int")
	case strings.HasSuffix(f, "_float"):
		return VarFloat, strings.TrimSuffix(f, "_float")
	case strings.HasSuffix(f, "_bool"):
		return VarBool, strings.TrimSuffix(f, "_bool")
	}
	return VarString, f
}

var timeLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	time.RFC3339,
}

func (varType VarType) convert(s string) interface{} {
	switch varType {
	default:
		return s
	case VarDateTime: //todo change it with more time layouts
		for _, timeLayout := range timeLayouts {
			t, err := time.Parse(timeLayout, s)
			if err == nil {
				return t
			}
		}
		return time.Unix(0, 0)
	case VarFloat:
		v, _ := conv.Float64(s)
		return v
	case VarInt:
		v, _ := conv.Int32(s)
		return v
	case VarBool:
		v, _ := conv.Bool(s)
		return v
	}
}
