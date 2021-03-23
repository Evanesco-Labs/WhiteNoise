PB = $(wildcard *.proto)
GO = $(PB:.proto=.pb.go)

all: $(GO)

%.pb.go: %.proto
		protoc --proto_path=$(PWD):$(PWD)/../.. --go-grpc_out=. $<
		protoc --proto_path=$(PWD):$(PWD)/../.. --go_out=. $<

clean:
		rm -f *.pb.go