BUILD=$(shell pwd)/main
SHA?=shahash

$(BUILD): clean
	go build -o main cmd/commitbuilder/main.go

docker:
	docker build -t ${SHA}-cb-build -f Dockerfile .
	docker run -d --restart unless-stopped -p 9000:8080 --name cb-${SHA} ${SHA}-cb-build

clean:
	rm -rf log; echo > /dev/null
	rm $(BUILD); echo > /dev/null
