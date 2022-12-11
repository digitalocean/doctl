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

// Contains constants used by various serverless command source files

const (
	cmdDeploy        = "deploy"
	cmdGetMetadata   = "get-metadata"
	cmdWatch         = "watch"
	flagURL          = "url"
	flagCode         = "code"
	flagSave         = "save"
	flagSaveEnv      = "save-env"
	flagSaveEnvJSON  = "save-env-json"
	flagSaveAs       = "save-as"
	flagWeb          = "web"
	flagNoWait       = "no-wait"
	flagParamFile    = "param-file"
	flagNameSort     = "name-sort"
	flagNameName     = "name" // avoid conflict with flagName, which is a function
	flagParam        = "param"
	flagLast         = "last"
	flagLogs         = "logs"
	flagResult       = "result"
	flagQuiet        = "quiet"
	flagSkip         = "skip"
	flagAction       = "action"
	flagCount        = "count"
	flagFull         = "full"
	flagLimit        = "limit"
	flagSince        = "since"
	flagUpto         = "upto"
	flagStrip        = "strip"
	flagFollow       = "follow"
	flagDeployed     = "deployed"
	flagPackage      = "package"
	flagFunction     = "function"
	flagOverwrite    = "overwrite"
	flagLanguage     = "language"
	flagInsecure     = "insecure"
	flagVerboseBuild = "verbose-build"
	flagVerboseZip   = "verbose-zip"
	flagYarn         = "yarn"
	flagRemoteBuild  = "remote-build"
	flagIncremental  = "incremental"
	flagEnv          = "env"
	flagBuildEnv     = "build-env"
	flagApihost      = "apihost"
	flagAuth         = "auth"
	flagInclude      = "include"
	flagExclude      = "exclude"
	flagJSON         = "json"
	keywordWeb       = "web"
	flagNoTriggers   = "no-triggers"
)
