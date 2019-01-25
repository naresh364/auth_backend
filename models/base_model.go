package models

import (
	"database/sql"
	"github.com/auth_backend/utils"
	"github.com/pkg/errors"
	"reflect"
	"strings"
	"time"
)

const DB_TIME_FORMAT = time.RFC3339
type BaseModel interface {
	SetId(int64)
	GetId() int64
	Register() *QueryBuilder
}

type BaseOrgModel interface {
	SetOrgId(int64)
	GetOrgId() int64
}

type BaseOwnerModel interface {
	GetOwner() int64
	SetOwner(int64)
}

type Operation struct {
	Name  string		`json:"name" validate:"required"`
	Value interface{} 	`json:"value" validate:"required"`
	Op    string 		`json:"op" validate:"oneof= = < > like in"`
	NextOp string		`json:"next_op" validate:"oneof=or and noop"`
}

func (op *Operation) createOpString(name string, fis map[string]FieldInfo, params *[]interface{}) (string, error) {
	kind := reflect.TypeOf(op.Value).Kind()
	var fn string
	if fi, ok := fis[op.Name]; !ok {
		return "", errors.New("Invalid field name "+op.Name)
	} else {
		fn = name+"."+utils.ToSnakeCase(fi.FN)
	}

	if kind == reflect.Array || kind == reflect.Slice {
		var count int
		for _, v := range op.Value.([]interface{}) {
			*params = append(*params, v)
			count++
		}
		return " "+fn +" in (? "+strings.Repeat(",?", count-1)+") ", nil
	} else if kind == reflect.String ||
		kind == reflect.Float64 ||
		kind == reflect.Float32 ||
		kind == reflect.Int ||
		kind == reflect.Int64 ||
		kind == reflect.Int32 ||
		kind == reflect.Int16 ||
		kind == reflect.Int8 ||
		kind == reflect.Uint||
		kind == reflect.Uint64 ||
		kind == reflect.Uint32 ||
		kind == reflect.Uint16 ||
		kind == reflect.Uint8 ||
		kind == reflect.Bool {
		*params = append(*params, op.Value)
		return " "+fn+" "+op.Op+" ? ", nil
	} else {
		return "", errors.New("Invalid value type passed : "+kind.String())
	}
}

type ReadMasker interface {
	maskRead() (map[string]interface{}, error)
}

type WriteMasker interface {
	maskWrite() (map[string]interface{}, error)
}


//query builders
//Implement this to create a custom create query
type QueryCreator interface {
	Create(builder *QueryBuilder) (string, []interface{}, error)
}

//Implement this to create a custom create query
type QueryReader interface {
	Read(qb *QueryBuilder, db *sql.DB, from int, limit int, desc bool, sortby string, cond string, p []interface{}) (*[]TableRow, error)
}

type QueryDeleter interface {
	Delete(qb *QueryBuilder, d int) error
}

type QueryUpdater interface {
	Update(qb *QueryBuilder, int, u map[string]interface{}) (error)
}

