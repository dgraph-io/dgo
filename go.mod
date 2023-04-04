module github.com/dgraph-io/dgo/v230

go 1.19

require (
	github.com/gogo/protobuf v1.3.2
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
	google.golang.org/grpc v1.27.0
)

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/golang/protobuf v1.4.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

retract v230.0.0 // needed to merge #158 for v230.0.0 release
