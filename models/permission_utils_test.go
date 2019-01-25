package models

import (
	"fmt"
	"github.com/auth_backend/utils"
	"sort"
	"strings"
	"testing"
)

func TestAddPermission(t *testing.T) {
	test_table := map[string]struct {
		shouldMatch bool
		valcounts int
		tp TablePermission
	}{
		"1.1:all" : {true, 0, TablePermission{Read: &Condition{}, Create:&Condition{}, Update: &Condition{}, Delete: &Condition{}}},
		"1.2:all2" : {false, 0, TablePermission{Read: &Condition{"col":{""}}, Create:&Condition{}, Update: &Condition{}, Delete: &Condition{}}},
		"1.3:ro" : {true, 0, TablePermission{Read: &Condition{}}},
		"1.4:ro_c" : {true, 1, TablePermission{Read: &Condition{"xc":{"c"}}}},
		"1.5:ro_1" : {true, 1, TablePermission{Read: &Condition{"xi":{"1"}}}},
		"1.6:ro_1.0" : {true, 1, TablePermission{Read: &Condition{"xf":{"1.0"}}}},
		"1.7:co" : {true, 0, TablePermission{Create: &Condition{}}},
		"1.8:co_c" : {true, 1, TablePermission{Create: &Condition{"xc":{"c"}}}},
		"1.9:co_1" : {true, 1, TablePermission{Create: &Condition{"xi":{"1"}}}},
		"2.0:co_1.0" : {true, 1, TablePermission{Create: &Condition{"xf":{"1.0"}}}},
		"2.1:uo" : {true, 0, TablePermission{Update: &Condition{}}},
		"2.2:uo_c" : {true, 1, TablePermission{Update: &Condition{"xc":{"c"}}}},
		"2.3:uo_1" : {true, 1, TablePermission{Update: &Condition{"xi":{"1"}}}},
		"2.4:uo_1.0" : {true, 1, TablePermission{Update: &Condition{"xf":{"1.0"}}}},
		"2.5:do" : {true, 0, TablePermission{Delete: &Condition{}}},
		"2.6:do_c" : {true, 1, TablePermission{Delete: &Condition{"xc":{"c"}}}},
		"2.7:do_1" : {true, 1, TablePermission{Delete: &Condition{"xi":{"1"}}}},
		"2.8:do_1.0" : {true, 1, TablePermission{Delete: &Condition{"xf":{"1.0"}}}},
		"2.9:multiv" : {true, 5, TablePermission{Read: &Condition{"xf":{"1.0", "2.2"}, "xc":{"a", "b", "c"}}}},
		"3.0:multip" : {true, 2, TablePermission{Read: &Condition{}, Create: &Condition{"xi":{"1", "2"}}}},
	}

	keys := make([]string, len(test_table))
	i :=0
	for k,_ := range test_table {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	ps := Permissions{Ps:make(map[string]*TablePermission)}
	for _, k := range keys {
		fmt.Printf("Executing test : %s\n",k)
		testkey := strings.Split(k, ":")[1]
		v := test_table[k]
		count := 0
		count += helperTestingSingleCase(t, testkey, &ps, v.tp.Read, PERMISSION_R)
		count += helperTestingSingleCase(t, testkey, &ps, v.tp.Create, PERMISSION_C)
		count += helperTestingSingleCase(t, testkey, &ps, v.tp.Update, PERMISSION_U)
		count += helperTestingSingleCase(t, testkey, &ps, v.tp.Delete, PERMISSION_D)
		utils.Equals(t, v.valcounts, count)
		if v.shouldMatch {
			utils.Equals(t, v.tp.Read, (ps.Ps[testkey]).Read)
			utils.Equals(t, v.tp.Create, (ps.Ps[testkey]).Create)
			utils.Equals(t, v.tp.Update, (ps.Ps[testkey]).Update)
			utils.Equals(t, v.tp.Delete, (ps.Ps[testkey]).Delete)
		}
	}
}

func helperTestingSingleCase(t *testing.T, table string, ps *Permissions, cond *Condition, pt string) int {
	if cond == nil {
		return 0
	}
	if len(*(cond)) == 0 {
		ps.addPermission(table, pt, "", "")
		return 0
	}
	ret := 0
	for kcol, vval := range *(cond) {
		for _,val := range vval {
			if err := ps.addPermission(table, pt, kcol, val); err == nil && val != "" {//empty vals are ignored
				ret++
			}
		}
	}
	return ret
}


func TestValidatePermission(t *testing.T) {

	fvdetails := map[string]FieldInfo {
		"XC" : {DBN:"xc"},
		"XI" : {DBN:"xi"},
		"XF" : {DBN:"xf"},
	}
	//setup permissions
	ps := Permissions{Ps:make(map[string]*TablePermission)}
	//table all has all the permissions
	ps.addPermission("all", PERMISSION_R,"", "")
	ps.addPermission("all", PERMISSION_C,"", "")
	ps.addPermission("all", PERMISSION_U,"", "")
	ps.addPermission("all", PERMISSION_D,"", "")
	//read only
	ps.addPermission("ro", PERMISSION_R,"", "")
	ps.addPermission("ro_c", PERMISSION_R,"xc", "b")
	ps.addPermission("ro_c", PERMISSION_R,"xc", "a")
	ps.addPermission("ro_c", PERMISSION_R,"xi", "1")
	ps.addPermission("ro_c", PERMISSION_R,"xi", "2")
	ps.addPermission("ro_c", PERMISSION_R,"xf", "1.0")
	ps.addPermission("ro_c", PERMISSION_R,"xf", "1.1")
	//delete only
	ps.addPermission("do", PERMISSION_D,"", "")
	ps.addPermission("do_c", PERMISSION_D,"xc", "b")
	ps.addPermission("do_c", PERMISSION_D,"xc", "a")
	ps.addPermission("do_c", PERMISSION_D,"xi", "1")
	ps.addPermission("do_c", PERMISSION_D,"xi", "2")
	ps.addPermission("do_c", PERMISSION_D,"xf", "1.0")
	ps.addPermission("do_c", PERMISSION_D,"xf", "1.1")
	//create only
	ps.addPermission("co", PERMISSION_C,"", "")
	ps.addPermission("co_c", PERMISSION_C,"xc", "b")
	ps.addPermission("co_c", PERMISSION_C,"xc", "a")
	ps.addPermission("co_c", PERMISSION_C,"xi", "1")
	ps.addPermission("co_c", PERMISSION_C,"xi", "2")
	ps.addPermission("co_c", PERMISSION_C,"xf", "1.0")
	ps.addPermission("co_c", PERMISSION_C,"xf", "1.1")
	//update only
	ps.addPermission("uo", PERMISSION_U,"", "")
	ps.addPermission("uo_c", PERMISSION_U,"xc", "b")
	ps.addPermission("uo_c", PERMISSION_U,"xc", "a")
	ps.addPermission("uo_c", PERMISSION_U,"xi", "1")
	ps.addPermission("uo_c", PERMISSION_U,"xi", "2")
	ps.addPermission("uo_c", PERMISSION_U,"xf", "1.0")
	ps.addPermission("uo_c", PERMISSION_U,"xf", "1.1")
	//mix
	ps.addPermission("mix", PERMISSION_R,"", "")// unconditional read
	ps.addPermission("mix", PERMISSION_D,"xc", "b") // conditional delete
	ps.addPermission("mix", PERMISSION_U,"", "")
	ps.addPermission("mix", PERMISSION_C,"xi", "1")
	ps.addPermission("mix", PERMISSION_C,"xi", "2")


	kvp_v_2 := map[string]interface{}{"a":"test", "XI": 10}
	kvp_v_4 := map[string]interface{}{"a":"test", "XI": 1, "XF":1.0, "XC":"b"}
	kvp_v_4_1 := map[string]interface{}{"a":"test", "XI": 1, "XF":1.1, "XC":"a"}
	kvp_iv_4 := map[string]interface{}{"a":"test", "XI": 9, "XF":1.1, "XC":"a"}

	test_table := map[string]struct{
		ra, ca, ua, da  bool //access
		rc, dc string // access condition
		kvp map[string]interface{} //used for create update, we tell it what values we are updating and if they can be done
		rlen, dlen int //param length for read, delete
	} {
		"1.1:all": {true, true, true, true, "","",nil,0,0},
		"1.2:noaccess": {false, false, false, false, "","",nil,0,0},

		"2.1:ro": {true, false, false, false, "","",nil,0,0},
		"2.2:ro_c": {true, false, false, false, "xc;xi;xf","",nil, 6,0},

		"2.3:do": {false, false, false, true, "","", nil, 0,0},
		"2.4:do_c": {false, false, false, true, "xc;xi;xf","",nil, 0,6},

		"2.5:co": {false, true, false, false, "","",kvp_v_2, 0,0},//no condition access
		"2.6:co_c": {false, true, false, false, "","",kvp_v_4, 0,0},
		"2.7:co_c": {false, true, false, false, "","", kvp_v_4_1, 0,0},
		"2.8:co_c": {false, false, false, false, "","", kvp_iv_4, 0,0},

		"2.9:uo": {false, false, true, false, "","", kvp_v_2, 0,0},//no condition access
		"2.a:uo_c": {false, false, true, false, "","", kvp_v_4, 0,0},
		"2.b:uo_c": {false, false, true, false, "","", kvp_v_4_1, 0,0},
		"2.c:uo_c": {false, false, false, false, "","", kvp_iv_4, 0,0},

		"3.1:mix": {true, true, true, true, "","xc", kvp_v_4, 0,1},
		"3.2:mix": {true, false, true, true, "","xc", kvp_v_2, 0,1},
	}

	keys := make([]string, len(test_table))
	i :=0
	for k,_ := range test_table {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		test := test_table[k]
		fmt.Printf("Executing %s\n", k)
		table := strings.Split(k, ":")[1]
		//read
		a,b,c := ps.HasReadAccess(table)
		utils.Equals(t, test.ra, a)
		if a {
			mstcnt := strings.Split(test.rc, ";")
			mismatch := false
			for _, v := range mstcnt {
				if !strings.Contains(b, v) {
					mismatch = true
				}
			}
			utils.Assert(t, !mismatch, " A : %s, E : %s", b, test.rc)
			clen := 0
			if c != nil {
				clen = len(*c)
			}
			utils.Equals(t, test.rlen, clen)
		}

		a,b,c = ps.HasDeleteAccess(table)
		utils.Equals(t, test.da, a)
		if a {
			mstcnt := strings.Split(test.dc, ";")
			mismatch := false
			for _, v := range mstcnt {
				if !strings.Contains(b, v) {
					mismatch = true
				}
			}
			utils.Assert(t, !mismatch, "A : %s, E : %s", b, test.dc)
			clen := 0
			if c != nil {
				clen = len(*c)
			}
			utils.Equals(t, test.dlen, clen)
		}

		if test.kvp == nil {
			test.kvp = kvp_v_2 //nil value updates permissions is rejected. WA for that
		}

		err := ps.hasAccessUC(table, test.kvp, fvdetails, PERMISSION_C)
		ne := (err == nil)
		utils.Equals(t, test.ca, ne)

		err = ps.hasAccessUC(table, test.kvp, fvdetails, PERMISSION_U)
		ne = (err == nil)
		utils.Equals(t, test.ua, ne)
	}
}
