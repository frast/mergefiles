package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/sys/unix" // Für Linux/macOS mmap
	// Für Windows: import "golang.org/x/sys/windows"
)

var (
	version              = "dev"
	date                 = "unknown"
	commit               = "HEAD"
)

// fileInfo speichert Informationen über eine zu verarbeitende Datei
type fileInfo struct {
	relativePath string // Relativer Pfad zur Verwendung im Header
	fullPath     string // Absoluter Pfad zum Lesen der Datei
	contentSize  int64  // Größe des Dateiinhalts
	headerSize   int64  // Größe des Headers (Pfad + Newline)
}

// stringSliceFlag ist ein benutzerdefinierter Flag-Typ für eine Liste von Strings
type stringSliceFlag []string

const (
	headerTemplate = "--- START FILE: %s ---\n```\n"
	footer      = "\n```\n--- END FILE ---\n\n"
	footerSize = int64(len(footer))
)

func (i *stringSliceFlag) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *stringSliceFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	// --- Argumente parsen ---
	srcDir := flag.String("dir", ".", "Quellverzeichnis (rekursiv durchsucht)")
	outFile := flag.String("out", "output.txt", "Zieldatei")
	promptKey := flag.String("prompt", "default", "Prompt")

	var extensionsFlag stringSliceFlag
	flag.Var(&extensionsFlag, "ext", "Dateierweiterung (kann mehrfach verwendet werden, z.B. -ext .txt -ext .md)")
	showVersion := flag.Bool("v", false, "Show version")

	flag.Parse()

	if *showVersion {
		fmt.Print(buildVersion()+"\n")
		return
	}

	// --- Eingaben validieren ---
	if *srcDir == "" || *outFile == "" {
		log.Fatal("Quellverzeichnis (-dir) und Zieldatei (-out) müssen angegeben werden.")
	}

	srcDirAbs, err := filepath.Abs(*srcDir)
	if err != nil {
		log.Fatalf("Fehler beim Ermitteln des absoluten Pfads für %s: %v", *srcDir, err)
	}

	info, err := os.Stat(srcDirAbs)
	if err != nil {
		log.Fatalf("Fehler beim Zugriff auf das Quellverzeichnis %s: %v", srcDirAbs, err)
	}
	if !info.IsDir() {
		log.Fatalf("Der angegebene Quellpfad %s ist kein Verzeichnis.", srcDirAbs)
	}


	conf, err := InitConfig()
	if err != nil {
		log.Fatalf("Error initializing configuration: %v", err)
	}

	prompt := conf.LookupPrompt(*promptKey)

	// --- Erweiterungen vorbereiten (als Set für schnelles Nachschlagen) ---
	extensions := make(map[string]struct{})
	processAllExtensions := len(extensionsFlag) == 0
	if !processAllExtensions {
		for _, ext := range extensionsFlag {
			// Stelle sicher, dass die Erweiterung mit einem Punkt beginnt und klein geschrieben ist
			normalizedExt := strings.ToLower(ext)
			if !strings.HasPrefix(normalizedExt, ".") {
				normalizedExt = "." + normalizedExt
			}
			extensions[normalizedExt] = struct{}{}
			fmt.Printf("Filtere nach Erweiterung: %s\n", normalizedExt)
		}
	} else {
		fmt.Println("Keine Erweiterungen angegeben, verarbeite alle Dateien.")
	}

	// --- Phase 1: Dateien sammeln und Größen berechnen ---
	fmt.Println("Phase 1: Sammle Dateien und berechne Gesamtgröße...")
	filesToProcess, totalSize, err := collectFilesAndSizes(prompt, srcDirAbs, extensions, processAllExtensions)
	if err != nil {
		log.Fatalf("Fehler beim Sammeln der Dateien: %v", err)
	}

	if len(filesToProcess) == 0 {
		log.Println("Keine passenden Dateien im Verzeichnis gefunden. Zieldatei wird nicht erstellt.")
		os.Exit(0)
	}

	fmt.Printf("Gefunden: %d Dateien, Gesamtgröße der Zieldatei: %d Bytes\n", len(filesToProcess), totalSize)

	// --- Phase 2: Zieldatei erstellen, mappen und parallel befüllen ---
	fmt.Println("Phase 2: Erstelle Zieldatei und schreibe Inhalte parallel...")
	err = createAndPopulateOutput(prompt, *outFile, totalSize, filesToProcess)
	if err != nil {
		log.Fatalf("Fehler beim Erstellen oder Schreiben der Zieldatei: %v", err)
	}

	fmt.Printf("Fertig! Die zusammengefasste Datei wurde erfolgreich erstellt: %s\n", *outFile)
}

// collectFilesAndSizes durchsucht das Verzeichnis rekursiv, filtert nach Erweiterungen
// und berechnet die Gesamtgröße der Zieldatei.
func collectFilesAndSizes(prompt string, rootDir string, extensions map[string]struct{}, processAll bool) ([]fileInfo, int64, error) {
	var files []fileInfo
	var totalSize int64 = int64(len([]byte(prompt)))

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Fehler beim Zugriff auf einen Pfad, z.B. Rechteprobleme. Loggen und weitermachen.
			fmt.Fprintf(os.Stderr, "Warnung: Fehler beim Zugriff auf %s: %v\n", path, err)
			return nil // Ignoriere diesen Eintrag und setze fort
		}

		// Überspringe Verzeichnisse
		if d.IsDir() {
			// Überspringe das Wurzelverzeichnis selbst bei der Pfadberechnung
			if path == rootDir {
				return nil
			}
			// Optional: Hier könnte man Verzeichnis-spezifische Logik hinzufügen
			return nil
		}

		// Filtere nach Erweiterung, wenn welche angegeben wurden
		if !processAll {
			ext := strings.ToLower(filepath.Ext(path))
			if _, found := extensions[ext]; !found {
				return nil // Diese Datei überspringen
			}
		}

		// Hole Datei-Informationen (insbesondere Größe)
		info, err := d.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warnung: Kann Informationen für %s nicht lesen: %v\n", path, err)
			return nil // Ignoriere diese Datei
		}

		// Berechne relativen Pfad für den Header
		relativePath, err := filepath.Rel(rootDir, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warnung: Kann relativen Pfad für %s nicht berechnen: %v\n", path, err)
			// Fallback: Absoluten Pfad verwenden oder überspringen? Wir überspringen hier.
			return nil
		}
		// Konvertiere Pfadtrenner zu "/" für Konsistenz im Header
		relativePath = filepath.ToSlash(relativePath)

		// Berechne Header und dessen Größe
		header := fmt.Sprintf(headerTemplate, relativePath)
		headerSize := int64(len(header))
		contentSize := info.Size()

		// Füge zur Liste hinzu und aktualisiere Gesamtgröße
		files = append(files, fileInfo{
			relativePath: relativePath,
			fullPath:     path,
			contentSize:  contentSize,
			headerSize:   headerSize,
		})
		totalSize += headerSize + contentSize + footerSize

		return nil
	})

	if err != nil {
		// Ein schwerwiegender Fehler während des WalkDir (sollte nicht passieren, wenn wir Fehler in der Callback-Funktion behandeln)
		return nil, 0, fmt.Errorf("schwerwiegender Fehler während des Durchsuchens von %s: %v", rootDir, err)
	}

	return files, totalSize, nil
}

// createAndPopulateOutput erstellt die Zieldatei, mappt sie in den Speicher
// und schreibt die Header und Inhalte der Quelldateien parallel hinein.
func createAndPopulateOutput(prompt string, outFilePath string, totalSize int64, files []fileInfo) error {
	// --- Zieldatei erstellen ---
	outFile, err := os.Create(outFilePath)
	if err != nil {
		return fmt.Errorf("kann Zieldatei %s nicht erstellen: %v", outFilePath, err)
	}
	defer outFile.Close()

	// --- Zieldatei auf die benötigte Größe bringen (wichtig für mmap!) ---
	if err := outFile.Truncate(totalSize); err != nil {
		return fmt.Errorf("kann Größe der Zieldatei %s nicht auf %d setzen: %v", outFilePath, totalSize, err)
	}

	// --- Datei in den Speicher mappen (Unix-spezifisch) ---
	// PROT_READ | PROT_WRITE: Lese- und Schreibzugriff erlauben
	// MAP_SHARED: Änderungen werden in die Datei zurückgeschrieben
	mappedData, err := unix.Mmap(int(outFile.Fd()), 0, int(totalSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("fehler beim Memory Mapping der Datei %s: %v", outFilePath, err)
	}
	// Sicherstellen, dass unmap am Ende aufgerufen wird
	defer func() {
		if err := unix.Munmap(mappedData); err != nil {
			fmt.Fprintf(os.Stderr, "Warnung: Fehler beim Unmappen der Datei %s: %v\n", outFilePath, err)
		}
	}()

	// --- Paralleles Schreiben vorbereiten ---
	var wg sync.WaitGroup
	var currentOffset int64 = 0 // Aktuelle Schreibposition im gemappten Speicher
	promptBytes := []byte(prompt)
	promptSize := int64(len(promptBytes))
	copy(mappedData[currentOffset:currentOffset+promptSize], promptBytes)
	currentOffset += promptSize

	for _, file := range files {
		wg.Add(1)

		// Kopiere Variablen für die Goroutine (verhindert Race Conditions bei Schleifenvariablen)
		fInfo := file
		offset := currentOffset

		go func() {
			defer wg.Done()

			// 1. Header schreiben
			header := fmt.Sprintf(headerTemplate, fInfo.relativePath)
			headerBytes := []byte(header)
			copy(mappedData[offset:offset+fInfo.headerSize], headerBytes)

			// 2. Dateiinhalt lesen und schreiben
			if fInfo.contentSize > 0 {
				srcFile, err := os.Open(fInfo.fullPath)
				if err != nil {
					// Logge Fehler, aber mache weiter (die Stelle bleibt leer oder enthält Nullen)
					fmt.Fprintf(os.Stderr, "\nFehler: Kann Quelldatei %s nicht öffnen: %v\n", fInfo.fullPath, err)
					return // Beende diese Goroutine
				}
				defer srcFile.Close()

				// Lese direkt in den gemappten Speicherbereich
				contentOffset := offset + fInfo.headerSize
				n, err := io.ReadFull(srcFile, mappedData[contentOffset:contentOffset+fInfo.contentSize])
				if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
					// Logge Lesefehler
					fmt.Fprintf(os.Stderr, "\nFehler: Kann Inhalt von %s nicht vollständig lesen (gelesen %d bytes): %v\n", fInfo.fullPath, n, err)
					// Optional: Fülle den Rest mit Nullen oder lasse es
				} else if int64(n) != fInfo.contentSize {
					fmt.Fprintf(os.Stderr, "\nWarnung: Gelesene Bytes (%d) für %s entsprechen nicht der erwarteten Größe (%d).\n", n, fInfo.fullPath, fInfo.contentSize)
				}
			}

			// 1. Footer schreiben
			footerOffset := offset + fInfo.headerSize + fInfo.contentSize
			copy(mappedData[footerOffset:footerOffset+footerSize], []byte(footer))

		}()

		// Aktualisiere den Offset für die nächste Datei
		currentOffset += fInfo.headerSize + fInfo.contentSize + footerSize
	}

	// --- Auf alle Goroutines warten ---
	wg.Wait()

	// --- Änderungen auf die Festplatte schreiben (Flush) ---
	if err := unix.Msync(mappedData, unix.MS_SYNC); err != nil {
		return fmt.Errorf("fehler beim Synchronisieren (msync) der gemappten Datei %s: %v", outFilePath, err)
	}

	fmt.Println("Paralleles Schreiben abgeschlossen.")
	return nil // Unmap und Schließen der Datei erfolgt durch defer
}

func buildVersion() string {
	result := version
	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}
	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}
	result = fmt.Sprintf("%s\ngoos: %s\ngoarch: %s", result, runtime.GOOS, runtime.GOARCH)
	return result
}