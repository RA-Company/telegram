.PHONY: tests tests-integration

cover-tests:
	@. ./.test.env && go clean -testcache && go test -cover -race ./...

tests:
	@. ./.test.env && go clean -testcache && go test -v -race ./...

tests-integration:
	@. ./.test.env && go test -count=1 -tags integration -cover -race ./...
