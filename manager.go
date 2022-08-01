package nsfw

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

const (
	DefaultCachePath = "./.models/"
)

var (
	ErrNoneCached = errors.New("no cached models")
)

type releaseInfo struct {
	LocalPath string `json:"-"`
	TagName   string `json:"tag_name"`
	Assets    []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestCached(folder string) (releaseInfo, error) {
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return releaseInfo{}, ErrNoneCached
	}

	latest := releaseInfo{TagName: "0"}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		if latest.isOlderDir(file.Name()) {
			latest = releaseInfo{
				LocalPath: folder + file.Name(),
				TagName:   file.Name(),
			}
			// Only get partial data until we make sure that it's the latest model
		}
	}

	if latest.TagName == "0" {
		return releaseInfo{}, ErrNoneCached
	}

	info, err := parseReleaseInfoFile(latest.LocalPath + "/meta.json")
	if err != nil {
		return releaseInfo{}, err
	}

	info.LocalPath = latest.LocalPath
	return info, nil
}

func parseReleaseInfoFile(filename string) (releaseInfo, error) {
	f, err := os.Open(filename)
	if err != nil {
		return releaseInfo{}, err
	}

	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return releaseInfo{}, err
	}

	var info releaseInfo
	err = jsoniter.Unmarshal(data, &info)
	if err != nil {
		return releaseInfo{}, err
	}

	return info, nil
}

func getLatestReleaseInfo() (releaseInfo, error) {
	resp, err := http.Get("https://api.github.com/repos/GantMan/nsfw_model/releases/latest")
	if err != nil {
		return releaseInfo{}, err
	}

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return releaseInfo{}, err
	}

	var info releaseInfo
	err = jsoniter.Unmarshal(body, &info)
	if err != nil {
		return releaseInfo{}, err
	}

	return info, nil
}

func (i releaseInfo) getTagPath() string {
	return strings.ReplaceAll(i.TagName, ".", "_")
}

func (i releaseInfo) getModelPath() Path {
	innerFolder := strings.TrimSuffix(i.Assets[0].Name, ".1.zip")
	return i.getModelFolder() + Path(innerFolder)
}

func (i releaseInfo) getModelFolder() Path {
	return Path(i.LocalPath + "/model/")
}

func (i releaseInfo) getZipPath() Path {
	return Path(i.LocalPath + "/model.zip")
}

func (i releaseInfo) getMetaPath() Path {
	return Path(i.LocalPath + "/meta.json")
}

func (i releaseInfo) isNewer(than releaseInfo) bool {
	if than.TagName == "" {
		return true
	}

	tag1, _ := strconv.Atoi(strings.ReplaceAll(i.TagName, ".", ""))
	tag2, _ := strconv.Atoi(strings.ReplaceAll(than.TagName, ".", ""))

	return tag1 > tag2
}

func (i releaseInfo) isOlderDir(than string) bool {
	tag1, _ := strconv.Atoi(strings.ReplaceAll(i.TagName, ".", ""))
	tag2, _ := strconv.Atoi(strings.ReplaceAll(than, "_", ""))

	return tag1 < tag2
}

func (i releaseInfo) download(filename Path) error {
	if len(i.Assets) == 0 {
		return fmt.Errorf("no assets for release '%s'", i.TagName)
	}

	if !strings.HasSuffix(string(filename), ".zip") {
		filename += ".zip"
	}

	if err := os.MkdirAll(filepath.Dir(string(filename)), 0770); err != nil {
		return err
	}

	out, err := os.Create(string(filename))
	if err != nil {
		return err
	}

	defer out.Close()

	resp, err := http.Get(i.Assets[0].DownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (i releaseInfo) saveMeta(filename Path) error {
	out, err := os.Create(string(filename))
	if err != nil {
		return err
	}

	defer out.Close()

	data, err := jsoniter.Marshal(i)
	if err != nil {
		return err
	}

	_, err = out.Write(data)
	if err != nil {
		return err
	}

	return nil
}
