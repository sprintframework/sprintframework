@echo off
setlocal

protoc.exe proto\*.proto -I proto -I third_party -I %GOPATH%\src\github.com\protocolbuffers\protobuf\src --go_out=plugins=grpc:. --grpc-gateway_out=logtostderr=true,allow_delete_body=true:. --swagger_out=logtostderr=true,allow_delete_body=true:.
move *.swagger.json resources\swagger\

go-bindata -pkg resources -o pkg\resources\bindata.go -nocompress -nomemcopy -fs -prefix "resources" resources\...
go-bindata -pkg assets -o pkg\assets\bindata.go -nocompress -nomemcopy -fs -prefix "assets" assets\...

for /f %%i in ('git describe --tags --always --dirty') do set VER=%%i
for /f "tokens=2 delims= " %%i in ('date /t') do set DATE=%%i

echo %VER%
echo %DATE%

rsrc -manifest sprint.manifest -o rsrc.syso -arch="amd64"
go test -cover ./...
go build -ldflags "-X main.Version=%VER% -X main.Build=%DATE%"

