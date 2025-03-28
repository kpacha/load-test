module github.com/kpacha/load-test

go 1.23.0

toolchain go1.23.6

require (
	github.com/gin-gonic/gin v1.1.5-0.20170702092826-d459835d2b07
	github.com/rakyll/hey v0.1.1-0.20180227211324-6369dbfd2e54
	github.com/rakyll/statik v0.1.7
)

require (
	github.com/gin-contrib/sse v0.0.0-20170109093832-22d885f9ecc7 // indirect
	github.com/golang/protobuf v1.0.0 // indirect
	github.com/mattn/go-isatty v0.0.3 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/ugorji/go v0.0.0-20180112141927-9831f2c3ac10 // indirect
	golang.org/x/net v0.0.0-20180218175443-cbe0f9307d01 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.0.0-20180224232135-f6cff0780e54 // indirect
	golang.org/x/text v0.3.0 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/go-playground/validator.v8 v8.18.2 // indirect
	gopkg.in/yaml.v2 v2.1.1 // indirect
)

replace github.com/rakyll/hey => github.com/kpacha/hey v0.1.1-0.20180227211324-6369dbfd2e54
