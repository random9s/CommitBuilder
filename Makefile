BUILD=$(shell pwd)/main

$(BUILD): clean
	go build -o main cmd/commitbuilder/main.go

docker: clean
	docker build -t cb-build -f Dockerfile .
	docker run -d --restart unless-stopped -p 9000:8080 --name cb cb-build

clean:
	rm -rf log; echo > /dev/null
	rm $(BUILD); echo > /dev/null
