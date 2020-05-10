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
	"errors"
	"fmt"
	"github.com/AletheiaWareLLC/bcgo"
	"github.com/golang/protobuf/proto"
	"io"
	"os"
)

const (
	MAX_DELTA_LENGTH = uint64(8 * 1024 * 1024) // 8Mb
)

func PathToDeltas(path string, max uint64, callback func(*Delta) error) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return ReaderToDeltas(file, max, callback)
}

func ReaderToDeltas(reader io.Reader, max uint64, callback func(*Delta) error) error {
	var offset int
	buffer := make([]byte, max)
	for {
		count, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		add := make([]byte, count)
		copy(add, buffer[:count])
		if err := callback(&Delta{
			Offset: uint64(offset),
			Add:    add,
		}); err != nil {
			return err
		}
		offset += count
	}
	return nil
}

func IterateDeltas(node *bcgo.Node, delta *bcgo.Channel, callback func([]byte, *bcgo.Record, *Delta) error) error {
	// Iterate through chain chronologically
	return bcgo.IterateChronologically(delta.Name, delta.Head, nil, node.Cache, node.Network, func(hash []byte, block *bcgo.Block) error {
		for _, entry := range block.Entry {
			// Unmarshal as Delta
			d := &Delta{}
			if err := proto.Unmarshal(entry.Record.Payload, d); err != nil {
				return err
			}
			if err := callback(entry.RecordHash, entry.Record, d); err != nil {
				return err
			}
		}
		return nil
	})
}

func DeltaToBuffer(delta *Delta, buffer []byte) (result []byte) {
	length := uint64(len(buffer))
	if delta.Offset <= length {
		result = append(result, buffer[:delta.Offset]...)
	}
	result = append(result, delta.Add...)
	index := delta.Offset + uint64(len(delta.Remove))
	if index < length {
		result = append(result, buffer[index:]...)
	}
	return
}

func DeltaToPath(delta *Delta, path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	offset := delta.Offset + uint64(len(delta.Remove))
	remaining := uint64(info.Size()) - offset
	buffer := make([]byte, remaining)
	if count, err := f.ReadAt(buffer, int64(offset)); err != nil {
		return err
	} else if uint64(count) != remaining {
		return errors.New(fmt.Sprintf("Could not read remaining; expected '%d', got '%d'", remaining, count))
	}
	if count, err := f.WriteAt(delta.Add, int64(delta.Offset)); err != nil {
		return err
	} else if count != len(delta.Add) {
		return errors.New(fmt.Sprintf("Could not write addition; expected '%d', got '%d'", len(delta.Add), count))
	}
	if count, err := f.WriteAt(buffer, int64(delta.Offset)+int64(len(delta.Add))); err != nil {
		return err
	} else if uint64(count) != remaining {
		return errors.New(fmt.Sprintf("Could not write remaining; expected '%d', got '%d'", remaining, count))
	}
	return f.Truncate(int64(delta.Offset) + int64(len(delta.Add)) + int64(remaining))
}
