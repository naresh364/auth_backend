package models

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/auth_backend/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"reflect"
	"strings"
)

type QueryBuilder struct {
	fields map[string]FieldInfo
	readQuery	string
	name string
	create Creator
}

type Creator func() BaseModel

func (bq *QueryBuilder) GetName() string {
	return bq.name
}

func (bq *QueryBuilder) GetInstance() BaseModel {
	return bq.create()
}

func (bq *QueryBuilder) GetFieldInfo() (map[string]FieldInfo) {
	return bq.fields
}

func (bq *QueryBuilder) GetReadQuery() string {
	return bq.readQuery
}

func (bq *QueryBuilder) InitFieldInfo(model BaseModel, creator Creator) *QueryBuilder {
	bq.fields,bq.name,bq.readQuery = initFieldInfo(model)
	bq.create = creator
	return bq
}

func (bq *QueryBuilder) ConvertFromMap(v TableRow) (BaseModel, error)  {
	bm := bq.GetInstance()
	if err := mapstructure.Decode(v, bm); err != nil {
		return nil, err
	}
	return bm, nil
}

func (bq *QueryBuilder) ConvertToJsonNames(m *[]TableRow) *[]TableRow {
	n := make([]TableRow, len(*m))
	fis := bq.GetFieldInfo()
	for i, ent := range *m {
		n[i] = make(TableRow)
		for k,v := range ent {
			if fi, ok := fis[k];ok {
				if fi.Json != "_" && !fi.NotReadable {
					n[i][fi.Json] = v
				}
			} else {
				logrus.Errorf("Invalid FN %s found while converting for table %s", k, bq.name)
			}
		}
	}
	return &n
}

// Uses reflection
func (bq *QueryBuilder) ConvertObj(m *[]TableRow) ([]BaseModel, error){
	var ret = make([]BaseModel, len(*m))
	for i, v := range *m {
		bm := bq.GetInstance()
		if err := mapstructure.Decode(v, bm); err != nil {
			return nil, err
		}
		ret[i] =  bm
	}
	return ret, nil
}

func (bq *QueryBuilder) Convert(rows *sql.Rows) (*[]TableRow, error){
	fis := bq.GetFieldInfo()
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))
	var ret = make([]TableRow, 0)

	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Make a slice for the values
	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		var c_row = make(TableRow)

		// Now do something with the data.
		// Here we just print each column as a string.
		for i, col := range values {
			cn := columns[i]
			if fi,ok := fis[columns[i]]; !ok {
				logrus.Errorf("Column %s not found for table %s", cn, bq.name)
			} else {
				if fv, err := fi.convertFromString(col); err != nil {
					logrus.Errorf("Trying to convert to wrong type for %s(%s)", bq.name, cn)
					return  nil, err
				} else {
					c_row[fi.FN] = fv
				}
			}
		}
		ret = append(ret, c_row)
	}
	return &ret, nil
}

//field validations based on operation types
type FieldInfo struct {
	FN          string
	DBN         string
	RO          bool
	UQ 			bool
	NotReadable bool
	IsPassword  bool
	IsRef		bool
	Json        string
	Type 		reflect.Type
}

func (fi FieldInfo) convertFromString(in []byte) (interface{}, error) {
	return utils.ConvertFromString(in, fi.Type, DB_TIME_FORMAT)
}

func (fi FieldInfo) isString() bool {
	kind := fi.Type.Kind()
	return kind == reflect.String
}

func (fi FieldInfo) isInt() bool {
	kind := fi.Type.Kind()
	if kind == reflect.Int ||
		kind == reflect.Int64 ||
		kind == reflect.Int32 ||
		kind == reflect.Int16 ||
		kind == reflect.Int8 ||
		kind == reflect.Uint||
		kind == reflect.Uint64 ||
		kind == reflect.Uint32 ||
		kind == reflect.Uint16 ||
		kind == reflect.Uint8 {
		return true
	}
	return false
}

func (fi FieldInfo) isFloat() bool {
	kind := fi.Type.Kind()
	if kind == reflect.Float64 ||
		kind == reflect.Float32 {
		return true
	}
	return false
}

/**
returns map[FieldName]FieldInfo
 */
func initFieldInfo(table BaseModel) (map[string]FieldInfo, string, string) {
	val := reflect.ValueOf(table).Elem()
	wv := make(map[string]FieldInfo)
	name := ""
	if t := reflect.TypeOf(table); t.Kind() == reflect.Ptr {
		name = t.Elem().Name()
	} else {
		name = t.Name()
	}

	readQuery := ""
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		tag := typeField.Tag

		fn := typeField.Name
		var fi FieldInfo
		if val, has := wv[fn]; has {
			fi = val
		} else {
			fi = FieldInfo{FN : fn}
		}

		fi.DBN = utils.ToSnakeCase(fn)
		fi.Type = typeField.Type
		fi.Json = tag.Get("json");

		v := tag.Get("v");
		if v != "" {
			values := strings.Split(v, ",")
			for _,f := range values {
				switch f {
				case "ro" : fi.RO = true
				case "uq" : fi.UQ = true
				case "noread" : fi.NotReadable = true
				case "password" : fi.IsPassword = true
				case "ref" : {
					fi.RO = true
					fi.IsRef = true
				}
				}
			}
		}
		if !fi.IsRef {
			readQuery += fi.DBN+","
		}
		wv[fn] = fi
		if fi.Json != "" && fi.Json != "_" {
			//insert the json key as well
			wv[fi.Json] = fi
		}
		wv[fi.DBN] = fi
	}
	readQuery = strings.TrimSuffix(readQuery, ",")
	return wv, utils.ToSnakeCase(name), readQuery
}

//Create string in k=?, k2=?, k3=? format
//BaseModel is the model on which we want the operation
//fvdetails are field info cached earlier
//returns query string with list of parameters to be passed
func buildCreateQuery(model BaseModel, fvdetails map[string]FieldInfo) (string, []interface{}){
	val := reflect.ValueOf(model).Elem();
	var kvm = make(map[string]interface{})
	for i:=0; i<val.NumField(); i++ {
		typeField := val.Type().Field(i)
		kvm[typeField.Name] = val.Field(i).Interface()
	}
	return buildUpdateQuery(kvm, fvdetails)
}

//Create string in k=?, k2=?, k3=? format
//kvp key value map to be updated
//fvdetails are field info cached earlier
//returns query string with list of parameters to be passed
func buildUpdateQuery(kvp map[string]interface{}, fvdetails map[string]FieldInfo) (string, []interface{}){
	var query string
	var vals []interface{}
	for k,v := range kvp {
		//only fields with RO != true can be inserted
		if finfo,ok := fvdetails[k]; ok && !finfo.IsRef && !finfo.RO {
			query += finfo.DBN + "=?, "
			vals = append(vals, v)
		}
	}

	return strings.TrimSuffix(query, ", "), vals
}

func validateFields(kvp *map[string]interface{}, fis *map[string]FieldInfo, checkRO bool) error {
	nf_fields := ""
	readonly := ""
	for k,_ := range *kvp {
		notfound := true
		for _, fi := range *fis {
			//TODO : should return multiple errors in one go
			if fi.Json != "_" && fi.Json == k {
				if checkRO && fi.RO {
					readonly += k+", "
				}
				notfound = false
				break
			}
		}
		if (notfound) {
			nf_fields += fmt.Sprintf("%s, ",k)
		}
	}

	if nf_fields != "" {
		nf_fields = strings.TrimSuffix(nf_fields, ", ")
		logrus.Debugf("Invalid fields passed will be ignored : "+ nf_fields)
	}
	if readonly != "" {
		readonly = strings.TrimSuffix(readonly, ", ")
		return errors.New("Cannot modify read only field " + readonly)
	}

	return nil
}


