
APP_NAME=gpt
DIST_DIR=dist

local:
	go build -o $(DIST_DIR)/$(APP_NAME) main.go

all: linux windows darwin

linux:
	GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/linux/$(APP_NAME) main.go
	tar -czf $(DIST_DIR)/linux/$(APP_NAME)-linux-amd64.tar.gz -C $(DIST_DIR)/linux $(APP_NAME)

windows:
	GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/win/$(APP_NAME).exe main.go
	zip -r -j $(DIST_DIR)/win/$(APP_NAME)-windows-amd64.zip $(DIST_DIR)/win/$(APP_NAME).exe

darwin:
	GOOS=darwin GOARCH=arm64 go build -o $(DIST_DIR)/mac/$(APP_NAME) main.go
	tar -czf $(DIST_DIR)/mac/$(APP_NAME)-darwin-arm64.tar.gz -C $(DIST_DIR)/mac $(APP_NAME)