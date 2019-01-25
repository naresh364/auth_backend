package models

import (
	"time"
)

type UserRolePermission struct {
	ID         int64     `json:"user_role_permission_id" v:"ro"`
	TableName  string    `json:"table_name" validate:"required"`
	ColumnName string    `json:"column_name" validate:"required"`
	Value      string    `json:"values"`
	Permission string     `json:"permission" validate="oneof c r u d *"`
	UserRoleId int64     `json:"user_role_id" validate:"required"`
	DateAdd    time.Time `json:"date_add" v:"ro"`
	DateUpd    time.Time `json:"date_upd" v:"ro"`
}

func (au *UserRolePermission) Register() *QueryBuilder {
	bq := QueryBuilder{}
	return bq.InitFieldInfo(&UserRolePermission{}, func() BaseModel {
		return &UserRolePermission{}
	})
}

func (au *UserRolePermission) getTableName() string {
	return string(au.TableName)
}

func (au *UserRolePermission) getColumnName() string {
	return string(au.ColumnName)
}

func (au *UserRolePermission) getPermission() string {
	return string(au.Permission)
}

func (au *UserRolePermission) getValue() string {
	return au.Value
}

func (au *UserRolePermission) SetId(id int64) {
	au.ID = id
}

func (au *UserRolePermission) GetId() int64 {
	return au.ID
}

