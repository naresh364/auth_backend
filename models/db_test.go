package models_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/auth_backend/models"
	"github.com/auth_backend/utils"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

var commandParams = flag.String("config", "", "Config file (Json format)")
var dbmHandler *models.DBRequestHandler
var super_user *models.UserData
var simple_user *models.UserData

var nu, nt, nrole, nup map[string]interface{}

func setUp(t *testing.T) *models.DBRequestHandler {
	_, filename, _, _ := runtime.Caller(0)
	ind := strings.LastIndex(filename, "/")
	dir := string([]byte(filename)[:ind])
	parentdir := dir+"/../"
	defer utils.Chdir(t, parentdir)()
	viper.SetConfigFile(*commandParams)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatal(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	env := viper.GetString("env")
	if env != "test" {
		t.Fatal("Invalid config files passed should have env test")
	}

	dbuser := viper.GetString("db.user")
	dbpass := viper.GetString("db.pass")
	dbhost := viper.GetString("db.host")
	dbdb := viper.GetString("db.db")
	if !strings.Contains(dbdb, "test") {
		t.Fatal("DB name should have test in its name")
	}

	initdbf := viper.GetString("db_setup_file")

	//setup logging
	var file *os.File
	var err error
	if initdbf != "" {
		file, err = os.Open(initdbf)
		if err != nil {
			t.Fatal("Unable to open db init file : "+err.Error())
		}
		defer file.Close()
	} else {
		t.Fatal("missing db init file")
	}

	content,err := ioutil.ReadFile(initdbf)
	strcon := string(content)
	dbexec := strings.Replace(strcon, "database_name_", dbdb, -1)

	cmd := exec.Command("mysql", "-u", dbuser, "-p"+dbpass, "-e", dbexec)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatal("Unable to execute db script : "+stderr.String())
	}

	dbconstr := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",dbuser, dbpass, dbhost, dbdb)
	db, err := sql.Open("mysql", dbconstr)
	if err != nil {
		t.Fatal("Unable to open connection to DB")
	}
	err = db.Ping()
	if err != nil {
		t.Fatal("Unable to open db connection : please add user:pass, host:port and db params")
	}

	dbh := models.InitDB(db, viper.GetString("org_col"), viper.GetString("owner_col"), 1, 1)

	nu = map[string]interface{}{
		"auth_user_id":0, //shold get ignored, is ro
		"username":"test_user", //is unique
		"email":"email@test.com", //is unique
		"password":"should not matter",
		"user_role_id":2,
		"is_active":1,
		"org_id":5,//this will get ignored
		"org":2,//this will be used, priviliged to super admin only
	}

	nrole = map[string]interface{}{
		"user_role_id":0, //shold get ignored, is ro
		"role":"test_user", //is unique
		"desc":nil, //is unique
	}

	nt = map[string]interface{}{
		"test_id":0, //should get ignored, is ro
		"name":"test_user", //is unique
		"s_value":"A",
		"i_value":-1,
		"u_value":1,
		"f_value":1.1,
		"d_value":1.2,
		"org":2,
		"owner":2,
	}

	nup = map[string]interface{}{
		"user_permission_id":0, //shold get ignored, is ro
		"auth_user_id":"test_user", //is unique
		"table_name":"test_table", //is unique
		"desc":nil, //is unique
	}

	return dbh
}

func mustHave(t *testing.T) {
	//all test need the variables initialized in these
	if dbmHandler == nil || super_user == nil || simple_user == nil {
		TestRegister(t)
		TestAuthenticate(t)
	}
}

func TestRegister(t *testing.T) {
	//will be used for tests
	dbmHandler = setUp(t)
	tt := &TestTable{}
	utils.Ok(t, dbmHandler.RegisterDB(tt.Register()))
	//double registration should fail
	err := dbmHandler.RegisterDB(tt.Register())
	utils.Assert(t, err != nil, "Duplicate table registration should have failed")
	tt2 := &TestTable2{}
	utils.Ok(t, dbmHandler.RegisterDB(tt2.Register()))
}

func TestAuthenticate(t *testing.T) {
	if dbmHandler == nil { t.Fatal("Database not initialized") }
	_, _, err := dbmHandler.Authenticate("test", "WrongPass")
	utils.Assert(t, err != nil, "Should have an error while logging in with wrong username")
	_, _, err = dbmHandler.Authenticate("su", "WrongPass")
	utils.Assert(t, err != nil, "Should have an error while logging in with wrong username")
	bm, ps, err := dbmHandler.Authenticate("su", "nkktest")
	utils.Assert(t, err == nil, "Should have logged in")
	au := bm.(*models.AuthUser)
	super_user = &models.UserData{Id: au.GetId(), Org_id:au.OrgId, Uuid:"su_uuid", P: ps}

	bm, ps, err = dbmHandler.Authenticate("simple_user", "nkktest")
	utils.Assert(t, err == nil, "Should have logged in")
	au = bm.(*models.AuthUser)
	simple_user = &models.UserData{Id: au.GetId(), Org_id:au.OrgId, Uuid:"simple_user_uuid", P: ps}
}


func TestCreate(t *testing.T) {
	mustHave(t)

	emptym := map[string]interface{}{}
	emptys := []string{}
	rofs := []string{"auth_user_id", "org_id"}

	test_table := map[string]struct{
		success bool
		input map[string]interface{}
		table string
		user *models.UserData
		remove []string
		change map[string]interface{}
		equal []string
		not_equal []string
	}{
		"1.1 no_user" : {false, nu, "auth_user", nil, emptys, emptym, emptys, emptys},
		"1.2 missing_req" : {false, nu, "auth_user", super_user,[]string{"user_role_id"}, emptym, emptys, emptys},
		"1.3 no_access" : {false, nu, "auth_user", simple_user,emptys, emptym, emptys, emptys},
		"1.4 ro_field" : {false, nu, "auth_user", super_user, emptys, map[string]interface{}{"auth_user_id":1}, []string{"username", "email", "user_role_id"}, rofs},
		"1.5 nosu_access_s_A" : {false, nt, "test_table", simple_user, []string{"test_id"}, map[string]interface{}{"s_value":"A", "name":"uq1"}, []string{"name", "s_value", "i_value", "u_value", "f_value", "d_value"}, []string{"test_id"}},
		"1.6 nosu_access_s_B" : {true, nt, "test_table", simple_user, []string{"test_id"}, map[string]interface{}{"f_value":"1.1", "name":"uq1"}, []string{"name", "s_value", "i_value", "u_value", "f_value", "d_value"}, []string{"test_id"}},
		"1.7 nosu_access_s_B_unque" : {false, nt, "test_table", simple_user, []string{"test_id"}, map[string]interface{}{"f_value":"1.1", "name":"uq1"}, []string{"name", "s_value", "i_value", "u_value", "f_value", "d_value"}, []string{"test_id"}},
		"1.8 sudo_no_org" : {false, nt, "test_table", super_user, []string{"test_id", "org", "owner"}, map[string]interface{}{"s_value":"A", "name":"uq2"}, []string{"name", "s_value", "i_value", "u_value", "f_value", "d_value"}, []string{"test_id"}},
		"1.9 sudo_w_org" : {true, nt, "test_table", super_user, []string{"test_id", "owner"}, map[string]interface{}{"s_value":"A", "name":"uq3"}, []string{"name", "s_value", "i_value", "u_value", "f_value", "d_value"}, []string{"test_id"}},
		"2.0 nosu_rolet" : {false, nrole, "user_role", simple_user, emptys, emptym, emptys, emptys},
	}

	keys := make([]string, len(test_table))
	i :=0
	for k,_ := range test_table {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		test := test_table[k] //to sort the table
		fmt.Printf("Executing : %s\n",k)
		qb := dbmHandler.GetQueryBuilders(test.table)
		utils.NotEquals(t, qb, nil)
		fis := qb.GetFieldInfo()
		v := map[string]interface{}{}
		mapstructure.Decode(&test.input, &v)
		for k,newv := range test.change {
			v[k] = newv
		}
		for _, k := range test.remove {
			delete(v, k)
		}
		data, err := json.Marshal(v)
		utils.Ok(t, err)
		bm, err := dbmHandler.SaveObj(data, test.table, test.user)
		utils.Equals(t, test.success, err==nil)
		if err == nil {
			owner := test.user.Id
			org := test.user.Org_id
			if test.user == super_user {
				if own, ok := v["owner"]; ok {
					sowner := fmt.Sprintf("%v", own)
					owner,_ = strconv.ParseInt(sowner, 10, 64)
				}
				if o, ok := v["org"]; ok {
					sorg := fmt.Sprintf("%v", o)
					org,_ = strconv.ParseInt(sorg, 10, 64)
				}
			}
			utils.NotEquals(t, bm, nil)
			decodeMap := map[string]interface{}{}
			validateMap := map[string]interface{}{}
			mapstructure.Decode(bm, &decodeMap)
			//from field name to json name
			for k,v := range decodeMap {
				if fi, ok := fis[k];ok{
					validateMap[fi.Json] = v
				} else {
					utils.Equals(t, true, ok)
				}
			}

			//validate owner & org
			if act_owner, ok := decodeMap["AuthUserId"]; ok {
				utils.Equals(t, owner, act_owner)
			}

			if act_org, ok := decodeMap["OrgId"]; ok {
				utils.Equals(t, org, act_org)
			}

			for _, eq := range test.equal {
				exp := fmt.Sprintf("%v", v[eq])
				act := fmt.Sprintf("%v", validateMap[eq])
				utils.Equals(t, exp, act)
			}
			for _, neq := range test.not_equal {
				exp := fmt.Sprintf("%v", v[neq])
				act := fmt.Sprintf("%v", validateMap[neq])
				utils.NotEquals(t, exp, act)
			}
		}
	}

}

func TestRead(t *testing.T) {
	mustHave(t)

}

func TestUpdate(t *testing.T) {
	mustHave(t)

}

func TestDelete(t *testing.T) {
	mustHave(t)

}

type TestTable struct {
	ID int64 			`json:"test_id" v:"ro"`
	Name string			`json:"name" validate:"required" v:"uq"`//is unique
	SValue string		`json:"s_value" validate:"oneof=A B C"`
	IValue int64 		`json:"i_value"`
	UValue uint  		`json:"u_value"`
	FValue float32		`json:"f_value"`
	DValue float64		`json:"d_value"`
	AuthUserId int64 	`json:"_" v:"ro"`
	OrgId int64 		`json:"_" v:"ro"`
	DateAdd time.Time 	`json:"date_add" v:"ro"`
	DateUpd time.Time 	`json:"date_upd" v:"ro"`
}

func (au *TestTable) Register() *models.QueryBuilder{
	qb := &models.QueryBuilder{}
	return qb.InitFieldInfo(&TestTable{}, func() models.BaseModel {
		return &TestTable{}
	})
}

func (au *TestTable) SetId(id int64) {
	au.ID = id
}

func (au *TestTable) GetId() int64 {
	return au.ID
}

func (au *TestTable) SetOrgId(id int64) {
	au.OrgId = id
}

func (au *TestTable) GetOrgId() int64 {
	return au.OrgId
}

func (au *TestTable) GetOwner() int64 {
	return au.AuthUserId
}

func (au *TestTable) SetOwner(id int64) {
	au.AuthUserId = id
}


type TestTable2 struct {
	ID int64 			`json:"test_id" v:"ro"`
	Name string			`json:"name" validate:"required" v:"uq"`//is unique
	IValue int64 		`json:"i_value"`
	AuthUserId int64 	`json:"_" v:"ro"`
	OrgId int64 		`json:"_" v:"ro"`
	TestTableId int64 	`json:"test_table_id" v:"ro"`
	DateAdd time.Time 	`json:"date_add" v:"ro"`
	DateUpd time.Time 	`json:"date_upd" v:"ro"`
}

func (au *TestTable2) Register() *models.QueryBuilder{
	qb := &models.QueryBuilder{}
	return qb.InitFieldInfo(&TestTable2{}, func() models.BaseModel {
		return &TestTable2{}
	})
}

func (au *TestTable2) SetId(id int64) {
	au.ID = id
}

func (au *TestTable2) GetId() int64 {
	return au.ID
}

func (au *TestTable2) SetOrgId(id int64) {
	au.OrgId = id
}

func (au *TestTable2) GetOrgId() int64 {
	return au.OrgId
}

func (au *TestTable2) GetOwner() int64 {
	return au.AuthUserId
}

func (au *TestTable2) SetOwner(id int64) {
	au.AuthUserId = id
}

