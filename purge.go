package main

import (
	"fmt"
	"github.com/knqyf263/go-rpm-version"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

func setupPurgeStaleFilesRoutine() *time.Ticker {
	ticker := time.NewTicker(time.Duration(24) * time.Hour) // purge files once a day
	go func() {
		if config.PurgeStrategy == PurgeStrategyTime {
			purgeStaleFiles(config.CacheDir, config.PurgeFilesAfter)
		}
		for {
			select {
			case <-ticker.C:
				if config.PurgeStrategy == PurgeStrategyTime {
					purgeStaleFiles(config.CacheDir, config.PurgeFilesAfter)
				}
			}
		}
	}()

	return ticker
}

// purgeStaleFiles purges files in the pacoloco cache
// it recursively scans `cacheDir`/pkgs and if the file access time is older than
// `now` - purgeFilesAfter(seconds) then the file gets removed
func purgeStaleFiles(cacheDir string, purgeFilesAfter int) {
	removeIfOlder := time.Now().Add(time.Duration(-purgeFilesAfter) * time.Second)
	pkgDir := filepath.Join(cacheDir, "pkgs")

	// Go through all files in the repos, and check if access time is older than `removeIfOlder`
	walkfn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		atimeUnix := info.Sys().(*syscall.Stat_t).Atim
		atime := time.Unix(int64(atimeUnix.Sec), int64(atimeUnix.Nsec))
		if atime.Before(removeIfOlder) {
			log.Printf("Remove stale file %v as its access time (%v) is too old", path, atime)
			err := os.Remove(path)
			if err != nil {
				log.Print(err)
			}
		}
		return nil
	}
	if err := filepath.Walk(pkgDir, walkfn); err != nil {
		log.Println(err)
	}
}

func purgeOldFiles(file string, keepAtMost int) {
	pkg := parsePackage(file)

	// find all files that might be the same package
	glob := filepath.Join(pkg.Directory, fmt.Sprintf("%s-*.pkg.tar*", pkg.Name))
	matches, err := filepath.Glob(glob)
	if err != nil {
		log.Print(err)
		return
	}

	// filter by those that are the same package
	canidates := make([]Package, 0, len(matches))
	for _, match := range matches {
		if canidate := parsePackage(match); canidate.Name == pkg.Name {
			canidates = append(canidates, canidate)
		}
	}

	// skip sorting if we don't have to remove any files
	if len(canidates) < keepAtMost {
		return
	}

	// remove the oldest (by version) packages
	sort.Slice(canidates, func(i, j int) bool {
		return canidates[i].Version.LessThan(canidates[j].Version)
	})
	for i := 0; i < len(canidates)-keepAtMost; i++ {
		canidate := canidates[i]
		log.Printf("Remove old file %v as there are more than %d version(s) of this package", canidate.FullPath, keepAtMost)
		if err := os.Remove(canidate.FullPath); err != nil {
			log.Print(err)
		}
	}
}

type Package struct {
	FullPath  string
	Directory string
	Name      string
	Version   version.Version
	Arch      string
	Extension string
}

func parsePackage(file string) Package {
	dir, base := filepath.Split(file)
	parts := strings.Split(base, "-")
	count := len(parts)
	last := strings.SplitN(parts[count-1], ".", 2)

	return Package{
		FullPath:  file,
		Directory: dir,
		Name:      strings.Join(parts[0:count-3], "-"),
		Version:   version.NewVersion(strings.Join(parts[count-3:count-1], "-")),
		Arch:      last[0],
		Extension: last[1],
	}
}
