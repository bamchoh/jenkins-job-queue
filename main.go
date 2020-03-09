package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	bolt "go.etcd.io/bbolt"

	"github.com/bamchoh/handler"

	"github.com/marcsauter/single"
)

var db *bolt.DB
var rootName []byte

func initDB(dbfile string) (err error) {
	if len(dbfile) == 0 {
		return fmt.Errorf("dbfile is empty")
	}

	if !filepath.IsAbs(dbfile) {
		execpath, err := os.Executable()
		if err != nil {
			return err
		}
		dbfile, err = filepath.Abs(filepath.Join(filepath.Dir(execpath), dbfile))
		if err != nil {
			return err
		}
	}
	db, err = bolt.Open(dbfile, 0666, nil)
	if err != nil {
		err = fmt.Errorf("open DB error: %s", err)
		fmt.Println(err)
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(rootName)
		if err != nil {
			err = fmt.Errorf("create bucket: %s", err)
			fmt.Println(err)
			return err
		}
		return err
	})
	return err
}

func main() {
	s := single.New("jenkins-job-scheduler")
	if err := s.CheckLock(); err != nil && err == single.ErrAlreadyRunning {
		log.Fatal("another instance of the app is already running, exiting")
	} else if err != nil {
		log.Fatalf("failed to acquire exclusive app lock: %v", err)
	}
	defer s.TryUnlock()

	dbname := flag.String("db", "", "database filename")
	addr := flag.String("addr", ":20000", "server address")
	bucket := flag.String("bucket", "MyBucket", "root bucket name")

	flag.Parse()

	rootName = []byte(*bucket)

	var err error
	err = initDB(*dbname)
	if err != nil {
		log.Fatalf("initDB error: %s", err)
	}
	defer db.Close()

	jh := handler.JobHandler{
		Db:       db,
		RootName: rootName,
		Ch:       make(chan int, 1),
		UpdateCh: make(chan int, 1),
	}

	go jh.Execute()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.GET("/", jh.MainPage())
	e.GET("/ws", jh.WebSocket())
	e.POST("/job", jh.Update())
	e.Logger.Fatal(e.Start(*addr))
}
