init:
	make pb-setup
	make grpc-gen
	make docker-setup
	go mod tidy
docker-setup:
	docker network inspect dev >/dev/null 2>&1 || docker network create dev
	docker compose pull 
	docker compose up -d --no-recreate
pb-setup:
	./pb-setup.sh
	make grpc-gen
grpc-gen:
	find ./pb -name "*.proto" -exec protoc --go_out=./pb --go-grpc_out=./pb {} +
run:
	go run .
test:
# go install github.com/vektra/mockery/v2@v2.50.4 beforehand
	mockery --name=\(HttpClient\|Clock\) --output ./mocks
	go test -v .
