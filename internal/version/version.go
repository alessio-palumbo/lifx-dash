package version

import "fmt"

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = ""
)

func Print() {
	fmt.Printf("Version: %s\nCommit: %s\nBuilt at: %s\n", Version, Commit, BuildTime)
}
