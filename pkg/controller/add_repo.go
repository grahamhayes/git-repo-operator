package controller

import (
	"github.com/grahamhayes/git-repo-operator/pkg/controller/repo"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, repo.Add)
}
