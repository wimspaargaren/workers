# Runs lint
lint:
	@echo Linting...
	@golangci-lint  -v --concurrency=3 --config=.golangci.yml --issues-exit-code=1 run \
	--out-format=colored-line-number 

$(GOBIN)/gofumpt:
	@GO111MODULE=on go get mvdan.cc/gofumpt
	@go mod tidy

gofumpt: | $(GOBIN)/gofumpt
	@gofumpt -w .

gci:
	@gci write --section Standard --section Default --section "Prefix(github.com/wimspaargaren/workers)" $(shell ls  -d $(PWD)/*)
	
test:
	@mkdir -p reports
	@go test -coverprofile=reports/codecoverage_all.cov ./... -cover -race -p=4
	@go tool cover -func=reports/codecoverage_all.cov > reports/functioncoverage.out
	@go tool cover -html=reports/codecoverage_all.cov -o reports/coverage.html
	@echo "View report at $(PWD)/reports/coverage.html"
	
coverage-report:
	@open reports/coverage.html
