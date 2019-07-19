package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"io"
	"os"

	"github.com/Dev43/arweave-go/transactor"
	"github.com/Dev43/arweave-go/tx"
	"github.com/Dev43/arweave-go/wallet"


	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
)

func init() {
	rootCmd.AddCommand(addCmd)
}

const TEMPGZIP = "temp.tar.gz"

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds the files into staging",
	Run: func(cmd *cobra.Command, args []string) {
		dirToUpload := "."
		if len(args) > 0 {
			dirToUpload = args[0]
		}

		r, err := git.PlainOpen(dirToUpload)
		if err != nil {
			panic(err)
		}

		err = ensureRepositoryIsClean(r, dirToUpload)
		if err != nil {
			panic(err)
		}

		// grab the whole directory, tar and zip it in memory

		directory, err := os.Open(dirToUpload)
		if err != nil {
			panic(err)
		}
		defer directory.Close()

		err = tarAndGzipDirectory(directory)
		if err != nil {
			panic(err)
		}

		// now we've created a tar and gzipped file, we need to load it in memory
		// create a JSON with the necessary information
		// and send it to the arweave network

		tarredData, err := ioutil.ReadFile(TEMPGZIP)
		if err != nil {
			panic(err)
		}

		commit, err := getLastCommit(r)
		if err != nil {
			panic(err)
		}

		conf, err := r.Config()
		if err != nil {
			panic(err)
		}
		// make this changeable
		repositoryURL := conf.Remotes["origin"].URLs[0]

		arweaveData := ArweaveRelease{
			Repository:  repositoryURL,
			LastCommit:  commit,
			LastRelease: "",
			Data:        base64.RawURLEncoding.EncodeToString(tarredData),
			Encoding:    []string{"tar", "gzip"},
		}

		toSend, err := json.Marshal(arweaveData)
		if err != nil {
			panic(err)
		}

		ar, err := transactor.NewTransactor("178.128.86.17")
		if err != nil {
			panic(err)
		}

		tx, err := sendToArweaveNetwork(ar, toSend)
		if err != nil {
			panic(err)
		}
		fmt.Println(tx.ID)

		ctx := context.TODO()

		pendingTx, err := ar.WaitMined(ctx, tx)
		if err != nil {
			panic(err)
		}
		fmt.Println(pendingTx.ID)

		// add this transaction into a new arweave file

		// finish with a commit of the arweave hash

		// commit, err := w.Commit("example go-git commit", &git.CommitOptions{
		// 	Author: &object.Signature{
		// 		Name:  "John Doe",
		// 		Email: "john@doe.org",
		// 		When:  time.Now(),
		// 	},
		// })

		// delete tarball
		err = deleteFile(TEMPGZIP)
		if err != nil {
			panic(err)
		}
	},
}


func ensureRepositoryIsClean(r *git.Repository, directory string) error {

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	s, err := w.Status()
	if err != nil {
		return err
	}
	fmt.Println(s.IsClean())
	// if !s.IsClean() {
	// 	return fmt.Errorf("Git directory is not clean, please stash your changes before continuing")
	// }

	return nil
}

func sendToArweaveNetwork(ar *transactor.Transactor, toSend []byte) (*tx.Transaction, error) {

	// create a new wallet instance
	w := wallet.NewWallet()
	// extract the key from the wallet instance
	err := w.ExtractKey("arweave.json")
	if err != nil {
		return nil, err
	}

	// create a transaction
	tx, err := ar.CreateTransaction(w, "0", toSend, "xblmNxr6cqDT0z7QIWBCo8V0UfJLd3CRDffDhF5Uh9g")
	if err != nil {
		return nil, err
	}
	// sign the transaction
	err = tx.Sign(w)
	if err != nil {
		return nil, err
	}

	fmt.Println(tx.EncodedID())

	// send the transaction
	resp, err := ar.SendTransaction(tx)
	if err != nil {
		return nil, err
	}

	fmt.Println(resp)
	return tx.Format(), err

}

func deleteFile(fileName string) error {
	return os.Remove(fileName)
}

func getLastCommit(r *git.Repository) (string, error) {
	ref, err := r.Head()
	if err != nil {
		return "", err
	}

	return ref.Hash().String(), nil
}

func tarAndGzipDirectory(directory *os.File) error {
	path := directory.Name()

	tarfile, err := os.Create(TEMPGZIP)
	if err != nil {
		panic(err)
	}
	defer tarfile.Close()

	fileWriter := gzip.NewWriter(tarfile) // add a gzip filter
	defer fileWriter.Close()

	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(path)
	}

	// Should GZIP and Tar ball our file
	targetWriter := tar.NewWriter(fileWriter)
	defer targetWriter.Close()

	return filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if info.Name() == TEMPGZIP {
			return nil
		}

		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, currentPath)
		}
		if info.IsDir() {
			return nil
		}
		header.Size = info.Size()

		if err := targetWriter.WriteHeader(header); err != nil {
			return err
		}

		file, err := os.Open(currentPath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(targetWriter, file)

		return err
	})

}
