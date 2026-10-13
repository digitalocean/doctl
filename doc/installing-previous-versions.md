# Installing a previous version of doctl

Every `doctl` release stays available on the
[Releases page](https://github.com/digitalocean/doctl/releases), so you can go
back to an earlier version at any time, for example from v2 to the last v1
release.

Your configuration is kept when you switch versions. Saved access tokens and
authentication contexts live in the same `config.yaml`, which both v1 and v2
read, so you do not need to sign in again.

The examples below install `1.179.0`. Replace it with the version you need,
without the leading `v`.

- [Default: download a release binary (any OS)](#default-download-a-release-binary-any-os)
- [macOS](#macos)
- [Linux](#linux)
- [Windows](#windows)
- [Docker and Go](#docker-and-go)
- [Returning to the latest version](#returning-to-the-latest-version)

## Default: download a release binary (any OS)

This works on every platform and does not depend on a package manager.

1. Find your archive on the
   [Releases page](https://github.com/digitalocean/doctl/releases). Archives are
   named `doctl-<version>-<os>-<arch>`:

   | OS | `<os>` | `<arch>` | Format |
   |---|---|---|---|
   | macOS, Apple silicon | `darwin` | `arm64` | `.tar.gz` |
   | macOS, Intel | `darwin` | `amd64` | `.tar.gz` |
   | Linux | `linux` | `amd64`, `arm64`, `386` | `.tar.gz` |
   | Windows | `windows` | `amd64`, `arm64`, `386` | `.zip` |

2. Download and extract it, then move `doctl` onto your `PATH`:

   ```sh
   V=1.179.0; OS=linux; ARCH=amd64
   curl -sL https://github.com/digitalocean/doctl/releases/download/v$V/doctl-$V-$OS-$ARCH.tar.gz | tar -xz
   sudo mv doctl /usr/local/bin/
   ```

3. Check the version:

   ```sh
   doctl version
   ```

If `doctl` was installed with a package manager, remove that copy first (see
your platform below). Otherwise the two copies conflict, and whichever comes
first on your `PATH` wins.

## macOS

### Homebrew

Homebrew only offers the latest version and removes the old one when it
upgrades, so it cannot install an earlier release. Replace the Homebrew copy
with the release binary instead:

```sh
brew uninstall doctl
V=1.179.0; ARCH=arm64          # Intel Macs: ARCH=amd64
curl -sL https://github.com/digitalocean/doctl/releases/download/v$V/doctl-$V-darwin-$ARCH.tar.gz | tar -xz
sudo mv doctl /usr/local/bin/
doctl version
```

To stay on your current version before an upgrade comes out, pin it.
`brew upgrade` then skips `doctl` until you unpin it:

```sh
brew pin doctl
brew unpin doctl               # when you are ready to upgrade
```

### Browser downloads

The `curl` commands above work as-is. If you download the archive with a web
browser instead, macOS may refuse to open `doctl` because it was downloaded
from the internet. Clear the flag and try again:

```sh
xattr -d com.apple.quarantine doctl
```

### MacPorts

The `doctl` port is community maintained. Use the release binary above to get
a specific version.

## Linux

### Snap

Snap keeps your previously installed version on the machine, so you can return
to it with one command. Hold the snap afterwards, or Snap's automatic updates
will install the latest version again:

```sh
sudo snap revert doctl
sudo snap refresh --hold doctl
doctl version
```

Run `sudo snap refresh --unhold doctl` to resume updates.

`snap revert` only goes back to the version you had installed before, and only
while Snap still keeps it. To install any other version, replace the snap with
the release binary:

```sh
sudo snap remove doctl
V=1.179.0; ARCH=amd64          # ARCH=arm64 on ARM machines
curl -sL https://github.com/digitalocean/doctl/releases/download/v$V/doctl-$V-linux-$ARCH.tar.gz | tar -xz
sudo mv doctl /usr/local/bin/
doctl version
```

The release binary is not sandboxed, so the `snap connect` steps from the
README are not needed.

### Arch Linux, Fedora, and Nix

These packages are maintained by their distributions and follow the latest
release. Use your distribution's own tools to downgrade, or remove the package
and use the release binary above.

## Windows

Download the `.zip` for your version and replace `doctl.exe`. In PowerShell:

```powershell
$v = "1.179.0"
Invoke-WebRequest "https://github.com/digitalocean/doctl/releases/download/v$v/doctl-$v-windows-amd64.zip" -OutFile doctl.zip
Expand-Archive doctl.zip -DestinationPath "$env:LOCALAPPDATA\doctl" -Force
& "$env:LOCALAPPDATA\doctl\doctl.exe" version
```

If `%LOCALAPPDATA%\doctl` is not on your `PATH` yet, add it under
**Settings > System > About > Advanced system settings > Environment Variables**,
then open a new terminal.

### Windows package managers

The Chocolatey, Scoop, and winget packages are community maintained, and
whether they keep older versions depends on each package. Where they do, these
commands install a specific version:

```powershell
choco install doctl --version 1.179.0 --allow-downgrade
scoop install doctl@1.179.0
winget search doctl                                   # find the package ID
winget install --id <package-id> --version 1.179.0
```

If none of them has the version you need, uninstall the package and use the
`.zip` above.

## Docker and Go

**Docker:** every release has its own image tag, and `<major>-latest` tracks
the newest release of a major version:

```sh
docker run --rm -it -e DIGITALOCEAN_ACCESS_TOKEN digitalocean/doctl:1.179.0 version
docker run --rm -it -e DIGITALOCEAN_ACCESS_TOKEN digitalocean/doctl:1-latest version
```

**Go:** with a Go toolchain installed, build any v1 release directly:

```sh
go install github.com/digitalocean/doctl/cmd/doctl@v1.179.0
```

## Returning to the latest version

Remove the release binary and reinstall with your package manager:

| Installed with | Return to the latest version |
|---|---|
| Release binary on macOS or Linux | `sudo rm /usr/local/bin/doctl`, then reinstall (for example `brew install doctl`) |
| Homebrew pin | `brew unpin doctl && brew upgrade doctl` |
| Snap hold | `sudo snap refresh --unhold doctl && sudo snap refresh doctl` |
| Release binary on Windows | Replace `doctl.exe` with the latest `.zip` from the Releases page |
| Docker | Use the `digitalocean/doctl:latest` tag |
