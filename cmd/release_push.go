package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"

	"io"
	"os"

	"github.com/Dev43/arweave-go/transactor"
	"github.com/Dev43/arweave-go/tx"
	"github.com/Dev43/arweave-go/wallet"

	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// tempGzip is the temporary file that gets created when tarring and Gzipping a directory. It is removed after completion
const tempGzip = "temp.tar.gz"
const repoRootTagVersion = "0.0.1"
const kvDelimiter = ":"
const repoRootTag = "arweave-git"
const arweaveFilename = ".arweave"
const arweaveDataversion = "0.0.1"

// Tags flag
var inputTags []string
var keyPath string
var withCommit bool

func init() {
	pushCmd.Flags().StringSliceVar(&inputTags, "tags", nil, "Comma seperated list of tags to add to the arweave transaction")
	pushCmd.Flags().StringVar(&keyPath, "key-path", "./arweave.json", "Path of the arweave key file to use")
	pushCmd.Flags().BoolVar(&withCommit, "commit", false, "Whether or not we want our push action to add a commit with the new transaction ID. Default is false.")
	releaseCmd.AddCommand(pushCmd)
}

var pushCmd = &cobra.Command{
	Use:   "push [directory]",
	Short: "Pushes a release into the weave",
	Run: func(cmd *cobra.Command, args []string) {
		dirToUpload := "."
		if len(args) > 0 {
			dirToUpload = args[0]
		}

		gitDir, err := git.PlainOpen(dirToUpload)
		if err != nil {
			log.Fatal(fmt.Errorf("could not open git directory %s, please ensure it is a git directory", dirToUpload))
		}

		err = ensureRepositoryIsClean(gitDir, dirToUpload)
		if err != nil {
			log.Fatal(err)
		}

		// grab the whole directory, tar and zip it in memory
		directory, err := os.Open(dirToUpload)
		if err != nil {
			log.Fatal(fmt.Errorf("could not open directory %s", dirToUpload))
		}
		defer directory.Close()

		defer deleteFile(tempGzip)

		err = tarAndGzipDirectory(directory)
		if err != nil {
			log.Fatal(fmt.Errorf("error when executing tar and gzip on directory %s", err.Error()))
		}

		tarredData, err := ioutil.ReadFile(tempGzip)
		if err != nil {
			log.Fatal(fmt.Errorf("error reading tar and gzip directory %s %s", tempGzip, err.Error()))
		}

		commit, err := getLastCommit(gitDir)
		if err != nil {
			log.Fatal(fmt.Errorf("could not get last commit of repository"))
		}

		conf, err := gitDir.Config()
		if err != nil {
			log.Fatal(err)
		}

		repositoryURL := ""
		_, ok := conf.Remotes["origin"]
		if ok {
			repositoryURL = conf.Remotes["origin"].URLs[0]
		}

		arweaveData := ArweaveRelease{
			Version:     arweaveDataversion,
			Repository:  repositoryURL,
			LastCommit:  commit,
			LastRelease: "",
			Data:        base64.RawURLEncoding.EncodeToString(tarredData),
			Encoding:    []string{"tar", "gzip"},
		}

		toSend, err := json.Marshal(arweaveData)
		if err != nil {
			log.Fatal(fmt.Errorf("could not marshall arweave data"))
		}

		w, err := initWallet(keyPath)
		if err != nil {
			log.Fatal(fmt.Errorf("could not initialize arweave wallet"))
		}

		tags := []map[string]interface{}{map[string]interface{}{repoRootTag: repoRootTagVersion}}
		for _, tag := range inputTags {
			key := tag
			value := tag
			if strings.Contains(tag, kvDelimiter) {
				split := strings.Split(tag, kvDelimiter)
				key = split[0]
				value = split[1]
			}
			tags[0][key] = value
		}

		txBuilder, err := createTransaction(ar, w, toSend, tags)
		if err != nil {
			log.Fatal(fmt.Errorf("could not create arweave transaction"))
		}
		_ = txBuilder

		tx, err := sendToArweaveNetwork(ar, w, txBuilder)
		if err != nil {
			log.Fatal(err)
		}

		ctx := context.TODO()
		pendingTx, err := ar.WaitMined(ctx, tx)
		if err != nil {
			log.Fatal(err)
		}

		hash := pendingTx.Hash()
		// add this transaction into a new arweave file
		arweaveFilePath := filepath.Dir(dirToUpload) + "/" + arweaveFilename
		err = appendAddressToArweaveFile(arweaveFilePath, commit, hash)
		if err != nil {
			log.Fatal(err)
		}

		// finish with a commit of the arweave hash
		if withCommit {
			err = createNewCommit(gitDir, hash)
			if err != nil {
				log.Fatal(err)
			}
		}

		fmt.Printf("Successfully uploaded repository %s. Transaction hash: %s \n", repositoryURL, hash)
	},
}

func createNewCommit(gitDir *git.Repository, hash string) error {
	w, err := gitDir.Worktree()
	if err != nil {
		return err
	}
	_, err = w.Add(arweaveFilename)
	if err != nil {
		return err
	}
	_, err = w.Commit(fmt.Sprintf("New release uploaded to the arweave with hash %s", hash), &git.CommitOptions{
		Author: &object.Signature{
			Name: repoRootTag,
			When: time.Now(),
		},
	})
	return err
}

func appendAddressToArweaveFile(filePath, commit, hash string) error {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	content := ArweaveFile{}
	if len(b) > 0 {
		err = json.Unmarshal(b, &content)
		if err != nil {
			return err
		}
	}
	content.Version = repoRootTagVersion
	if len(content.Releases) == 0 {
		content.Releases = map[int64]ReleaseInfo{}
	}
	content.Releases[time.Now().Unix()] = ReleaseInfo{
		Commit: commit,
		Hash:   hash,
	}
	toWrite, err := json.Marshal(content)
	if err != nil {
		return err
	}
	_, err = f.WriteAt(toWrite, 0)

	return err
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
	if !s.IsClean() {
		return fmt.Errorf("Git directory is not clean, please commit or stash your changes before continuing")
	}

	return nil
}

func initWallet(filePath string) (*wallet.Wallet, error) {
	// create a new wallet instance
	w := wallet.NewWallet()
	// extract the key from the wallet instance
	err := w.LoadKeyFromFile(filePath)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func createTransaction(ar *transactor.Transactor, w *wallet.Wallet, toSend []byte, tags []map[string]interface{}) (*tx.Transaction, error) {
	// create a transaction
	txBuilder, err := ar.CreateTransaction(context.TODO(), w, "0", toSend, "")
	if err != nil {
		return nil, err
	}
	txBuilder.SetTags(tags)
	return txBuilder, nil

}

func sendToArweaveNetwork(ar *transactor.Transactor, w *wallet.Wallet, tx *tx.Transaction) (*tx.Transaction, error) {

	// sign the transaction
	signedTx, err := tx.Sign(w)
	if err != nil {
		return nil, err
	}

	// send the transaction
	_, err = ar.SendTransaction(context.TODO(), signedTx)
	if err != nil {
		return nil, err
	}

	return signedTx, err

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

	tarfile, err := os.Create(tempGzip)
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
		if info.Name() == tempGzip {
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
