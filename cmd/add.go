package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"path/filepath"
	"strings"

	"io"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
)

func init() {
	rootCmd.AddCommand(balanceCmd)
}

var balanceCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds the files into staging",
	Run: func(cmd *cobra.Command, args []string) {
		dirToUpload := "."
		if len(args) > 0 {
			dirToUpload = args[0]
		}

		info, err := os.Stat(dirToUpload)
		if err != nil {
			panic(err)
		}
		fmt.Println(info)

		fmt.Println(filepath.Dir("."))
		fmt.Println(filepath.Base(dirToUpload))

		err = ensureRepositoryIsClean(dirToUpload)
		if err != nil {
			panic(err)
		}

		// grab the whole directory, tar and zip it in memory

		dir, err := os.Open(dirToUpload)
		if err != nil {
			panic(err)
		}
		defer dir.Close()

		err = tarDirectory(dirToUpload, dir)
		if err != nil {
			panic(err)
		}

		// finish with a commit of the arweave hash

		// commit, err := w.Commit("example go-git commit", &git.CommitOptions{
		// 	Author: &object.Signature{
		// 		Name:  "John Doe",
		// 		Email: "john@doe.org",
		// 		When:  time.Now(),
		// 	},
		// })

	},
}

func ensureRepositoryIsClean(directory string) error {
	r, err := git.PlainOpen(directory)
	if err != nil {
		return err
	}

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

func tarDirectory(path string, w io.Writer) error {

	tarfile, err := os.Create("temp.tar.gz")
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
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(currentPath, path))
			fmt.Println(header.Name)
		}
		if info.IsDir() {
			return nil
		}

		if err := targetWriter.WriteHeader(header); err != nil {
			return err
		}

		file, err := os.Open(currentPath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(targetWriter, file)
		fmt.Println("the err", err)

		return err

	})

}
