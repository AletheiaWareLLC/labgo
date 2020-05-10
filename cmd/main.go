/*
 * Copyright 2020 Aletheia Ware LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/AletheiaWareLLC/bcgo"
	"github.com/AletheiaWareLLC/cryptogo"
	"github.com/AletheiaWareLLC/labgo"
	"io"
	"log"
	"os"
)

var peer = flag.String("peer", "", "Lab peer")

func PrintUsage(output io.Writer) {
	fmt.Fprintln(output, "Lab Usage:")
	fmt.Fprintf(output, "\t%s - display usage\n", os.Args[0])
	fmt.Fprintf(output, "\t%s init - initializes environment, generates key pair, and registers alias\n", os.Args[0])
	fmt.Fprintln(output)
	fmt.Fprintf(output, "\t%s create <path> - creates a new experiment from the given path\n", os.Args[0])
	fmt.Fprintf(output, "\t%s open <experiment> - opens an existing experiment\n", os.Args[0])
	fmt.Fprintf(output, "\t%s save <experiment> <path> - saves an existing experiment to the given path\n", os.Args[0])
}

func PrintLegalese(output io.Writer) {
	fmt.Fprintln(output, "Lab Legalese:")
	fmt.Fprintln(output, "Lab is made available by Aletheia Ware LLC [https://aletheiaware.com] under the Terms of Service [https://aletheiaware.com/terms-of-service.html] and Privacy Policy [https://aletheiaware.com/privacy-policy.html].")
	fmt.Fprintln(output, "This beta version of Lab is made available under the Beta Test Agreement [https://aletheiaware.com/lab-beta-test-agreement.html].")
	fmt.Fprintln(output, "By continuing to use this software you agree to the Terms of Service, Privacy Policy, and Beta Test Agreement.")
}

func PrintNode(output io.Writer, node *bcgo.Node) error {
	fmt.Fprintln(output, node.Alias)
	publicKeyBytes, err := cryptogo.RSAPublicKeyToPKIXBytes(&node.Key.PublicKey)
	if err != nil {
		return err
	}
	fmt.Fprintln(output, base64.RawURLEncoding.EncodeToString(publicKeyBytes))
	return nil
}

func main() {
	// Parse command line flags
	flag.Parse()

	// Set log flags
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load config files (if any)
	err := bcgo.LoadConfig()
	if err != nil {
		log.Fatal("Could not load config: %w", err)
	}

	// Get root directory
	rootDir, err := bcgo.GetRootDirectory()
	if err != nil {
		log.Fatal("Could not get root directory: %w", err)
	}

	// Get cache directory
	cacheDir, err := bcgo.GetCacheDirectory(rootDir)
	if err != nil {
		log.Fatal("Could not get cache directory: %w", err)
	}

	// Create file cache
	cache, err := bcgo.NewFileCache(cacheDir)
	if err != nil {
		log.Fatal("Could not create file cache: %w", err)
	}

	// Create network of peers
	network := bcgo.NewTCPNetwork()
	for _, p := range bcgo.SplitRemoveEmpty(*peer, ",") {
		if err := network.Connect(p, []byte("")); err != nil {
			fmt.Println(err)
		}
	}

	// Handle args
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "init":
			PrintLegalese(os.Stdout)
			node, err := labgo.Init(rootDir, cache, network, &bcgo.PrintingMiningListener{Output: os.Stdout})
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Initialized")
			if err := PrintNode(os.Stdout, node); err != nil {
				log.Fatal(err)
			}
		case "create":
			if len(args) > 1 {
				node, err := bcgo.GetNode(rootDir, cache, network)
				if err != nil {
					log.Fatal(err)
				}
				experiment, err := labgo.CreateFromPaths(node, &bcgo.PrintingMiningListener{Output: os.Stdout}, args[1])
				if err != nil {
					log.Fatal(err)
				}
				log.Println(experiment)
			} else {
				log.Fatal("Usage: create [path]")
			}
		case "open":
			if len(args) > 1 {
				node, err := bcgo.GetNode(rootDir, cache, network)
				if err != nil {
					log.Fatal(err)
				}
				experiment, err := labgo.Open(node, args[1])
				if err != nil {
					log.Fatal(err)
				}
				log.Println(experiment)
			} else {
				log.Fatal("Usage: open [experiment]")
			}
		case "save":
			if len(args) > 2 {
				node, err := bcgo.GetNode(rootDir, cache, network)
				if err != nil {
					log.Fatal(err)
				}
				experiment, err := labgo.Open(node, args[1])
				if err != nil {
					log.Fatal(err)
				}
				if err := labgo.Save(node, experiment, args[2]); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal("Usage: save [experiment] [path]")
			}
		default:
			log.Fatal("Cannot handle: ", args[0])
		}
	} else {
		PrintUsage(os.Stdout)
	}
}
