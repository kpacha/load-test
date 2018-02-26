# load-test
HTTP Load testing web tool powered by `hey`

### New load test

![new load test](docs/new.png)

### Inspect test report

![test report grafs](docs/snapshot_1.png)

![test report data](docs/snapshot_2.png)

## Install

1. Clone the repo

```
$ go get github.com/kpacha/load-test
```

2. Install dependencies and build

```
$ cd $GOPATH/src/github.com/kpacha/load-test
$ make prepare all
```

And the `load-test` binary should be in your `$GOPATH/bin` folder. Make sure it's also in your `$PATH`!

## Run

Check the help for details on the accepted flags...

```
$ load-test -h
Usage of ./load-test:
  -f string
    	path to use as store (default ".")
  -p int
    	port to expose the html ui (default 7879)
```

And then just run it!

```
$ load-test
[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export GIN_MODE=release
 - using code:	gin.SetMode(gin.ReleaseMode)

[GIN-debug] POST   /test                     --> main.(*Server).(main.testHandler)-fm (1 handlers)
[GIN-debug] GET    /browse/:id               --> main.(*Server).(main.browseHandler)-fm (1 handlers)
[GIN-debug] GET    /                         --> main.(*Server).(main.homeHandler)-fm (1 handlers)
[GIN-debug] Listening and serving HTTP on :7879
```

And the web will be running at http://localhost:7879/

## TODO

- Expose the data collected per request in the test browser
- Search for ulrs and tests names
- Support curstom request headers and body
- Support complex use cases