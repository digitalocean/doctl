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

//  Samples
//  We include all the samples that we have, historically, from legacy Nimbella.
//  Not all are used because not all runtimes are deployed.

var (
	javascriptSample = `function main(args) {
    let name = args.name || 'stranger'
    let greeting = 'Hello ' + name + '!'
    console.log(greeting)
    return {"body": greeting}
}
`
	typescriptSample = `export function main(args: {}): {} {
    let name: string = args['name'] || 'stranger'
    let greeting: string = 'Hello ' + name + '!'
    console.log(greeting)
    return { body: greeting }
}
`
	pythonSample = `def main(args):
      name = args.get("name", "stranger")
      greeting = "Hello " + name + "!"
      print(greeting)
      return {"body": greeting}
`
	swiftSample = `func main(args: [String:Any]) -> [String:Any] {
      if let name = args["name"] as? String {
          let greeting = "Hello \\(name)!"
          print(greeting)
          return [ "greeting" : greeting ]
      } else {
          let greeting = "Hello stranger!"
          print(greeting)
          return [ "body" : greeting ]
      }
  }
`
	phpSample = `<?php
  function main(array $args) : array
  {
      $name = $args["name"] ?? "stranger";
      $greeting = "Hello $name!";
      echo $greeting;
      return ["body" => $greeting];
  }
`
	javaSample = `import com.google.gson.JsonObject;

  public class Main {
      public static JsonObject main(JsonObject args) {
          String name = "stranger";
          if (args.has("name"))
              name = args.getAsJsonPrimitive("name").getAsString();
          String greeting = "Hello " + name + "!";
          JsonObject response = new JsonObject();
          response.addProperty("body", greeting);
          return response;
      }
  }
`
	goSample = `package main

func Main(args map[string]interface{}) map[string]interface{} {
	name, ok := args["name"].(string)
	if !ok {
		name = "stranger"
	}
	msg := make(map[string]interface{})
	msg["body"] = "Hello " + name + "!"
	return msg
}
`
	rustSample = `extern crate serde_json;

use serde_derive::{Deserialize, Serialize};
use serde_json::{Error, Value};

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
struct Input {
    #[serde(default = "stranger")]
    name: String,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
struct Output {
    body: String,
}

fn stranger() -> String {
    "stranger".to_string()
}

pub fn main(args: Value) -> Result<Value, Error> {
    let input: Input = serde_json::from_value(args)?;
    let output = Output {
        body: format!("Hello {}", input.name),
    };
    serde_json::to_value(output)
}
`
	denoSample = `export default function main(args: {[key: string]: any}) {
  return {
    body: ` + "`" + `Hello \${args.name || "stranger"}!\` + "`" + `,
  };
};
`
	rubySample = `def main(args)
name = args["name"] || "stranger"
greeting = "Hello #{name}!"
puts greeting
{ "body" => greeting }
end
`
	csharpSample = `using System;
using Newtonsoft.Json.Linq;

namespace Nimbella.Example.Dotnet
{
    public class Hello
    {
        public JObject Main(JObject args)
        {
            string name = "stranger";
            if (args.ContainsKey("name")) {
                name = args["name"].ToString();
            }
            JObject message = new JObject();
            message.Add("body", new JValue($"Hello {name}!"));
            return (message);
        }
    }
}
`

	// samples is the official table of samples
	samples = map[string]string{
		"deno":       denoSample,
		"cs":         csharpSample,
		"csharp":     csharpSample,
		"go":         goSample,
		"golang":     goSample,
		"java":       javaSample,
		"javascript": javascriptSample,
		"js":         javascriptSample,
		"php":        phpSample,
		"py":         pythonSample,
		"python":     pythonSample,
		"ruby":       rubySample,
		"rust":       rustSample,
		"swift":      swiftSample,
		"ts":         typescriptSample,
		"typescript": typescriptSample,
	}

	// gitignores contains the contents of a "standard" .gitigore file
	// Note that we do not attempt to list typical IDE and editor temporaries here.
	// It is considered best practice for developers to list these in a personal global
	// ignore file (`core.excludesfile` in the git config) and not in a committed .gitignore.
	gitignores = `.nimbella
.deployed
__deployer__.zip
__pycache__/
node_modules
package-lock.json
.DS_Store
`

	// ignoreForTypescript is added when the project is typescript
	ignoreForTypescript = "lib/\n"

	// packageJSONForTypescript is a canned package.json for a minimal typescript project
	packageJSONForTypescript = `{
  "main": "lib/hello.js",
  "devDependencies": {
    "typescript": "^4"
  },
  "scripts": {
    "build": "tsc -b"
  }
}
`

	// tsconfigJSON is a canned tsconfig.json for typescript project
	tsconfigJSON = `{
  "compilerOptions": {
    "baseUrl": ".",
    "esModuleInterop": true,
    "importHelpers": true,
    "module": "commonjs",
    "outDir": "lib",
    "rootDir": "src",
    "target": "es2019"
  },
  "include": [
    "src/**/*"
  ]
}
`
)
