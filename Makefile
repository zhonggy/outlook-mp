BINARY := bin/outlook-manager
GOFLAGS := -trimpath -ldflags="-s -w"

.PHONY: build web backend run test vet clean

build: web backend

web:
	cd web && npm install && npm run build

backend:
	@mkdir -p web/dist
	@test -f web/dist/index.html || echo '<!DOCTYPE html><html><body>run make web first</body></html>' > web/dist/index.html
	go build $(GOFLAGS) -o $(BINARY) ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin web/node_modules
	mkdir -p web/dist
	echo '<!DOCTYPE html><html><body>placeholder</body></html>' > web/dist/index.html
