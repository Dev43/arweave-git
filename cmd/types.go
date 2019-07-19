package cmd

type ArweaveRelease struct {
	Repository  string   `json:"repository"`
	LastCommit  string   `json:"last_commit"`
	LastRelease string   `json:"last_release"`
	Data        string   `json:"data"`
	Encoding    []string `json:"encoding"`
}
