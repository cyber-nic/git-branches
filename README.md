# Git Branch Manager

Simple terminal tool to list local Git branches with their last commit date and ahead/behind status, and provide a clickable `delete` link to remove branches.

## Features

- Lists all local branches sorted by creation order.
- Shows last commit timestamp.
- Displays commits ahead/behind compared to `main` (or `master`).
<!-- - Provides a `delete` link per row for branch cleanup. -->

## Prerequisites

- Go 1.18+ installed
- `git` available in your PATH

## Installation

### From source

1. Clone the repository:

   ```sh
   git clone https://github.com/cyber-nic/git-branches.git
   cd git-branches
   ```

2. Build and deploy the binary to `$(HOME)/go/bin`:

   ```sh
   make deploy
   ```

   This compiles `main.go` and places `git-branches` in `$(HOME)/go/bin`.

### Via `go install`

If you prefer a quick install, run:

```sh
go install github.com/cyber-nic/git-branches@latest
```

Ensure `$(HOME)/go/bin` is in your `PATH`:

```sh
export PATH="$HOME/go/bin:$PATH"
```

## Usage

Run the tool inside any Git repository:

```sh
git-branches
```

- Press **Enter** to refresh the branch list.
- Click the **delete** link on any row to copy the `git branch -d <branch>` command, then confirm removal.

## Development

- To rebuild locally without deploying:

  ```sh
  go build -o git-branches main.go
  ```

- To run tests (if any):

  ```sh
  go test ./...
  ```

## License

MIT © Nicolas Delorme
