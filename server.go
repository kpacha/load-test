package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kpacha/load-test/db"
	"github.com/kpacha/load-test/requester"
)

type Server struct {
	Engine   *gin.Engine
	DB       db.DB
	Executor Executor
	Addr     string
}

func (s *Server) Run() {
	names := []string{}
	fs, _ := ioutil.ReadDir("templates")
	for _, f := range fs {
		names = append(names, "templates/"+f.Name())
	}
	s.Engine.SetFuncMap(template.FuncMap{
		"formatLatency": formatLatency,
	})
	s.Engine.LoadHTMLFiles(names...)

	s.Engine.POST("/test", s.testHandler)
	s.Engine.GET("/browse/:id", s.browseHandler)
	s.Engine.GET("/", s.homeHandler)
	s.Engine.Run(s.Addr)
}

func (s *Server) homeHandler(c *gin.Context) {
	keys, err := s.DB.Keys()
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.HTML(200, "index.html", gin.H{
		"keys": keys,
	})
}

func (s *Server) browseHandler(c *gin.Context) {
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
	c.HTML(200, "browse.html", gin.H{
		"keys":    keys,
		"reports": reports,
		"id":      id,
	})
}

func (s *Server) testHandler(c *gin.Context) {
	target := c.PostForm("url")
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		fmt.Println(err)
		c.AbortWithError(500, err)
		return
	}
	if err := s.Executor.Run(c, Plan{
		Name:     c.PostForm("name"),
		Min:      getInt(c, "min"),
		Max:      getInt(c, "max"),
		Steps:    getInt(c, "steps"),
		Duration: time.Duration(getInt(c, "duration")) * time.Second,
		Sleep:    time.Duration(getInt(c, "sleep")) * time.Second,
		Request:  req,
	}); err != nil {
		fmt.Println(err)
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(301, "/")
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
