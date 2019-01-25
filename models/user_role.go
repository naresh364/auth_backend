package models

import (
	"time"
)

type UserRole struct {
	ID int64 			`json:"user_role_id" v:"ro"`
	Role string 		`json:"role" validate:"required" v:"uq"`
	Desc string			`json:"desc"`
	DateAdd time.Time 	`json:"date_add" v:"ro"`
	DateUpd time.Time 	`json:"date_upd" v:"ro"`
}

func (au *UserRole) Register() *QueryBuilder {
	bq := QueryBuilder{}
	return bq.InitFieldInfo(&UserRole{}, func() BaseModel {
		return &UserRole{}
	})
}

func (au *UserRole) SetId(id int64) {
	au.ID = id
}

func (au *UserRole) GetId() int64 {
	return au.ID
}

