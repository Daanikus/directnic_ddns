package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

const configFileName = "directnic_ddns.toml"

var configDirs = []string{".", "/etc"}

func loadConfig() (*toml.Tree, error) {
	for _, dir := range configDirs {
		filename := filepath.Join(dir, configFileName)
		file, err := os.Open(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, errors.Wrap(err, "opening config")
		}
		defer file.Close()

		tree, err := toml.LoadReader(file)
		if err != nil {
			return nil, errors.Wrap(err, "reading config")
		}
		return tree, nil
	}
	return nil, errors.New("no config found")
}

func loadUpdateURL() (string, error) {
	config, err := loadConfig()
	if err != nil {
		return "", errors.Wrap(err, "loading config")
	}
	updateURL, ok := config.Get("update-url").(string)
	if !ok {
		return "", errors.New("invalid/missing update-url in config")
	}
	return updateURL, nil
}

func externalIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org/")
	if err != nil {
		return "", errors.Wrap(err, "GET failed")
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("GET failed with %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "body read failed")
	}

	return string(body), nil
}

func updateEntry(updateURL, addr string) error {
	resp, err := http.Get(updateURL + addr)
	if err != nil {
		return errors.Wrap(err, "update failed")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("update failed with %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "body read failed")
	}

	if !bytes.Contains(body, []byte("success")) {
		return errors.Errorf("update failed: %s", body)
	}
	return nil
}

func main() {
	log.SetFlags(0)

	updateURL, err := loadUpdateURL()
	if err != nil {
		log.Println(err)
		return
	}

	addr, err := externalIP()
	if err != nil {
		log.Printf("failed to retrieve external ip: %v", err)
		return
	}

	log.Printf("external address: %s", addr)

	if err := updateEntry(updateURL, addr); err != nil {
		log.Printf("failed to update external ip: %v", err)
	}
	log.Println("external ip updated")
}
