package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/auth_backend/models"
	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"syscall"
)

func isProduction(env string) bool {
	return env == "prod"
}

func initLogging(env string, file *os.File) {

	if isProduction(env) {
		log.SetFormatter(&log.JSONFormatter{})
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetFormatter(&log.TextFormatter{})
		log.SetReportCaller(true)
		log.SetLevel(log.DebugLevel)
	}
	if file == nil {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(file)
	}
}

func main() {
	commandParams := flag.String("config", "", "Config file (Json format)")
	flag.Parse()
	if *commandParams == "" {
		log.Fatal("Need config file to start server")
	}
	//load config params
	viper.SetConfigFile(*commandParams)
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	env := viper.GetString("env")
	logf := viper.GetString("log")

	//setup logging
	var file *os.File
	if logf != "" {
		file, err := os.OpenFile(logf, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf(reflect.TypeOf(file).String())
		defer file.Close()
	}
	initLogging(env, file)
	dbuser := viper.GetString("db.user")
	dbpass := viper.GetString("db.pass")
	dbhost := viper.GetString("db.host")
	dbdb := viper.GetString("db.db")
	dbconstr := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",dbuser, dbpass, dbhost, dbdb)
	db, err := sql.Open("mysql", dbconstr)
	if err != nil {
		log.Error(err.Error())
		log.Fatal("Unable to open db connection : please add user:pass, host:port and db params")
	}
	err = db.Ping()
	if err != nil {
		log.Error(err.Error())
		log.Fatal("Unable to open db connection : please add user:pass, host:port and db params")
	}

	dbHandler := models.InitDB(db, viper.GetString("org_col"),
		viper.GetString("owner_col"), viper.GetInt("sudo"), viper.GetInt("sudo_org"));

	//redis
	redis_client := redis.NewClient(&redis.Options{
		Addr : viper.GetString("redis.addr"),
		Password: viper.GetString("redis.password"),
		DB : viper.GetInt("redis.db"),
	})

	if _, err := redis_client.Ping().Result(); err != nil {
		log.Fatal(fmt.Errorf("redis config error: %s \n", err))
	}

	port := viper.GetInt64("port")
	if port <= 1000 {
		port = 3000
	}
	router := httprouter.New()
	ac := &AuthController{dbHandler:dbHandler, redis_client:redis_client};
	routing(&Server{dbHandler, redis_client, ac}, router, port)

	// Respect OS stop signals.
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a termination signal.
	<-c
	defer db.Close()
	defer redis_client.Close()
}

type Server struct {
	DBh *models.DBRequestHandler
	RedisC *redis.Client
	ac *AuthController
}

func routing(s *Server, router *httprouter.Router, port int64 ) {
	router.POST("/api/v1/auth/setpassword", setPassword(s));
	//TODO
	//router.POST("/api/v1/auth/signup", setPassword(s));
	router.POST("/api/v1/auth/login", handleLogin(s));
	router.POST("/api/v1/auth/logout", BasicAuth(handleLogout, s));
	router.POST("/api/v1/data/:table/add", BasicAuth(handleCreate, s));
	router.POST("/api/v1/data/:table/update/:id", BasicAuth(handleUpdate, s));
	router.GET("/api/v1/data/:table/list", BasicAuth(handleRead, s));
	router.DELETE("/api/v1/data/:table/delete/:id", BasicAuth(handleDelete, s));
	log.Fatal(http.ListenAndServe(":"+strconv.FormatInt(port,10), router))
}
