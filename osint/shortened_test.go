package osint

import (
	"github.com/caio-ishikawa/netscout/shared"
	"os"
	"testing"
)

func TestUnzipSingleDownload(t *testing.T) {
	comms := shared.NewCommsChannels()

	su := ShortenedUrlFinder{
		DeletePostDownload: true,
		TargetHost:         "test",
		ZipFilePath:        "../testfiles/testdir.zip",
		DestinationPath:    "../testfiles",
		Comms:              comms,
	}
	err := su.unzipSingleDownload(0)
	if err != nil {
		t.Error(err)
	}

	if _, err = os.Stat("../testfiles/testdir"); err != nil {
		t.Errorf("unzipped file 'testdir' does not exist")
	}

	if _, err := os.Stat("../testfiles/0test.txt.xz"); err != nil {
		t.Errorf("0test.txt.xz file does not exist")
	}
}
