/*
 *    Copyright 2019 Insolar Technologies
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package storage

import (
	"github.com/insolar/insolar/core"
)

// JetParent returns a parent of the jet
func JetParent(id core.JetID) core.JetID {
	depth, prefix := id.Depth(), id.Prefix()
	if depth == 0 {
		return id
	}

	return *core.NewJetID(depth-1, ResetBits(prefix, depth-1))
}

// ResetBits returns a new byte slice with all bits in 'value' reset, starting from 'start' number of bit. If 'start'
// is bigger than len(value), the original slice will be returned.
func ResetBits(value []byte, start uint8) []byte {
	if int(start) >= len(value)*8 {
		return value
	}

	startByte := start / 8
	startBit := start % 8

	result := make([]byte, len(value))
	copy(result, value[:startByte])

	// Reset bits in starting byte.
	mask := byte(0xFF)
	mask <<= 8 - byte(startBit)
	result[startByte] = value[startByte] & mask

	return result
}
