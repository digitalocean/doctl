# Change Log

All changes to `doctl` will be documented in this file.

## [1.13.0] - UNRELEASED

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
