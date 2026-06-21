APP       := OCGTool.app
BINARY    := ocg

.PHONY: all build app run clean

all: app

build: $(BINARY)

$(BINARY): go.mod go.sum main.go ocg.go usage.go login.go
	CGO_ENABLED=1 go build -o $@ .

app: $(BINARY)
	rm -rf $(APP)
	mkdir -p $(APP)/Contents/MacOS
	cp $(BINARY) $(APP)/Contents/MacOS/
	cp Info.plist $(APP)/Contents/
	@echo "✓ $(APP) built"
	@echo "  Drag to /Applications or run: open $(APP)"

run: app
	open $(APP)

clean:
	rm -rf $(BINARY) $(APP)
	go clean -cache
	@echo "✓ cleaned"
