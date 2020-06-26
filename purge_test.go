package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

func TestTimePurge(t *testing.T) {
	purgeFilesAfter := 3600 * 24 * 30 // purge files if they are not accessed for 30 days

	testPacolocoDir, err := ioutil.TempDir(os.TempDir(), "*-pacoloco-repo")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(testPacolocoDir) // clean up

	testRepo := path.Join(testPacolocoDir, "pkgs", "purgerepo")
	err = os.MkdirAll(testRepo, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	fileToRemove := path.Join(testRepo, "toremove")
	fileToStay := path.Join(testRepo, "tostay")
	fileOutsideRepo := path.Join(testPacolocoDir, "outsiderepo")

	thresholdTime := time.Now().Add(time.Duration(-purgeFilesAfter) * time.Second)

	os.Create(fileToRemove)
	os.Chtimes(fileToRemove, thresholdTime.Add(-time.Hour), thresholdTime.Add(-time.Hour))

	os.Create(fileToStay)
	os.Chtimes(fileToStay, thresholdTime.Add(time.Hour), thresholdTime.Add(-time.Hour))

	os.Create(fileOutsideRepo)
	os.Chtimes(fileToRemove, thresholdTime.Add(-time.Hour), thresholdTime.Add(-time.Hour))

	purgeStaleFiles(testPacolocoDir, purgeFilesAfter)

	_, err = os.Stat(fileToRemove)
	if !os.IsNotExist(err) {
		t.Fail()
	}

	_, err = os.Stat(fileToStay)
	if err != nil {
		t.Fail()
	}

	_, err = os.Stat(fileOutsideRepo) // files outside of the pkgs cache should not be touched
	if err != nil {
		t.Fail()
	}
}

func setupPurgeCountRepo(tempdir string) (string, [4]string, [3]string, [2]string) {
	testRepo := path.Join(tempdir, "pkgs", "purgerepo")
	err := os.MkdirAll(testRepo, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	var pkgWithTooManyVersions [4]string
	for i := 0; i < len(pkgWithTooManyVersions); i++ {
		pkgWithTooManyVersions[i] = path.Join(testRepo, fmt.Sprintf("toomany-1-%d-any.pkg.tar", i+1))
		os.Create(pkgWithTooManyVersions[i])
	}

	var pkgWithJustEnoughVersions [3]string
	for i := 0; i < len(pkgWithJustEnoughVersions); i++ {
		pkgWithJustEnoughVersions[i] = path.Join(testRepo, fmt.Sprintf("justenough-1-%d-any.pkg.tar", i+1))
		os.Create(pkgWithJustEnoughVersions[i])
	}

	var pkgWithTooFewVersions [2]string
	for i := 0; i < len(pkgWithTooFewVersions); i++ {
		pkgWithTooFewVersions[i] = path.Join(testRepo, fmt.Sprintf("toofew-1-%d-any.pkg.tar", i+1))
		os.Create(pkgWithTooFewVersions[i])
	}

	return testRepo, pkgWithTooManyVersions, pkgWithJustEnoughVersions, pkgWithTooFewVersions
}

func TestCountPurge(t *testing.T) {
	purgeKeepAtMost := 3

	testPacolocoDir, err := ioutil.TempDir(os.TempDir(), "*-pacoloco-repo")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(testPacolocoDir)

	testRepo, pkgWithTooManyVersions, pkgWithJustEnoughVersions, pkgWithTooFewVersions := setupPurgeCountRepo(testPacolocoDir)

	purgeOldFiles(pkgWithTooManyVersions[0], purgeKeepAtMost)
	purgeOldFiles(pkgWithJustEnoughVersions[0], purgeKeepAtMost)
	purgeOldFiles(pkgWithTooFewVersions[0], purgeKeepAtMost)

	var matches []string

	matches, err = filepath.Glob(filepath.Join(testRepo, "toomany-*.pkg.tar"))
	if err != nil {
		log.Fatal(err)
	}
	if len(matches) != 3 {
		t.Fail()
	}

	matches, err = filepath.Glob(filepath.Join(testRepo, "justenough-*.pkg.tar"))
	if err != nil {
		log.Fatal(err)
	}
	if len(matches) != 3 {
		t.Fail()
	}

	matches, err = filepath.Glob(filepath.Join(testRepo, "toofew-*.pkg.tar"))
	if err != nil {
		log.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fail()
	}

}

func TestAllCountPurge(t *testing.T) {
	log.SetOutput(os.Stderr)

	testPacolocoDir, err := ioutil.TempDir(os.TempDir(), "*-pacoloco-repo")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(testPacolocoDir)

	setupPurgeCountRepo(testPacolocoDir)

	purgeAllOldPackages(&Config{
		CacheDir:        testPacolocoDir,
		PurgeStrategy:   PurgeStrategyCount,
		PurgeKeepAtMost: 3,
	})
}
