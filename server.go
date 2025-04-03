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
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kpacha/load-test/db"
	"github.com/kpacha/load-test/requester"
)

type Server interface {
	Run(ctx context.Context, addr string) error
}

func NewServer(engine *gin.Engine, db db.DB, executor Executor, isDevel bool) (*SimpleServer, error) {
	s := &SimpleServer{
		Engine:   engine,
		DB:       db,
		Executor: executor,
		IsDevel:  isDevel,
	}
	tmpl, err := s.getHTMLTemplate()
	if err != nil {
		return nil, err
	}
	s.Engine.SetHTMLTemplate(tmpl)

	s.Engine.POST("/test", s.testHandler)
	s.Engine.GET("/flush-cache", s.flushAllCacheHandler)
	s.Engine.GET("/flush-cache/:id", s.flushCacheHandler)
	s.Engine.GET("/browse/:id", s.browseHandler)
	s.Engine.GET("/download/:id", s.downloadHandler)
	s.Engine.GET("/", s.homeHandler)

	return s, nil
}

var templateFilePattern = "templates/*.html"

type SimpleServer struct {
	Engine   *gin.Engine
	DB       db.DB
	Executor Executor
	IsDevel  bool
}

func (s *SimpleServer) getHTMLTemplate() (*template.Template, error) {
	funcMap := template.FuncMap{
		"formatLatency": formatLatency,
	}
	if s.IsDevel {
		tmpl, err := template.ParseGlob(templateFilePattern)
		return tmpl.Funcs(funcMap), err
	}

	buff := new(bytes.Buffer)

	for _, name := range []string{
		"templates/browse.html",
		"templates/index.html",
		"templates/partials.html",
	} {
		f, err := fs.Open(name)
		if err != nil {
			fmt.Printf("opening file %s: %s\n", name, err.Error())
			return nil, err
		}
		defer f.Close()

		buff.ReadFrom(f)
	}

	return template.New("main").Funcs(funcMap).Parse(buff.String())
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

var (
	mutex = new(sync.Mutex)
	cache = map[string]gin.H{}
)

func (s *SimpleServer) flushAllCacheHandler(c *gin.Context) {
	mutex.Lock()
	defer mutex.Unlock()
	cache = map[string]gin.H{}
	c.Redirect(301, "/")
}

func (s *SimpleServer) flushCacheHandler(c *gin.Context) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(cache, c.Param("id"))
	c.Redirect(301, "/")
}

func (s *SimpleServer) browseHandler(c *gin.Context) {
	id := c.Param("id")
	mutex.Lock()
	defer mutex.Unlock()

	if res, ok := cache[id]; ok {
		keys, _ := s.DB.Keys()
		res["keys"] = keys
		c.HTML(200, "browse", res)
		return
	}

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
	result := gin.H{
		"reports": reports,
		"id":      id,
	}
	cache[id] = result

	keys, err := s.DB.Keys()
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	result["keys"] = keys

	c.HTML(200, "browse", result)
}

func (s *SimpleServer) downloadHandler(c *gin.Context) {
	id := c.Param("id")
	mutex.Lock()
	defer mutex.Unlock()

	if res, ok := cache[id]; ok {
		keys, _ := s.DB.Keys()
		res["keys"] = keys
		c.JSON(200, res)
		return
	}

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
	result := gin.H{
		"reports": reports,
		"id":      id,
	}
	cache[id] = result

	keys, err := s.DB.Keys()
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	result["keys"] = keys

	c.JSON(200, result)
}

func (s *SimpleServer) testHandler(c *gin.Context) {
	req, err := getRequest(c)
	if err != nil {
		fmt.Println(err.Error())
		c.AbortWithError(500, err)
		return
	}
	name := c.PostForm("name")
	log.Println("starting the test", name)

	if _, err := s.Executor.Run(c, Plan{
		Name:     name,
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
