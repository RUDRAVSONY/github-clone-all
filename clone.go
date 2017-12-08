package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

type cloner struct {
	git     string
	dist    string
	ret     [4]chan error
	done    [4]bool
	lastErr error
}

func newCloner(dist string) *cloner {
	c := &cloner{dist: dist}

	c.git = os.Getenv("GIT_EXECUTABLE_PATH")
	if c.git == "" {
		c.git = "git"
	}

	for i := range c.ret {
		c.ret[i] = make(chan error)
	}

	for i := range c.done {
		c.done[i] = true
	}

	return c
}

// Note: This function is run in another goroutine. It should not share the state with cloner so it should not be a method of cloner.
func clone(git, repo, dist string, done chan error) {
	log.Println("Cloning", repo)
	url := fmt.Sprintf("https://github.com/%s.git", repo)
	dir := fmt.Sprintf("%s/%s", dist, repo)
	cmd := exec.Command(git, "clone", "--depth=1", "--single-branch", url, dir)
	err := cmd.Run()
	log.Println("Cloned:", repo)
	done <- err
}

func (cl *cloner) waitOne() (idx int) {
	var err error

	select {
	case err = <-cl.ret[0]:
		cl.done[0] = true
		idx = 0
	case err = <-cl.ret[1]:
		cl.done[1] = true
		idx = 1
	case err = <-cl.ret[2]:
		cl.done[2] = true
		idx = 2
	case err = <-cl.ret[3]:
		cl.done[3] = true
		idx = 3
	}

	if err != nil {
		log.Println("Failed to clone:", err)
		cl.lastErr = err
	}
	return
}

func (cl *cloner) waitDone() {
	for !cl.done[0] || !cl.done[1] || !cl.done[2] || !cl.done[3] {
		cl.waitOne()
	}
}

func (cl *cloner) cloneWith(idx int, repo string) {
	cl.done[idx] = false
	go clone(cl.git, repo, cl.dist, cl.ret[idx])
}

// Clones the repository in other goroutine
func (cl *cloner) clone(repo string) {
	for i, done := range cl.done {
		if done {
			cl.cloneWith(i, repo)
			return
		}
	}
	cl.cloneWith(cl.waitOne(), repo)
}
