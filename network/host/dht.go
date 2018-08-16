/*
 *    Copyright 2018 INS Ecosystem
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

package host

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/huandu/xstrings"
	"github.com/insolar/insolar/network/host/message"
	"github.com/insolar/insolar/network/host/node"
	"github.com/insolar/insolar/network/host/relay"
	"github.com/insolar/insolar/network/host/routing"
	"github.com/insolar/insolar/network/host/rpc"
	"github.com/insolar/insolar/network/host/store"
	"github.com/insolar/insolar/network/host/transport"
	"github.com/jbenet/go-base58"
)

// DHT represents the state of the local node in the distributed hash table.
type DHT struct {
	tables  []*routing.HashTable
	options *Options

	origin *node.Origin

	transport transport.Transport
	store     store.Store
	rpc       rpc.RPC
	relay     relay.Relay
	proxy     relay.Proxy
	auth      AuthInfo
	subnet    Subnet
}

// AuthInfo collects some information about authentication.
type AuthInfo struct {
	// Sent/received unique auth keys.
	SentKeys     map[string][]byte
	ReceivedKeys map[string][]byte

	authenticatedNodes map[string]bool

	mut sync.Mutex
}

// Subnet collects some information about self network part
type Subnet struct {
	SubnetIDs        map[string][]string // key - ip, value - id
	HomeSubnetKey    string              // key of home subnet fo SubnetIDs
	PossibleRelayIDs []string
	PossibleProxyIDs []string
	HighKnownNodes   HighKnownOuterNodesNode
}

// HighKnownOuterNodesNode collects an information about node in home subnet which have a more known outer nodes.
type HighKnownOuterNodesNode struct {
	ID                  string
	OuterNodes          int // high known outer nodes by ID node
	SelfKnownOuterNodes int
}

// Options contains configuration options for the local node.
type Options struct {
	// The nodes being used to bootstrap the insolar. Without a bootstrap
	// node there is no way to connect to the insolar. NetworkNodes can be
	// initialized via insolar.NewNode().
	BootstrapNodes []*node.Node

	// The time after which a key/value pair expires;
	// this is a time-to-live (TTL) from the original publication date.
	ExpirationTime time.Duration

	// Seconds after which an otherwise unaccessed bucket must be refreshed.
	RefreshTime time.Duration

	// The interval between Kademlia replication events, when a node is
	// required to publish its entire database.
	ReplicateTime time.Duration

	// The time after which the original publisher must
	// republish a key/value pair. Currently not implemented.
	RepublishTime time.Duration

	// The maximum time to wait for a response from a node before discarding
	// it from the bucket.
	PingTimeout time.Duration

	// The maximum time to wait for a response to any message.
	MessageTimeout time.Duration
}

// NewDHT initializes a new DHT node.
func NewDHT(store store.Store, origin *node.Origin, transport transport.Transport, rpc rpc.RPC, options *Options, proxy relay.Proxy) (dht *DHT, err error) {
	tables, err := newTables(origin)
	if err != nil {
		return nil, err
	}

	rel := relay.NewRelay()

	dht = &DHT{
		options:   options,
		origin:    origin,
		rpc:       rpc,
		transport: transport,
		tables:    tables,
		store:     store,
		relay:     rel,
		proxy:     proxy,
	}

	if options.ExpirationTime == 0 {
		options.ExpirationTime = time.Second * 86410
	}

	if options.RefreshTime == 0 {
		options.RefreshTime = time.Second * 3600
	}

	if options.ReplicateTime == 0 {
		options.ReplicateTime = time.Second * 3600
	}

	if options.RepublishTime == 0 {
		options.RepublishTime = time.Second * 86400
	}

	if options.PingTimeout == 0 {
		options.PingTimeout = time.Second * 1
	}

	if options.MessageTimeout == 0 {
		options.MessageTimeout = time.Second * 10
	}

	dht.auth.authenticatedNodes = make(map[string]bool)
	dht.auth.SentKeys = make(map[string][]byte)
	dht.auth.ReceivedKeys = make(map[string][]byte)

	dht.subnet.SubnetIDs = make(map[string][]string)

	return dht, nil
}

func newTables(origin *node.Origin) ([]*routing.HashTable, error) {
	tables := make([]*routing.HashTable, len(origin.IDs))

	for i, id := range origin.IDs {
		ht, err := routing.NewHashTable(id, origin.Address)
		if err != nil {
			return nil, err
		}

		tables[i] = ht
	}

	return tables, nil
}

func (dht *DHT) getExpirationTime(ctx context.Context, key []byte) time.Time {
	ht := dht.htFromCtx(ctx)

	bucket := routing.GetBucketIndexFromDifferingBit(key, ht.Origin.ID)
	var total int
	for i := 0; i < bucket; i++ {
		total += ht.GetTotalNodesInBucket(i)
	}
	closer := ht.GetAllNodesInBucketCloserThan(bucket, key)
	score := total + len(closer)

	if score == 0 {
		score = 1
	}

	if score > routing.MaxContactsInBucket {
		return time.Now().Add(dht.options.ExpirationTime)
	}

	day := dht.options.ExpirationTime
	seconds := day.Nanoseconds() * int64(math.Exp(float64(routing.MaxContactsInBucket/score)))
	dur := time.Second * time.Duration(seconds)
	return time.Now().Add(dur)
}

// Store stores data on the insolar. This will trigger an iterateStore loop.
// The base58 encoded identifier will be returned if the store is successful.
func (dht *DHT) Store(ctx Context, data []byte) (id string, err error) {
	key := store.NewKey(data)
	expiration := dht.getExpirationTime(ctx, key)
	replication := time.Now().Add(dht.options.ReplicateTime)
	err = dht.store.Store(key, data, replication, expiration, true)
	if err != nil {
		return "", err
	}
	_, _, err = dht.iterate(ctx, routing.IterateStore, key, data)
	if err != nil {
		return "", err
	}
	str := base58.Encode(key)
	return str, nil
}

// Get retrieves data from the transport using key. Key is the base58 encoded
// identifier of the data.
func (dht *DHT) Get(ctx Context, key string) ([]byte, bool, error) {
	keyBytes := base58.Decode(key)
	if len(keyBytes) != routing.MaxContactsInBucket {
		return nil, false, errors.New("invalid key")
	}

	value, exists := dht.store.Retrieve(keyBytes)
	if !exists {
		var err error
		value, _, err = dht.iterate(ctx, routing.IterateFindValue, keyBytes, nil)
		if err != nil {
			return nil, false, err
		}
		if value != nil {
			exists = true
		}
	}

	return value, exists, nil
}

// FindNode returns target node's real insolar address.
func (dht *DHT) FindNode(ctx Context, key string) (*node.Node, bool, error) {
	keyBytes := base58.Decode(key)
	if len(keyBytes) != routing.MaxContactsInBucket {
		return nil, false, errors.New("invalid key")
	}
	ht := dht.htFromCtx(ctx)

	if ht.Origin.ID.Equal(keyBytes) {
		return ht.Origin, true, nil
	}

	var targetNode *node.Node
	var exists = false
	routeSet := ht.GetClosestContacts(1, keyBytes, nil)

	if routeSet.Len() > 0 && routeSet.FirstNode().ID.Equal(keyBytes) {
		targetNode = routeSet.FirstNode()
		exists = true
	} else if dht.proxy.ProxyNodesCount() > 0 {
		address, _ := node.NewAddress(dht.proxy.GetNextProxyAddress())
		targetNode = &node.Node{ID: keyBytes, Address: address}
		return targetNode, true, nil
	} else {
		log.Println("Node not found in routing table. Iterating through insolar...")
		_, closest, err := dht.iterate(ctx, routing.IterateFindNode, keyBytes, nil)
		if err != nil {
			return nil, false, err
		}
		for i := range closest {
			if closest[i].ID.Equal(keyBytes) {
				targetNode = closest[i]
				exists = true
			}
		}
	}

	return targetNode, exists, nil
}

// NumNodes returns the total number of nodes stored in the local routing table.
func (dht *DHT) NumNodes(ctx Context) int {
	ht := dht.htFromCtx(ctx)
	return ht.TotalNodes()
}

// GetOriginID returns the base58 encoded identifier of the local node.
func (dht *DHT) GetOriginID(ctx Context) string {
	ht := dht.htFromCtx(ctx)
	return ht.Origin.ID.String()
}

// Listen begins listening on the socket for incoming Messages.
func (dht *DHT) Listen() error {
	start := make(chan bool)
	stop := make(chan bool)

	go dht.handleDisconnect(start, stop)
	go dht.handleMessages(start, stop)
	go dht.handleStoreTimers(start, stop)

	return dht.transport.Start()
}

// Bootstrap attempts to bootstrap the insolar using the BootstrapNodes provided
// to the Options struct. This will trigger an iterateBootstrap to the provided
// BootstrapNodes.
func (dht *DHT) Bootstrap() error {
	if len(dht.options.BootstrapNodes) == 0 {
		return nil
	}
	var futures []transport.Future
	wg := &sync.WaitGroup{}
	cb := NewContextBuilder(dht)

	for _, ht := range dht.tables {
		futures = dht.iterateBootstrapNodes(ht, cb, wg, futures)
	}

	for _, f := range futures {
		go func(future transport.Future) {
			select {
			case result := <-future.Result():
				// If result is nil, channel was closed
				if result != nil {
					ctx, err := cb.SetNodeByID(result.Receiver.ID).Build()
					// TODO: must return error here
					if err != nil {
						log.Fatal(err)
					}
					dht.addNode(ctx, routing.NewRouteNode(result.Sender))
				}
				wg.Done()
				return
			case <-time.After(dht.options.MessageTimeout):
				future.Cancel()
				wg.Done()
				return
			}
		}(f)
	}

	wg.Wait()
	return dht.iterateHt(cb)
}

func (dht *DHT) iterateHt(cb ContextBuilder) error {
	for _, ht := range dht.tables {
		ctx, err := cb.SetNodeByID(ht.Origin.ID).Build()
		if err != nil {
			return err
		}

		if dht.NumNodes(ctx) > 0 {
			_, _, err = dht.iterate(ctx, routing.IterateBootstrap, ht.Origin.ID, nil)
			return err
		}
	}
	return nil
}

func (dht *DHT) iterateBootstrapNodes(
	ht *routing.HashTable,
	cb ContextBuilder,
	wg *sync.WaitGroup,
	futures []transport.Future,
) []transport.Future {
	ctx, err := cb.SetNodeByID(ht.Origin.ID).Build()
	if err != nil {
		return futures
	}
	for _, bn := range dht.options.BootstrapNodes {
		request := message.NewPingMessage(ht.Origin, bn)

		if bn.ID == nil {
			res, err := dht.transport.SendRequest(request)
			if err != nil {
				continue
			}
			wg.Add(1)
			futures = append(futures, res)
		} else {
			routeNode := routing.NewRouteNode(bn)
			dht.addNode(ctx, routeNode)
		}
	}
	return futures
}

// Disconnect will trigger a StopRelay from the insolar.
func (dht *DHT) Disconnect() {
	dht.transport.Stop()
}

// Iterate does an iterative search through the insolar. This can be done
// for multiple reasons. These reasons include:
//     iterateStore - Used to store new information in the insolar.
//     iterateFindNode - Used to find node in the insolar given node abstract address.
//     iterateFindValue - Used to find a value among the insolar given a key.
//     iterateBootstrap - Used to bootstrap the insolar.
func (dht *DHT) iterate(ctx Context, t routing.IterateType, target []byte, data []byte) (value []byte, closest []*node.Node, err error) {
	ht := dht.htFromCtx(ctx)
	routeSet := ht.GetClosestContacts(routing.ParallelCalls, target, []*node.Node{})

	// We keep track of nodes contacted so far. We don't contact the same node
	// twice.
	var contacted = make(map[string]bool)

	// According to the Kademlia white paper, after a round of FIND_NODE RPCs
	// fails to provide a node closer than closestNode, we should send a
	// FIND_NODE RPC to all remaining nodes in the route set that have not
	// yet been contacted.
	queryRest := false

	// We keep a reference to the closestNode. If after performing a search
	// we do not find a closer node, we stop searching.
	if routeSet.Len() == 0 {
		return nil, nil, nil
	}

	closestNode := routeSet.FirstNode()

	checkAndRefreshTimeForBucket(t, ht, target)

	var removeFromRouteSet []*node.Node

	for {
		var futures []transport.Future
		var futuresCount int

		futures, removeFromRouteSet = dht.sendMessageToAlphaNodes(routeSet, queryRest, t, ht, contacted, target, futures, removeFromRouteSet)

		routeSet.RemoveMany(routing.RouteNodesFrom(removeFromRouteSet))

		futuresCount = len(futures)

		resultChan := make(chan *message.Message)
		dht.setUpResultChan(futures, ctx, resultChan)

		value, closest, err = dht.checkFuturesCountAndGo(t, &queryRest, routeSet, futuresCount, resultChan, target, closest)
		if (err == nil) || ((err != nil) && (err.Error() != "do nothing")) {
			return value, closest, err
		}

		sort.Sort(routeSet)

		var tmpValue []byte
		var tmpClosest []*node.Node
		var tmpNode *node.Node
		tmpValue, tmpClosest, tmpNode, err = dht.iterateIsDone(t, &queryRest, routeSet, data, ht, closestNode)
		if err == nil {
			return tmpValue, tmpClosest, err
		} else if tmpNode != nil {
			closestNode = tmpNode
		}
	}
}

func (dht *DHT) iterateIsDone(
	t routing.IterateType,
	queryRest *bool,
	routeSet *routing.RouteSet,
	data []byte,
	ht *routing.HashTable,
	closestNode *node.Node,
) (value []byte, closest []*node.Node, close *node.Node, err error) {

	if routeSet.FirstNode().ID.Equal(closestNode.ID) || *(queryRest) {
		switch t {
		case routing.IterateBootstrap:
			if !(*queryRest) {
				*queryRest = true
				err = errors.New("do nothing")
				return nil, nil, nil, err
			}
			return nil, routeSet.Nodes(), nil, nil
		case routing.IterateFindNode, routing.IterateFindValue:
			return nil, routeSet.Nodes(), nil, nil
		case routing.IterateStore:
			for i, receiver := range routeSet.Nodes() {
				if i >= routing.MaxContactsInBucket {
					return nil, nil, nil, nil
				}

				msg := message.NewBuilder().Sender(ht.Origin).Receiver(receiver).Type(message.TypeStore).Request(
					&message.RequestDataStore{
						Data: data,
					}).Build()

				future, _ := dht.transport.SendRequest(msg)
				// We do not need to handle result of this message
				future.Cancel()
			}
			return nil, nil, nil, nil
		}
	} else {
		err = errors.New("do nothing")
		return nil, nil, routeSet.FirstNode(), err
	}
	err = errors.New("do nothing")
	return nil, nil, nil, err
}

func (dht *DHT) checkFuturesCountAndGo(
	t routing.IterateType,
	queryRest *bool,
	routeSet *routing.RouteSet,
	futuresCount int,
	resultChan chan *message.Message,
	target []byte,
	close []*node.Node,
) ([]byte, []*node.Node, error) {

	var err error
	var results []*message.Message
	var selected bool
	if futuresCount > 0 {
	Loop:
		for {
			results, selected = dht.selectResultChan(resultChan, &futuresCount, results)
			if selected {
				break Loop
			}
		}

		_, close, err = resultsIterate(t, results, routeSet, target)
		if close != nil {
			return nil, close, err
		}
	}

	if !*queryRest && routeSet.Len() == 0 {
		return nil, close, nil
	}
	err = errors.New("do nothing")
	return nil, close, err
}

func resultsIterate(
	t routing.IterateType,
	results []*message.Message,
	routeSet *routing.RouteSet,
	target []byte,
) (value []byte, closest []*node.Node, err error) {

	for _, result := range results {
		if result.Error != nil {
			routeSet.Remove(routing.NewRouteNode(result.Sender))
			continue
		}
		switch t {
		case routing.IterateBootstrap, routing.IterateFindNode, routing.IterateStore:
			responseData := result.Data.(*message.ResponseDataFindNode)
			if len(responseData.Closest) > 0 && responseData.Closest[0].ID.Equal(target) {
				return nil, responseData.Closest, nil
			}
			routeSet.AppendMany(routing.RouteNodesFrom(responseData.Closest))
		case routing.IterateFindValue:
			responseData := result.Data.(*message.ResponseDataFindValue)
			routeSet.AppendMany(routing.RouteNodesFrom(responseData.Closest))
			if responseData.Value != nil {
				// TODO When an iterateFindValue succeeds, the initiator must
				// store the key/value pair at the closest receiver seen which did
				// not return the value.
				return responseData.Value, nil, nil
			}
		}
	}
	return nil, nil, nil
}

func checkAndRefreshTimeForBucket(t routing.IterateType, ht *routing.HashTable, target []byte) {
	if t == routing.IterateBootstrap {
		bucket := routing.GetBucketIndexFromDifferingBit(target, ht.Origin.ID)
		ht.ResetRefreshTimeForBucket(bucket)
	}
}

func (dht *DHT) selectResultChan(
	resultChan chan *message.Message,
	futuresCount *int,
	results []*message.Message,
) ([]*message.Message, bool) {
	select {
	case result := <-resultChan:
		if result != nil {
			results = append(results, result)
		} else {
			*futuresCount--
		}
		if len(results) == *futuresCount {
			close(resultChan)
			return results, true
		}
	case <-time.After(dht.options.MessageTimeout):
		close(resultChan)
		return results, true
	}
	return results, false
}

func (dht *DHT) setUpResultChan(futures []transport.Future, ctx Context, resultChan chan *message.Message) {
	for _, f := range futures {
		go func(future transport.Future) {
			select {
			case result := <-future.Result():
				if result == nil {
					// Channel was closed
					return
				}
				dht.addNode(ctx, routing.NewRouteNode(result.Sender))
				resultChan <- result
				return
			case <-time.After(dht.options.MessageTimeout):
				future.Cancel()
				return
			}
		}(f)
	}
}

func (dht *DHT) sendMessageToAlphaNodes(
	routeSet *routing.RouteSet,
	queryRest bool,
	t routing.IterateType,
	ht *routing.HashTable,
	contacted map[string]bool,
	target []byte,
	futures []transport.Future,
	removeFromRouteSet []*node.Node,
) (resultFutures []transport.Future, resultRouteSet []*node.Node) {
	// Next we send Messages to the first (closest) alpha nodes in the
	// route set and wait for a response

	for i, receiver := range routeSet.Nodes() {
		// Contact only alpha nodes
		if i >= routing.ParallelCalls && !queryRest {
			break
		}

		// Don't contact nodes already contacted
		if (contacted)[string(receiver.ID)] {
			continue
		}

		(contacted)[string(receiver.ID)] = true

		messageBuilder := message.NewBuilder().Sender(ht.Origin).Receiver(receiver)
		messageBuilder = getMessageBuilder(t, messageBuilder, target)
		msg := messageBuilder.Build()

		// Send the async queries and wait for a response
		res, err := dht.transport.SendRequest(msg)
		if err != nil {
			// Node was unreachable for some reason. We will have to remove
			// it from the route set, but we will keep it in our routing
			// table in hopes that it might come back online in the f.
			removeFromRouteSet = append(removeFromRouteSet, msg.Receiver)
			continue
		}

		futures = append(futures, res)
	}
	return futures, removeFromRouteSet
}

func getMessageBuilder(t routing.IterateType, messageBuilder message.Builder, target []byte) message.Builder {
	switch t {
	case routing.IterateBootstrap, routing.IterateFindNode:
		return messageBuilder.Type(message.TypeFindNode).Request(&message.RequestDataFindNode{Target: target})
	case routing.IterateFindValue:
		return messageBuilder.Type(message.TypeFindValue).Request(&message.RequestDataFindValue{Target: target})
	case routing.IterateStore:
		return messageBuilder.Type(message.TypeFindNode).Request(&message.RequestDataFindNode{Target: target})
	default:
		panic("Unknown iterate type")
	}
}

// addNode adds a node into the appropriate k bucket
// we store these buckets in big-endian order so we look at the bits
// from right to left in order to find the appropriate bucket
func (dht *DHT) addNode(ctx Context, node *routing.RouteNode) {
	ht := dht.htFromCtx(ctx)
	index := routing.GetBucketIndexFromDifferingBit(ht.Origin.ID, node.ID)

	// Make sure node doesn't already exist
	// If it does, mark it as seen
	if ht.DoesNodeExistInBucket(index, node.ID) {
		ht.MarkNodeAsSeen(node.ID)
		return
	}

	ht.Lock()
	defer ht.Unlock()

	bucket := ht.RoutingTable[index]

	if len(bucket) == routing.MaxContactsInBucket {
		// If the bucket is full we need to ping the first node to find out
		// if it responds back in a reasonable amount of time. If not -
		// we may remove it
		n := bucket[0].Node
		request := message.NewPingMessage(ht.Origin, n)
		future, err := dht.transport.SendRequest(request)
		if err != nil {
			bucket = append(bucket, node)
			bucket = bucket[1:]
		} else {
			select {
			case <-future.Result():
				return
			case <-time.After(dht.options.PingTimeout):
				bucket = bucket[1:]
				bucket = append(bucket, node)
			}
		}
	} else {
		bucket = append(bucket, node)
	}

	ht.RoutingTable[index] = bucket
}

func (dht *DHT) handleDisconnect(start, stop chan bool) {
	multiplexCount := 0

	for {
		select {
		case <-start:
			multiplexCount++
		case <-dht.transport.Stopped():
			for i := 0; i < multiplexCount; i++ {
				stop <- true
			}
			dht.transport.Close()
			return
		}
	}
}

func (dht *DHT) handleStoreTimers(start, stop chan bool) {
	start <- true

	ticker := time.NewTicker(time.Second)
	cb := NewContextBuilder(dht)
	for {
		dht.selectTicker(ticker, &cb, stop)
	}
}

func (dht *DHT) selectTicker(ticker *time.Ticker, cb *ContextBuilder, stop chan bool) {
	select {
	case <-ticker.C:
		keys := dht.store.GetKeysReadyToReplicate()
		for _, ht := range dht.tables {
			ctx, err := cb.SetNodeByID(ht.Origin.ID).Build()
			// TODO: do something sane with error
			if err != nil {
				log.Fatal(err)
			}
			// Refresh
			for i := 0; i < routing.KeyBitSize; i++ {
				if time.Since(ht.GetRefreshTimeForBucket(i)) > dht.options.RefreshTime {
					id := ht.GetRandomIDFromBucket(routing.MaxContactsInBucket)
					_, _, err = dht.iterate(ctx, routing.IterateBootstrap, id, nil)
					if err != nil {
						continue
					}
				}
			}

			// Replication
			for _, key := range keys {
				value, _ := dht.store.Retrieve(key)
				_, _, err2 := dht.iterate(ctx, routing.IterateStore, key, value)
				if err2 != nil {
					continue
				}
			}
		}

		// Expiration
		dht.store.ExpireKeys()
	case <-stop:
		ticker.Stop()
		return
	}
}

func (dht *DHT) handleMessages(start, stop chan bool) {
	start <- true

	cb := NewContextBuilder(dht)
	for {
		select {
		case msg := <-dht.transport.Messages():
			if msg == nil || !msg.IsForMe(*dht.origin) {
				continue
			}

			var ctx Context
			ctx = buildContext(cb, msg)
			ht := dht.htFromCtx(ctx)

			if ht.Origin.ID.Equal(msg.Receiver.ID) || !dht.relay.NeedToRelay(msg.Sender.Address.String()) {
				dht.dispatchMessageType(ctx, msg, ht)
			} else {
				targetNode, exist, err := dht.FindNode(ctx, msg.Receiver.ID.String())
				if err != nil {
					log.Println(err)
				} else if !exist {
					log.Printf("Target node addr: %s, ID: %s not found", msg.Receiver.Address.String(), msg.Receiver.ID.String())
				} else {
					// need to relay incoming message
					request := &message.Message{Sender: &node.Node{Address: dht.origin.Address, ID: msg.Sender.ID},
						Receiver:  &node.Node{ID: msg.Receiver.ID, Address: targetNode.Address},
						Type:      msg.Type,
						RequestID: msg.RequestID,
						Data:      msg.Data}
					dht.sendRelayedRequest(request, ctx)
				}
			}
		case <-stop:
			return
		}
	}
}

func (dht *DHT) sendRelayedRequest(request *message.Message, ctx Context) {
	future, err := dht.transport.SendRequest(request)
	if err != nil {
		log.Println(err)
	}
	select {
	case rsp := <-future.Result():
		if rsp == nil {
			// Channel was closed
			log.Println("chanel closed unexpectedly")
		}
		dht.addNode(ctx, routing.NewRouteNode(rsp.Sender))

		response := rsp.Data.(*message.ResponseDataRPC)
		if response.Success {
			log.Println(response.Result)
		}
		log.Println(response.Error)
	case <-time.After(dht.options.MessageTimeout):
		future.Cancel()
		log.Println("timeout")
	}
}

func buildContext(cb ContextBuilder, msg *message.Message) Context {
	var ctx Context
	var err error
	if msg.Receiver.ID == nil {
		ctx, err = cb.SetDefaultNode().Build()
	} else {
		ctx, err = cb.SetNodeByID(msg.Receiver.ID).Build()
	}
	if err != nil {
		// TODO: Do something sane with error!
		log.Println(err) // don't return this error cuz don't know what to do with
	}
	return ctx
}

func (dht *DHT) dispatchMessageType(ctx Context, msg *message.Message, ht *routing.HashTable) {
	messageBuilder := message.NewBuilder().Sender(ht.Origin).Receiver(msg.Sender).Type(msg.Type)
	switch msg.Type {
	case message.TypeFindNode:
		dht.processFindNode(ctx, msg, messageBuilder)
	case message.TypeFindValue:
		dht.processFindValue(ctx, msg, messageBuilder)
	case message.TypeStore:
		dht.processStore(ctx, msg, messageBuilder)
	case message.TypePing:
		dht.processPing(ctx, msg, messageBuilder)
	case message.TypeRPC:
		dht.processRPC(ctx, msg, messageBuilder)
	case message.TypeRelay:
		dht.processRelay(ctx, msg, messageBuilder)
	case message.TypeCheckOrigin:
		dht.processCheckOriginRequest(ctx, msg, messageBuilder)
	case message.TypeAuth:
		dht.processAuthentication(ctx, msg, messageBuilder)
	case message.TypeObtainIP:
		dht.processObtainIPRequest(ctx, msg, messageBuilder)
	case message.TypeRelayOwnership:
		dht.processRelayOwnership(ctx, msg, messageBuilder)
	case message.TypeKnownOuterNodes:
		dht.processKnownOuterNodes(ctx, msg, messageBuilder)
	}
}

func (dht *DHT) processRelayOwnership(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	data := msg.Data.(*message.RequestRelayOwnership)

	if data.Ready {
		dht.subnet.PossibleProxyIDs = append(dht.subnet.PossibleProxyIDs, msg.Sender.ID.String())
	} else {
		for i, j := range dht.subnet.PossibleProxyIDs {
			if j == msg.Sender.ID.String() {
				dht.subnet.PossibleProxyIDs = append(dht.subnet.PossibleProxyIDs[:i], dht.subnet.PossibleProxyIDs[i+1:]...)
				err := dht.AuthenticationRequest(ctx, "begin", msg.Sender.ID.String())
				if err != nil {
					log.Println("error to send auth request: ", err)
				}
				err = dht.RelayRequest(ctx, "start", msg.Sender.ID.String())
				if err != nil {
					log.Println("error to send relay request: ", err)
				}
				break
			}
		}
	}
	response := &message.ResponseRelayOwnership{Accepted: true}

	err := dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
	if err != nil {
		log.Println("Failed to send response:", err.Error())
	}
}

func (dht *DHT) processFindNode(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	ht := dht.htFromCtx(ctx)
	data := msg.Data.(*message.RequestDataFindNode)
	dht.addNode(ctx, routing.NewRouteNode(msg.Sender))
	closest := ht.GetClosestContacts(routing.MaxContactsInBucket, data.Target, []*node.Node{msg.Sender})
	response := &message.ResponseDataFindNode{
		Closest: closest.Nodes(),
	}
	err := dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
	if err != nil {
		log.Println("Failed to send response:", err.Error())
	}
}

func (dht *DHT) processFindValue(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	ht := dht.htFromCtx(ctx)
	data := msg.Data.(*message.RequestDataFindValue)
	dht.addNode(ctx, routing.NewRouteNode(msg.Sender))
	value, exists := dht.store.Retrieve(data.Target)
	response := &message.ResponseDataFindValue{}
	if exists {
		response.Value = value
	} else {
		closest := ht.GetClosestContacts(routing.MaxContactsInBucket, data.Target, []*node.Node{msg.Sender})
		response.Closest = closest.Nodes()
	}
	err := dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
	if err != nil {
		log.Println("Failed to send response:", err.Error())
	}
}

func (dht *DHT) processStore(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	data := msg.Data.(*message.RequestDataStore)
	dht.addNode(ctx, routing.NewRouteNode(msg.Sender))
	key := store.NewKey(data.Data)
	expiration := dht.getExpirationTime(ctx, key)
	replication := time.Now().Add(dht.options.ReplicateTime)
	err := dht.store.Store(key, data.Data, replication, expiration, false)
	if err != nil {
		log.Println("Failed to store data:", err.Error())
	}
}

func (dht *DHT) processPing(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	err := dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(nil).Build())
	if err != nil {
		log.Println("Failed to send response:", err.Error())
	}
}

func (dht *DHT) processRPC(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	data := msg.Data.(*message.RequestDataRPC)
	dht.addNode(ctx, routing.NewRouteNode(msg.Sender))
	result, err := dht.rpc.Invoke(msg.Sender, data.Method, data.Args)
	response := &message.ResponseDataRPC{
		Success: true,
		Result:  result,
		Error:   "",
	}
	if err != nil {
		response.Success = false
		response.Error = err.Error()
	}
	err = dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
	if err != nil {
		log.Println("Failed to send response:", err.Error())
	}
}

// Precess relay request.
func (dht *DHT) processRelay(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	var err error
	if !dht.auth.authenticatedNodes[msg.Sender.ID.String()] {
		log.Print("relay request from unknown node rejected")
		response := &message.ResponseRelay{
			State: relay.NoAuth,
		}

		err = dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
	} else {
		data := msg.Data.(*message.RequestRelay)
		dht.addNode(ctx, routing.NewRouteNode(msg.Sender))

		var state relay.State

		switch data.Command {
		case message.StartRelay:
			err = dht.relay.AddClient(msg.Sender)
			state = relay.Started
		case message.StopRelay:
			err = dht.relay.RemoveClient(msg.Sender)
			state = relay.Stopped
		default:
			state = relay.Unknown
		}

		if err != nil {
			state = relay.Error
		}

		response := &message.ResponseRelay{
			State: state,
		}

		err = dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
	}
	if err != nil {
		log.Println("Failed to send response:", err.Error())
	}
}

func (dht *DHT) processAuthentication(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	data := msg.Data.(*message.RequestAuth)
	switch data.Command {
	case message.BeginAuth:
		if dht.auth.authenticatedNodes[msg.Sender.ID.String()] {
			// TODO: whats next?
			response := &message.ResponseAuth{
				Success:       false,
				AuthUniqueKey: nil,
			}

			err := dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
			if err != nil {
				log.Println("Failed to send response:", err)
			}
			break
		}
		key := make([]byte, 512)
		_, err := rand.Read(key) // crypto/rand
		if err != nil {
			log.Println("failed to create auth key. ", err)
			return
		}
		dht.auth.SentKeys[msg.Sender.ID.String()] = key
		response := &message.ResponseAuth{
			Success:       true,
			AuthUniqueKey: key,
		}

		err = dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
		if err != nil {
			log.Println("Failed to send response:", err)
		}
		// TODO process verification msg.Sender node
		// confirmed
		err = dht.CheckOriginRequest(ctx, msg.Sender.ID.String())
		if err != nil {
			log.Println("error: ", err)
		}
	case message.RevokeAuth:
		delete(dht.auth.authenticatedNodes, msg.Sender.ID.String())
		response := &message.ResponseAuth{
			Success:       true,
			AuthUniqueKey: nil,
		}

		err := dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
		if err != nil {
			log.Println("Failed to send response:", err)
		}
	default:
		log.Println("unknown auth command")
	}
}

func (dht *DHT) processCheckOriginRequest(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	dht.auth.mut.Lock()
	defer dht.auth.mut.Unlock()
	if key, ok := dht.auth.ReceivedKeys[msg.Sender.ID.String()]; ok {
		response := &message.ResponseCheckOrigin{AuthUniqueKey: key}
		err := dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
		if err != nil {
			log.Println("Failed to send check origin response:", err)
		}
	} else {
		log.Println("CheckOrigin request from unregistered node")
	}
}

func (dht *DHT) processObtainIPRequest(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	response := &message.ResponseObtainIP{IP: msg.RemoteAddress}
	err := dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
	if err != nil {
		log.Println("Failed to send obtain IP response:", err)
	}
}

// RelayRequest sends relay request to target.
func (dht *DHT) RelayRequest(ctx Context, command, targetID string) error { // target - node ID
	var typedCommand message.CommandType
	targetNode, exist, err := dht.FindNode(ctx, targetID)
	if err != nil {
		return err
	}
	if !exist {
		err = errors.New("target for relay request not found")
		return err
	}

	switch command {
	case "start":
		typedCommand = message.StartRelay
	case "stop":
		typedCommand = message.StopRelay
	default:
		err = errors.New("unknown command")
		return err
	}
	request := message.NewRelayMessage(typedCommand, dht.htFromCtx(ctx).Origin, targetNode)
	future, err := dht.transport.SendRequest(request)

	if err != nil {
		log.Println(err.Error())
		return err
	}

	select {
	case rsp := <-future.Result():
		if rsp == nil {
			err = errors.New("chanel closed unexpectedly")
			return err
		}

		response := rsp.Data.(*message.ResponseRelay)
		err = dht.handleRelayResponse(ctx, response, targetID)
		if err != nil {
			return err
		}

	case <-time.After(dht.options.MessageTimeout):
		future.Cancel()
		err = errors.New("timeout")
		return err
	}

	return nil
}

func (dht *DHT) handleRelayResponse(ctx Context, response *message.ResponseRelay, targetID string) error {
	var err error
	switch response.State {
	case relay.Stopped:
		// stop use this address as relay
		dht.proxy.RemoveProxyNode(targetID)
		err = nil
	case relay.Started:
		// start use this address as relay
		dht.proxy.AddProxyNode(targetID)
		err = nil
	case relay.NoAuth:
		err = errors.New("unable to execute relay because this node not authenticated")
	case relay.Unknown:
		err = errors.New("unknown relay command")
	case relay.Error:
		err = errors.New("relay request error")
	default:
		// unknown state/failed to change state
		err = errors.New("unknown response state")
	}
	return err
}

func (dht *DHT) handleCheckOriginResponse(response *message.ResponseCheckOrigin, targetID string) {
	if bytes.Equal(response.AuthUniqueKey, dht.auth.SentKeys[targetID]) {
		delete(dht.auth.SentKeys, targetID)
		dht.auth.authenticatedNodes[targetID] = true
	}
}

// CheckOriginRequest send a request to check target node originality
func (dht *DHT) CheckOriginRequest(ctx Context, targetID string) error {
	targetNode, exist, err := dht.FindNode(ctx, targetID)
	if err != nil {
		return err
	}
	if !exist {
		err = errors.New("target for relay request not found")
		return err
	}

	request := message.NewCheckOriginMessage(dht.htFromCtx(ctx).Origin, targetNode)
	future, err := dht.transport.SendRequest(request)

	if err != nil {
		log.Println(err.Error())
		return err
	}

	select {
	case rsp := <-future.Result():
		if rsp == nil {
			err = errors.New("chanel closed unexpectedly")
			return err
		}

		response := rsp.Data.(*message.ResponseCheckOrigin)
		dht.handleCheckOriginResponse(response, targetID)

	case <-time.After(dht.options.MessageTimeout):
		future.Cancel()
		err = errors.New("timeout")
		return err
	}

	return nil
}

// AuthenticationRequest sends an authentication request.
func (dht *DHT) AuthenticationRequest(ctx Context, command, targetID string) error {
	targetNode, exist, err := dht.FindNode(ctx, targetID)
	if err != nil {
		return err
	}
	if !exist {
		err = errors.New("target for auth request not found")
		return err
	}

	origin := dht.htFromCtx(ctx).Origin
	var authCommand message.CommandType
	switch command {
	case "begin":
		authCommand = message.BeginAuth
	case "revoke":
		authCommand = message.RevokeAuth
	default:
		err = errors.New("unknown command")
		return err
	}
	request := message.NewAuthMessage(authCommand, origin, targetNode)
	future, err := dht.transport.SendRequest(request)

	if err != nil {
		log.Println(err.Error())
		return err
	}

	select {
	case rsp := <-future.Result():
		if rsp == nil {
			err = errors.New("chanel closed unexpectedly")
			return err
		}

		response := rsp.Data.(*message.ResponseAuth)
		err = dht.handleAuthResponse(response, targetNode.ID.String())
		if err != nil {
			return err
		}

	case <-time.After(dht.options.MessageTimeout):
		future.Cancel()
		err = errors.New("timeout")
		return err
	}

	return nil
}

func (dht *DHT) handleAuthResponse(response *message.ResponseAuth, target string) error {
	var err error
	if (len(response.AuthUniqueKey) != 0) && response.Success {
		dht.auth.mut.Lock()
		defer dht.auth.mut.Unlock()
		dht.auth.ReceivedKeys[target] = response.AuthUniqueKey
		err = nil
	} else {
		if response.Success && (len(response.AuthUniqueKey) == 0) { // revoke success
			return err
		}
		if !response.Success {
			err = errors.New("authentication unsuccessful")
		} else if len(response.AuthUniqueKey) == 0 {
			err = errors.New("wrong auth unique key received")
		}
	}
	return err
}

// ObtainIPRequest is request to self IP obtaining.
func (dht *DHT) ObtainIPRequest(ctx Context, targetID string) error {
	targetNode, exist, err := dht.FindNode(ctx, targetID)
	if err != nil {
		return err
	}
	if !exist {
		err = errors.New("target for relay request not found")
		return err
	}

	origin := dht.htFromCtx(ctx).Origin
	request := message.NewObtainIPMessage(origin, targetNode)

	future, err := dht.transport.SendRequest(request)

	if err != nil {
		log.Println(err.Error())
		return err
	}

	select {
	case rsp := <-future.Result():
		if rsp == nil {
			err = errors.New("chanel closed unexpectedly")
			return err
		}

		response := rsp.Data.(*message.ResponseObtainIP)
		dht.handleObtainIPResponse(response, targetNode.ID.String())

	case <-time.After(dht.options.MessageTimeout):
		future.Cancel()
		err = errors.New("timeout")
		return err
	}

	return nil
}

func (dht *DHT) handleObtainIPResponse(response *message.ResponseObtainIP, target string) {
	if response.IP != "" {
		dht.subnet.SubnetIDs[response.IP] = append(dht.subnet.SubnetIDs[response.IP], target)
	}
}

// RemoteProcedureCall calls remote procedure on target node.
func (dht *DHT) RemoteProcedureCall(ctx Context, target string, method string, args [][]byte) (result []byte, err error) {
	targetNode, exists, err := dht.FindNode(ctx, target)
	ht := dht.htFromCtx(ctx)

	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("targetNode not found")
	}

	request := &message.Message{
		Sender:   ht.Origin,
		Receiver: targetNode,
		Type:     message.TypeRPC,
		Data: &message.RequestDataRPC{
			Method: method,
			Args:   args,
		},
	}

	if target == dht.GetOriginID(ctx) {
		return dht.rpc.Invoke(request.Sender, method, args)
	}

	// Send the async queries and wait for a future
	future, err := dht.transport.SendRequest(request)
	if err != nil {
		return nil, err
	}

	select {
	case rsp := <-future.Result():
		if rsp == nil {
			// Channel was closed
			return nil, errors.New("chanel closed unexpectedly")
		}
		dht.addNode(ctx, routing.NewRouteNode(rsp.Sender))

		response := rsp.Data.(*message.ResponseDataRPC)
		if response.Success {
			return response.Result, nil
		}
		return nil, errors.New(response.Error)
	case <-time.After(dht.options.MessageTimeout):
		future.Cancel()
		return nil, errors.New("timeout")
	}
}

// ObtainIP starts to self IP obtaining.
func (dht *DHT) ObtainIP(ctx Context) {
	for _, table := range dht.tables {
		for i := range table.RoutingTable {
			for j := range table.RoutingTable[i] {
				err := dht.ObtainIPRequest(ctx, table.RoutingTable[i][j].ID.String())
				if err != nil {
					log.Println("error to obtain IP: ", err)
				}
			}
		}
	}
}

// GetDistance returns a distance between id1 and id2.
func (dht *DHT) GetDistance(id1, id2 []byte) *big.Int {
	buf1 := new(big.Int).SetBytes(id1)
	buf2 := new(big.Int).SetBytes(id2)
	return new(big.Int).Xor(buf1, buf2)
}

func (dht *DHT) getHomeSubnetKey() string {
	var result string
	for key, subnet := range dht.subnet.SubnetIDs {
		first := subnet[0]
		first = xstrings.Reverse(first)
		first = strings.SplitAfterN(first, ".", 2)[1] // remove X.X.X.this byte
		first = strings.SplitAfterN(first, ".", 2)[1] // remove X.X.this byte
		first = xstrings.Reverse(first)
		for j := 1; j < len(subnet); j++ {
			if !strings.Contains(subnet[j], first) {
				result = ""
				break
			} else {
				result = key
			}
		}
	}
	return result
}

func (dht *DHT) countOuterNodes() {
	if len(dht.subnet.SubnetIDs) > 1 {
		for key, nodes := range dht.subnet.SubnetIDs {
			if key == dht.subnet.HomeSubnetKey {
				continue
			}
			dht.subnet.HighKnownNodes.SelfKnownOuterNodes += len(nodes)
		}
	}
}

// AnalyzeNetwork is func to analyze the network after IP obtaining.
func (dht *DHT) AnalyzeNetwork(ctx Context) {
	dht.subnet.HomeSubnetKey = dht.getHomeSubnetKey()
	dht.countOuterNodes()
	dht.subnet.HighKnownNodes.OuterNodes = dht.subnet.HighKnownNodes.SelfKnownOuterNodes
	nodes := dht.subnet.SubnetIDs[dht.subnet.HomeSubnetKey]
	for _, ids := range nodes {
		err := dht.knownOuterNodesRequest(ids, dht.subnet.HighKnownNodes.OuterNodes)
		if err != nil {
			log.Println("error to send outer nodes request: ", err)
		}
	}
	if len(dht.subnet.SubnetIDs) == 1 {
		if dht.subnet.HomeSubnetKey == "" { // current node have a static IP
			for _, subnetIDs := range dht.subnet.SubnetIDs {
				dht.sendRelayOwnership(subnetIDs)
			}
		}
	}
}

func (dht *DHT) sendRelayOwnership(subnetIDs []string) {
	for _, id := range subnetIDs {
		err := dht.relayOwnershipRequest(id, true)
		log.Println(err.Error())
	}
}

func (dht *DHT) handleRelayOwnership(response *message.ResponseRelayOwnership, target string) {
	if response.Accepted {
		dht.subnet.PossibleRelayIDs = append(dht.subnet.PossibleRelayIDs, target)
	}
}

func (dht *DHT) relayOwnershipRequest(target string, ready bool) error {
	ctx, err := NewContextBuilder(dht).SetDefaultNode().Build()
	if err != nil {
		return err
	}
	targetNode, exist, err := dht.FindNode(ctx, target)
	if err != nil {
		return err
	}
	if !exist {
		err = errors.New("target for relay request not found")
		return err
	}

	request := message.NewRelayOwnershipMessage(dht.htFromCtx(ctx).Origin, targetNode, true)
	future, err := dht.transport.SendRequest(request)

	if err != nil {
		return err
	}

	select {
	case rsp := <-future.Result():
		if rsp == nil {
			return err
		}

		response := rsp.Data.(*message.ResponseRelayOwnership)
		dht.handleRelayOwnership(response, target)

	case <-time.After(dht.options.MessageTimeout):
		future.Cancel()
		err = errors.New("timeout")
		return err
	}

	return nil
}

func (dht *DHT) processKnownOuterNodes(ctx Context, msg *message.Message, messageBuilder message.Builder) {
	data := msg.Data.(*message.RequestKnownOuterNodes)

	ID := dht.subnet.HighKnownNodes.ID
	nodes := dht.subnet.HighKnownNodes.OuterNodes
	if data.OuterNodes > nodes {
		ID = data.ID
		nodes = data.OuterNodes
	}
	response := &message.ResponseKnownOuterNodes{
		ID:         ID,
		OuterNodes: nodes,
	}

	err := dht.transport.SendResponse(msg.RequestID, messageBuilder.Response(response).Build())
	if err != nil {
		log.Println("Failed to send response:", err.Error())
	}
}

func (dht *DHT) knownOuterNodesRequest(targetID string, nodes int) error {
	ctx, err := NewContextBuilder(dht).SetDefaultNode().Build()
	if err != nil {
		return err
	}
	targetNode, exist, err := dht.FindNode(ctx, targetID)
	if err != nil {
		return err
	}
	if !exist {
		err = errors.New("target for relay request not found")
		return err
	}

	request := message.NewKnownOuterNodesMessage(dht.htFromCtx(ctx).Origin, targetNode, nodes)
	future, err := dht.transport.SendRequest(request)

	if err != nil {
		return err
	}

	select {
	case rsp := <-future.Result():
		if rsp == nil {
			return err
		}

		response := rsp.Data.(*message.ResponseKnownOuterNodes)
		dht.handleKnownOuterNodes(ctx, response, targetID)

	case <-time.After(dht.options.MessageTimeout):
		future.Cancel()
		err = errors.New("timeout")
		return err
	}

	return nil
}

func (dht *DHT) handleKnownOuterNodes(ctx Context, response *message.ResponseKnownOuterNodes, targetID string) {
	if response.OuterNodes > dht.subnet.HighKnownNodes.OuterNodes { // update data
		dht.subnet.HighKnownNodes.OuterNodes = response.OuterNodes
		dht.subnet.HighKnownNodes.ID = response.ID
	}
	if (response.OuterNodes > dht.subnet.HighKnownNodes.SelfKnownOuterNodes) &&
		(dht.proxy.ProxyNodesCount() == 0) {
		err := dht.AuthenticationRequest(ctx, "begin", targetID)
		if err != nil {
			log.Println("error to send auth request: ", err)
		}
		err = dht.RelayRequest(ctx, "start", targetID)
		if err != nil {
			log.Println("error to send relay request: ", err)
		}
	}
}

func (dht *DHT) htFromCtx(ctx Context) *routing.HashTable {
	htIdx := ctx.Value(ctxTableIndex).(int)
	return dht.tables[htIdx]
}
