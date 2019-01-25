package models

import (
	"time"
)

type Artist struct {
	ID int64 			`json:"user_role_id" v:"ro"`
	Name string			`json:"name" validate:"required"`
	Gender string		`json:"gender" validate:"oneof M F N"`
	AuthUserId int64 	`json:"_" validate:"required" v:"ro"`
	OrgId int64 		`json:"_" validate:"required" v:"ro"`
	DateAdd time.Time 	`json:"date_add" v:"ro"`
	DateUpd time.Time 	`json:"date_upd" v:"ro"`
}

func (au *Artist) Register() *QueryBuilder{
	qb := &QueryBuilder{}
	return qb.InitFieldInfo(&Artist{}, func() BaseModel {
		return &Artist{}
	})
}

func (au *Artist) SetId(id int64) {
	au.ID = id
}

func (au *Artist) GetId() int64 {
	return au.ID
}

func (au *Artist) SetOrgId(id int64) {

}

func (au *Artist) GetOrgId() int64 {
	return -1
}

func (au *Artist) GetOwner() int64 {
	return 0
}

func (au *Artist) SetOwner(id int64) {
	au.AuthUserId = id
}

