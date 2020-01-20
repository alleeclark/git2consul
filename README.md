# go-git2consul

[![Go Report Card](https://goreportcard.com/badge/github.com/alleeclark/git2consul)](https://goreportcard.com/report/github.com/alleeclark/git2consul) [![Build Status](https://dev.azure.com/alleeclark0813/git2consul/_apis/build/status/alleeclark.git2consul%20(2)?branchName=master)](https://dev.azure.com/alleeclark0813/git2consul/_build/latest?definitionId=3&branchName=master)[![](https://godoc.org/github.com/nathany/looper?status.svg)](http://godoc.org/github.com/alleeclark/git2consul)


**Description**:  git2consul syncs content from a git repo into consul.

  - **Technology stack**: Written in go with bindings to libtgit2. Libgit2 provides easier access to lowerlevel git calls than other git projects. Supports prometheus metric format for metric collection. Leverages Consul for locking and service registration.
  - **Status**:  Early stages testing, but release coming soon.

----

## Usage

At least two commits are needed in the repository to be able to diff against the latest tree and previous tree.

git2consul does not yet manage it's sycing service in the background. You must run it in a cron and use the same cron frequency as the `--since` flag. I will write a systemD service and timer file to help get started.

Run a sync from git2consul on a 1 second interval. This will sync only new commits from git into Consul.
```bash
git2consul --consul-addr="172.17.0.1:8500" --git-url="https://github.com/alleeclark/test-git2consul.git" sync --since 1
```

Register git2consul as a consul service
service registration
```bash
git2consul operator register
```

## Credits and references

1.[ Projects that inspired you](https://github.com/breser/git2consul)


#### Features
- Commit only changes to consul on an interval
- Full sync
