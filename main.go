package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const dirPerm = 0o755

var typeMap = map[string]string{
	".jpg":  "pics",
	".jpeg": "pics",
	".png":  "pics",
	".gif":  "pics",

	".pdf":  "docs",
	".docx": "docs",
	".txt":  "docs",

	".mp4": "video",
	".avi": "video",
	".mov": "video",

	".mp3": "audio",
	".wav": "audio",

	".zip": "archives",
	".tar": "archives",
	".rar": "archives",
}

func waitForCompleteFile(path string, checkInterval time.Duration, attempts int) error {
	var prevSize int64 = -1
	for i := 0; i < attempts; i++ {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}
		if size := info.Size(); size == prevSize {
			return nil
		} else {
			prevSize = size
		}
		time.Sleep(checkInterval)
	}
	return fmt.Errorf("file %s is still changing", path)
}

func ensureDir(p string) error {
	return os.MkdirAll(p, dirPerm)
}

func insideDir(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

func moveFileWithUniqueName(srcPath, dstDir string, dryRun bool) error {
	ext := strings.ToLower(filepath.Ext(srcPath))
	folder := typeMap[ext]
	if folder == "" {
		folder = "other"
	}

	targetFolder := filepath.Join(dstDir, folder)
	if err := ensureDir(targetFolder); err != nil {
		return fmt.Errorf("create folder %s: %w", targetFolder, err)
	}

	if err := waitForCompleteFile(srcPath, 500*time.Millisecond, 5); err != nil {
		return err
	}

	base := filepath.Base(srcPath)
	dstPath := filepath.Join(targetFolder, base)

	for i := 1; ; i++ {
		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			break
		}
		dstPath = filepath.Join(targetFolder, fmt.Sprintf("%d_%s", i, base))
	}

	if dryRun {
		fmt.Printf("[dry-run] %s → %s\n", srcPath, dstPath)
		return nil
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		if errCopy := copyFile(srcPath, dstPath); errCopy != nil {
			return fmt.Errorf("move %s → %s: rename=%v, copy=%w", srcPath, dstPath, err, errCopy)
		}
		if errRm := os.Remove(srcPath); errRm != nil {
			return fmt.Errorf("remove source after copy %s: %w", srcPath, errRm)
		}
	}

	fmt.Printf("Moved: %s → %s\n", srcPath, dstPath)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

func organizeOnce(srcDir, dstDir string, noRecursive, dryRun bool) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && filepath.Clean(path) == filepath.Clean(dstDir) {
			return filepath.SkipDir
		}

		if d.IsDir() {
			if noRecursive && filepath.Clean(path) != filepath.Clean(srcDir) {
				return filepath.SkipDir
			}
			return nil
		}

		if insideDir(dstDir, path) {
			return nil
		}

		if err := moveFileWithUniqueName(path, dstDir, dryRun); err != nil {
			log.Println("move failed:", err)
		}
		return nil
	})
}

func watchDir(srcDir, dstDir string, noRecursive, dryRun bool, stop <-chan struct{}) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("new watcher: %w", err)
	}
	defer watcher.Close()

	addDir := func(dir string) {
		if filepath.Clean(dir) == filepath.Clean(dstDir) {
			return
		}
		if err := watcher.Add(dir); err != nil {
			log.Println("watch add error:", err)
		}
	}

	if noRecursive {
		addDir(srcDir)
	} else {
		if err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				addDir(path)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("walk for watching: %w", err)
		}
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
					info, err := os.Stat(event.Name)
					if err == nil {
						if info.IsDir() {
							if !noRecursive {
								addDir(event.Name)
							}
							continue
						}
						if !insideDir(dstDir, event.Name) {
							if err := moveFileWithUniqueName(event.Name, dstDir, dryRun); err != nil {
								log.Println("move failed:", err)
							}
						}
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("watcher error:", err)
			}
		}
	}()

	fmt.Println("Watching:", srcDir)
	<-stop
	return nil
}

func main() {
	srcDir := flag.String("source", ".", "Source directory")
	dstDir := flag.String("dest", "./organized", "Destination directory")
	mode := flag.String("mode", "once", "Mode: once or watch")
	dryRun := flag.Bool("dry", false, "Dry run (no changes)")
	noRecursive := flag.Bool("no-recursive", false, "Do not traverse subdirectories")
	flag.Parse()

	if err := ensureDir(*dstDir); err != nil {
		log.Fatal(err)
	}

	switch *mode {
	case "once":
		if err := organizeOnce(*srcDir, *dstDir, *noRecursive, *dryRun); err != nil {
			log.Fatal(err)
		}
	case "watch":
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		stop := make(chan struct{})

		go func() {
			<-sigCh
			fmt.Println("\nStopping watcher...")
			close(stop)
		}()

		if err := watchDir(*srcDir, *dstDir, *noRecursive, *dryRun, stop); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Println("Invalid mode. Use 'once' or 'watch'.")
	}
}
