package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

type release struct {
	TagName     string         `json:"tag_name"`
	HTMLURL     string         `json:"html_url"`
	Body        string         `json:"body"`
	PublishedAt string         `json:"published_at"`
	Assets      []releaseAsset `json:"assets"`
}

func main() {
	var installerPath string
	var version string
	var listenAddr string
	var readyFile string

	flag.StringVar(&installerPath, "installer", "", "installer asset to serve")
	flag.StringVar(&version, "version", "", "release version to serve")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:0", "HTTP listen address")
	flag.StringVar(&readyFile, "ready-file", "", "file to receive the base URL after startup")
	flag.Parse()

	if installerPath == "" || version == "" || readyFile == "" {
		log.Fatal("--installer, --version, and --ready-file are required")
	}
	installerPath, err := filepath.Abs(installerPath)
	if err != nil {
		log.Fatal(err)
	}
	info, err := os.Stat(installerPath)
	if err != nil {
		log.Fatal(err)
	}
	if info.IsDir() {
		log.Fatal("installer path is a directory")
	}
	digest, err := fileSHA256(installerPath)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	baseURL := "http://" + listener.Addr().String()
	assetName := filepath.Base(installerPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(release{
			TagName:     version,
			HTMLURL:     baseURL + "/release-notes",
			Body:        "Local upgrade acceptance test",
			PublishedAt: time.Now().UTC().Format(time.RFC3339),
			Assets: []releaseAsset{{
				Name:               assetName,
				BrowserDownloadURL: baseURL + "/assets/" + assetName,
				Digest:             "sha256:" + digest,
			}},
		})
	})
	mux.HandleFunc("/assets/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || strings.TrimPrefix(request.URL.Path, "/assets/") != assetName {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		file, err := os.Open(installerPath)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		writer.Header().Set("Content-Type", "application/octet-stream")
		writer.Header().Set("Content-Length", fmt.Sprint(info.Size()))
		_, _ = io.Copy(writer, file)
	})

	if err := os.MkdirAll(filepath.Dir(readyFile), 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(readyFile, []byte(baseURL+"\n"), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("serving version=%q asset=%q url=%q", version, installerPath, baseURL)
	log.Fatal(http.Serve(listener, mux))
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
