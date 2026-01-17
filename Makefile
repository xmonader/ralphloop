BINARY_NAME=ralphloop
CMD_PATH=./cmd/ralphloop/main.go

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)

clean:
	rm -f $(BINARY_NAME)

run-plan:
	./$(BINARY_NAME) plan "Test goal"

run-run:
	./$(BINARY_NAME) run

tidy:
	go mod tidy

lint:
	golangci-lint run

all: build
