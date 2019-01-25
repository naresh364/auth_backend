package models

import (
	"database/sql"
	"errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"time"
)

//ID should always be the first element, should be ro
type AuthUser struct {
	ID         int64     `json:"auth_user_id" v:"ro"`
	Username   string    `json:"username" validate:"min=3,max=12" v:"uq"`
	Email      string    `json:"email" validate:"email" v:"uq"`
	Password   string    `json:"_" v:"password,noread"`
	UserRoleId int64     `json:"user_role_id" validate:"required"`
	IsActive   int8	     `json:"is_active"`
	OrgId      int64	 `json:"_" v:"ro"`
	FacebookId string	 `json:"_"`
	GoogleId   string	 `json:"_"`
	DateAdd    time.Time `json:"date_add" v:"ro"`
	DateUpd    time.Time `json:"date_upd" v:"ro"`
	UserRole   UserRole  `json:"user_role" v:"ref" validate:"structonly"`
}

func (au *AuthUser) Register() *QueryBuilder {
	bq := QueryBuilder{}
	return bq.InitFieldInfo(&AuthUser{}, func() BaseModel {
		return &AuthUser{}
	})
}

func (au *AuthUser) SetId(id int64) {
	au.ID = id
}

func (au *AuthUser) GetId() int64 {
	return au.ID
}

func (au *AuthUser) SetOrgId(id int64) {
	au.OrgId = id
}

func (au *AuthUser) GetOrgId() int64 {
	return au.OrgId
}

func (au *AuthUser) maskRead() (map[string]interface{}, error) {
	return nil,nil
}

func (au *AuthUser) maskWrite() (map[string]interface{}, error) {
	var ret = make(map[string]interface{})
	//we need atleast one to work with
	if au.Username == "" && au.GoogleId == "" && au.FacebookId == "" {
		return nil, errors.New("Invalid username/FB Id/Google Id")
	}
	if au.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(au.Password), bcrypt.MinCost)
		if err != nil {
			log.Error(err)
			return nil, errors.New("Unable to hash password for AuthUser")
		}
		au.Password = string(hash)
		ret["password"] = au.Password
	}
	//TODO : autogenerate these
	return ret,nil
}

func (au *AuthUser) validatePassword(p string) error {
	return bcrypt.CompareHashAndPassword([]byte(au.Password), []byte(p))
}

var converter = func (rows *sql.Rows) ([]BaseModel, error){
	var resp []BaseModel
	for rows.Next() {
		inst := AuthUser{}
		err := rows.Scan(&inst.ID, &inst.Username, &inst.Email,
			&inst.UserRoleId, &inst.IsActive, &inst.OrgId,
			&inst.FacebookId, &inst.GoogleId,
			&inst.DateAdd, &inst.DateUpd)
		if err != nil {
			return nil, err
		}
		resp = append(resp, &inst)
	}
	return resp,nil
}

//func (qb *AuthUser) Read(db *sql.DB,from int, limit int, desc bool,
//	sortby string, cond string, params []interface{}) ([]BaseModel, error) {
//
//	q := "select au.id, au.username, au.email, au.user_role_id, au.is_active," +
//		" au.org_id, au.facebook_id, au.google_id, au.date_add, au.date_upd, ur.id, ur.role," +
//		" ur.desc from auth_user au join user_role ur on ur.id=au.user_role_id "+
//		cond+" limit "+strconv.Itoa(from)+","+strconv.Itoa(limit)
//	log.Debug("Query :"+q)
//
//	var ret []BaseModel
//
//	if rows,err := db.Query(q, params...); err != nil {
//		return nil, err
//	} else {
//		if m, err := Convert(rows); err != nil {
//			return nil, err
//		} else {
//			for _,r := range *m {
//				au := AuthUser{}
//				ur := UserRole{}
//				au.ID = int64(r["au.id"])
//
//
//			}
//
//		}
//		var resp []BaseModel
//		for rows.Next() {
//			inst := AuthUser{}
//			err := rows.Scan(&inst.ID, &inst.Username, &inst.Email, &inst.Password,
//				&inst.UserRoleId, &inst.IsActive, &inst.OrgId, &inst.FacebookId, &inst.GoogleId,
//				&inst.DateAdd, &inst.DateUpd, &inst.UserRole.ID, &inst.UserRole.Role, &inst.UserRole.Desc,
//				&inst.UserRole.DateAdd, &inst.UserRole.DateUpd)
//			if err != nil {
//				return nil, err
//			}
//			inst.maskRead()
//			resp = append(resp, &inst)
//		}
//		return resp,nil
//	}
//}

