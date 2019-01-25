package models

import (
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"nkk_backend/server_errors"
	"reflect"
	"strconv"
	"strings"
)

const (
	PERMISSION_R = "r"
	PERMISSION_C = "c"
	PERMISSION_U = "u"
	PERMISSION_D = "d"
)

type BasePermissionModel interface {
	getTableName() string
	getColumnName() string
	getPermission() string
	getValue() string
	GetId() int64
}

type Condition map[string][]string

type TablePermission struct {
	Read 	*Condition
	Create 	*Condition
	Update 	*Condition
	Delete 	*Condition
}

type Permissions struct {
	Ps map[string]*TablePermission
}

func (p *Permissions) hasAccessRD(table string, pt string) (bool, string, *[]interface{}){
	if tp, ok := p.Ps[table]; ok {
		var tocheck *Condition
		switch pt {
		case PERMISSION_R : tocheck = tp.Read
		case PERMISSION_D : tocheck = tp.Delete
		default:
			return false, "", nil
		}
		if tocheck != nil {
			query := ""
			var params []interface{}

			for k, perms := range *tocheck {
				query += " "+k+" in (? "+strings.Repeat(",?", len(perms)-1)+") and "
				for _, per := range perms {
					params = append(params, per)
				}
			}
			return true, query, &params
		}
	}
	//does not have access
	return false, "", nil
}

func hasCUPermissionForCondition(col string, val string, conds *Condition) bool {
	wasfound := false
	for k, vals := range *conds {
		if k == col {
			wasfound = true
			fval, err := strconv.ParseFloat(val, 64)
			for _, v := range vals {
				if err == nil {
					cval, err := strconv.ParseFloat(v, 64)
					if err != nil {continue;} //assume do not match
					if v != "" && cval == fval { return true} //matched as numbers
				}

				//continue with string compare
				if v != "" && v == val {
					//new values should be one of the values that user have access to
					return true
				}
			}
		}
	}

	//in case this col does not exists in conditions, we don't have any conditional access for this column
	return !wasfound
}

func (p *Permissions) hasAccessUC(table string, kvp map[string]interface{}, fvdetails map[string]FieldInfo, pt string) (error){
	if kvp == nil || len(kvp) == 0 {
		return errors.New(fmt.Sprintf("No update values provided for table %s", table))
	}

	if tp, ok := p.Ps[table]; ok {
		var tocheck *Condition
		switch pt {
		case PERMISSION_C : tocheck = tp.Create
		case PERMISSION_U : tocheck = tp.Update
		default:
			return errors.New("Invalid permission type")
		}

		if tocheck != nil {
			for k,v := range kvp {
				if finfo,ok := fvdetails[k]; ok {
					if finfo.RO { errors.New("trying to update readonly field")}
					col := finfo.DBN
					val := fmt.Sprintf("%v", v)
					if !hasCUPermissionForCondition(col, val, tocheck) {
						log.Debugf("Cannot update %s to %s", col, val)
						return errors.New(fmt.Sprintf("Cannot update %s to %s", col, val))
					}
				}
			}
			//if we have passed above loop means no cond failed
			return nil
		}
	}
	//does not have access
	return errors.New("Do not have write permission")

}

func (ps *Permissions) hasCreateAccessForValue(table string, model BaseModel, fvdetails map[string]FieldInfo) error {
	val := reflect.ValueOf(model).Elem();
	var kvm = make(map[string]interface{})
	for i:=0; i<val.NumField(); i++ {
		typeField := val.Type().Field(i)
		kvm[typeField.Name] = val.Field(i).Interface()
	}
	return ps.hasAccessUC(table, kvm, fvdetails, PERMISSION_C)
}

//Check if for given permission it has read access to the table
//returns true if has access along with query to be append for which access is given
//for example if someone has a read access only if col1 in (x1,x2,x3) and col2 in (x4)
// it will return the query along with params
func (p *Permissions) HasReadAccess(table string) (bool, string, *[]interface{})  {
	return p.hasAccessRD(table, PERMISSION_R)
}

//Check if for given permission it has delete access to the table
//returns true if has access along with query to be append for which access is given
//for example if someone has a read access only if col1 in (x1,x2,x3) and col2 in (x4)
// it will return the query along with params
func (p *Permissions) HasDeleteAccess(table string) (bool, string, *[]interface{})  {
	return p.hasAccessRD(table, PERMISSION_D)
}

//Check if for given permission it has update access to the table
//returns true if has access along with query to be append for which access is given
//for example if someone has a read access only if col1 in (x1,x2,x3) and col2 in (x4)
// it will return the query along with params
func (p *Permissions) HasUpdateAccess(table string, kvp map[string]interface{},
						fvdetails map[string]FieldInfo) (error)  {
	return p.hasAccessUC(table, kvp, fvdetails, PERMISSION_U)
}

//Check if for given permission it has create access to the table
//returns true if has access along with query to be append for which access is given
// it will return the query along with params
func (p *Permissions) HasCreateAccess(table string, model BaseModel,
						fvdetails map[string]FieldInfo) (error)  {
	return p.hasCreateAccessForValue(table, model, fvdetails);
}


//Adding a new Permissions
func (p *Permissions) addPermission(table string, pt string, col string, val string) error {
	if p.Ps[table] == nil {
		p.Ps[table] = &TablePermission{}
	}
	tp := p.Ps[table]
	var perm *Condition
	wasnil := false
	switch pt {
	case PERMISSION_C:
		if tp.Create == nil { tp.Create = &Condition{}; wasnil = true}
		perm = tp.Create
	case PERMISSION_U:
		if tp.Update == nil { tp.Update = &Condition{}; wasnil = true}
		perm = tp.Update
	case PERMISSION_R:
		if tp.Read == nil { tp.Read = &Condition{};  wasnil = true}
		perm =  tp.Read
	case PERMISSION_D:
		if tp.Delete == nil { tp.Delete = &Condition{}; wasnil = true}
		perm = tp.Delete
	default:
		return errors.New("Invalid permission type passed :" + pt)
	}

	if col == "" {
		//we have unconditional access to this table
		if wasnil {
			return nil
		} else {
			//we have got unconditional access, remove others and just add it
			log.Debugf("we have got unconditional access for table %s, removing all column conditions : last conditions : %s", table, *perm)
			(*perm) = Condition{}
		}
	} else {
		if val == "" {
			log.Debugf("empty val passed for col %s, table %s", col, table)
			return errors.New(fmt.Sprintf("empty val passed for col %s, table %s", col, table))
		}
	}

	if col_vals, ok := (*perm)[col]; !ok {
		(*perm)[col] = []string{val}
	} else {
		for _, cv := range col_vals {
			if cv == val {
				//duplicate permission, avoid adding
				return server_errors.DUPLICATE_ENTRY
			}
		}
		col_vals = append(col_vals, val)
		(*perm)[col] = col_vals
	}
	return nil
}

func checkPermissionValue(pv string) bool {
	switch pv {
	case PERMISSION_R:
	case PERMISSION_C:
	case PERMISSION_D:
	case PERMISSION_U:
	default:
		return false
	}
	return true
}

func isValidPermission(rp BasePermissionModel, rm *DBRequestHandler) error {
	pv := rp.getPermission()
	if !checkPermissionValue(pv) {
		msg := fmt.Sprintf("permission (%d) is not valid %s. Will be ignored", rp.GetId(), rp.getPermission())
		log.Debug(msg)
		return errors.New(msg)
	}

	if mv, ok := rm.queryBuilders[rp.getTableName()]; ok {
		fis := mv.GetFieldInfo()

		for _, fi := range fis {
			if rp.getColumnName() == fi.DBN {
				if _, err := validateValue(fi, rp); err != nil {
					return err
				} else {
					return nil
				}
			}
		}
	}
	return nil
}

func validateValue(fi FieldInfo, rp BasePermissionModel) (interface{}, error) {
	if rp.getValue() == "" {
		//assuming that nil is a valid value
		return nil, nil
	}
	val := string(rp.getValue())
	if fi.isInt() {
		if v, err := strconv.Atoi(val); err != nil {
			return v, errors.New(fmt.Sprintf("permission (%d) does not have valid value %s. Will be ignored", rp.GetId(), rp.getValue()))
		}
	} else if fi.isFloat() {
		if v, err := strconv.ParseFloat(val, 64); err != nil {
			return v, errors.New(fmt.Sprintf("permission (%d) does not have valid value %s. Will be ignored", rp.GetId(), rp.getValue()))
		}
	}
	return rp.getValue(), nil
}

func cacheUserPermissions(au *AuthUser, allp []BasePermissionModel, rm *DBRequestHandler) *Permissions {
	len := len(allp)
	if len == 0 {
		log.Debugf("User %s does not have any permission", au.Username)
		return nil
	}
	ps := &Permissions{make(map[string]*TablePermission)}
	for _,rp := range allp {
		if _, ok := rm.queryBuilders[rp.getTableName()]; ok {
			if err := isValidPermission(rp, rm); err != nil {
				log.Error(err.Error())
				continue
			}
			ps.addPermission(rp.getTableName(), rp.getPermission(), rp.getColumnName(), rp.getValue())
		} else {
			log.Errorf("User %s (%d) has an invalid table name %s. Will be ignored", au.Username, rp.GetId(), rp.getTableName())
		}
	}
	return ps
}

