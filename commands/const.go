/*
Copyright 2018 The Doctl Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

// Contains constants shared by multiple source files of the package

const (
	actionGet          = "action/get"
	flagURL            = "url"
	flagCode           = "code"
	flagSave           = "save"
	flagSaveEnv        = "save-env"
	flagSaveEnvJson    = "save-env-json"
	flagSaveAs         = "save-as"
	actionInvoke       = "action/invoke"
	flagWeb            = "web"
	flagNoWait         = "no-wait"
	flagParamFile      = "param-file"
	actionList         = "action/list"
	flagNameSort       = "name-sort"
	flagNameName       = "name" // avoid conflict with flagName, which is a function
	flagParam          = "param"
	dashdashParam      = "--param"
	activationGet      = "activation/get"
	flagLast           = "last"
	flagLogs           = "logs"
	flagResult         = "result"
	flagQuiet          = "quiet"
	flagSkip           = "skip"
	flagAction         = "action"
	activationList     = "activation/list"
	flagCount          = "count"
	flagFull           = "full"
	flagLimit          = "limit"
	flagSince          = "since"
	flagUpto           = "upto"
	activationLogs     = "activation/logs"
	flagStrip          = "strip"
	flagFollow         = "follow"
	flagDeployed       = "deployed"
	flagPackage        = "package"
	activationResult   = "activation/result"
	flagFunction       = "function"
	projectCreate      = "project/create"
	flagOverwrite      = "overwrite"
	flagLanguage       = "language"
	projectDeploy      = "project/deploy"
	flagInsecure       = "insecure"
	flagVerboseBuild   = "verbose-build"
	flagVerboseZip     = "verbose-zip"
	flagYarn           = "yarn"
	flagRemoteBuild    = "remote-build"
	flagIncremental    = "incremental"
	flagEnv            = "env"
	flagBuildEnv       = "build-env"
	flagApihost        = "apihost"
	flagAuth           = "auth"
	flagInclude        = "include"
	flagExclude        = "exclude"
	projectGetMetadata = "project/get-metadata"
	flagJSON           = "json"
	projectWatch       = "project/watch"
	keywordWeb         = "web"
)
