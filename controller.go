package main

import (
	"encoding/json"
	"github.com/auth_backend/models"
	"github.com/auth_backend/utils"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type HandleRequest func(s *Server,w http.ResponseWriter, r *http.Request, ps httprouter.Params, ud *models.UserData)

func setPassword(s *Server) httprouter.Handle{
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeResp(w, http.StatusBadRequest, errors.New("password required"), nil)
			return
		}
		var creds map[string]string
		if err = json.Unmarshal(body, &creds); err != nil {
			writeResp(w, http.StatusBadRequest, err, nil)
			return
		}
		var pass,token string
		var ok bool
		if pass, ok = creds["password"]; !ok {
			writeResp(w, http.StatusBadRequest, errors.New("password required"),
				map[string]string{"password": "required"})
			return
		}
		if token, ok = creds["token"]; !ok {
			writeResp(w, http.StatusBadRequest, errors.New("token required"),
				map[string]string{"token": "required"})
			return
		}

		if err := s.ac.setPassword(token, pass); err != nil {
			writeResp(w, http.StatusBadRequest, err, nil)
		} else {
			writeResp(w, http.StatusOK, nil, map[string]string{"status": "success"})
		}
	}
}


func handleLogin(s *Server) httprouter.Handle{
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeResp(w, http.StatusBadRequest, errors.New("username/password required"), nil)
			return
		}
		var creds map[string]string
		if err = json.Unmarshal(body, &creds); err != nil {
			writeResp(w, http.StatusBadRequest, err, nil)
			return
		}
		var pass, un string
		var ok bool
		if pass, ok = creds["password"]; !ok {
			writeResp(w, http.StatusBadRequest, errors.New("password required"),
				map[string]string{"password": "required"})
		}
		if un, ok = creds["username"]; !ok {
			writeResp(w, http.StatusBadRequest, errors.New("username required"),
				map[string]string{"username": "required"})
		}
		if uuid, err := s.ac.login(un, pass); err != nil {
			switch err {
			case models.INVALID_CREDENTIALS : writeResp(w, http.StatusUnauthorized, err, nil)
			default:
				writeResp(w, http.StatusBadRequest, err, nil)
			}
		} else {
			writeResp(w, http.StatusOK, nil, map[string]string{"token": uuid})
		}
	}
}

func handleLogout(s *Server,w http.ResponseWriter, r *http.Request, ps httprouter.Params, ud *models.UserData) {
	if err := s.ac.logout(ud); err != nil {
		writeResp(w, http.StatusInternalServerError, models.SERVER_ERROR, nil)
		return
	}
	writeResp(w, http.StatusOK, nil, map[string]string{"status": "ok"})
}

func handleRead(s *Server,w http.ResponseWriter, r *http.Request, ps httprouter.Params, ud *models.UserData) {
	table := ps.ByName("table")
	w.Header().Set("Content-Type", "application/json")
	qValues := r.URL.Query()

	sortBy := qValues.Get("sortBy")
	var from, limit int
	var desc bool
	var err error

	if f:= qValues.Get("from"); f != "" {
		if from, err = strconv.Atoi(f); err != nil {
			writeResp(w, http.StatusBadRequest, errors.New("from should be a number"), nil)
			return
		}

	}
	if f:= qValues.Get("limit"); f != "" {
		if limit, err = strconv.Atoi(f); err != nil {
			writeResp(w, http.StatusBadRequest, errors.New("limit should be a number"), nil)
			return
		}

	}
	if f:= qValues.Get("desc"); f == "true" {
		desc = true
	}

	body, err := ioutil.ReadAll(r.Body)
	if (err != nil) {
		body = []byte("{}")
	}
	if res, e := s.DBh.ReadObjJson(table, body, from, limit, desc, sortBy, ud); e != nil {
		writeResp(w, http.StatusBadRequest, e, nil)
	} else {
		writeResp(w, http.StatusOK, nil, res)
	}
}

func handleDelete(s *Server,w http.ResponseWriter, r *http.Request, ps httprouter.Params, ud *models.UserData) {
	table := ps.ByName("table")
	id := ps.ByName("id")
	w.Header().Set("Content-Type", "application/json")
	vid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		writeResp(w, http.StatusBadRequest, err, nil)
		return
	}
	if err := s.DBh.DeleteObj(table, vid, ud); err == nil {
		writeResp(w, http.StatusOK, err, "")
	} else {
		writeResp(w, http.StatusBadRequest, err, nil)
		return
	}
}


func handleUpdate(s *Server,w http.ResponseWriter, r *http.Request, ps httprouter.Params, ud *models.UserData) {
	table := ps.ByName("table")
	id := ps.ByName("id")
	w.Header().Set("Content-Type", "application/json")
	body, err := ioutil.ReadAll(r.Body)
	if (err != nil) {
		writeResp(w, http.StatusBadRequest, err, nil)
		return
	}
	vid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		writeResp(w, http.StatusBadRequest, err, nil)
		return
	}
	if res, err := s.DBh.UpdateObj(table, vid, body, ud); err == nil {
		writeResp(w, http.StatusOK, nil, res)
	} else {
		writeResp(w, http.StatusBadRequest, err, nil)
		return
	}
}


func handleCreate(s *Server,w http.ResponseWriter, r *http.Request, ps httprouter.Params, ud *models.UserData) {
	table := ps.ByName("table")
	w.Header().Set("Content-Type", "application/json")
	body, err := ioutil.ReadAll(r.Body)
	if (err != nil) {
		writeResp(w, http.StatusBadRequest, err, nil)
		return
	}
	if dbModel, err := s.DBh.SaveObj(body, table, ud); err == nil {
		if s.DBh.IsAuthTable(table) {
			//this needs special care to create a new password,
			if tok, err := s.ac.NewUserCreate(dbModel, ud); err != nil {
				logrus.Error("Unable to generate token for this user")
			} else {
				//TODO: should mail this to user
				logrus.Debug("token="+tok)
			}
		}
		writeResp(w, http.StatusOK, nil, dbModel)
	} else {
		writeResp(w, http.StatusBadRequest, err, nil)
		return
	}
}

func BasicAuth(request HandleRequest, s *Server) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Get the Basic Authentication credentials
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if len(auth) > len (prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
			token := auth[len(prefix):]
			if ud, err := s.ac.authenticate(token); err == nil {
				//when access is checked
				request(s, w, r, ps, ud)
				return
			} else if err != models.USER_NOT_AUTHENTICATED {
				writeResp(w, http.StatusInternalServerError, models.SERVER_ERROR, nil)
				return
			}
		}

		// Request Basic Authentication otherwise
		w.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
}

func writeResp(w http.ResponseWriter, status int, err error, data interface{}) {
	w.WriteHeader(status)
	var msg string
	if err != nil {
		logrus.Error(err.Error())
		msg = err.Error()
	} else {
		msg = "success"
	}
	var d []byte
	if data != nil {
		var e error
		if d , e = json.Marshal(data); e != nil {
			resp, _ := utils.JsonResponse(status, "Operations completed, error while creating response", "")
			logrus.Error(e.Error())
			w.Write(resp)
		} else {
			v := string(d)
			if v == "null" || v == "" {
				v = "{}"//send an empty json
			}
			resp, _ := utils.JsonResponse(status, msg, v)
			w.Write(resp)
		}
	} else {
		resp, _ := utils.JsonResponse(status, msg, "{}")
		w.Write(resp)
	}
}



