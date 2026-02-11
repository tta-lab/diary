.PHONY: help gifs append-gif edit-gif clean install-vhs install-ttyd

help:
	@echo "diary-cli GIF generation"
	@echo ""
	@echo "Usage:"
	@echo "  make gifs              Generate all demo GIFs"
	@echo "  make append-gif        Generate append workflow GIF"
	@echo "  make edit-gif          Generate edit workflow GIF"
	@echo "  make install-vhs       Install VHS (required for GIF generation)"
	@echo "  make install-ttyd      Install ttyd (required by VHS)"
	@echo "  make clean             Remove generated GIFs"

# Check if VHS is installed
check-vhs:
	@command -v vhs >/dev/null 2>&1 || (echo "VHS not found. Run 'make install-vhs'" && exit 1)

# Check if ttyd is installed
check-ttyd:
	@command -v ttyd >/dev/null 2>&1 || (echo "ttyd not found. Run 'make install-ttyd'" && exit 1)

# Install VHS via Homebrew
install-vhs:
	@echo "Installing VHS..."
	@brew install charmbracelet/tap/vhs || echo "Install failed. See: https://github.com/charmbracelet/vhs"

# Install ttyd (required by VHS)
install-ttyd:
	@echo "Installing ttyd..."
	@brew install ttyd || echo "Install failed. See: https://github.com/tsl0922/ttyd"

# Generate all GIFs
gifs: check-vhs check-ttyd append-gif edit-gif
	@echo "✓ All demo GIFs generated in docs/screenshots/"

# Generate append workflow GIF
append-gif: check-vhs check-ttyd docs/screenshots
	@echo "Recording append demo..."
	@vhs < docs/append-demo.tape
	@echo "✓ Created docs/screenshots/append-workflow.gif"

# Generate edit workflow GIF
edit-gif: check-vhs check-ttyd docs/screenshots
	@echo "Recording edit demo..."
	@vhs < docs/edit-demo.tape
	@echo "✓ Created docs/screenshots/edit-workflow.gif"

# Create screenshots directory if it doesn't exist
docs/screenshots:
	@mkdir -p docs/screenshots

# Clean up generated GIFs
clean:
	@echo "Removing generated GIFs..."
	@rm -f docs/screenshots/append-workflow.gif
	@rm -f docs/screenshots/edit-workflow.gif
	@echo "✓ Cleaned"

.PHONY: check-vhs check-ttyd
