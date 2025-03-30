# Makefile

# Name der resultierenden Binärdatei
BINARY_NAME=mergefiles

# Go-Pakete für Formatierung, Tests etc. (./... bedeutet alle im aktuellen Verzeichnis und darunter)
PKG=./...
# Hauptpaket für den Build (das Verzeichnis mit der main-Funktion)
MAIN_PKG=.

# Standard Go Build Flags (können über die Kommandozeile überschrieben werden)
GOFLAGS=-v

# Linker Flags: -s entfernt Debug-Symbole, -w entfernt DWARF-Informationen -> kleinere Binärdatei
# Für einen Release-Build verwenden
LDFLAGS=-ldflags="-s -w"

# Standardziel: Wird ausgeführt, wenn nur 'make' aufgerufen wird
all: build

# Baut die Binärdatei
build: tidy
	@echo "Building $(BINARY_NAME)..."
	@go build $(GOFLAGS) -o $(BINARY_NAME) $(MAIN_PKG)

# Baut eine kleinere Release-Binärdatei
release: tidy
	@echo "Building $(BINARY_NAME) (Release)..."
	@go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PKG)

# Entfernt die gebaute Binärdatei
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)

# Führt go mod tidy aus, um Abhängigkeiten zu bereinigen
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Formatiert den Code
fmt:
	@echo "Formatting code..."
	@go fmt $(PKG)

# Führt Tests aus (füge Tests hinzu, um dies sinnvoll zu nutzen!)
test:
	@echo "Running tests..."
	@go test $(PKG)

# Baut und führt das Programm aus. Argumente können mit ARGS übergeben werden.
# Beispiel: make run ARGS="-dir . -out output.log -ext .go"
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) $(ARGS)

# Deklariert Ziele, die keine Dateien sind
.PHONY: all build release clean tidy fmt test run