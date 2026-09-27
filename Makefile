.PHONY: tests tests-integration

tests:
	@go test -count=1 -cover -race ./...

tests-integration:
	@. ./.test.env && go test -count=1 -tags integration -cover -race ./...
