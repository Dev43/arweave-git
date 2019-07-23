package cmd

// ArweaveRelease is the structure of the saved data on the arweave blockchain
type ArweaveRelease struct {
	Version     string   `json:"version"`
	Repository  string   `json:"repository"`
	LastCommit  string   `json:"last_commit"`
	LastRelease string   `json:"last_release"`
	Data        string   `json:"data"`
	Encoding    []string `json:"encoding"`
}

// ArweaveFile is the structure of the .arweave file saved in the repository's directory
type ArweaveFile struct {
	Version  string                `json:"version"`
	Releases map[int64]ReleaseInfo `json:"releases"`
}

// ReleaseInfo contains the arweave hash and the git commit of the release
type ReleaseInfo struct {
	Hash   string `json:"hash"`
	Commit string `json:"Commit"`
}
