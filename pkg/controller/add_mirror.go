package controller

import (
	"github.com/grahamhayes/git-repo-operator/pkg/controller/mirror"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, mirror.Add)
}
