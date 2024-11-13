
.PHONY: install
install:
	@echo "Installing..."
	@go build  -o ${GOPATH}/bin/jdocgen cmd/jdocgen/main.go
	@echo "Done."