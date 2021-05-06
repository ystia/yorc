# Contributing to the Ystia Orchestrator project

**First off**, thanks for taking time to contribute!

The following is a set of guidelines for contributing to Yorc.
Feel free to provide feedback about it in an
issue or pull request.

Don't be afraid to contribute, if something is unclear then just ask or submit the issue or pull request
anyways. The worst that can happen is that you'll be politely asked to change something.

## How to build Yorc from source

Go 1.11+ is required. The easiest way to install it to follow [the official guide](https://golang.org/doc/install)

Here is how to install and setup the Yorc project:

    sudo apt-get install build-essential git curl
    # Or
    sudo yum install build-essential git curl

    # Install Go

    git clone https://github.com/ystia/yorc.git
    cd yorc

    # Build
    make tools
    make

Using `make` you will automatically generate required code, run unit tests and format code with `go fmt`.

## How to contribute

You can contribute to the Yorc project in several ways. All of them are welcome.

* Report a documentation issue
* Report a bug
* Report an improvement request
* Propose a PR that fixes one of the above, we will try to tag `good first issue` issues that are a good starting point for contributing

### Report an issue

Use [Github issues](https://github.com/ystia/yorc/issues) to report issues.
Please try to answer most of the questions asked in the issue template.

### Propose a Pull Request

**Working on your first Pull Request?** You can learn how from this *free* series [How to Contribute to an Open Source Project on GitHub](https://egghead.io/series/how-to-contribute-to-an-open-source-project-on-github)

Use [Github pull requests](https://github.com/ystia/yorc/pulls) to propose a PR.
Please try to answer most of the questions asked in the pull request template.

## Coding Style

<!-- From the moby project https://github.com/moby/moby -->

Unless explicitly stated, we follow all coding guidelines from the Go
community. While some of these standards may seem arbitrary, they somehow seem
to result in a solid, consistent codebase.

It is possible that the code base does not currently comply with these
guidelines. We are not looking for a massive PR that fixes this, since that
goes against the spirit of the guidelines. All new contributions should make a
best effort to clean up and make the code base better than they left it.
Obviously, apply your best judgement. Remember, the goal here is to make the
code base easier for humans to navigate and understand. Always keep that in
mind when nudging others to comply.

The rules:

1. All code should be formatted with `go fmt`.
2. All code should pass the default levels of
   [`golint`](https://github.com/golang/lint).
3. All code should follow the guidelines covered in [Effective
   Go](http://golang.org/doc/effective_go.html) and [Go Code Review
   Comments](https://github.com/golang/go/wiki/CodeReviewComments).
4. Comment the code. Tell us the why, the history and the context.
5. Document _all_ declarations and methods, even private ones. Declare
   expectations, caveats and anything else that may be important. If a type
   gets exported, having the comments already there will ensure it's ready.
6. Variable name length should be proportional to its context and no longer.
   `noCommaALongVariableNameLikeThisIsNotMoreClearWhenASimpleCommentWouldDo`.
   In practice, short methods will have short variable names and globals will
   have longer names.
7. No underscores in package names. If you need a compound name, step back,
   and re-examine why you need a compound name. If you still think you need a
   compound name, lose the underscore.
8. All tests should run with `go test` and outside tooling should not be
   required. No, we don't need another unit testing framework. Assertion
   packages are acceptable if they provide _real_ incremental value.
9. Even though we call these "rules" above, they are actually just
    guidelines. Since you've read all the rules, you now know that.

If you are having trouble getting into the mood of idiomatic Go, we recommend
reading through [Effective Go](https://golang.org/doc/effective_go.html). The
[Go Blog](https://blog.golang.org) is also a great resource. Drinking the
kool-aid is a lot easier than going thirsty.

## Release Yorc

Releases are now handled by a [GitHub Action Workflow](https://github.com/ystia/yorc/actions/workflows/release.yml).
Contributors with `members` role on this project can trigger this workflow. it requires as an input the release version
in semver format (leading 'v' should be omitted).

This workflow will:

1. call the `build/release.sh` script
2. checkout the generated tag
3. generate the distribution and a changelog for this version
4. create a GH Release and upload assets
5. publish the GH Release
