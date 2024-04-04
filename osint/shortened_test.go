package osint

import (
	"os"
	"testing"

	"github.com/caio-ishikawa/netscout/shared"
)

func TestUnzipSingleDownload(t *testing.T) {
	comms := shared.NewCommsChannels()

	su := ShortenedUrlFinder{
		DeletePostDownload: false,
		TargetHost:         "test",
		ZipFilePath:        "../testfiles/testdir.zip",
		DestinationPath:    "../testfiles",
		Comms:              comms,
	}
	err := su.unzipSingleDownload(0)
	if err != nil {
		t.Error(err)
	}

	if _, err = os.Stat("../testfiles/0testinner.zip"); err != nil {
		t.Errorf("unzipped file '0testinner.zip' does not exist")
	}

	if _, err = os.Stat("../testfiles/0testinner2.zip"); err != nil {
		t.Errorf("unzipped file '0testinner2' does not exist")
	}
}

func TestUnzipAllDownloads(t *testing.T) {
	comms := shared.NewCommsChannels()

	su := ShortenedUrlFinder{
		DeletePostDownload: true,
		TargetHost:         "test",
		ZipFilePath:        "../testfiles/testdir.zip",
		DestinationPath:    "../testfiles",
		Comms:              comms,
	}

	err := su.UnzipAllDownloads()
	if err != nil {
		t.Error(err)
	}

	if _, err = os.Stat("../testfiles/0text1.txt.xz"); err != nil {
		t.Errorf("unzipped file '0test1.txt.xz' does not exist")
	}

	if _, err = os.Stat("../testfiles/1text2.txt.xz"); err != nil {
		t.Errorf("unzipped file '1test2.txt.xz' does not exist")
	}
}

func TestDecompressXZ(t *testing.T) {
	comms := shared.NewCommsChannels()

	su := ShortenedUrlFinder{
		DeletePostDownload: true,
		TargetHost:         "test.com",
		ZipFilePath:        "../testfiles/testdir.zip",
		DestinationPath:    "../testfiles",
		Comms:              comms,
	}

	receivedData := 0
	expectedData := 1

	receivedWarning := 0
	expectedWarning := 0

	go func() {
		for {
			select {
			case <-comms.DataChan:
				receivedData++
			case <-comms.WarningChan:
				expectedWarning++
			case <-comms.ShortenedDoneChan:
				if receivedData != expectedData {
					t.Errorf("shortened url scanner expected %v msgs; got %v", expectedData, receivedData)
					t.Fail()
				}
				if receivedWarning != expectedWarning {
					t.Errorf("shortened url scanner expected %v warnings; got %v", expectedWarning, receivedWarning)
					t.Fail()
				}
				return
			}
		}
	}()

	su.UnzipAllDownloads()
}
