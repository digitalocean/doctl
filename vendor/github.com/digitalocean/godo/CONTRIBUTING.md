# Contributing

We love contributions! You are welcome to open a pull request, but it's a good idea to
open an issue and discuss your idea with us first.

Once you are ready to open a PR, please keep the following guidelines in mind:

1. Code should be `go fmt` compliant.
1. Types, structs and funcs should be documented.
1. Tests pass.

## Getting set up

`godo` uses go modules. Just fork this repo, clone your fork and off you go!

## Running tests

When working on code in this repository, tests can be run via:

```sh
go test -mod=vendor .
```

## Versioning

Godo follows [semver](https://www.semver.org) versioning semantics.
New functionality should be accompanied by increment to the minor
version number. Any code merged to main is subject to release.

## Releasing

> [!NOTE]  
> This section is for maintainers. 

Releasing a new version of godo is currently a partially manual process.

1. Run the `Prepare Release` workflow against the `main` branch. This workflow will update `CHANGELOG.md` and the `libraryVersion` in `godo.go` with the next version number.
   - Be sure to set the **_next_** version to the upcoming version. For example, if the latest version is `1.2.3` and you intent to release `1.2.4`, specifiy `1.2.4`.
   - Review the generated PR and merge it to main.
4. Once the pull request has been merged, [draft a new release](https://github.com/digitalocean/godo/releases/new).
5. Update the `Tag version` and `Release title` field with the new godo version.  Be sure the version has a `v` prefixed in both places. Ex `v1.8.0`.
6. Copy the changelog bullet points to the description field.
7. Publish the release.

## Go Version Support

This project follows the support [policy of Go](https://go.dev/doc/devel/release#policy)
as its support policy. The two latest major releases of Go are supported by the project.
[CI workflows](.github/workflows/ci.yml) should test against both supported versions.
[go.mod](./go.mod) should specify the oldest of the supported versions to give
downstream users of godo flexibility.
