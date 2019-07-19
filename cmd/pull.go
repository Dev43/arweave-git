package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dev43/arweave-go/transactor"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pulls a release from the weave",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		hash := args[0]

		// set destination if needed.. default is a new folder with the name of the git folder

		ar, err := transactor.NewTransactor("178.128.86.17")
		if err != nil {
			panic(err)
		}

		tx, err := ar.Client.GetTransaction(hash)
		if err != nil {
			panic(err)
		}

		decodedRaw, err := base64.RawURLEncoding.DecodeString(tx.Data)
		if err != nil {
			panic(err)
		}
		// unmarshal the data
		gitInfo := ArweaveRelease{}
		json.Unmarshal(decodedRaw, &gitInfo)

		fileNames := strings.Split(gitInfo.Repository, "/")
		fileName := strings.Replace(fileNames[len(fileNames)-1], ".git", "", -1)

		decodedData, err := base64.RawURLEncoding.DecodeString(gitInfo.Data)
		reader := bytes.NewReader(decodedData)

		err = untarAndUnzip(fileName, reader)
		if err != nil {
			panic(err)
		}

	},
}

func untarAndUnzip(folderName string, data io.Reader) error {
	gzipReader, err := gzip.NewReader(data)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(folderName, header.Name)
		info := header.FileInfo()
		dirName, _ := filepath.Split(path)

		// here we need to change the folders permissions so we can actually write into them
		if err = os.MkdirAll(dirName, os.ModePerm); err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}
	return nil
}
