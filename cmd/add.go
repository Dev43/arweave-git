package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"path/filepath"

	"io"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
)

func init() {
	rootCmd.AddCommand(balanceCmd)
}

const TEMP_GZIP = "temp.tar.gz"

var balanceCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds the files into staging",
	Run: func(cmd *cobra.Command, args []string) {
		dirToUpload := "."
		if len(args) > 0 {
			dirToUpload = args[0]
		}

		err := ensureRepositoryIsClean(dirToUpload)
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

		err = untarAndGzipDirectory(TEMP_GZIP)
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

func tarAndGzipDirectory(directory *os.File) error {
	path := directory.Name()

	tarfile, err := os.Create(TEMP_GZIP)
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
		if info.Name() == TEMP_GZIP {
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

func untarAndGzipDirectory(fileName string) error {
	reader, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer reader.Close()
	gzipReader, err := gzip.NewReader(reader)
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

		path := filepath.Join("temp", header.Name)
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
