package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/auth_backend/utils"
	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v9"
	"regexp"
	"strings"
)

const (
	MAX_READ_LIMIT = 500
	MAX_UPDATE_LIMIT = 9
)

type DBRequestHandler struct {
	queryBuilders              map[string]*QueryBuilder
	db                         *sql.DB
	validate                   *validator.Validate
	auth_table                 string
	auth_role_permission_table string
	org_table                  string
	role_table                 string
	auth_permission_table      string
	su 						   *UserData
	orgcol					   string
	ownercol			   	   string
}

func (dbm * DBRequestHandler) IsAuthTable(table string) bool {
	return (table == dbm.auth_table)
}

func (dbm * DBRequestHandler) GetQueryBuilders(table string) *QueryBuilder {
	return dbm.queryBuilders[table]
}

type TableRow map[string]interface{}

//this will be used to control user access
type UserData struct {
	Id int64
	Uuid string
	Org_id int64
	P *Permissions
}

func (rm *DBRequestHandler) isSU(ud *UserData) bool {
	return ud.Id == rm.su.Id
}

func (rm *DBRequestHandler) RegisterDB(bq *QueryBuilder) error {
	if _, ok := rm.queryBuilders[bq.GetName()]; ok {
		return errors.New("A query builder is already registered with this name : "+bq.GetName())
	}
	rm.queryBuilders[bq.GetName()] = bq
	return nil
}

func InitDB(db *sql.DB, org string, owner string, sudo int, sudo_org int) *DBRequestHandler {
	au := (&AuthUser{}).Register()
	ur := (&UserRole{}).Register()
	up := (&UserPermission{}).Register()
	urp := (&UserRolePermission{}).Register()
	o := (&Org{}).Register()

	rm := DBRequestHandler{db : db,
		validate:                   validator.New(),
		queryBuilders:              make(map[string]*QueryBuilder),
		//authentication related tables
		auth_table:                 au.GetName(),
		auth_permission_table:      up.GetName(),
		auth_role_permission_table: urp.GetName(),
		org_table:                  o.GetName(),
		role_table :                ur.GetName(),
		orgcol:                     org,
		ownercol:					owner,
	}

	rm.su = &UserData{Id:int64(sudo), Org_id:int64(sudo_org)}//super user
	rm.queryBuilders[au.GetName()] = au
	rm.queryBuilders[up.GetName()] = up
	rm.queryBuilders[urp.GetName()] = urp
	rm.queryBuilders[o.GetName()] = o
	rm.queryBuilders[ur.GetName()] = ur

	log.Infof("Tables registered : %d", len(rm.queryBuilders))

	return &rm
}


func (rm *DBRequestHandler) Authenticate(username string, pass string) (BaseModel, *Permissions, error) {
	t_rm := rm.queryBuilders[rm.auth_table]
	if m, err := rm.ReadObjOps(rm.auth_table,
		[]Operation{{Name:"username", Value:username, Op:"=", NextOp:"noop"}},
		0,500,true,"", rm.su); err != nil {
			log.Error(err.Error())
			return nil, nil, err
	} else {
		switch len(*m) {
		default:
			log.Errorf("Multiple users found for %s",username)
			return nil, nil, INVALID_CREDENTIALS
		case 0 :
			return nil, nil, INVALID_CREDENTIALS
		case 1:
			var au *AuthUser
			if l, err := t_rm.ConvertObj(m); err != nil {
				return nil, nil, err
			} else {
				au = l[0].(*AuthUser)
			}
			if err = au.validatePassword(pass); err != nil {
				log.Error(err)
				return au, nil, INVALID_CREDENTIALS
			}
			var rolep, userp *[]TableRow
			if rolep, err = rm.ReadObjOps(rm.auth_role_permission_table,
				[]Operation{{Name:"user_role_id", Value:au.UserRoleId, Op:"=", NextOp:"noop"}},
				0,500000,true,"", rm.su); err != nil {
				log.Error(err.Error())
				return nil, nil, err
			}

			var conv_rolep []BaseModel
			if conv_rolep, err = rm.queryBuilders[rm.auth_role_permission_table].ConvertObj(rolep); err != nil {
				log.Errorf("User %s role permission could not be converted to model", au.Username)
				return nil, nil, errors.New("Unable to read permissions for user "+au.Username)
			}

			if userp, err = rm.ReadObjOps(rm.auth_permission_table,
				[]Operation{{Name:"auth_user_id", Value:au.ID, Op:"=", NextOp:"noop"}},
				0,500000,true,"", rm.su); err != nil {
				log.Error(err.Error())
				return nil, nil, err
			}

			var conv_userp []BaseModel
			if conv_userp, err = rm.queryBuilders[rm.auth_permission_table].ConvertObj(userp); err != nil {
				log.Errorf("User %s permission could not be converted to model", au.Username)
				return nil, nil, errors.New("Unable to read permissions for user "+au.Username)
			}

			lenr := len(conv_rolep)
			lenu := len(conv_userp)

			allp := make([]BasePermissionModel, lenr+lenu)
			for i, p := range conv_rolep{
				allp[i] = p.(BasePermissionModel)
			}
			for i, p := range conv_userp {
				allp[lenr+i] = p.(BasePermissionModel)
			}

			log.Debugf("User %s has %d role, %d role permissions, %d user permissions",
				au.Username, au.UserRoleId, lenr, lenu)

			ps := cacheUserPermissions(au, allp, rm)
			//if au.IsActive <= 0 {
			//	return au, server_errors.INACTIVE_USER
			//}
			return au, ps, nil
		}
	}
}

func (rm *DBRequestHandler) ReadObjJson(table string, data []byte, from int, limit int,
	desc bool, sortby string, ud *UserData) (*[]TableRow, error) {
	var ops []Operation

	if t_rm, ok := rm.queryBuilders[table]; !ok {
		return nil, errors.New("Invalid table")
	} else {
		if len(data) > 0 {
			err := json.Unmarshal(data, &ops)
			if (err != nil) {
				return nil, err
			}
		}
		if v, err := rm.ReadObjOps(table, ops, from, limit, desc, sortby, ud); err != nil {
			return nil, err
		} else {
			return t_rm.ConvertToJsonNames(v), err
		}
	}
}

func (rm *DBRequestHandler) ReadObjOps(table string, ops []Operation, from int, limit int,
						desc bool, sortby string, ud *UserData) (*[]TableRow, error) {
	if t_rm, ok := rm.queryBuilders[table]; ok {
		var err error
		fis := t_rm.GetFieldInfo()
		sel := t_rm.GetReadQuery()
		if v, ok := fis[sortby]; !ok && sortby != ""  {
			return nil, errors.New(sortby+" is not a valid field to sort by")
		} else {
			sortby = utils.ToSnakeCase(v.FN)
		}

		if ops != nil && len(ops) > 0 {
			errmap := make(map[string]string)
			for _, op := range ops {
				err = rm.validate.Struct(op)
				if err != nil {
					for _, e := range err.(validator.ValidationErrors) {
						fd := strings.Split(e.Namespace(), ".")
						n := fd[len(fd)-1]
						if v, ok := fis[n]; ok && v.Json != "" {
							n = v.Json
						}
						errmap[n] = e.Tag()
					}
				}
			}
			if len(errmap) > 0 {
				resp, _ := json.Marshal(errmap)
				return nil, errors.New(string(resp));
			}
		}

		//get query and params for this object
		var scond,defq string
		var params []interface{}
		for _, v := range ops {
			cond, err := v.createOpString(t_rm.GetName(), fis, &params)
			if err != nil {
				return nil, err
			}
			scond += cond
			if v.NextOp != "noop" {
				scond+= " "+v.NextOp
			}
		}
		accessq := ""
		var accessp *[]interface{}

		if !rm.isSU(ud) {
			//SU has read access to everything
			if ok, accessq, accessp = ud.P.HasReadAccess(table); !ok {
				log.Debugf("User %d does not have read access to %s", ud.Id, table)
				return nil, UNAUTHORIZED
			}

			if ud.Org_id > 0 {
				//limit user access to its org only
				if scond != "" {
					scond += " and "
				}
				scond += " "+rm.orgcol+"=? "
				params = append(params, ud.Org_id)
			} else if !rm.isSU(ud) {
				log.Debugf("user %d does not have read access to due to invalid org_id", ud.Org_id)
				return nil, UNAUTHORIZED
			}

			if accessp != nil && len(*accessp) > 0 {
				if scond != "" {
					scond += " and ((" + accessq + ") or auth_user_id=?)"
				} else {
					scond = " (" + accessq + ") or auth_user_id=?"
				}
				params = append(params, *accessp...)
				params = append(params, ud.Id)
			}
		}

		if scond != "" {
			scond = " where "+scond
		}

		if defq, err = DefaultRead(t_rm.GetName(), &from, &limit, desc, &sortby, scond); err != nil {
			return nil, err
		}

		if qobj, ok := t_rm.GetInstance().(QueryReader); ok {
			//in case there is a custom read
			return qobj.Read(t_rm, rm.db, from, limit, desc, sortby, scond, params)
		} else {
			if (defq != "") {
				q := "select "+sel+" " + defq;
				log.Debug("SQL Query: "+q)
				rows, err := rm.db.Query(q, params...)
				if err != nil {
					return nil, err
				}

				if res, err := t_rm.Convert(rows); err != nil {
					return nil, err
				} else {
					return res, nil
				}
			} else {
				log.Errorf("Unable to execute query, query builder failed (%s)", table)
				return nil, errors.New("Unable to execute query for "+table);
			}
		}
	} else {
		log.Errorf("%s is not registered",table);
		return nil, errors.New("not found :"+table)
	}

}

//Creating a new function rather than update, as had to change to many things to accommodate this
func (rm *DBRequestHandler) SetPassword(id int64, pass string) error {
	t_rm := rm.queryBuilders[rm.auth_table]
	var obj *AuthUser
	if id <=0 {
		return errors.New("object id invalid")
	} else {
		if found, err := findById(t_rm, rm.db, id); err != nil {
			return err
		} else {
			if found == nil || len(found) != 1 {
				return errors.New("Object with mentioned id could not be found")
			}
			obj = found[0].(*AuthUser)
		}
	}

	obj.Password = pass
	obj.maskWrite()
	q := fmt.Sprintf("update %s set password=? where id=?",t_rm.GetName())
	log.Debug("Update: "+q)
	insForm, err := rm.db.Prepare(q)
	if err != nil {
		return err
	}
	if s, err := insForm.Exec(obj.Password, id); err != nil {
		return err
	} else {
		if upd, err := s.RowsAffected(); err != nil {
			return err
		} else {
			log.Debugf("Update the password for id %d",upd)
			return nil
		}
	}
}

//Update fields of an entry
func (rm *DBRequestHandler) UpdateObj(table string, id int64, data []byte, ud *UserData) (map[string]interface{}, error) {
	if t_rm, ok := rm.queryBuilders[table]; ok {
		fis := t_rm.GetFieldInfo()
		var kvp map[string]interface{}
		var exist BaseModel
		err := json.Unmarshal(data, &kvp)
		if (err != nil) {
			return nil, err
		}

		if id <=0 {
			return nil, errors.New("object id invalid")
		} else {
			if found, err := findById(t_rm, rm.db, id); err != nil {
				return nil, err
			} else {
				if found == nil || len(found) != 1 {
					return nil, errors.New("Object with mentioned id could not be found")
				}
				exist = found[0]
			}
		}


		//we do manual validation rather than depending on validator
		if len(kvp) > MAX_UPDATE_LIMIT {
			return nil, errors.New("Cannot modify so many field in a single update")
		}

		if err := validateFields(&kvp, &fis, true); err != nil {
			return nil, err
		}

		if err := validateUnique(rm.db, t_rm, &kvp); err != nil {
			return nil, err
		}

		covrt_obj := t_rm.GetInstance()
		err = json.Unmarshal(data, covrt_obj)
		if (err != nil) {
			log.Errorf(err.Error())
			return nil, err
		}

		if err = rm.validatePermissionUpdates(covrt_obj, t_rm, ud); err != nil {
			return nil, err
		}

		var org, owner int64
		if rm.isSU(ud){
			if org, owner, err = assignOrgOwnerForSu(covrt_obj, &kvp, &fis, ud, false); err != nil {
				return nil, err
			}
		} else {
			//not allowed to update these
			delete(kvp, "org")
			delete(kvp, "owner")
		}

		//in case there are few fields need to be changed before writing to db
		orig_vals := make(map[string]interface{})
		if wm, ok := covrt_obj.(WriteMasker); ok {
			ufis, e := wm.maskWrite()
			if e != nil {
				return nil, e
			}
			//write back updated field to kvp
			for k, v := range ufis {
				if _, ok := kvp[k]; ok {
					orig_vals[k] = kvp[k]
					kvp[k] =v
				}
			}
		}

		accessq := ""
		var accessp *[]interface{}

		if rm.isSU(ud) {
			log.Debugf("User %d is SU. granting access", ud.Id)
		} else if bom, ok := exist.(BaseOwnerModel);ok && bom.GetOwner() == ud.Id {
			//owner has all the access
			log.Debugf("User %d is owner of %d in table %s. granting access", ud.Id, exist.GetId(), table)
		} else {
			if err = ud.P.HasUpdateAccess(table, kvp, fis); err != nil {
				return nil, UNAUTHORIZED
			}
		}


		//get query and params for this object
		var q string
		var params []interface{}
		q, params = buildUpdateQuery(kvp, fis)

		if (q != "") {

			if org > 0 {
				q += ", "+rm.orgcol+"=? "
				params = append(params, org)
			}

			if owner > 0 {
				q+= ", "+rm.ownercol+"=? "
				params = append(params, owner)
			}

			q = fmt.Sprintf("update %s set %s where id=?",t_rm.GetName(), q)
			params = append(params, id)

			//can only update if belong to same org and is not SU
			if !rm.isSU(ud) {
				if _,ok := exist.(BaseOrgModel); ok {
					//limit user access to its org only
					q+= " and "+rm.orgcol+"=? "
					params = append(params, ud.Org_id)
				}
			}

			//append update conditions as well
			if accessp != nil && len(*accessp) > 0 {
				q+= " and "+accessq
				params = append(params, *accessp...)
			}

			//final query
			log.Debug("Update: "+q)
			insForm, err := rm.db.Prepare(q)
			if err != nil {
				log.Errorf(err.Error())
				return nil, err
			}
			if s, err := insForm.Exec(params...); err != nil {
				log.Errorf(err.Error())
				return nil, err
			} else {
				if upd, err := s.RowsAffected(); err != nil {
					log.Errorf(err.Error())
					return nil, err
				} else {
					log.Debugf("Update the fields for id %d",upd)
					//copy back the prev values
					for k,v := range orig_vals {
						kvp[k] = v
					}
					return kvp, nil
				}
			}
		} else {
			log.Errorf("Unable to execute query, no key-value pairs can be build for given input")
			return nil, errors.New("Unable to execute query for "+table);
		}

	} else {
		log.Errorf("%s is not registered",table);
		return nil, errors.New("not found :"+table)
	}
}

func (rm *DBRequestHandler) SaveObj(data []byte, table string, ud *UserData) (BaseModel, error) {
	if t_rm, ok := rm.queryBuilders[table]; ok {
		fi := t_rm.GetFieldInfo()
		obj := t_rm.GetInstance()
		err := json.Unmarshal(data, obj)
		if (err != nil) {
			log.Errorf("Unable to decode data %s\n error : %s", data, err.Error())
			return nil, err
		}
		err = rm.validate.Struct(obj)
		if (err != nil) {
			errmap := make(map[string]string)
			for _, e := range err.(validator.ValidationErrors) {
				fd := strings.Split(e.Namespace(), ".")
				n := fd[len(fd)-1]
				if v, ok := fi[n]; ok && v.Json != "" {
					n = v.Json
				}
				errmap[n] = e.Tag()
			}
			resp, _ := json.Marshal(errmap)
			return nil, errors.New(string(resp));
		}

		if ud == nil {
			return nil, UNAUTHORIZED
		}

		//get raw values into a map
		var vmap = make(map[string]interface{})
		err = json.Unmarshal(data, &vmap)
		if (err != nil) {
			return nil, err
		}

		if err = rm.validatePermissionUpdates(obj, t_rm, ud); err != nil {
			return nil, err
		}

		var org, owner int64

		if !rm.isSU(ud) {
			if _, ok := obj.(BaseOrgModel);ok {
				org = ud.Org_id
			}
			if _, ok := obj.(BaseOwnerModel);ok {
				owner = ud.Id
			}
			if err := ud.P.HasCreateAccess(table, obj, fi); err != nil {
				return nil, UNAUTHORIZED
			}
		} else { //it does not belong to anyone as of now
			if org, owner, err = assignOrgOwnerForSu(obj, &vmap, &fi, ud, true); err != nil {
				return nil, err
			}
		}

		//in case there are few fields need to be changed before writing to db
		if wm, ok := obj.(WriteMasker); ok {
			_, e := wm.maskWrite()
			if e != nil {
				return nil, e
			}
		}

		if err := validateFields(&vmap, &fi, true); err != nil {
			return nil, err
		}

		if err := validateUnique(rm.db, t_rm, &vmap); err != nil {
			return nil, err
		}

		//get query and params for this object
		var q string
		var params []interface{}
		if qobj, ok := t_rm.GetInstance().(QueryCreator); ok {
			if q, params, err = qobj.Create(t_rm); err != nil {
				return nil, err
			}
		} else {
			q, params = buildCreateQuery(obj, fi)
		}
		if (q != "") {
			q = "insert into "+t_rm.GetName()+" set "+q+" "
			if org > 0 {
				q+=", "+rm.orgcol+"=?"
				params = append(params, org)
			}
			if owner > 0 {
				q+=", "+rm.ownercol+"=?"
				params = append(params, owner)
			}
			insForm, err := rm.db.Prepare(q)
			if err != nil {
				log.Errorf(err.Error())
				return nil, err
			}
			if s, err := insForm.Exec(params...); err != nil {
				log.Errorf(err.Error())
				return nil, err
			} else {
				if ins_id, err := s.LastInsertId(); err != nil {
					log.Errorf(err.Error())
					return nil, err
				} else {
					obj.SetId(ins_id)
					if orgObj, ok := obj.(BaseOrgModel); ok {
						orgObj.SetOrgId(org)
					}
					if ownObj, ok := obj.(BaseOwnerModel); ok {
						ownObj.SetOwner(owner)
					}
					//in case there are few fields need to be changed before sending to user
					if rm, ok := obj.(ReadMasker);ok {
						rm.maskRead()
					}
					return obj.(BaseModel), nil
				}
			}
		} else {
			log.Errorf("Unable to execute query, no key-value pairs can be build for (%s):(%s)", table, obj)
			return nil, errors.New("Unable to execute query for "+table);
		}

	} else {
		log.Errorf("%s is not registered",table);
		return nil, errors.New("not found :"+table)
	}

}

func (rm *DBRequestHandler) DeleteObj(table string, id int64, ud *UserData) error {
	if t_rm, ok := rm.queryBuilders[table]; ok {
		var exist BaseModel

		if id <=0 {
			log.Errorf("Invalid object id")
			return errors.New("object id invalid")
		} else {
			if found, err := findById(t_rm, rm.db, id); err != nil {
				log.Errorf(err.Error())
				return err
			} else {
				if found == nil || len(found) != 1{
					log.Errorf("obj not found for id %d",id)
					return errors.New("Object with mentioned id could not be found")
				}
				exist = found[0]
			}
		}

		accessq := ""
		var accessp *[]interface{}

		if rm.isSU(ud) {
			log.Infof("User %d is SU. granting delete access for row %d", ud.Id, id)
		} else if bom,ok := exist.(BaseOwnerModel); ok && bom.GetOwner() == ud.Id {
			//owner has all the access
			log.Infof("User %d is owner of %d in table %s. granting access", ud.Id, exist.GetId(), table)
		} else {
			if ok, accessq, accessp = ud.P.HasDeleteAccess(table); !ok {
				return UNAUTHORIZED
			}
		}

		params := []interface{}{id}

		q := "delete from "+t_rm.GetName()+" where id=? "
		if _, ok := exist.(BaseOrgModel);!rm.isSU(ud) && ok {
			//limit user access to its org only
			q+= " and "+rm.orgcol+"=? "
			params = append(params, ud.Org_id)
		}
		if accessp != nil && len(*accessp) > 0 {
			q+= "and "+accessq
			params = append(params, *accessp...)
		}
		insForm, err := rm.db.Prepare(q)
		if err != nil {
			return err
		}
		if s, err := insForm.Exec(params...); err != nil {
			return err
		} else {
			if ins_id, err := s.RowsAffected(); err != nil {
				log.Error(err)
				//this should fail only if user does not have access to it
				return UNAUTHORIZED
			} else {
				log.Debugf("Object deleted : %d",ins_id)
				return nil
			}
		}
	} else {
		log.Errorf("%s is not registered",table);
		return errors.New("not found :"+table)
	}

}

func DefaultRead(name string, from *int, limit *int, desc bool, sortby *string, cond string) (string, error) {
	if name == "" {
		return "", errors.New("table name missing in query")
	}
	if *from < 0 || *limit <0 {
		return "", errors.New("from, limit should be > 0")
	}
	if *limit == 0 || *limit > MAX_READ_LIMIT {
		*limit = MAX_READ_LIMIT
		log.Debugf("Limit of rows for %s is set to max %d", name, MAX_READ_LIMIT)
	}
	s := "id "
	//assuming sortby has been verified to be a field
	if *sortby != "" {
		valid := regexp.MustCompile("^[A-Za-z0-9_]+$")
		if !valid.MatchString(*sortby) {
			return "", errors.New("Invalid sortBy passed :"+*sortby)
		}
		s = *sortby
	}
	*sortby = s
	d := ""
	if desc {
		d = "desc"
	} else {
		d = "asc"
	}

	return fmt.Sprintf("from "+name+" %s order by %s %s limit %d,%d ",cond,s,d,*from,*limit), nil
}

func findById(t_rm *QueryBuilder, db *sql.DB, id int64) ([]BaseModel, error) {
	sel := t_rm.GetReadQuery()
	rows, err := db.Query("select "+sel+" from "+t_rm.GetName()+" where id=?", id)
	if err != nil {
		return nil, err
	}
	if m,err := t_rm.Convert(rows); err != nil {
		return nil, err
	} else {
		return t_rm.ConvertObj(m)
	}
}

func validateUnique(db *sql.DB, t_rm *QueryBuilder, vmap *map[string]interface{}) error {
	//check for unique keys
	uq_str := ""
	var uq_params []interface{}
	fi := t_rm.GetFieldInfo()
	for k,val := range *vmap {
		for _, f := range fi {
			if f.UQ && k == f.Json {
				uq_str += " "+f.DBN+"=? or"
				uq_params = append(uq_params, val)
				break;
			}
		}
	}

	if len(uq_params) > 0 {
		uq_str = strings.TrimSuffix(uq_str, "or")
		//make sure these does not already exists
		q := "select count(*) from "+t_rm.GetName()+" where "+uq_str;
		log.Debug("Query: "+q)
		if rows, err := db.Query(q, uq_params...); err != nil {
			log.Errorf("query error : %s", err.Error())
			return err
		} else {
			var count int
			rows.Next() // this should always return atleast one row
			if err := rows.Scan(&count); err != nil {
				log.Errorf("query error : %s", err.Error())
				return err
			} else if count > 0 {
				return errors.New("entry with mentioned data already exists")
			}
		}
	}
	return nil
}

func assignOrgOwnerForSu(obj BaseModel, vmap *map[string]interface{}, fi *map[string]FieldInfo, ud *UserData, isCreate bool) (int64, int64, error) {
	var org, owner int64
	if _, ok := obj.(BaseOrgModel);ok {
		if o, ok := (*vmap)["org"]; ok {
			//check if org_id exists in input, su can set this
			val := fmt.Sprintf("%v", o)
			if v, err := utils.ConvertFromString([]byte(val), (*fi)["org_id"].Type, DB_TIME_FORMAT); err == nil {
				org = v.(int64)
			} else {
				log.Errorf("Invalid format for org")
				return 0,0, errors.New("Invalid format for org")
			}
		} else if isCreate{//only create need the org
			log.Errorf("missing org in request")
			return 0,0, errors.New("missing org in input")
		}
	}

	if _, ok := obj.(BaseOwnerModel);ok {
		if o, ok := (*vmap)["owner"]; ok {
			//check if org_id exists in input, su can set this
			val := fmt.Sprintf("%v", o)
			if v, err := utils.ConvertFromString([]byte(val), (*fi)["auth_user_id"].Type, DB_TIME_FORMAT); err == nil {
				owner = v.(int64)
			} else {
				log.Errorf("Invalid format for owner")
				return 0,0, errors.New("Invalid format for owner")
			}
		} else if isCreate { // update owner only if
			//owner = creator
			owner = ud.Id
		}
	}
	return org, owner, nil
}

//validates user is not trying to add/update columns in permission table which should not have been done
//User cannot add/modify permissions for role/role_permission table
//User cannot add access based on org, owner columns, they are appended automatically
//Validates tablename & column name
//Validates value type
//I can only give access of what I have access to
func (dbr *DBRequestHandler)validatePermissionUpdates(bm BaseModel, qb *QueryBuilder, ud *UserData) error {
	issu := dbr.isSU(ud)

	var model BasePermissionModel
	var ok bool
	if model, ok = bm.(BasePermissionModel); !ok {
		return nil // no validation needed
	}

	fis := qb.GetFieldInfo()

	found := false
	for k, _ := range dbr.queryBuilders {
		if k == model.getTableName() {
			found = true
			break
		}
	}
	if (!found) {
		return errors.New(fmt.Sprintf("Table %s not registered", model.getTableName()))
	}

	found = false
	for _, fi := range fis {
		if fi.DBN == model.getColumnName() {
			found = true
			if model.getValue() == "" {
				msg := fmt.Sprintf("%s col has empty value", fi.DBN)
				log.Debug(msg)
				return errors.New(msg)
			}
			if _, err := utils.ConvertFromString([]byte(model.getValue()), fi.Type, DB_TIME_FORMAT); err != nil {
				msg := fmt.Sprintf("%s col has invalid value : %s", fi.DBN, model.getValue())
				log.Debug(msg)
				return errors.New(msg)
			}
			break
		}
	}

	if (!found) {
		msg := fmt.Sprintf("col has invalid column name : %s", model.getColumnName())
		log.Debug(msg)
		return errors.New(msg)
	}

	if model.getColumnName() == dbr.orgcol {
		return errors.New("can not add org column condition on table")
	}

	if model.getColumnName() == dbr.ownercol {
		return errors.New("can not add owner column condition on table")
	}

	tn := model.getTableName()
	if !issu {
		if tn == dbr.role_table || tn == dbr.auth_role_permission_table || tn == dbr.org_table {
			return errors.New("given table cannot be accessed")
		}
	}

	myps := ud.P
	found = false
	for mytn, tablep := range myps.Ps {
		if mytn == tn {
			var cond *Condition
			switch model.getPermission() {
			case PERMISSION_R : cond = tablep.Read
			case PERMISSION_C : cond = tablep.Create
			case PERMISSION_U : cond = tablep.Update
			case PERMISSION_D : cond = tablep.Delete
			default:
				return errors.New("Invalid permission type passed")
			}
			if cond == nil {
				return errors.New("cannot add permission for given table")
			}
			for k,vals := range *cond {
				if k == model.getColumnName() {
					for _,v := range vals {
						if v == model.getValue() {
							found = true
							break
						}
					}
				}
			}
		}
	}
	if !found {
		return errors.New("cannot add permission for given table")
	}

	return nil
}
