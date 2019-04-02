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

package rootdomain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto/sha3"

	"github.com/insolar/insolar/application/proxy/account"
	"github.com/insolar/insolar/application/proxy/ethstore"
	"github.com/insolar/insolar/application/proxy/member"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/logicrunner/goplugin/foundation"
)

// RootDomain is smart contract representing entrance point to system
type RootDomain struct {
	foundation.BaseContract
	RootMember    core.RecordRef   `json:"rootMember"`
	OracleMembers []core.RecordRef `json:"oracleMembers"`
	NodeDomain    core.RecordRef   `json:"node_domain"`
}

var INSATTR_CreateMember_API = true

func decodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	return b
}

func hash(msg string) string {

	hash := sha3.NewKeccak256()

	var buf []byte
	//hash.Write([]byte{0xcc})
	hash.Write(decodeHex(msg))
	buf = hash.Sum(nil)

	return hex.EncodeToString(buf)
}

// CreateMember processes create member request
func (rd *RootDomain) CreateMember(name string, key string) (string, error) {
	memberHolder := member.New(name, key)
	memberObject, err := memberHolder.AsChild(rd.GetReference())
	if err != nil {
		return "", fmt.Errorf("[ CreateMember ] Can't save as child: %s", err.Error())
	}

	ethAddr := hash(key)

	var balance uint
	iterator, err := rd.NewChildrenTypedIterator(ethstore.GetPrototype())
	if err != nil {
		return "", fmt.Errorf("[ CreateMember ] Can't get children: %s", err.Error())
	}

	var ethStoreObject *ethstore.EthStore
	for iterator.HasNext() {
		cref, err := iterator.Next()
		if err != nil {
			return "", fmt.Errorf("[ CreateMember ] Can't get next child: %s", err.Error())
		}

		ethStoreObject = ethstore.GetObject(cref)

		b, err := ethStoreObject.IsEthEquals(ethAddr)
		if err != nil {
			return "", fmt.Errorf("[ CreateMember ] Can't compare ethereuum address: %s", err.Error())
		}

		if b {
			break
		}
	}

	if ethStoreObject != nil {
		balance, err = ethStoreObject.Activate()
		if err != nil {
			return "", fmt.Errorf("[ CreateMember ] Can't activate store: %s", err.Error())
		}

		accountHolder := account.New(ethAddr, balance)
		_, err = accountHolder.AsDelegate(memberObject.GetReference())
		if err != nil {
			return "", fmt.Errorf("[ CreateMember ] Can't save account based on ethStore as delegate: %s", err.Error())
		}
	} else {
		accountHolder := account.New(ethAddr, balance)
		accountObject, err := accountHolder.AsDelegate(memberObject.GetReference())
		if err != nil {
			return "", fmt.Errorf("[ CreateMember ] Can't save clear account as delegate: %s", err.Error())
		}

		ethStoreHolder := ethstore.NewByIns(accountObject.Reference.String(), ethAddr)
		_, err = ethStoreHolder.AsDelegate(rd.GetReference())
		if err != nil {
			return "", fmt.Errorf("[ CreateMember ] Can't save ethStore as delegate: %s", err.Error())
		}
	}

	return memberObject.GetReference().String(), nil
}

// SaveEthTx validate and save ethTx from oracle and increase ins-account balance
func (rd *RootDomain) SaveEthTx(ethAddr string, amount uint, ethTxHash string, oracleName string) (interface{}, error) {

	iterator, err := rd.NewChildrenTypedIterator(ethstore.GetPrototype())
	if err != nil {
		return "", fmt.Errorf("[ SaveEthTx ] Can't get children: %s", err.Error())
	}

	var ethStoreObject *ethstore.EthStore
	for iterator.HasNext() {
		cref, err := iterator.Next()
		if err != nil {
			return "", fmt.Errorf("[ SaveEthTx ] Can't get next child: %s", err.Error())
		}

		ethStoreObject = ethstore.GetObject(cref)

		b, err := ethStoreObject.IsEthEquals(ethAddr)
		if err != nil {
			return "", fmt.Errorf("[ SaveEthTx ] Can't compare ethereuum address: %s", err.Error())
		}

		if b {
			break
		}
	}

	oracleMap := map[string]bool{}
	for _, om := range rd.OracleMembers {
		oracleMap[om.String()] = false
	}

	if ethStoreObject != nil {

		confirmedAmount, err := ethStoreObject.SaveEthTx(oracleName, ethTxHash, amount, oracleMap)
		if err != nil {
			return "", fmt.Errorf("[ SaveEthTx ] Can't save transaction in : %s", err.Error())
		}

		if confirmedAmount != amount {
			return "", fmt.Errorf("[ SaveEthTx ] confirmed amount isn't equal executed amount : %s", err.Error())
		}

		if confirmedAmount != 0 {
			insAddr, err := ethStoreObject.GetInsAddr()
			if err != nil {
				return "", fmt.Errorf("[ SaveEthTx ] Can't get insolar address : %s", err.Error())
			}
			if insAddr != "" {
				insAddrRef, err := core.NewRefFromBase58(insAddr)
				if err != nil {
					return nil, fmt.Errorf("[ SaveEthTx ] Failed to parse insolar address reference: %s", err.Error())
				}

				accountObject := account.GetObject(*insAddrRef)
				err = accountObject.AddToBalance(confirmedAmount)
				if err != nil {
					return nil, fmt.Errorf("[ SaveEthTx ] Failed to add confirmed amount to account balance: %s", err.Error())
				}
			}
		}
	} else {

		ethStoreHolder := ethstore.NewByEth(ethAddr)
		ethstoreObject, err := ethStoreHolder.AsDelegate(rd.GetReference())
		if err != nil {
			return "", fmt.Errorf("[ SaveEthTx ] Can't save ethStore as delegate: %s", err.Error())
		}
		_, err = ethstoreObject.SaveEthTx(oracleName, ethTxHash, amount, oracleMap)
		if err != nil {
			return "", fmt.Errorf("[ SaveEthTx ]  Can't save transaction in new ethStore: %s", err.Error())
		}
	}

	return nil, nil
}

// Returns root member's reference
func (rd *RootDomain) GetRootMemberRef() (*core.RecordRef, error) {
	return &rd.RootMember, nil
}

// Returns oracle members's reference
func (rd *RootDomain) GetOracleMemberRef() ([]core.RecordRef, error) {
	return rd.OracleMembers, nil
}

// Returns user name and balance in map
func (rd *RootDomain) getUserInfoMap(m *member.Member) (map[string]interface{}, error) {
	a, err := account.GetImplementationFrom(m.GetReference())
	if err != nil {
		return nil, fmt.Errorf("[ getUserInfoMap ] Can't get implementation: %s", err.Error())
	}

	name, err := m.GetName()
	if err != nil {
		return nil, fmt.Errorf("[ getUserInfoMap ] Can't get name: %s", err.Error())
	}

	balance, err := a.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("[ getUserInfoMap ] Can't get total balance: %s", err.Error())
	}
	return map[string]interface{}{
		"member":  name,
		"balance": balance,
	}, nil
}

// DumpUserInfo processes dump user info request
func (rd *RootDomain) DumpUserInfo(reference string) ([]byte, error) {
	caller := *rd.GetContext().Caller
	ref, err := core.NewRefFromBase58(reference)
	if err != nil {
		return nil, fmt.Errorf("[ DumpUserInfo ] Failed to parse reference: %s", err.Error())
	}
	if *ref != caller && caller != rd.RootMember {
		return nil, fmt.Errorf("[ DumpUserInfo ] You can dump only yourself")
	}
	m := member.GetObject(*ref)

	res, err := rd.getUserInfoMap(m)
	if err != nil {
		return nil, fmt.Errorf("[ DumpUserInfo ] Problem with making request: %s", err.Error())
	}

	return json.Marshal(res)
}

// DumpAllUsers processes dump all users request
func (rd *RootDomain) DumpAllUsers() ([]byte, error) {
	if *rd.GetContext().Caller != rd.RootMember {
		return nil, fmt.Errorf("[ DumpAllUsers ] Only root can call this method")
	}
	res := []map[string]interface{}{}
	iterator, err := rd.NewChildrenTypedIterator(member.GetPrototype())
	if err != nil {
		return nil, fmt.Errorf("[ DumpAllUsers ] Can't get children: %s", err.Error())
	}

	for iterator.HasNext() {
		cref, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("[ DumpAllUsers ] Can't get next child: %s", err.Error())
		}

		if cref == rd.RootMember {
			continue
		}
		m := member.GetObject(cref)
		userInfo, err := rd.getUserInfoMap(m)
		if err != nil {
			return nil, fmt.Errorf("[ DumpAllUsers ] Problem with making request: %s", err.Error())
		}
		res = append(res, userInfo)
	}
	resJSON, _ := json.Marshal(res)
	return resJSON, nil
}

var INSATTR_Info_API = true

// Info returns information about basic objects
func (rd *RootDomain) Info() (interface{}, error) {
	res := map[string]interface{}{
		"root_member":    rd.RootMember.String(),
		"oracle_members": rd.OracleMembers,
		"node_domain":    rd.NodeDomain.String(),
	}
	resJSON, err := json.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("[ Info ] Can't marshal res: %s", err.Error())
	}
	return resJSON, nil
}

// GetNodeDomainRef returns reference of NodeDomain instance
func (rd *RootDomain) GetNodeDomainRef() (core.RecordRef, error) {
	return rd.NodeDomain, nil
}

// NewRootDomain creates new RootDomain
func NewRootDomain() (*RootDomain, error) {
	return &RootDomain{}, nil
}
