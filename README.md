# Git Arweave Bridge

Eternally save a code releases to the Arweave

## Requirements

- Go 1.12
- Go modules activated
- Arweave keyfile loaded with some arweave

### Build

To build the project and create an executable, do `go build`. To install the executable, do `go install`

## Usage

### Release

Currently this tool is only designed to work with full releases. Also, there is currently a size limit of 2 MB per transactions on the arweave network, so this tool will only work with smaller repositories.

Currently you have access to 2 actions `push` and `pull`

### Pull

Pull is the simplest action to take, it allows you to pull a repository from the weave using the transaction hash only. Also it is free. The repository needs to have the same formatting as done in this package for it to pull properly. Currently all repository are tarred and compressed using Gzip before being pushed to the arweave.

An example for pull is `arweave-git release pull <ADDRESS> ` which will actually pull this repository.

### Push

Push is slightly more complex, where we need to first tar and compress the repository we wish to upload, sign it with an arweave wallet address that has enough AR to pay for the transaction fees.

An example would be: `arweave-git release push <DIR> <FLAGS>`

For more information on the workings of the command line and all the flags, please visit the [docs](docs/git_docs.md).
