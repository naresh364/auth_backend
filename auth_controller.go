package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/auth_backend/models"
	"strconv"
	"time"
)

type AuthController struct {
	redis_client *redis.Client
	dbHandler *models.DBRequestHandler
}

const (
	REDIS_USER_ID_KEY = "USER_ID_KEY:"
	REDIS_USER_UUID_KEY = "USER_UUID_KEY:"
	REDIS_USER_EXPIRY = 10*24 //hours
	REDIS_PASSWORD_TOKEN = "REDIS_PASSWORD_TOKEN:"
	REDIS_PASSWORD_EXPIRY = 12 //hours
	)

func (ac *AuthController) authenticate(uuid string) (*models.UserData, error) {
	k := fmt.Sprintf("%s%s",REDIS_USER_UUID_KEY, uuid)
	str, err := ac.redis_client.Get(k).Result();
	if err == redis.Nil {
		return nil, server_errors.USER_NOT_AUTHENTICATED
	} else if err != nil {
		return nil, err
	} else {
		log.Debug("Redis: " +str)
		ud := &models.UserData{}
		if err = json.Unmarshal([]byte(str), ud); err != nil {
			return nil, err
		}
		return ud, err
	}
}

func (ac *AuthController) setPassword(token string, pass string) error {
	//if token == "su" {
	//	if err := ac.dbHandler.SetPassword(1, pass); err != nil {
	//		log.Error(err.Error())
	//		return err
	//	}
	//}
	k := fmt.Sprintf("%s%s",REDIS_PASSWORD_TOKEN, token)
	str, err := ac.redis_client.Get(k).Result();
	if err == redis.Nil {
		return errors.New("invalid token")
	} else if err != nil {
		return err
	} else {
		if id,err := strconv.ParseInt(str, 10, 64); err != nil {
			log.Error(str)
			return errors.New("data not valid")
		} else {
			if err = ac.dbHandler.SetPassword(id, pass); err != nil {
				log.Error(err.Error())
				return err
			} else {
				ac.redis_client.Del(k).Result()
				return nil
			}
		}
	}
}

func (ac *AuthController) logout(data *models.UserData) error {
	k1 := fmt.Sprintf("%s%d",REDIS_USER_ID_KEY, data.Id)
	k2 := fmt.Sprintf("%s%s",REDIS_USER_UUID_KEY, data.Uuid)

	if str, err := ac.redis_client.Del(k1, k2).Result(); err != nil {
		log.Error(err)
		return errors.New("Unable to logout at this time")
	} else {
		log.Info(str)
		return nil
	}
}

func (ac *AuthController) NewUserCreate(u models.BaseModel, creator *models.UserData) (string, error) {
	if uuid, err := uuid.NewV4(); err != nil {
		return "", errors.New("Unable to generate unique identifier at this time. Please try again")
	} else {
		k := fmt.Sprintf("%s%s",REDIS_PASSWORD_TOKEN, uuid.String())
		if str, err := ac.redis_client.Set(k, u.GetId(), REDIS_PASSWORD_EXPIRY*time.Hour).Result(); err != nil {
			return "", err
		} else {
			log.Debug("Redis: " + str)
			return uuid.String(), nil
		}
	}
}

func (ac *AuthController) login(username string, pass string) (string, error) {
	if user, perms, err := ac.dbHandler.Authenticate(username, pass); err != nil {
		return "",err
	} else {
		if uuid, err := uuid.NewV4(); err != nil {
			return "", errors.New("Unable to generate unique identifier at this time. Please try again")
		} else {
			var orgid int64
			if bom, ok := user.(models.BaseOrgModel); !ok {
				return "", err
			} else {
				orgid = bom.GetOrgId()
			}
			ud := &models.UserData{user.GetId(), uuid.String(), orgid, perms}
			var udjson []byte
			if udjson, err = json.Marshal(ud); err != nil {
				return "", err
			}
			k := fmt.Sprintf("%s%d", REDIS_USER_ID_KEY, user.GetId())
			if str, err := ac.redis_client.Set(k, udjson, REDIS_USER_EXPIRY*time.Hour).Result(); err != nil {
				return "", err
			} else {
				log.Debug("Redis: " + str)
				k := fmt.Sprintf("%s%s", REDIS_USER_UUID_KEY, uuid.String())
				if str, err := ac.redis_client.Set(k, udjson, REDIS_USER_EXPIRY*time.Hour).Result(); err != nil {
					return "", err
				} else {
					log.Debug("Redis: " + str)
					return uuid.String(), nil
				}
			}
		}
	}
}
