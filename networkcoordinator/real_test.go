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

package networkcoordinator

import (
	"context"
	"testing"

	"github.com/insolar/insolar/certificate"
	"github.com/insolar/insolar/insolar"
	"github.com/insolar/insolar/insolar/message"
	"github.com/insolar/insolar/insolar/reply"
	"github.com/insolar/insolar/testutils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func mockContractRequester(t *testing.T, nodeRef insolar.Reference, ok bool, r []byte) insolar.ContractRequester {
	cr := testutils.NewContractRequesterMock(t)
	cr.SendRequestFunc = func(ctx context.Context, ref *insolar.Reference, method string, args []interface{}) (insolar.Reply, error) {
		require.Equal(t, nodeRef, *ref)
		require.Equal(t, "GetNodeInfo", method)
		require.Equal(t, 0, len(args))
		if ok {
			return &reply.CallMethod{
				Result: r,
			}, nil
		}
		return nil, errors.New("test_error")
	}
	return cr
}

func TestRealNetworkCoordinator_New(t *testing.T) {
	coord := newRealNetworkCoordinator(nil, nil, nil, nil)
	require.Equal(t, &realNetworkCoordinator{}, coord)
}

func TestRealNetworkCoordinator_GetCert(t *testing.T) {
	nodeRef := testutils.RandomRef()
	certNodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))

	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() insolar.NetworkState {
		return insolar.CompleteNetworkState
	}

	mb := mockMessageBus(t, true, &nodeRef, &certNodeRef)
	cm := mockCertificateManager(t, &certNodeRef, &certNodeRef, true)
	cs := mockCryptographyService(t, true)

	coord := newRealNetworkCoordinator(cm, cr, mb, cs)
	ctx := context.Background()
	result, err := coord.GetCert(ctx, &nodeRef)
	require.NoError(t, err)

	cert := result.(*certificate.Certificate)
	require.Equal(t, "test_node_public_key", cert.PublicKey)
	require.Equal(t, nodeRef.String(), cert.Reference)
	require.Equal(t, "virtual", cert.Role)
	require.Equal(t, 0, cert.MajorityRule)
	require.Equal(t, uint(0), cert.MinRoles.Virtual)
	require.Equal(t, uint(0), cert.MinRoles.HeavyMaterial)
	require.Equal(t, uint(0), cert.MinRoles.LightMaterial)
	require.Equal(t, []string{}, cert.PulsarPublicKeys)
	require.Equal(t, "test_root_domain_ref", cert.RootDomainReference)
	require.Equal(t, 1, len(cert.BootstrapNodes))
	require.Equal(t, "test_discovery_public_key", cert.BootstrapNodes[0].PublicKey)
	require.Equal(t, []byte("test_network_sign"), cert.BootstrapNodes[0].NetworkSign)
	require.Equal(t, "test_discovery_host", cert.BootstrapNodes[0].Host)
	require.Equal(t, []byte("test_sig"), cert.BootstrapNodes[0].NodeSign)
	require.Equal(t, certNodeRef.String(), cert.BootstrapNodes[0].NodeRef)
}

func TestRealNetworkCoordinator_GetCert_getNodeInfoError(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, false, nil)

	coord := newRealNetworkCoordinator(nil, cr, nil, nil)
	ctx := context.Background()
	_, err := coord.GetCert(ctx, &nodeRef)
	require.EqualError(t, err, "[ GetCert ] Couldn't get node info: [ GetCert ] Couldn't call GetNodeInfo: test_error")
}

func TestRealNetworkCoordinator_GetCert_DeserializeError(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, []byte(""))

	coord := newRealNetworkCoordinator(nil, cr, nil, nil)
	ctx := context.Background()
	_, err := coord.GetCert(ctx, &nodeRef)
	require.EqualError(t, err, "[ GetCert ] Couldn't get node info: [ GetCert ] Couldn't extract response: [ NodeInfoResponse ] Can't unmarshal response: [ UnMarshalResponse ]: [ Deserialize ]: EOF")
}

func TestRealNetworkCoordinator_GetCert_UnsignedCertificateError(t *testing.T) {
	nodeRef := testutils.RandomRef()
	certNodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))

	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() insolar.NetworkState {
		return insolar.CompleteNetworkState
	}

	cm := mockCertificateManager(t, &certNodeRef, &certNodeRef, false)
	coord := newRealNetworkCoordinator(cm, cr, nil, nil)
	ctx := context.Background()
	_, err := coord.GetCert(ctx, &nodeRef)
	require.EqualError(t, err, "[ GetCert ] Couldn't create certificate: test_error")
}

func TestRealNetworkCoordinator_GetCert_SignCertError(t *testing.T) {
	nodeRef := testutils.RandomRef()
	certNodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))

	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() insolar.NetworkState {
		return insolar.CompleteNetworkState
	}

	cm := mockCertificateManager(t, &certNodeRef, &certNodeRef, true)
	cs := mockCryptographyService(t, false)

	coord := newRealNetworkCoordinator(cm, cr, nil, cs)
	ctx := context.Background()
	_, err := coord.GetCert(ctx, &nodeRef)
	require.EqualError(t, err, "[ GetCert ] Couldn't request cert sign: [ SignCert ] Couldn't sign: test_error")
}

func TestRealNetworkCoordinator_requestCertSignSelfDiscoveryNode(t *testing.T) {
	nodeRef := testutils.RandomRef()
	certNodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))

	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() insolar.NetworkState {
		return insolar.CompleteNetworkState
	}

	mb := mockMessageBus(t, true, &nodeRef, &certNodeRef)

	cm := mockCertificateManager(t, &certNodeRef, &certNodeRef, true)
	cs := mockCryptographyService(t, true)

	coord := newRealNetworkCoordinator(cm, cr, mb, cs)
	ctx := context.Background()
	dNode := certificate.BootstrapNode{
		PublicKey:   "test_discovery_public_key",
		Host:        "test_discovery_host",
		NetworkSign: []byte("test_network_sign"),
		NodeRef:     certNodeRef.String(),
	}
	result, err := coord.requestCertSign(ctx, &dNode, &nodeRef)
	require.NoError(t, err)
	require.Equal(t, []byte("test_sig"), result)
}

func TestRealNetworkCoordinator_requestCertSignOtherDiscoveryNode(t *testing.T) {
	nodeRef := testutils.RandomRef()
	certNodeRef := testutils.RandomRef()
	discoveryNodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))

	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() insolar.NetworkState {
		return insolar.CompleteNetworkState
	}

	mb := mockMessageBus(t, true, &nodeRef, &discoveryNodeRef)

	cm := mockCertificateManager(t, &certNodeRef, &discoveryNodeRef, true)

	coord := newRealNetworkCoordinator(cm, cr, mb, nil)
	ctx := context.Background()
	dNode := certificate.BootstrapNode{
		PublicKey:   "test_discovery_public_key",
		Host:        "test_discovery_host",
		NetworkSign: []byte("test_network_sign"),
		NodeRef:     discoveryNodeRef.String(),
	}
	result, err := coord.requestCertSign(ctx, &dNode, &nodeRef)
	require.NoError(t, err)
	require.Equal(t, []byte("test_sig"), result)
}

func TestRealNetworkCoordinator_requestCertSignSelfDiscoveryNode_signCertError(t *testing.T) {
	nodeRef := testutils.RandomRef()
	certNodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, false, nil)

	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() insolar.NetworkState {
		return insolar.CompleteNetworkState
	}

	cm := mockCertificateManager(t, &certNodeRef, &certNodeRef, true)
	coord := newRealNetworkCoordinator(cm, cr, nil, nil)
	ctx := context.Background()
	dNode := certificate.BootstrapNode{
		PublicKey:   "test_discovery_public_key",
		Host:        "test_discovery_host",
		NetworkSign: []byte("test_network_sign"),
		NodeRef:     certNodeRef.String(),
	}
	_, err := coord.requestCertSign(ctx, &dNode, &nodeRef)
	require.EqualError(t, err, "[ SignCert ] Couldn't extract response: [ GetCert ] Couldn't call GetNodeInfo: test_error")
}

func TestRealNetworkCoordinator_requestCertSignOtherDiscoveryNode_CurrentPulseError(t *testing.T) {
	nodeRef := testutils.RandomRef()
	certNodeRef := testutils.RandomRef()
	discoveryNodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))

	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() insolar.NetworkState {
		return insolar.CompleteNetworkState
	}

	mb := mockMessageBus(t, false, &nodeRef, &discoveryNodeRef)
	cm := mockCertificateManager(t, &certNodeRef, &certNodeRef, true)

	coord := newRealNetworkCoordinator(cm, cr, mb, nil)
	ctx := context.Background()
	dNode := certificate.BootstrapNode{
		PublicKey:   "test_discovery_public_key",
		Host:        "test_discovery_host",
		NetworkSign: []byte("test_network_sign"),
		NodeRef:     discoveryNodeRef.String(),
	}
	_, err := coord.requestCertSign(ctx, &dNode, &nodeRef)
	require.EqualError(t, err, "test_error")
}

func TestRealNetworkCoordinator_requestCertSignOtherDiscoveryNode_SendError(t *testing.T) {
	nodeRef := testutils.RandomRef()
	certNodeRef := testutils.RandomRef()
	discoveryNodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))

	ns := testutils.NewNetworkSwitcherMock(t)
	ns.GetStateFunc = func() insolar.NetworkState {
		return insolar.CompleteNetworkState
	}

	mb := mockMessageBus(t, false, &nodeRef, &discoveryNodeRef)

	cm := mockCertificateManager(t, &certNodeRef, &discoveryNodeRef, true)

	coord := newRealNetworkCoordinator(cm, cr, mb, nil)
	ctx := context.Background()
	dNode := certificate.BootstrapNode{
		PublicKey:   "test_discovery_public_key",
		Host:        "test_discovery_host",
		NetworkSign: []byte("test_network_sign"),
		NodeRef:     discoveryNodeRef.String(),
	}
	_, err := coord.requestCertSign(ctx, &dNode, &nodeRef)
	require.EqualError(t, err, "test_error")
}

func TestRealNetworkCoordinator_signCertHandler(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))
	cs := mockCryptographyService(t, true)

	coord := newRealNetworkCoordinator(nil, cr, nil, cs)
	ctx := context.Background()
	result, err := coord.signCertHandler(ctx, &message.Parcel{Msg: &message.NodeSignPayload{NodeRef: &nodeRef}})
	require.NoError(t, err)
	require.Equal(t, []byte("test_sig"), result.(*reply.NodeSign).Sign)
}

func TestRealNetworkCoordinator_signCertHandler_NodeInfoError(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, false, nil)

	coord := newRealNetworkCoordinator(nil, cr, nil, nil)
	ctx := context.Background()
	_, err := coord.signCertHandler(ctx, &message.Parcel{Msg: &message.NodeSignPayload{NodeRef: &nodeRef}})
	require.EqualError(t, err, "[ SignCert ] Couldn't extract response: [ SignCert ] Couldn't extract response: [ GetCert ] Couldn't call GetNodeInfo: test_error")
}

func TestRealNetworkCoordinator_signCertHandler_SignError(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))
	cs := mockCryptographyService(t, false)

	coord := newRealNetworkCoordinator(nil, cr, nil, cs)
	ctx := context.Background()
	_, err := coord.signCertHandler(ctx, &message.Parcel{Msg: &message.NodeSignPayload{NodeRef: &nodeRef}})
	require.EqualError(t, err, "[ SignCert ] Couldn't extract response: [ SignCert ] Couldn't sign: test_error")
}

func TestRealNetworkCoordinator_signCert(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))
	cs := mockCryptographyService(t, true)

	coord := newRealNetworkCoordinator(nil, cr, nil, cs)
	ctx := context.Background()
	result, err := coord.signCert(ctx, &nodeRef)
	require.NoError(t, err)
	require.Equal(t, []byte("test_sig"), result)
}

func TestRealNetworkCoordinator_signCert_NodeInfoError(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, false, nil)

	coord := newRealNetworkCoordinator(nil, cr, nil, nil)
	ctx := context.Background()
	_, err := coord.signCert(ctx, &nodeRef)
	require.EqualError(t, err, "[ SignCert ] Couldn't extract response: [ GetCert ] Couldn't call GetNodeInfo: test_error")
}

func TestRealNetworkCoordinator_signCert_SignError(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))
	cs := mockCryptographyService(t, false)

	coord := newRealNetworkCoordinator(nil, cr, nil, cs)
	ctx := context.Background()
	_, err := coord.signCert(ctx, &nodeRef)
	require.EqualError(t, err, "[ SignCert ] Couldn't sign: test_error")
}

func TestRealNetworkCoordinator_getNodeInfo(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, mockReply(t))

	coord := newRealNetworkCoordinator(nil, cr, nil, nil)
	ctx := context.Background()
	key, role, err := coord.getNodeInfo(ctx, &nodeRef)
	require.NoError(t, err)
	require.Equal(t, "test_node_public_key", key)
	require.Equal(t, "virtual", role)
}

func TestRealNetworkCoordinator_getNodeInfo_SendRequestError(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, false, nil)

	coord := newRealNetworkCoordinator(nil, cr, nil, nil)
	ctx := context.Background()
	_, _, err := coord.getNodeInfo(ctx, &nodeRef)
	require.EqualError(t, err, "[ GetCert ] Couldn't call GetNodeInfo: test_error")
}

func TestRealNetworkCoordinator_getNodeInfo_ExtractError(t *testing.T) {
	nodeRef := testutils.RandomRef()

	cr := mockContractRequester(t, nodeRef, true, []byte(""))

	coord := newRealNetworkCoordinator(nil, cr, nil, nil)
	ctx := context.Background()
	_, _, err := coord.getNodeInfo(ctx, &nodeRef)
	require.EqualError(t, err, "[ GetCert ] Couldn't extract response: [ NodeInfoResponse ] Can't unmarshal response: [ UnMarshalResponse ]: [ Deserialize ]: EOF")
}
