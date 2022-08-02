package nsfw

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"

	tg "github.com/galeone/tfgo"
	"github.com/sirupsen/logrus"
)

type Path string

func GetLocalModelPath() (Path, error) {
	cached, err := getLatestCached(DefaultCachePath)
	if err != nil {
		return "", err
	}

	return cached.getModelPath(), nil
}

func GetLatestModelPath() (Path, error) {
	logrus.Info("Fetching latest release info form repository")
	latest, err := getLatestReleaseInfo()
	if err != nil {
		return "", err
	}

	logrus.Infof("Latest version is '%s'", latest.TagName)

	cached, err := getLatestCached(DefaultCachePath)
	if err != nil {
		if err == ErrNoneCached {
			logrus.Info("No models cached")
		} else {
			logrus.Info("Errored while fetching cached data. Trying with remote")
		}
	} else {
		logrus.Infof("Latest cached version is '%s'", cached.TagName)
	}

	if !latest.isNewer(cached) {
		logrus.Info("Local version is up to date")

		return cached.getModelPath(), nil
	}

	logrus.Info("Downloading new version from repository")

	latest.LocalPath = DefaultCachePath + latest.getTagPath()
	err = latest.download(latest.getZipPath())
	if err != nil {
		return "", err
	}

	err = latest.saveMeta(latest.getMetaPath())
	if err != nil {
		return "", err
	}

	logrus.Info("Unzipping model")

	err = latest.unpack()
	if err != nil {
		return "", err
	}

	logrus.Info("Model cached successfully")
	return latest.getModelPath(), nil
}

func (i releaseInfo) unpack() error {
	err := i.unzip()
	if err != nil {
		return err
	}

	err = i.cleanup()
	if err != nil {
		logrus.Errorf("Unable to cleanup model file at '%s'", i.getZipPath().String())
	}

	return nil
}

func (i releaseInfo) unzip() error {
	r, err := zip.OpenReader(string(i.getZipPath()))
	if err != nil {
		return err
	}

	defer r.Close()

	for _, f := range r.File {
		filePath := filepath.Join(i.getModelFolder().String(), f.Name)

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}

			continue
		}

		if err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}

func (i releaseInfo) cleanup() error {
	return os.Remove(i.getZipPath().String())
}

func (p Path) GetModel() *tg.Model {
	return tg.LoadModel(string(p), []string{"serve"}, nil)
}

func (p Path) String() string {
	return string(p)
}
