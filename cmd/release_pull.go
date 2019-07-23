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
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	releaseCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:   "pull [address]",
	Short: "Pulls a release from the weave",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		address := args[0]

		// set destination if needed.. default is a new folder with the name of the git folder
		tx, err := ar.Client.GetTransaction(context.TODO(), address)
		if err != nil {
			log.Fatal(fmt.Errorf("could not retrieve transaction with hash %s %s", address, err.Error()))
		}
		b, err := base64.RawURLEncoding.DecodeString(tx.Data())
		if err != nil {
			log.Fatal(fmt.Errorf("error decoding transaction data field"))
		}
		decodedStringRaw := string(b)
		// unmarshal the data
		gitInfo := ArweaveRelease{}
		err = json.Unmarshal([]byte(decodedStringRaw), &gitInfo)
		if err != nil {
			log.Fatal(fmt.Errorf("error unmarshaling arweave transaction data into information struct"))
		}

		fileNames := strings.Split(gitInfo.Repository, "/")
		fileName := strings.Replace(fileNames[len(fileNames)-1], ".git", "", -1)

		decodedData, err := base64.RawURLEncoding.DecodeString(gitInfo.Data)
		if err != nil {
			log.Fatal(fmt.Errorf("error decoding data field of information struct"))
		}
		reader := bytes.NewReader([]byte(decodedData))
		err = untarAndUnzip(fileName, reader)
		if err != nil {
			log.Fatal(fmt.Errorf("error untarring and unzipping data %s", err.Error()))
		}
		fmt.Printf("Successfully downloaded repository %s \n", gitInfo.Repository)

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
