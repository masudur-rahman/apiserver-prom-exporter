package api

import (
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/macaron.v1"
	"xorm.io/xorm"
	logx "xorm.io/xorm/log"

	_ "github.com/lib/pq"
)

var engine *xorm.Engine

type Worker struct {
	Username string `json:"username" xorm:"pk not null unique"`

	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`

	City     string `json:"city"`
	Division string `json:"division"`

	Position string `json:"position"`
	Salary   int64  `json:"salary"`

	CreatedAt time.Time `xorm:"created"`
	UpdatedAt time.Time `xorm:"updated"`
	DeletedAt time.Time `xorm:"deleted"`
	Version   int       `xorm:"version"`
}

// List of workers and authenticated users
var Workers []Worker
var authUser = make(map[string]string)

var srvr http.Server
var byPass bool = true
var stopTime int16

func StartXormEngine() {
	var err error
	connStr := "user=postgres password=postgres host=127.0.0.1 port=5432 dbname=apiserver sslmode=disable"

	engine, err = xorm.NewEngine("postgres", connStr)
	if err != nil {
		log.Fatalln(err)
	}

	logFile, err := os.Create("apiserver-prom-exporter.log")
	if err != nil {
		log.Println(err)
	}
	logger := logx.NewSimpleLogger(logFile)
	logger.ShowSQL(true)
	engine.SetLogger(logger)

	if engine.TZLocation, err = time.LoadLocation("Asia/Dhaka"); err != nil {
		log.Println(err)
	}
}

// Handler Functions....

func Welcome(ctx *macaron.Context) {
	start := time.Now()
	ctx.JSON(http.StatusOK, "Congratulations...! Your API Server is up and running... :) ")

	duration := time.Since(start)
	promHttpRequestTotal.With(prometheus.Labels{"url": "/", "method": "GET", "code": strconv.Itoa(http.StatusOK)}).Inc()
	promHttpRequestDurationSeconds.With(prometheus.Labels{"url": "/", "method": "GET"}).Observe(duration.Seconds())
}

func WelcomeToAppsCode(ctx *macaron.Context) {
	start := time.Now()
	ctx.JSON(http.StatusOK, "Welcome to AppsCode Ltd.. Available Links are : `/appscode/workers`, `/appscode/workers/{username}`")

	duration := time.Since(start)
	promHttpRequestTotal.With(prometheus.Labels{"url": "/appscode/", "method": "GET", "code": strconv.Itoa(http.StatusOK)}).Inc()
	promHttpRequestDurationSeconds.With(prometheus.Labels{"url": "/appscode/", "method": "GET"}).Observe(duration.Seconds())
}

func ShowAllWorkers(ctx *macaron.Context) {
	start := time.Now()
	var workers []Worker
	if err := engine.Find(&workers); err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
	}

	ctx.JSON(http.StatusOK, workers)

	duration := time.Since(start)
	promHttpRequestTotal.With(prometheus.Labels{"url": "/appscode/workers/", "method": "GET", "code": strconv.Itoa(http.StatusOK)}).Inc()
	promHttpRequestDurationSeconds.With(prometheus.Labels{"url": "/appscode/workers/", "method": "GET"}).Observe(duration.Seconds())
}

func ShowSingleWorker(ctx *macaron.Context) {
	start := time.Now()
	worker := new(Worker)
	worker.Username = ctx.Params("username")
	exist, err := engine.Get(worker)
	if err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
	} else if !exist {
		ctx.Error(http.StatusNotFound, "404 - Content Not Found")
		return
	}

	ctx.JSON(http.StatusOK, worker)

	duration := time.Since(start)
	promHttpRequestTotal.With(prometheus.Labels{"url": fmt.Sprintf("/appscode/workers/%s/", worker.Username), "method": "GET", "code": strconv.Itoa(http.StatusOK)}).Inc()
	promHttpRequestDurationSeconds.With(prometheus.Labels{"url": fmt.Sprintf("/appscode/workers/%s/", worker.Username), "method": "GET"}).Observe(duration.Seconds())
}

func AddNewWorker(ctx *macaron.Context, worker Worker) {
	start := time.Now()

	if worker.Username == "" {
		ctx.Error(http.StatusNotAcceptable, "Username must be provided")
		return
	}

	newWorker := new(Worker)
	newWorker.Username = worker.Username
	if exist, err := engine.Get(newWorker); err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	} else if exist {
		ctx.Error(http.StatusConflict)
	}

	// Check if it exists in deleted accounts
	newWorker = new(Worker)
	newWorker.Username = worker.Username
	if exist, err := engine.Unscoped().Get(newWorker); err != nil {
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	} else if exist {
		ctx.Error(http.StatusConflict)
		return
	}

	session := engine.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}
	if _, err := session.Insert(&worker); err != nil {
		if err = session.Rollback(); err != nil {
			log.Println(err)
			ctx.Error(http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := session.Commit(); err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.JSON(http.StatusCreated, worker)

	duration := time.Since(start)
	promHttpRequestTotal.With(prometheus.Labels{"url": "/appscode/workers/", "method": "POST", "code": strconv.Itoa(http.StatusOK)}).Inc()
	promHttpRequestDurationSeconds.With(prometheus.Labels{"url": "/appscode/workers/", "method": "POST"}).Observe(duration.Seconds())
}

func UpdateWorkerProfile(ctx *macaron.Context, newWorker Worker) {
	start := time.Now()
	worker := new(Worker)
	worker.Username = ctx.Params("username")
	exist, err := engine.Get(worker)
	if err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	} else if !exist {
		ctx.Error(http.StatusNotFound)
		return
	}

	if newWorker.Username != worker.Username {
		ctx.Error(http.StatusMethodNotAllowed, "405 - Username can't be changed")
		return
	}

	// Updated information assignment
	worker.FirstName = newWorker.FirstName
	worker.LastName = newWorker.LastName
	worker.City = newWorker.City
	worker.Division = newWorker.Division
	worker.Salary = newWorker.Salary

	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()

	session := engine.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := session.ID(worker.Username).Update(worker); err != nil {
		log.Println(err)
		if err = session.Rollback(); err != nil {
			log.Println(err)
			ctx.Error(http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := session.Commit(); err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, worker)

	duration := time.Since(start)
	promHttpRequestTotal.With(prometheus.Labels{"url": fmt.Sprintf("/appscode/workers/%s/", worker.Username), "method": "PUT", "code": strconv.Itoa(http.StatusOK)}).Inc()
	promHttpRequestDurationSeconds.With(prometheus.Labels{"url": fmt.Sprintf("/appscode/workers/%s/", worker.Username), "method": "PUT"}).Observe(duration.Seconds())
}

func DeleteWorker(ctx *macaron.Context) {
	start := time.Now()
	worker := new(Worker)
	worker.Username = ctx.Params("username")
	exist, err := engine.Get(worker)
	if err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	} else if !exist {
		ctx.Error(http.StatusNotFound)
		return
	}

	session := engine.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := session.ID(worker.Username).Delete(worker); err != nil {
		log.Println(err)
		if err = session.Rollback(); err != nil {
			log.Println(err)
			ctx.Error(http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := session.Commit(); err != nil {
		log.Println(err)
		ctx.Error(http.StatusInternalServerError, err.Error())
		return
	}

	ctx.Status(http.StatusOK)

	duration := time.Since(start)
	promHttpRequestTotal.With(prometheus.Labels{"url": fmt.Sprintf("/appscode/workers/%s/", worker.Username), "method": "DELETE", "code": strconv.Itoa(http.StatusOK)}).Inc()
	promHttpRequestDurationSeconds.With(prometheus.Labels{"url": fmt.Sprintf("/appscode/workers/%s/", worker.Username), "method": "DELETE"}).Observe(duration.Seconds())
}

// Creating initial worker profiles
func CreateInitialWorkerProfile() {
	Workers = make([]Worker, 0)
	worker := Worker{
		Username:  "masud",
		FirstName: "Masudur",
		LastName:  "Rahman",
		City:      "Madaripur",
		Division:  "Dhaka",
		Position:  "Software Engineer",
		Salary:    55,
	}
	Workers = append(Workers, worker)

	worker = Worker{
		Username:  "fahim",
		FirstName: "Fahim",
		LastName:  "Abrar",
		City:      "Chittagong",
		Division:  "Chittagong",
		Position:  "Software Engineer",
		Salary:    55,
	}
	Workers = append(Workers, worker)

	worker = Worker{
		Username:  "tahsin",
		FirstName: "Tahsin",
		LastName:  "Rahman",
		City:      "Chittagong",
		Division:  "Chittagong",
		Position:  "Software Engineer",
		Salary:    55,
	}
	Workers = append(Workers, worker)

	worker = Worker{
		Username:  "jenny",
		FirstName: "Jannatul",
		LastName:  "Ferdows",
		City:      "Chittagong",
		Division:  "Chittagong",
		Position:  "Software Engineer",
		Salary:    55,
	}
	Workers = append(Workers, worker)

	if exist, _ := engine.IsTableExist(new(Worker)); !exist {
		if err := engine.CreateTables(new(Worker)); err != nil {
			log.Fatalln(err)
		}
	}

	session := engine.NewSession()
	defer session.Close()

	if err := session.Begin(); err != nil {
		log.Fatalln(err)
	}

	for _, user := range Workers {
		if _, err := session.Insert(&user); err != nil {
			if err = session.Rollback(); err != nil {
				log.Fatalln(err)
			}
		}
	}
	if err := session.Commit(); err != nil {
		log.Fatalln(err)
	}

	authUser["masud"] = "pass"
	authUser["admin"] = "admin"

}

func basicAuth(ctx *macaron.Context) (bool, error) {
	if byPass {
		return true, nil
	}
	authHeader := ctx.Req.Header.Get("Authorization")
	if authHeader == "" {
		return false, errors.New("Authorization Needed...!")
	}

	authInfo := strings.SplitN(authHeader, " ", 2)

	userInfo, err := base64.StdEncoding.DecodeString(authInfo[1])

	if err != nil {
		return false, errors.New("Error while decoding...!")
	}
	userPass := strings.SplitN(string(userInfo), ":", 2)

	if len(userPass) != 2 {
		return false, errors.New("Authorization failed...!")
	}

	if pass, exist := authUser[userPass[0]]; exist {
		if pass != userPass[1] {
			return false, errors.New("Unauthorized User")
		} else {
			return true, nil
		}
	} else {
		return false, errors.New("Authorization failed...!")
	}
}

func reqAuthentication() macaron.Handler {
	return func(ctx *macaron.Context) {
		if authorized, err := basicAuth(ctx); !authorized {
			ctx.Error(http.StatusUnauthorized, err.Error())
		}
	}
}

func AssignValues(port string, bypass bool, stop int16) {
	srvr.Addr = ":" + port
	byPass = bypass
	stopTime = stop
}

func StartTheApp() {
	m := macaron.Classic()
	m.Use(macaron.Renderer())

	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	srvr.WriteTimeout = time.Second * 15
	srvr.ReadTimeout = time.Second * 15
	srvr.IdleTimeout = time.Second * 60

	srvr.Handler = m

	StartXormEngine()
	CreateInitialWorkerProfile()

	m.Get("/", Welcome)
	m.Group("appscode", func() {
		m.Get("/", WelcomeToAppsCode)
		m.Group("/workers", func() {
			m.Get("/", ShowAllWorkers)
			m.Get("/:username", ShowSingleWorker)
			m.Post("/", AddNewWorker)
			m.Put("/:username", UpdateWorkerProfile)
			m.Delete("/:username", DeleteWorker)
		}, reqAuthentication())
	})

	m.Get("/metrics", promhttp.HandlerFor(prom, promhttp.HandlerOpts{}))

	log.Printf("Starting the server at 127.0.0.1%s\n", srvr.Addr)

	go func() {
		if err := srvr.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()
	// Channel to interrupt the server from keyboard
	channel := make(chan os.Signal, 1)

	signal.Notify(channel, os.Interrupt)
	<-channel

	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()

	//  Shutting down the server
	log.Println("Shutting down the server...!")

	time.Sleep(time.Second * time.Duration(stopTime))

	if err := srvr.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}

	log.Println("The server has been shut down...!")
	if err := engine.Close(); err != nil {
		log.Fatalln(err)
	}

	os.Exit(0)
}
