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

package labgo

import (
	"crypto/rsa"
	"encoding/base64"
	"github.com/AletheiaWareLLC/aliasgo"
	"github.com/AletheiaWareLLC/bcgo"
	"github.com/AletheiaWareLLC/bcnetgo"
	"github.com/AletheiaWareLLC/cryptogo"
	"github.com/golang/protobuf/proto"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	EXPERIMENT_HASH_LENGTH = 16
	CHANNEL_THRESHOLD      = bcgo.THRESHOLD_H

	LAB_PREFIX = "Lab-"
	//LAB_PREFIX_CHAT = "Lab-Chat-" // labgo.Delta Chain
	LAB_PREFIX_FILE = "Lab-File-" // labgo.Delta Chain
	//LAB_PREFIX_DRAW = "Lab-Draw-" // labgo.Delta Chain
	LAB_PREFIX_PATH = "Lab-Path-" // labgo.Path Chain
)

type Experiment struct {
	ID string
	//Chat *bcgo.Channel
	//Draw *bcgo.Channel
	Path *bcgo.Channel
}

func Init(rootDir string, cache bcgo.Cache, network bcgo.Network, listener bcgo.MiningListener) (*bcgo.Node, error) {
	// Create Node
	node, err := bcgo.GetNode(rootDir, cache, network)
	if err != nil {
		return nil, err
	}

	// Register Alias
	if err := aliasgo.Register(node, listener); err != nil {
		return nil, err
	}

	return node, nil
}

/*
func OpenChatChannel(experimentId string) *bcgo.Channel {
	// TODO(v2) add validator to ensure Delta Payload can be unmarshalled as protobuf
	return bcgo.OpenPoWChannel(LAB_PREFIX_CHAT+experimentId, CHANNEL_THRESHOLD)
}

func OpenDrawChannel(experimentId string) *bcgo.Channel {
	// TODO(v2) add validator to ensure Delta Payload can be unmarshalled as protobuf
	return bcgo.OpenPoWChannel(LAB_PREFIX_DRAW+experimentId, CHANNEL_THRESHOLD)
}
*/

func OpenFileChannel(fileId string) *bcgo.Channel {
	// TODO(v2) add validator to ensure Delta Payload can be unmarshalled as protobuf
	return bcgo.OpenPoWChannel(LAB_PREFIX_FILE+fileId, CHANNEL_THRESHOLD)
}

func OpenPathChannel(experimentId string) *bcgo.Channel {
	// TODO(v2) add validator to ensure Path Payload can be unmarshalled as protobuf
	return bcgo.OpenPoWChannel(LAB_PREFIX_PATH+experimentId, CHANNEL_THRESHOLD)
}

func Clean(node *bcgo.Node, experimentId string) error {
	// TODO remove all blocks from cache
	/*
		// Open Lab-Chat-<id> Chain
		c := OpenChatChannel(experimentId)
		// Open Lab-Draw-<id> Chain
		d := OpenDrawChannel(experimentId)
		// Open Lab-Path-<id> Chain
		p := OpenPathChannel(experimentId)
		if err := bcgo.Read(p.Name, p.Head, nil, node.Cache, node.Network, "", nil, nil, func(entry *bcgo.BlockEntry, key, data []byte) error {
			// Get channel from node
			file, err := node.GetChannel(LAB_PREFIX_FILE + base64.RawURLEncoding.EncodeToString(entry.RecordHash))
			if err != nil {
				return err
			}
			// TODO remove all blocks from cache
			return nil
		})
	*/
	return nil
}

func CreateFromReader(node *bcgo.Node, listener bcgo.MiningListener, uri string, reader io.ReadCloser) (*Experiment, error) {
	// Generate ID
	id, err := cryptogo.RandomString(EXPERIMENT_HASH_LENGTH)
	if err != nil {
		return nil, err
	}
	// Create Lab-Path-<id> Chain
	p := OpenPathChannel(id)
	node.AddChannel(p)
	if uri != "" && reader != nil {
		// TODO truncate uri to remove file:///Users/foobar/...
		log.Println(uri)
		path := strings.Split(uri, string(os.PathSeparator))
		log.Println(path)
		if _, _, err := CreatePathFromReader(node, listener, p, path, reader); err != nil {
			return nil, err
		}
	}
	return &Experiment{
		ID: id,
		//Chat: c,
		//Draw: d,
		Path: p,
	}, nil
}

func CreateFromPaths(node *bcgo.Node, listener bcgo.MiningListener, paths ...string) (*Experiment, error) {
	// Generate ID
	id, err := cryptogo.RandomString(EXPERIMENT_HASH_LENGTH)
	if err != nil {
		return nil, err
	}
	// Create Lab-Chat-<id> Chain
	//c := OpenChatChannel(id)
	//node.AddChannel(c)
	// Create Lab-Draw-<id> Chain
	//d := OpenDrawChannel(id)
	//node.AddChannel(d)
	// Create Lab-Path-<id> Chain
	p := OpenPathChannel(id)
	node.AddChannel(p)
	// Read paths into Lab-File-<id> Chain
	for _, path := range paths {
		if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// Skip directories
				return nil
			}
			if info.Mode()&os.ModeSymlink == os.ModeSymlink {
				// Skip symbolic links
				return nil
			}
			// TODO truncate path to remove /Users/foobar/...
			log.Println(path, info.Name(), info.Size())
			// Create fileId from path
			fileHash, err := WriteProto(node, listener, p, &Path{
				Path: strings.Split(path, string(os.PathSeparator)),
			})
			if err != nil {
				return nil
			}
			// Create Lab-File-<id> Chain
			file := OpenFileChannel(base64.RawURLEncoding.EncodeToString(fileHash))
			node.AddChannel(file)
			return PathToDeltas(path, MAX_DELTA_LENGTH, func(d *Delta) error {
				_, err := WriteProto(node, listener, file, d)
				return err
			})
		}); err != nil {
			return nil, err
		}
	}
	return &Experiment{
		ID: id,
		//Chat: c,
		//Draw: d,
		Path: p,
	}, nil
}

func CreatePath(node *bcgo.Node, listener bcgo.MiningListener, channel *bcgo.Channel, path []string) (string, *bcgo.Channel, error) {
	// Create fileId from path
	fileHash, err := WriteProto(node, listener, channel, &Path{
		Path: path,
	})
	if err != nil {
		return "", nil, err
	}
	// Create Lab-File-<id> Chain
	id := base64.RawURLEncoding.EncodeToString(fileHash)
	file := OpenFileChannel(id)
	node.AddChannel(file)
	return id, file, nil
}

func CreatePathFromReader(node *bcgo.Node, listener bcgo.MiningListener, channel *bcgo.Channel, path []string, reader io.ReadCloser) (string, *bcgo.Channel, error) {
	id, file, err := CreatePath(node, listener, channel, path)
	if err != nil {
		return "", nil, err
	}
	if err := ReaderToDeltas(reader, MAX_DELTA_LENGTH, func(d *Delta) error {
		_, err := WriteProto(node, listener, file, d)
		return err
	}); err != nil {
		return "", nil, err
	}
	return id, file, nil
}

func Open(node *bcgo.Node, experimentId string) (*Experiment, error) {
	// Open Lab-Chat-<id> Chain
	//c := OpenChatChannel(experimentId)
	// Open Lab-Draw-<id> Chain
	//d := OpenDrawChannel(experimentId)
	// Open Lab-Path-<id> Chain
	p := OpenPathChannel(experimentId)

	for _, c := range []*bcgo.Channel{
		//c,
		//d,
		p,
	} {
		// Load channel
		if err := c.LoadCachedHead(node.Cache); err != nil {
			log.Println(err)
		}
		if node.Network != nil {
			// Pull channel from network
			if err := c.Pull(node.Cache, node.Network); err != nil {
				log.Println(err)
			}
		}
		// Add channel to node
		node.AddChannel(c)
	}

	return &Experiment{
		ID: experimentId,
		//Chat: c,
		//Draw: d,
		Path: p,
	}, nil
}

func Save(node *bcgo.Node, experiment *Experiment, path string) error {
	p := experiment.Path
	return bcgo.Read(p.Name, p.Head, nil, node.Cache, node.Network, "", nil, nil, func(entry *bcgo.BlockEntry, key, data []byte) error {
		// Unmarshal as Path
		p := &Path{}
		if err := proto.Unmarshal(data, p); err != nil {
			return err
		}
		filePath := filepath.Join(append([]string{path}, p.Path...)...)
		// Get channel from node
		file, err := node.GetChannel(LAB_PREFIX_FILE + base64.RawURLEncoding.EncodeToString(entry.RecordHash))
		if err != nil {
			return err
		}
		return IterateDeltas(node, file, func(h []byte, r *bcgo.Record, d *Delta) error {
			return DeltaToPath(d, filePath)
		})
	})
}

func Serve(node *bcgo.Node, cache bcgo.Cache, network *bcgo.TCPNetwork) {
	// Serve Connect Requests
	go bcnetgo.BindTCP(bcgo.PORT_CONNECT, bcnetgo.ConnectPortTCPHandler(network))
	// Serve Block Requests
	go bcnetgo.BindTCP(bcgo.PORT_GET_BLOCK, bcnetgo.BlockPortTCPHandler(cache, network))
	// Serve Head Requests
	go bcnetgo.BindTCP(bcgo.PORT_GET_HEAD, bcnetgo.HeadPortTCPHandler(cache, network))
	// Serve Block Updates
	go bcnetgo.BindTCP(bcgo.PORT_BROADCAST, bcnetgo.BroadcastPortTCPHandler(cache, network, func(name string) (*bcgo.Channel, error) {
		channel, err := node.GetChannel(name)
		if err != nil {
			if strings.HasPrefix(name, LAB_PREFIX) {
				channel = bcgo.OpenPoWChannel(name, CHANNEL_THRESHOLD)
				// Load channel
				if err := channel.LoadCachedHead(cache); err != nil {
					log.Println(err)
				}
				// Pull channel
				if err := channel.Pull(cache, network); err != nil {
					log.Println(err)
				}
				// Add channel to node
				node.AddChannel(channel)
			} else {
				return nil, err
			}
		}
		return channel, nil
	}))
}

func WriteProto(node *bcgo.Node, listener bcgo.MiningListener, channel *bcgo.Channel, protobuf proto.Message) ([]byte, error) {
	// Create protobuf record
	hash, record, err := ProtoToRecord(node.Alias, node.Key, bcgo.Timestamp(), protobuf)
	if err != nil {
		return nil, err
	}

	// Write Record to Cache
	if err := node.Cache.PutBlockEntry(channel.Name, &bcgo.BlockEntry{
		RecordHash: hash,
		Record:     record,
	}); err != nil {
		return nil, err
	}

	// Mine Channel
	if _, _, err := node.Mine(channel, CHANNEL_THRESHOLD, listener); err != nil {
		return nil, err
	}

	if node.Network != nil {
		// Push Update to Peers
		if err := channel.Push(node.Cache, node.Network); err != nil {
			return nil, err
		}
	}
	return hash, nil
}

func ProtoToRecord(alias string, key *rsa.PrivateKey, timestamp uint64, protobuf proto.Message) ([]byte, *bcgo.Record, error) {
	// Marshal Protobuf
	data, err := proto.Marshal(protobuf)
	if err != nil {
		return nil, nil, err
	}

	// Create Record
	_, record, err := bcgo.CreateRecord(timestamp, alias, key, nil, nil, data)
	if err != nil {
		return nil, nil, err
	}

	hash, err := cryptogo.HashProtobuf(record)
	if err != nil {
		return nil, nil, err
	}

	return hash, record, nil
}
