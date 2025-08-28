.PHONY: protos

# go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
# go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
# brew install protobuf
protos:
	mkdir -p ./pkg/protos/gen
	cd ./pkg/protos && protoc --go_out=gen --go_opt=paths=source_relative --go-grpc_out=gen --go-grpc_opt=paths=source_relative match.proto