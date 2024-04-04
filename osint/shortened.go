package osint

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/caio-ishikawa/netscout/shared"
)

const MODULE_NAME = "ARCHIVE.ORG"

// Errors
const (
	noUploadResponseErr          = "no response from archive.org's uploads query"
	unsuccessfullDownloadReqErr  = "request to download shortened url data failed"
	unexpectedShortenedUrlFormat = "invalid format for shortened URL entry"
)

const latestUpload = "https://archive.org/advancedsearch.php?q=collection:(UrlteamWebCrawls)&sort=-publicdate&fl[]=identifier,download&rows=1&output=json"
const base = "https://archive.org/compress"

// Represents the urlteam's latest upload to archive.org
type LatestUpload struct {
	Response struct {
		Docs []struct {
			Identifier string `json:"identifier"`
		} `json:"docs"`
	} `json:"response"`
}

// Responsible for storing data necessary for downloading, unzipping, and reading from the archive.org file
type ShortenedUrlFinder struct {
	DeletePostDownload bool
	TargetHost         string
	ZipFilePath        string
	DestinationPath    string
	Comms              shared.CommsChannels
}

func NewShortenedUrlFinder(host string, comms shared.CommsChannels) ShortenedUrlFinder {
	return ShortenedUrlFinder{
		DeletePostDownload: true,
		TargetHost:         host,
		ZipFilePath:        "shortened.zip",
		DestinationPath:    "data",
		Comms:              comms,
	}
}

// Unzips the downloaded file and the zipped inner files
func (su *ShortenedUrlFinder) UnzipAllDownloads() error {
	if err := su.unzipSingleDownload(0); err != nil {
		return err
	}

	fmt.Println(su.ZipFilePath)

	files, err := filepath.Glob(filepath.Join(su.DestinationPath, "*.zip"))
	if err != nil {
		return err
	}

	// check for inner zipped files
	if len(files) == 0 {
		return nil
	}

	// unzip files
	for i, file := range files {
		su.ZipFilePath = file
		if err := su.unzipSingleDownload(i); err != nil {
			return err
		}
	}

	return nil
}

// Unzips single file
func (su *ShortenedUrlFinder) unzipSingleDownload(index int) error {
	reader, err := zip.OpenReader(su.ZipFilePath)
	if err != nil {
		fmt.Println("fuck")
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		fullPath := filepath.Join(su.DestinationPath, file.Name)
		split := strings.Split(fullPath, "/")
		// filePath := su.DestinationPath + "/" + split[len(split)-1]

		if file.FileInfo().IsDir() {
			// os.Mkdir(filePath, os.ModePerm)
			continue
		}

		// if not a directory, a unique name is needed to avoid conflicts
		uniqueFilePath := su.DestinationPath + "/" + strconv.Itoa(index) + split[len(split)-1]

		if err := os.MkdirAll(filepath.Dir(uniqueFilePath), os.ModePerm); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		dst, err := os.Create(uniqueFilePath)
		if err != nil {
			return err
		}
		defer dst.Close()

		_, err = io.Copy(dst, src)
		if err != nil {
			return err
		}
	}

	// If DeletePostDownload delete the files after processing
	if su.DeletePostDownload {
		if err := os.Remove(su.ZipFilePath); err != nil {
			return err
		}
	}

	return nil
}

// Decompresses .xz.txt files downloaded from archive.org
// TODO: run the command and capture output in real-time to output via comms channels
func (su *ShortenedUrlFinder) DecompressXZ() {
	files, err := filepath.Glob(filepath.Join(su.DestinationPath, "*.txt.xz"))
	if err != nil {
		su.Comms.WarningChan <- err.Error()
		return
	}

	fmt.Println(len(files))

	for _, file := range files {
		cmd := exec.Command("xzcat", file)

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			su.Comms.WarningChan <- err.Error()
			return
		}

		err = cmd.Start()
		if err != nil {
			su.Comms.WarningChan <- err.Error()
			return
		}

		fmt.Println("command started")

		var wg sync.WaitGroup
		wg.Add(1)

		go func(wg *sync.WaitGroup) {
			defer wg.Done()

			reader := bufio.NewReader(stdoutPipe)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break
					}

					su.Comms.WarningChan <- err.Error()
					return
				}

				parsed, err := su.parseLine(line)
				if err != nil {
					// parsing errors will be omitted to reduce noise
					continue
				}

				su.Comms.DataChan <- parsed
			}
		}(&wg)

		// wait for end of output before decompressing the next file
		wg.Wait()
	}

	close(su.Comms.ShortenedDoneChan)
	return
}

// Parses line from .txt.xz file, extract URL and returns a ScannedItem object
func (su *ShortenedUrlFinder) parseLine(line string) (shared.ScannedItem, error) {
	split := strings.Split(line, "|")
	if len(split) != 2 {
		return shared.ScannedItem{}, fmt.Errorf(unexpectedShortenedUrlFormat)
	}

	formatted := strings.ReplaceAll(split[1], "\n", "")
	url, err := url.Parse(formatted)
	if err != nil {
		return shared.ScannedItem{}, err
	}

	return shared.ScannedItem{
		Url:    *url,
		Source: shared.ShortenedUrl,
	}, nil
}

// Scans the files from archive.org looking for URLs containing the host string
func (su *ShortenedUrlFinder) scanForHost(host string) {}

// Downloads shortened URL data to local directory
func (su *ShortenedUrlFinder) DownloadShortenedURLs() error {
	downloadURL, err := su.craftDownloadURL()
	if err != nil {
		return err
	}

	fmt.Println("downloading...")
	resp, err := http.Get(downloadURL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf(unsuccessfullDownloadReqErr)
	}

	filePath := "./" + filepath.Base(su.ZipFilePath)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("downloaded!")
	return nil
}

// Returns download URL for the lastest .zip file uploaded to archive.org containing shortened URL data
func (su *ShortenedUrlFinder) craftDownloadURL() (url.URL, error) {
	resp, err := http.Get(latestUpload)
	if err != nil {
		return url.URL{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return url.URL{}, err
	}

	var results LatestUpload
	if err := json.Unmarshal(body, &results); err != nil {
		return url.URL{}, err
	}

	if len(results.Response.Docs) == 0 {
		return url.URL{}, fmt.Errorf(noUploadResponseErr)
	}

	identifier := results.Response.Docs[0].Identifier

	urlStr := fmt.Sprintf("%s/%s/formats=ZIP&file=/%s.zip", base, identifier, identifier)
	u, err := url.Parse(urlStr)
	if err != nil {
		return url.URL{}, err
	}

	return *u, nil
}
