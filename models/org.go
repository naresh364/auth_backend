package models

import (
	"time"
)

type Org struct {
	ID int64 			`json:"org_id" v:"ro"`
	Name string 		`json:"name" validate:"required"`
	DateAdd time.Time 	`json:"date_add" v:"ro"`
	DateUpd time.Time 	`json:"date_upd" v:"ro"`
}

func (au *Org) Register() *QueryBuilder {
	bq := QueryBuilder{}
	return bq.InitFieldInfo(&Org{}, func() BaseModel {
		return &Org{}
	})
}

func (au *Org) SetId(id int64) {
	au.ID = id
}

func (au *Org) GetId() int64 {
	return au.ID
}

