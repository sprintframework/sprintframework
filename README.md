# sprint framework

![build workflow](https://github.com/sprintframework/sprintframework/actions/workflows/build.yaml/badge.svg)

The goal of this project is to provide framework for building enterprise grade applications by using independent components, instead of monolith cross-dependency architecture. There are certain benefits of this approach:
* independent testing of components (unit testing)
* ability to build application differently for each environment
* ability to build different grade of applications based on the same code base.

This repository represents list of services, clients, servers that are using dependency injection framework `beans`.
The best practices of building enterprise applications are collected in this framework.
Each application should have single application context, that is inherited by single core context, that could derive
multiple servers and one client context.

### Application context

Basic application knowledge, like application name and version. Application flags and stateless application config with the
set of supported commandline commands are here.

### Core context

Base application services, that does not have external interfaces called Core. The initial list of them are ConfigService,
DatabaseService, Storage and etc. Since `sprint` framework uses encrypted storage, during core context creation should be
provided bootstrap token. For this purpose exist `sprint.CoreScanner` interface that provides list of beans for core context, 
which would inherit from application basic context.

### Server context

Server network application practically can have multiple servers that possible to locate in one server context, or if application
needs to separate TLS certificate creation or other server related instances then it is reasonable to use multiple server contexts.
Especially, in case of separation of protected public API servers from unprotected internal API servers.
Each server needs to know listening address, therefore there is a special interface for server scanner: `sprint.ServerScanner` which is 
provides beans and server context would inherit core context.
 
### Client context
Since application could have multiple servers it is still possible to have a single client. 
Each client needs to know the address to connect, therefore there is a special interface for client scanner: `sprint.ClientScanner` which is 
provides beans and client context would inherit application basic context.

```
ApplicationContext -> [Core Context, Client Context...]
Core Context -> [Server Context...]
```

### Quick start

Build from source on golang v1.17
```
go install github.com/codeallergy/go-bindata/go-bindata@v1.0.0
make
```

### Configure (mac):

Install golang 1.17.13
```
wget [go1.17.13.darwin-amd64.pkg](https://go.dev/dl/go1.17.13.darwin-amd64.pkg)
```

Fix Goland IDE
```
sudo nano /usr/local/go/src/runtime/internal/sys/zversion.go
```
Add line:
```
const TheVersion = `go1.17`
```

Setup env
```
export PATH=$PATH:~/go/bin
export GOROOT=/usr/local/go  
export GOPATH=~/go
```

Setup env (optional)
```
export GONOSUMDB=github.com
```

Create dirs
```
mkdir ~/go
mkdir ~/go/bin
mkdir ~/go/src
```

Download and install Protoc 3.20.3 for your OS (optional)
```
open https://github.com/protocolbuffers/protobuf/releases/tag/v3.20.3
wget https://github.com/protocolbuffers/protobuf/releases/download/v3.20.3/protoc-3.20.3-osx-x86_64.zip
unzip protoc-3.20.3-osx-x86_64.zip -d ~/go
```

Verify
```
protoc --version
libprotoc 3.20.3
```

Install protocol buffers src (optional)
```
cd ~/go/src/github.com/protocolbuffers
git clone https://github.com/protocolbuffers/protobuf
cd protobuf 
git checkout v3.20.3
```

Install protoc golang compiler
```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
```

Install gRPC compiler
```
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
```

Install gRPC gateway
```
go get -u github.com/grpc-ecosystem/grpc-gateway/v2@v2.15.2
```

Compile and install protoc-gen-grpc-gateway
```
cd ~/go/src/github.com/grpc-ecosystem
git clone https://github.com/grpc-ecosystem/grpc-gateway
cd grpc-gateway
git checkout v2.15.2
cd ~/go/src/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
go install
```

Compile and install protoc-gen-openapiv2
```
cd ~/go/src/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-openapiv2
go install
```

Install support libs
```
go install github.com/codeallergy/go-bindata/go-bindata@v1.0.0
go install github.com/google/go-licenses@v1.2.1
```

# Verify

```
cd ~/go/bin
ls
go-bindata
go-licenses
protoc-gen-go		
protoc-gen-go-grpc	
protoc-gen-grpc-gateway	
protoc-gen-openapiv2
```
