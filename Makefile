
APP_NAME=gpt
DIST_DIR=dist

local:
	go build -o $(DIST_DIR)/$(APP_NAME) main.go

all: linux windows darwin

linux:
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/linux/$(APP_NAME) main.go

windows:
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/win/$(APP_NAME).exe main.go

darwin:
	GOOS=darwin GOARCH=arm64 go build -o $(DIST_DIR)/mac/$(APP_NAME) main.go