module github.com/sprintframework/sprintframework

go 1.17

//replace github.com/codeallergy/glue => ../../codeallergy/glue

//replace github.com/sprintframework/sprint => ../../sprintframework/sprint

require (
	github.com/codeallergy/base62 v1.1.0
	github.com/codeallergy/glue v1.1.4
	github.com/codeallergy/properties v1.1.0
	github.com/codeallergy/uuid v1.1.0
	github.com/fsnotify/fsnotify v1.6.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.15.2
	github.com/hashicorp/go-hclog v1.5.0
	github.com/keyvalstore/badgerstore v1.3.1
	github.com/keyvalstore/store v1.3.1
	github.com/mailgun/mailgun-go/v4 v4.8.1
	github.com/pkg/errors v0.9.1
	github.com/sprintframework/cert v1.0.0
	github.com/sprintframework/dns v1.0.0 // indirect
	github.com/sprintframework/nat v1.0.0
	github.com/sprintframework/sprint v1.4.1
	github.com/stretchr/testify v1.8.4
	go.uber.org/atomic v1.10.0
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.10.0
	golang.org/x/net v0.11.0 // indirect
	golang.org/x/sync v0.3.0
	google.golang.org/grpc v1.53.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/keyvalstore/bboltstore v1.3.1
	github.com/keyvalstore/boltstore v1.3.1
	github.com/keyvalstore/cachestore v1.3.1
	github.com/sprintframework/sprintpb v1.3.0
)

require (
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/badger/v3 v3.2103.5 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.13.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v23.1.21+incompatible // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sprintframework/certpb v1.0.0 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/oauth2 v0.9.0 // indirect
	golang.org/x/sys v0.9.0 // indirect
	golang.org/x/term v0.9.0 // indirect
	golang.org/x/text v0.10.0 // indirect
	google.golang.org/genproto v0.0.0-20230303212802-e74f57abe488 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
