/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package whisk

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// ActionFunction is the signature of an action in OpenWhisk
type ActionFunction func(event json.RawMessage) (json.RawMessage, error)

// actual implementation of a read-eval-print-loop
func repl(fn ActionFunction, in io.Reader, out io.Writer) {
	// read loop
	reader := bufio.NewReader(in)
	for {
		event, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}
		result, err := fn(event)
		if err != nil {
			fmt.Fprintf(out, "{ error: %q}\n", err.Error())
			continue
		}
		fmt.Fprintln(out, string(result))
	}
}

// Start will start a loop reading in stdin and outputting in fd3
// This is expected to be uses for implementing Go actions
func Start(fn ActionFunction) {
	out := os.NewFile(3, "pipe")
	defer out.Close()
	repl(fn, os.Stdin, out)
}

// StartWithArgs will execute the function for each arg
// If there are no args it will start a read-write loop on the function
// Expected to be used as starting point for implementing Go Actions
// as whisk.StartWithArgs(function, os.Args[:1])
// if args are 2 (command and one parameter) it will invoke the function once
// otherwise it will stat the function in a read-write loop
func StartWithArgs(action ActionFunction, args []string) {
	// handle command line argument
	if len(args) > 0 {
		for _, arg := range args {
			log.Println(arg)
			result, err := action([]byte(arg))
			if err == nil {
				fmt.Println(string(result))
			} else {
				log.Println(err)
			}
		}
		return
	}
	Start(action)
}
