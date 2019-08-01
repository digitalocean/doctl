# Change Log

Changelog moved to Release Notes in [Github Releases](https://github.com/digitalocean/doctl/releases)

## [1.24.1] = 2019-07-29

- PR #525 - @hilary - release containers to dockerhub

## [1.24.0] = 2019-07-29

- PR #523 - @hilary - explicitly include .kube/config.lock in snapcraft.yml
- PR #522 - @bentranter - Migrate from mockery to gomock
- PR #521 - @hilary - release in one step leveraging goreleaser
- PR #520 - @hilary - update vendored modules
- PR #519 - @hilary - add shellcheck to travis config
- PR #518 - @hilary - update snapcraft config per snapcraft
- PR #517 - @hilary - remove antique bintray install/release code
- PR #515 - @hilary - Bump and tag
- PR #514 - @hilary - Remove hardcoded version
- PR #513 - @hilary - fix make docker_build

## [1.23.1] = 2019-07-22

- PR #510 - @bentranter - Add .exe suffix to Windows binary filenames
- PR #508 - @bentranter - Fix distfile basename in staging script

## [1.23.0] = 2019-07-22

- PR #505 - @sunny-b - dbaas: add private-network-uuid to create, migrate, and replica create
- PR #503 - @hilary - try proposed work-around for snap in production
- PR #502 - @hilary - adopt shellcheck
- PR #501 - @hilary - add descriptions to `make help`

## [1.22.0] = 2019-07-18

- PR #498 - @hilary - Makefile improvements
- PR #497 - @andrewsomething - Re-add the logic setting LDFLAGS in scripts/stage.sh (Fixes: #496).
- PR #494 - @bentranter - Write JSON for nil slices as [] instead of null
- PR #491 - @hilary - build snap using go mod vendor
- PR #490 - @hilary - consolidate version logic

## [1.21.1] - 2019-07-14

- PR #488 - @hilary - add make changelog using github-release-notes
- PR #487 - @hilary - clean build dir before building
- PR #486 - @hilary - ensure tags are up to date before using

## [1.21.0] - 2019-07-11

- PR #482 - @eddiezane - Allow completion of aliases in zsh
- PR #477 - @hilary - describe current status of snap in README
- PR #474 - @DazWilkin - Support "Are you sure?" prompting
- PR #473 - @KritR - Added NixPkgs as Package Manager Option
- PR #472 - @hilary - migrate from go dep to go modules

## [1.20.1] - 2019-06-14

 - #471 add macport to README - @hilary
 - #469 set default location for config file to configPath - @hilary
 - #468 advise when not to use snap, revise README for flow - @hilary
 - #467 set snap version using tag - @hilary

## [1.20.0] - 2019-06-11

 - #465 fix typo in CONTRIBUTING.md - @senechko
 - #463 Add kubernetes delete-node and replace-node commands, deprecate and hide recycle - @bouk
 - #456 tag the release with the release tag - @hilary

## [1.19.0] - 2019-05-31

 - #454 Ensure all 'get' and 'list' commands support the 'format' flag - @andrewsomething
 - #453 fix doctl compute ssh in snap - @hilary
 - #450 update snap build - @hilary
 - #447 Unhide the command "completion" + prevent it from being autocompleted - @kamaln7

## [1.18.0] - 2019-05-15

 - #443 Remove beta flag from Kubernetes commands. - @adamwg
 - #442 Add support for Kubernetes cluster upgrades. - @adamwg
 - #440 Add flag to set local KubeConfig's current-context. - @eddiezane
 - #426 Add support for configuring Kubernetes maintenance windows. - @fatih

## [1.17.0] - 2019-05-08

 - #438 Remove need to opt-in to database commands. - @andrewsomething
 - #420 Allow creating Volumes from a Snapshots. - @bentranter

## [1.16.0] - 2019-04-25

- #431 Godo v1.13.0 + Tag Support for Volumes + Vol Snapshots - @jcodybaker
- #430 fix --tag for kubernetes create/update - @jcodybaker
- #429 remove flaky tip from travis - @hilary
- #428 Disable terminal ECHO flag when prompting for auth token - @waynr
- #427 add mock assertions for database service - @sunny-b

## [1.15.0] - 2019-04-9

- #422 update CONTRIBUTING.md with info on how to update vendored code - @mregmi
- #421 Add support for managed databases - @sunny-b

## [1.14.0] - 2019-03-11

- #415 Add support for custom domains in Spaces CDN - @xornivore
- #414 Clean up out of sync vendor deps - @xornivore
- #408 k8s: Fix case where kube.Get returns a nil cluster - @bouk
- #401 k8s: Fetch credentials after cluster is provisioned - @bouk
- #398 Link to docs to create a Github token - @bouk
- #392 Simplify newline trimming in retrieveUserInput - @timoreimann

## [1.13.0] - 2019-01-16

- #393 Fix linter violations - @timoreimann
- #391 doks: Fix node-pool flags when creating cluster - @bouk
- #388 errors: Don't print superfluous newlines when logging - @bouk
- #387 k8s: Use ExecCredential for authentication - @bouk

## [1.12.2] - 2018-12-09

- #383 Fix bad default for cluster node sizes, improve help and warn of kubeconfig expiry date.

## [1.12.1] - 2018-12-09

- #354 volumes: Fix droplet ID display when listing volumes - @adamwg
- #357 Change snap to classic confinement - @itbm
- #361 Update help for multi-argument commands - @bengadbois
- #369 No longer require ip-address when creating domains - @andrewsomething
- #372 Add handling of kubeconfig git files for kubernetes commands - @aybabtme
- #379 Expose suitable regions, versions and node sizes in kubernetes commands - @aybabtme

## [1.12.0] - 2018-11-26

- #370 Projects API is no longer in beta. See https://developers.digitalocean.com/documentation/v2/#projects for more details - @mchitten
- #365 Add support for kubernetes API [beta] - @aybabtme

## [1.11.0] - 2018-10-01

- #348 Add support for projects API [beta] - @mchitten

## [1.10.0] - 2018-10-01

- #348 Add support for tagging Images. - @hugocorbucci

## [1.9.0] - 2018-08-27

- #343 Add support for Spaces CDN. - @sunny-b

## [1.8.3] - 2018-06-13

- #326 Fix required arguments (#325). - @adamwg

## [1.8.2] - 2018-06-12

- #323 Add support for formatted volumes - @adamwg

## [1.8.1] - 2018-05-09

### Added
- #313 Add support for Let's Encrypt certificates - @viola

## [1.8.0] - 2018-04-09

- #295 commands: fix configuration file location for windows xp users - @xmudrii
- #296 Confirm dialog for deleting by ID now specifies number of droplets to be deleted - @justinbeal
- #299 Implement context switching, allowing for multiple configured API access keys - @kamaln7

## [1.7.2] - 2018-03-07

- #186 ssh: windows support for command forwarding - @xmudrii
- #280 commands: show public images for distros and apps by default - @mudrii
- #294 Respect access token flag when calling init as well - @mauricio
- #291 Adds `SizeSlug` to format fields - @lxfontes
- #284 commands: General simplifications - @ferhatelmas
- #282 xdg: fix config path when XDG_CONFIG_HOME is set - @mudrii
- #278 firewall: omit the port field for the icmp - @caglar10ur

## [1.7.1] - 2017-06-06

- #267 Add flag for overriding API endpoint - @utlemming

## [1.7.0] - 2017-06-06

### Added
- #234 Implement firewall commands - @viola

## [1.6.1] - 2017-05-17

### Added
- #202 Including missing API endpoints for doctl - @xmudrii
- #206 Bash and ZSH completion - @xmudrii
- #220 domains: Add TTL field - @xmudrii

### Changed
- #210 Deprecate tag rename (PUT /v2/tags/:name) - @mchitten
- #208 Remove Detach function - @xmudrii
- #215 Allow certificate-chain-path to be optional - @viola
- #214 Rename DetachByDropletID function to Detach - @xmudrii
- #217 Minimize Docker build context - @SISheogorath
- #228 Upgrading doctl version - @mauricio
- #224 consistency changes: ask for confirm added to all delete actions - @xmudrii
- #222 docs: improve package manager part in readme - @xmudrii

### Fixed
- #198 Hiding public images by deault - @xmudrii
- #194 Use apk's --no-cache option instead of updating and removing cache - @cspicer
- #201 fix vektra/{errors,mockery} to static vendoring instead submodule - @zchee
- #223 completion: fix command description typos - @xmudrii
- #225 completion: make completion code generation independent on auth status - @xmudrii

## [1.6.0] - 2017-03-10

### Added
- #146 Add the option to run doctl within a docker container - @FuriKuri
- #153 Add ability to delete snapshots and confirmation before delete - @xmudrii
- #158 Add private IPv4 address to output - @johscheuer
- #161 Add multiple tag delete, tag delete confirmation - @xmudrii
- #165 Allow "tags" attribute for droplet create - @mchitten
- #169 Implement new unified Snapshots API - @xmudrii
- #173 Add new actions for Volume and Volume-Action - @xmudrii
- #191 Add certificate commands - @viola
- #193 Implement load-balancer commands - @viola

### Changed
- #160 Improve tag handling for droplet create - @akshaychhajed
- #171 Add shorthand flags - @xmudrii

### Fixed
- #159 Add Volume filtering - @xmudrii
- #177 Fix output color - @xmudrii
- #178 Implement command forwarding for external SSH - @xmudrii
- #196 Add type transfer to image-action transfer - @xmudrii

## [1.5.0] - 2016-10-10

### Added
- #144 SSH to private Droplet IP - @haz-mat
- #148 Add confirmation for destructive actions - @xmudrii

### Changed
- #121 Allow untagging droplets - @bryanl

### Fixed
- #126 Fix location of config in help - @bryanl
- #132 Update download location - @xmudrii
- #135 Fix location of doctl configuration - @xmudrii
- #143 Send progress report to stderr - @kkpoon

## [1.4.0] - 2016-08-02

### Added
- #111 Replace `auth login` with `auth init` - @bryanl

### Changed
- #118 Add doctl version to user agent - @bryanl
- #122 Add SSH agent forward support for Windows - @tbalthazar

### Fixed
- #113 Update SSH client support for Windows - @tbalthazar
- #117 Update download links - @garykrige
- #123 Use Windows compatible ANSI colors - @bryanl
- #125 Create valid JSON when creating multiple droplets - @snoopdouglas

## [1.3.1] - 2016-07-13

### Changed
- #99 Build test bin in out directory - @bryanl
- #104 Remove beta status for storage - @bryanl

### Fixed

- #100 password-reset, not power-reset - @aybabtme

## [1.3.0] - 2016-06-25

### Added
- #88 Add a --ssh-agent-forwarding-flag - @tbalthazar

### Changed
- #92 Rename drive to volume - @aybabtme
- #93 Extract token retrieval process - @bryanl
- #98 Remove output when deleting droplets - @bryanl

### Fixed
- #85 Don't report new release available when already installed - @andrewsomething
- #87 Update help output in documentation - @gmontalvoriv
- #97 User list images returns distributions - @bryanl

## [1.2.0] - 2016-06-07

### Added
- #37 Add a script to regenerate test mocks - @bryanl

### Changed
- #79 Ensure pagination is 1 indexed, and not 0 indexed - @jphines

### Fixed
- #68 Respect ssh-user flag - @vkurchatkin
- #69 Fix type in README - @aybabtme
- #70 Fix type in README - @aybabtme
- #74 Add more specific install instructions to README - @aybabtme
- #80 Fix a typo in usage text - @davidkuridza

## [1.1.0] - 2016-04-22

### Added
- #52 Add tagging commands - @bryanl
- #56 Add support for Drive beta - @aybabtme
- #58 Add support for beta features - @bryanl
- #63 Add ci build for windows - @bryanl

### Changed
- #53 Remove unused dependencies - @aybabtme
- #54 Rename root path - @aybabtme
- #57 Clean up version message - @aybabtme
- #61 Print drive columns if beta is enabled - @aybabtme

### Fixed
- #60 Disable tracing by default - @bryanl

## [1.0.2] - 2016-04-14

### Added
- #51 Adding change log - @bryanl

### Changed
- #41 All compute actions with `list` now have `ls` alias - @andrewsomething
- #44 Clean up references to doit - @aybabtme

### Fixed
- #49 Fix image argument to allow rebuilding droplets - @bryanl
