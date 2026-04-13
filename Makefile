.PHONY: gen

SWAGGER_DIR_GEN = ./internal/api/swagger/generated

gen:
	mkdir -p  ${SWAGGER_DIR_GEN} || true
	find . -name '*mocks.go' -exec rm -rf {} \; # удалить ранее сгенерированные моки
	find ${SWAGGER_DIR_GEN} ! -name 'middleware.go' -type f -exec rm -fr {} +;
	go generate -x ./...
	swagger generate server -t ${SWAGGER_DIR_GEN} -f ./internal/api/swagger/swagger.yaml --exclude-main
	go run ./cmd/rmfn/... ${SWAGGER_DIR_GEN}/restapi/configure_signature_service.go setupMiddlewares setupGlobalMiddleware

run:
	go run ./cmd/server/... 8080