package pkg

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

func NewGitRepo(url string, directory string, branch string) (*git.Repository, string, error) {
	// CheckArgs("<url>", "<directory>", "<branch>")
	// url, directory, branch := os.Args[1], os.Args[2], os.Args[3]
	comp, _ := ExtractURLComponent(url)
	directory = path.Join(directory, comp)

	// Clone the given repository to the given directory
	fmt.Printf("git clone %s %s", url, directory)
	r, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil && err.Error() != "repository already exists" {
		return nil, "", fmt.Errorf("Error cloning: %v", err)
	}

	// ... retrieving the commit being pointed by HEAD
	fmt.Println("git show-ref --head HEAD")
	ref, err := r.Head()
	if err != nil {
		return nil, "", fmt.Errorf("Error getting HEAD: %v", err)
	}

	fmt.Println(ref.Hash())

	w, err := r.Worktree()
	if err != nil {
		return nil, "", fmt.Errorf("Error getting worktree: %v", err)
	}

	// ... checking out branch
	fmt.Printf("git checkout %s", branch)

	branchRefName := plumbing.NewBranchReferenceName(branch)
	branchCoOpts := git.CheckoutOptions{
		Branch: plumbing.ReferenceName(branchRefName),
		Force:  true,
	}
	if err := w.Checkout(&branchCoOpts); err != nil {
		fmt.Printf("local checkout of branch '%s' failed, will attempt to fetch remote branch of same name.", branch)
		fmt.Printf("like `git checkout <branch>` defaulting to `git checkout -b <branch> --track <remote>/<branch>`")

		mirrorRemoteBranchRefSpec := fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch)
		err = fetchOrigin(r, mirrorRemoteBranchRefSpec)
		if err != nil {
			return nil, "", fmt.Errorf("Error fetching origin: %v", err)
		}

		err = w.Checkout(&branchCoOpts)
		if err != nil {
			return nil, "", fmt.Errorf("Error checking out branch: %v", err)
		}
	}
	if err != nil {
		return nil, "", fmt.Errorf("Error checking out branch: %v", err)
	}

	fmt.Printf("checked out branch: %s", branch)

	// ... retrieving the commit being pointed by HEAD (branch now)
	fmt.Printf("git show-ref --head HEAD")
	ref, err = r.Head()
	if err != nil {
		return nil, "", fmt.Errorf("Error getting HEAD: %v", err)
	}
	fmt.Println(ref.Hash())
	return r, directory, nil
}

func fetchOrigin(repo *git.Repository, refSpecStr string) error {
	remote, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("Error getting remote: %v", err)
	}

	var refSpecs []config.RefSpec
	if refSpecStr != "" {
		refSpecs = []config.RefSpec{config.RefSpec(refSpecStr)}
	}

	if err = remote.Fetch(&git.FetchOptions{
		RefSpecs: refSpecs,
	}); err != nil {
		if err == git.NoErrAlreadyUpToDate {
			fmt.Print("refs already up to date")
		} else {
			return fmt.Errorf("fetch origin failed: %v", err)
		}
	}

	return nil
}

func ExtractURLComponent(urlStr string) (string, error) {
	// https://github.com/comfyanonymous/ComfyUI.git would produce ComfyUI
	// Parse the URL to extract the path
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// Extract the last part of the path
	lastPart := path.Base(u.Path)

	// Remove the extension, if present
	return strings.TrimSuffix(lastPart, path.Ext(lastPart)), nil
}
