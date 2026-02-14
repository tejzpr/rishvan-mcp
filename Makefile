.PHONY: all frontend build clean

all: frontend build

frontend:
	cd frontend && npm install && npx vite build

build: frontend
	go build -o rishvan-mcp .

clean:
	rm -rf frontend/dist frontend/node_modules rishvan-mcp
