package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kpacha/load-test/db"
	"github.com/kpacha/load-test/requester"
)

type Server interface {
	Run(ctx context.Context, addr string) error
}

func NewServer(engine *gin.Engine, db db.DB, executor Executor) (*SimpleServer, error) {
	s := &SimpleServer{
		Engine:   engine,
		DB:       db,
		Executor: executor,
	}
	s.Engine.SetFuncMap(template.FuncMap{
		"formatLatency": formatLatency,
	})
	s.Engine.LoadHTMLGlob(templateFilePattern)

	s.Engine.POST("/test", s.testHandler)
	s.Engine.GET("/browse/:id", s.browseHandler)
	s.Engine.GET("/", s.homeHandler)

	return s, nil
}

var templateFilePattern = "templates/*.html"

type SimpleServer struct {
	Engine   *gin.Engine
	DB       db.DB
	Executor Executor
}

func (s *SimpleServer) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.Engine,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown Server ...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server Shutdown:", err)
		return err
	}
	log.Println("Server exiting")
	return nil
}

func (s *SimpleServer) homeHandler(c *gin.Context) {
	keys, err := s.DB.Keys()
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.HTML(200, "index", gin.H{
		"keys": keys,
	})
}

func (s *SimpleServer) browseHandler(c *gin.Context) {
	id := c.Param("id")
	r, err := s.DB.Get(id)
	switch err {
	case db.ErrNotFound:
		c.AbortWithStatus(http.StatusNotFound)
		return
	case nil:
	default:
		c.AbortWithError(500, err)
		return
	}

	reports := []requester.Report{}
	if err := json.NewDecoder(r).Decode(&reports); err != nil {
		c.AbortWithError(500, err)
		return
	}
	keys, err := s.DB.Keys()
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.HTML(200, "browse", gin.H{
		"keys":    keys,
		"reports": reports,
		"id":      id,
	})
}

func (s *SimpleServer) testHandler(c *gin.Context) {
	req, err := getRequest(c)
	if err != nil {
		fmt.Println(err.Error())
		c.AbortWithError(500, err)
		return
	}

	if _, err := s.Executor.Run(c, Plan{
		Name:     c.PostForm("name"),
		Min:      getInt(c, "min"),
		Max:      getInt(c, "max"),
		Steps:    getInt(c, "steps"),
		Duration: time.Duration(getInt(c, "duration")) * time.Second,
		Sleep:    time.Duration(getInt(c, "sleep")) * time.Second,
		Request:  req,
	}); err != nil {
		fmt.Println(err.Error())
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(301, "/")
}

func getRequest(c *gin.Context) (*http.Request, error) {
	target := c.PostForm("url")
	method := c.PostForm("req_method")
	if method == "" {
		method = "GET"
	}
	body := ioutil.NopCloser(bytes.NewBufferString(c.PostForm("body")))
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		fmt.Println("building request:", err.Error())
		return nil, err
	}

	req.Header = parseHeaders(c.PostForm("headers"))

	return req, nil
}

func parseHeaders(headersTxt string) http.Header {
	res := http.Header{}
	headersTxt = strings.Replace(strings.Trim(headersTxt, " "), "\r", "", -1)
	if headersTxt == "" {
		return res
	}
	lines := strings.Split(headersTxt, "\n")
	for i := range lines {
		if lines[i] == "" {
			continue
		}
		index := strings.Index(lines[i], ":")
		if index < 1 {
			continue
		}
		res.Set(strings.Trim(lines[i][:index], " "), strings.Trim(lines[i][index+1:], " "))
	}
	return res
}

func getInt(c *gin.Context, key string) int {
	i, err := strconv.Atoi(c.PostForm(key))
	if err != nil {
		return -1
	}
	return i
}

func formatLatency(l float64) string {
	return time.Duration(int64(l * float64(time.Second))).String()
}
