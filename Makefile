all: build release

build:
	go get -u ./...
	go build

release:
	./create_releases.sh

clean:
	rm -rf ./releases
