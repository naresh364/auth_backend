package utils

import (
	"errors"
	"github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"time"
)

func ConvertFromString(in []byte, p reflect.Type, timef string) (interface{}, error) {
	val := string(in)
	switch p.Kind() {
	case reflect.Int :
		if cv,err := strconv.Atoi(val); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Int8 :
		if cv,err := strconv.ParseInt(val, 10,8); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Int16 :
		if cv,err := strconv.ParseInt(val, 10,16); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Int32 :
		if cv,err := strconv.ParseInt(val, 10,32); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Int64 :
		if cv,err := strconv.ParseInt(val, 10,64); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Uint :
		if cv,err := strconv.ParseUint(val, 10, 32); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Uint8 :
		if cv,err := strconv.ParseUint(val, 10,8); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Uint16 :
		if cv,err := strconv.ParseUint(val, 10,16); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Uint32 :
		if cv,err := strconv.ParseUint(val, 10,32); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Uint64 :
		if cv,err := strconv.ParseUint(val, 10,64); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Bool:
		if cv,err := strconv.ParseBool(val); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Float32 :
		if cv,err := strconv.ParseFloat(val, 32); err!=nil { return nil, err} else { return cv, nil}
	case reflect.Float64:
		if cv,err := strconv.ParseFloat(val, 64); err!=nil { return nil, err} else { return cv, nil}
	case reflect.String:
		return string(val),nil
	default:
		//using timestamp
		if p.Name() == "Time" {
			if t, err := time.Parse(timef, val); err != nil {
				logrus.Errorf("Convert to time failed for val %s", val)
				return nil, err
			} else {
				return t, nil
			}

		}
		logrus.Errorf("Invalid type not implemented : "+p.Name())
		return nil, errors.New("Invalid type, not implemented : "+p.Name())
	}
}

