package models

import (
	"time"
)

type UserPermission struct {
	ID 		   int64 	`json:"user_permission_id" v:"ro"`
	AuthUserId int64	`json:"auth_user_id" validate:required`
	TableName  string   `json:"table_name" validate:"required"`
	ColumnName string   `json:"column_name" validate:"required"`
	Value      string   `json:"value"`
	Permission string  	`json:"permission" validate="oneof c r u d *"`
	OrgId	   int64	`json:"org_id" validate:"required"`
	DateAdd time.Time 	`json:"date_add" v:"ro"`
	DateUpd time.Time 	`json:"date_upd" v:"ro"`
}

func (au *UserPermission) Register() *QueryBuilder {
	bq := QueryBuilder{}
	return bq.InitFieldInfo(&UserPermission{}, func() BaseModel {
		return &UserPermission{}
	})
}

func (au *UserPermission) getTableName() string {
	return string(au.TableName)
}

func (au *UserPermission) getColumnName() string {
	return string(au.ColumnName)
}

func (au *UserPermission) getPermission() string {
	return string(au.Permission)
}

func (au *UserPermission) getValues() string {
	return au.Value
}


func (au *UserPermission) SetId(id int64) {
	au.ID = id
}

func (au *UserPermission) GetId() int64 {
	return au.ID
}

func (au *UserPermission) SetOrgId(id int64) {
	au.OrgId = id
}

func (au *UserPermission) GetOrgId() int64 {
	return au.OrgId
}

