.PHONY : proto-vivaldi

grpc_dir := ./communication/rpc/grpc

module := "github.com/sebastianopriscan/GNCFD"

vivaldi_proto_loc := $(grpc_dir)/vivaldi/proto

proto-vivaldi : $(wildcard $(vivaldi_proto_loc)/*.proto)
	protoc --proto_path=$(vivaldi_proto_loc) --go_out=. --go_opt=module=$(module) --go-grpc_out=. --go-grpc_opt=module=$(module) $(wildcard $(vivaldi_proto_loc)/*.proto)
