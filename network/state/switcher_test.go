//
// Modified BSD 3-Clause Clear License
//
// Copyright (c) 2019 Insolar Technologies GmbH
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted (subject to the limitations in the disclaimer below) provided that
// the following conditions are met:
//  * Redistributions of source code must retain the above copyright notice, this list
//    of conditions and the following disclaimer.
//  * Redistributions in binary form must reproduce the above copyright notice, this list
//    of conditions and the following disclaimer in the documentation and/or other materials
//    provided with the distribution.
//  * Neither the name of Insolar Technologies GmbH nor the names of its contributors
//    may be used to endorse or promote products derived from this software without
//    specific prior written permission.
//
// NO EXPRESS OR IMPLIED LICENSES TO ANY PARTY'S PATENT RIGHTS ARE GRANTED
// BY THIS LICENSE. THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS
// AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES,
// INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY
// AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL
// THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
// INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
// BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS
// OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
// Notwithstanding any other provisions of this license, it is prohibited to:
//    (a) use this software,
//
//    (b) prepare modifications and derivative works of this software,
//
//    (c) distribute this software (including without limitation in source code, binary or
//        object code form), and
//
//    (d) reproduce copies of this software
//
//    for any commercial purposes, and/or
//
//    for the purposes of making available this software to third parties as a service,
//    including, without limitation, any software-as-a-service, platform-as-a-service,
//    infrastructure-as-a-service or other similar online service, irrespective of
//    whether it competes with the products or services of Insolar Technologies GmbH.
//

package state

import (
	"context"
	"sync"
	"testing"

	"github.com/insolar/insolar/component"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/testutils/network"
	"github.com/stretchr/testify/require"
)

func mockSwitcherWorkAround(t *testing.T, isBootstrapped bool) *network.SwitcherWorkAroundMock {
	swaMock := network.NewSwitcherWorkAroundMock(t)
	swaMock.IsBootstrappedFunc = func() bool {
		return isBootstrapped
	}
	return swaMock
}

func mockMessageBusLocker(t *testing.T) *messageBusLockerMock {
	mblMock := NewmessageBusLockerMock(t)
	mblMock.UnlockFunc = func(p context.Context) {}
	return mblMock
}

func TestNewNetworkSwitcher(t *testing.T) {
	nodeNet := network.NewNodeNetworkMock(t)
	switcherWorkAround := mockSwitcherWorkAround(t, false)
	messageBusLocker := mockMessageBusLocker(t)

	switcher, err := NewNetworkSwitcher()
	require.NoError(t, err)

	cm := &component.Manager{}
	cm.Inject(nodeNet, switcherWorkAround, messageBusLocker, switcher)

	require.Equal(t, nodeNet, switcher.NodeNetwork)
	require.Equal(t, switcherWorkAround, switcher.SwitcherWorkAround)
	require.Equal(t, messageBusLocker, switcher.MBLocker)
	require.Equal(t, insolar.NoNetworkState, switcher.state)
	require.Equal(t, sync.RWMutex{}, switcher.stateLock)
}

func TestGetState(t *testing.T) {
	switcher, err := NewNetworkSwitcher()
	require.NoError(t, err)

	state := switcher.GetState()
	require.Equal(t, insolar.NoNetworkState, state)
}

func TestOnPulseNoChange(t *testing.T) {
	switcher, err := NewNetworkSwitcher()
	require.NoError(t, err)
	switcherWorkAround := mockSwitcherWorkAround(t, false)
	nodeNet := network.NewNodeNetworkMock(t)
	messageBusLocker := mockMessageBusLocker(t)

	cm := &component.Manager{}
	cm.Inject(switcherWorkAround, switcher, nodeNet, messageBusLocker)

	err = switcher.OnPulse(context.Background(), insolar.Pulse{})
	require.NoError(t, err)
	require.Equal(t, insolar.NoNetworkState, switcher.state)
	require.Equal(t, uint64(0), messageBusLocker.UnlockCounter)
}

func TestOnPulseStateChanged(t *testing.T) {
	switcher, err := NewNetworkSwitcher()
	require.NoError(t, err)
	switcherWorkAround := mockSwitcherWorkAround(t, true)
	nodeNet := network.NewNodeNetworkMock(t)
	messageBusLocker := mockMessageBusLocker(t)

	cm := &component.Manager{}
	cm.Inject(switcherWorkAround, switcher, nodeNet, messageBusLocker)

	err = switcher.OnPulse(context.Background(), insolar.Pulse{})
	require.NoError(t, err)
	require.Equal(t, insolar.CompleteNetworkState, switcher.state)
	require.Equal(t, uint64(1), messageBusLocker.UnlockCounter)
}

func TestGetStateAfterStateChanged(t *testing.T) {
	switcher, err := NewNetworkSwitcher()
	require.NoError(t, err)
	switcherWorkAround := mockSwitcherWorkAround(t, true)
	nodeNet := network.NewNodeNetworkMock(t)
	messageBusLocker := mockMessageBusLocker(t)

	cm := &component.Manager{}
	cm.Inject(switcherWorkAround, switcher, nodeNet, messageBusLocker)

	err = switcher.OnPulse(context.Background(), insolar.Pulse{})
	require.NoError(t, err)
	require.Equal(t, insolar.CompleteNetworkState, switcher.state)

	state := switcher.GetState()
	require.Equal(t, insolar.CompleteNetworkState, state)
}
