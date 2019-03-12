/*
 *    Copyright 2019 Insolar
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

package insolar

import (
	"github.com/insolar/insolar/core"
)

// ID represents universal insolar ID.
type ID = core.RecordID

// Reference represents universal insolar Reference. Reference consists of two IDs. First ID is an affinity ID,
// second is referred object ID.
type Reference = core.RecordRef