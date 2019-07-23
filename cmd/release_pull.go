package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pulls a release from the weave",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		hash := args[0]

		// set destination if needed.. default is a new folder with the name of the git folder
		tx, err := ar.Client.GetTransaction(context.TODO(), hash)
		if err != nil {
			panic(err)
		}
		b, _ := base64.RawURLEncoding.DecodeString(tx.Data())
		decodedRaw := string(b)
		if err != nil {
			panic(err)
		}
		// unmarshal the data
		gitInfo := ArweaveRelease{}
		json.Unmarshal([]byte(decodedRaw), &gitInfo)

		fileNames := strings.Split(gitInfo.Repository, "/")
		fileName := strings.Replace(fileNames[len(fileNames)-1], ".git", "", -1)

		decodedData, err := base64.RawURLEncoding.DecodeString(gitInfo.Data)
		reader := bytes.NewReader([]byte(decodedData))
		fmt.Println(gitInfo.Data)

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
