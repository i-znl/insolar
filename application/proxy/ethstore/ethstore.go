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

package ethstore

import (
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/logicrunner/goplugin/foundation"
	"github.com/insolar/insolar/logicrunner/goplugin/proxyctx"
)

type Tx struct {
	Balance   uint
	OracleMap map[string]bool
}

// PrototypeReference to prototype of this contract
// error checking hides in generator
var PrototypeReference, _ = core.NewRefFromBase58("11114vu9tJRExCgiHkFGbGw77xPmtBekU2Lo8wHMVW.11111111111111111111111111111111")

// EthStore holds proxy type
type EthStore struct {
	Reference core.RecordRef
	Prototype core.RecordRef
	Code      core.RecordRef
}

// ContractConstructorHolder holds logic with object construction
type ContractConstructorHolder struct {
	constructorName string
	argsSerialized  []byte
}

// AsChild saves object as child
func (r *ContractConstructorHolder) AsChild(objRef core.RecordRef) (*EthStore, error) {
	ref, err := proxyctx.Current.SaveAsChild(objRef, *PrototypeReference, r.constructorName, r.argsSerialized)
	if err != nil {
		return nil, err
	}
	return &EthStore{Reference: ref}, nil
}

// AsDelegate saves object as delegate
func (r *ContractConstructorHolder) AsDelegate(objRef core.RecordRef) (*EthStore, error) {
	ref, err := proxyctx.Current.SaveAsDelegate(objRef, *PrototypeReference, r.constructorName, r.argsSerialized)
	if err != nil {
		return nil, err
	}
	return &EthStore{Reference: ref}, nil
}

// GetObject returns proxy object
func GetObject(ref core.RecordRef) (r *EthStore) {
	return &EthStore{Reference: ref}
}

// GetPrototype returns reference to the prototype
func GetPrototype() core.RecordRef {
	return *PrototypeReference
}

// GetImplementationFrom returns proxy to delegate of given type
func GetImplementationFrom(object core.RecordRef) (*EthStore, error) {
	ref, err := proxyctx.Current.GetDelegate(object, *PrototypeReference)
	if err != nil {
		return nil, err
	}
	return GetObject(ref), nil
}

// NewByEth is constructor
func NewByEth(ethAddr string) *ContractConstructorHolder {
	var args [1]interface{}
	args[0] = ethAddr

	var argsSerialized []byte
	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		panic(err)
	}

	return &ContractConstructorHolder{constructorName: "NewByEth", argsSerialized: argsSerialized}
}

// NewByIns is constructor
func NewByIns(insAddr string, ethAddr string) *ContractConstructorHolder {
	var args [2]interface{}
	args[0] = insAddr
	args[1] = ethAddr

	var argsSerialized []byte
	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		panic(err)
	}

	return &ContractConstructorHolder{constructorName: "NewByIns", argsSerialized: argsSerialized}
}

// GetReference returns reference of the object
func (r *EthStore) GetReference() core.RecordRef {
	return r.Reference
}

// GetPrototype returns reference to the code
func (r *EthStore) GetPrototype() (core.RecordRef, error) {
	if r.Prototype.IsEmpty() {
		ret := [2]interface{}{}
		var ret0 core.RecordRef
		ret[0] = &ret0
		var ret1 *foundation.Error
		ret[1] = &ret1

		res, err := proxyctx.Current.RouteCall(r.Reference, true, "GetPrototype", make([]byte, 0), *PrototypeReference)
		if err != nil {
			return ret0, err
		}

		err = proxyctx.Current.Deserialize(res, &ret)
		if err != nil {
			return ret0, err
		}

		if ret1 != nil {
			return ret0, ret1
		}

		r.Prototype = ret0
	}

	return r.Prototype, nil

}

// GetCode returns reference to the code
func (r *EthStore) GetCode() (core.RecordRef, error) {
	if r.Code.IsEmpty() {
		ret := [2]interface{}{}
		var ret0 core.RecordRef
		ret[0] = &ret0
		var ret1 *foundation.Error
		ret[1] = &ret1

		res, err := proxyctx.Current.RouteCall(r.Reference, true, "GetCode", make([]byte, 0), *PrototypeReference)
		if err != nil {
			return ret0, err
		}

		err = proxyctx.Current.Deserialize(res, &ret)
		if err != nil {
			return ret0, err
		}

		if ret1 != nil {
			return ret0, ret1
		}

		r.Code = ret0
	}

	return r.Code, nil
}

// IsEthCreated is proxy generated method
func (r *EthStore) IsEthCreated() (bool, error) {
	var args [0]interface{}

	var argsSerialized []byte

	ret := [2]interface{}{}
	var ret0 bool
	ret[0] = &ret0
	var ret1 *foundation.Error
	ret[1] = &ret1

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return ret0, err
	}

	res, err := proxyctx.Current.RouteCall(r.Reference, true, "IsEthCreated", argsSerialized, *PrototypeReference)
	if err != nil {
		return ret0, err
	}

	err = proxyctx.Current.Deserialize(res, &ret)
	if err != nil {
		return ret0, err
	}

	if ret1 != nil {
		return ret0, ret1
	}
	return ret0, nil
}

// IsEthCreatedNoWait is proxy generated method
func (r *EthStore) IsEthCreatedNoWait() error {
	var args [0]interface{}

	var argsSerialized []byte

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return err
	}

	_, err = proxyctx.Current.RouteCall(r.Reference, false, "IsEthCreated", argsSerialized, *PrototypeReference)
	if err != nil {
		return err
	}

	return nil
}

// GetInsAddr is proxy generated method
func (r *EthStore) GetInsAddr() (string, error) {
	var args [0]interface{}

	var argsSerialized []byte

	ret := [2]interface{}{}
	var ret0 string
	ret[0] = &ret0
	var ret1 *foundation.Error
	ret[1] = &ret1

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return ret0, err
	}

	res, err := proxyctx.Current.RouteCall(r.Reference, true, "GetInsAddr", argsSerialized, *PrototypeReference)
	if err != nil {
		return ret0, err
	}

	err = proxyctx.Current.Deserialize(res, &ret)
	if err != nil {
		return ret0, err
	}

	if ret1 != nil {
		return ret0, ret1
	}
	return ret0, nil
}

// GetInsAddrNoWait is proxy generated method
func (r *EthStore) GetInsAddrNoWait() error {
	var args [0]interface{}

	var argsSerialized []byte

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return err
	}

	_, err = proxyctx.Current.RouteCall(r.Reference, false, "GetInsAddr", argsSerialized, *PrototypeReference)
	if err != nil {
		return err
	}

	return nil
}

// IsEthEquals is proxy generated method
func (r *EthStore) IsEthEquals(ethAddr string) (bool, error) {
	var args [1]interface{}
	args[0] = ethAddr

	var argsSerialized []byte

	ret := [2]interface{}{}
	var ret0 bool
	ret[0] = &ret0
	var ret1 *foundation.Error
	ret[1] = &ret1

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return ret0, err
	}

	res, err := proxyctx.Current.RouteCall(r.Reference, true, "IsEthEquals", argsSerialized, *PrototypeReference)
	if err != nil {
		return ret0, err
	}

	err = proxyctx.Current.Deserialize(res, &ret)
	if err != nil {
		return ret0, err
	}

	if ret1 != nil {
		return ret0, ret1
	}
	return ret0, nil
}

// IsEthEqualsNoWait is proxy generated method
func (r *EthStore) IsEthEqualsNoWait(ethAddr string) error {
	var args [1]interface{}
	args[0] = ethAddr

	var argsSerialized []byte

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return err
	}

	_, err = proxyctx.Current.RouteCall(r.Reference, false, "IsEthEquals", argsSerialized, *PrototypeReference)
	if err != nil {
		return err
	}

	return nil
}

// IsInsEquals is proxy generated method
func (r *EthStore) IsInsEquals(insAddr string) (bool, error) {
	var args [1]interface{}
	args[0] = insAddr

	var argsSerialized []byte

	ret := [2]interface{}{}
	var ret0 bool
	ret[0] = &ret0
	var ret1 *foundation.Error
	ret[1] = &ret1

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return ret0, err
	}

	res, err := proxyctx.Current.RouteCall(r.Reference, true, "IsInsEquals", argsSerialized, *PrototypeReference)
	if err != nil {
		return ret0, err
	}

	err = proxyctx.Current.Deserialize(res, &ret)
	if err != nil {
		return ret0, err
	}

	if ret1 != nil {
		return ret0, ret1
	}
	return ret0, nil
}

// IsInsEqualsNoWait is proxy generated method
func (r *EthStore) IsInsEqualsNoWait(insAddr string) error {
	var args [1]interface{}
	args[0] = insAddr

	var argsSerialized []byte

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return err
	}

	_, err = proxyctx.Current.RouteCall(r.Reference, false, "IsInsEquals", argsSerialized, *PrototypeReference)
	if err != nil {
		return err
	}

	return nil
}

// Activate is proxy generated method
func (r *EthStore) Activate() (uint, error) {
	var args [0]interface{}

	var argsSerialized []byte

	ret := [2]interface{}{}
	var ret0 uint
	ret[0] = &ret0
	var ret1 *foundation.Error
	ret[1] = &ret1

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return ret0, err
	}

	res, err := proxyctx.Current.RouteCall(r.Reference, true, "Activate", argsSerialized, *PrototypeReference)
	if err != nil {
		return ret0, err
	}

	err = proxyctx.Current.Deserialize(res, &ret)
	if err != nil {
		return ret0, err
	}

	if ret1 != nil {
		return ret0, ret1
	}
	return ret0, nil
}

// ActivateNoWait is proxy generated method
func (r *EthStore) ActivateNoWait() error {
	var args [0]interface{}

	var argsSerialized []byte

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return err
	}

	_, err = proxyctx.Current.RouteCall(r.Reference, false, "Activate", argsSerialized, *PrototypeReference)
	if err != nil {
		return err
	}

	return nil
}

// ActivateTx is proxy generated method
func (r *EthStore) ActivateTx(ethTx string) (uint, error) {
	var args [1]interface{}
	args[0] = ethTx

	var argsSerialized []byte

	ret := [2]interface{}{}
	var ret0 uint
	ret[0] = &ret0
	var ret1 *foundation.Error
	ret[1] = &ret1

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return ret0, err
	}

	res, err := proxyctx.Current.RouteCall(r.Reference, true, "ActivateTx", argsSerialized, *PrototypeReference)
	if err != nil {
		return ret0, err
	}

	err = proxyctx.Current.Deserialize(res, &ret)
	if err != nil {
		return ret0, err
	}

	if ret1 != nil {
		return ret0, ret1
	}
	return ret0, nil
}

// ActivateTxNoWait is proxy generated method
func (r *EthStore) ActivateTxNoWait(ethTx string) error {
	var args [1]interface{}
	args[0] = ethTx

	var argsSerialized []byte

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return err
	}

	_, err = proxyctx.Current.RouteCall(r.Reference, false, "ActivateTx", argsSerialized, *PrototypeReference)
	if err != nil {
		return err
	}

	return nil
}

// SaveEthTx is proxy generated method
func (r *EthStore) SaveEthTx(oracleName string, ethTx string, balance uint, oracleMap map[string]bool) (uint, error) {
	var args [4]interface{}
	args[0] = oracleName
	args[1] = ethTx
	args[2] = balance
	args[3] = oracleMap

	var argsSerialized []byte

	ret := [2]interface{}{}
	var ret0 uint
	ret[0] = &ret0
	var ret1 *foundation.Error
	ret[1] = &ret1

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return ret0, err
	}

	res, err := proxyctx.Current.RouteCall(r.Reference, true, "SaveEthTx", argsSerialized, *PrototypeReference)
	if err != nil {
		return ret0, err
	}

	err = proxyctx.Current.Deserialize(res, &ret)
	if err != nil {
		return ret0, err
	}

	if ret1 != nil {
		return ret0, ret1
	}
	return ret0, nil
}

// SaveEthTxNoWait is proxy generated method
func (r *EthStore) SaveEthTxNoWait(oracleName string, ethTx string, balance uint, oracleMap map[string]bool) error {
	var args [4]interface{}
	args[0] = oracleName
	args[1] = ethTx
	args[2] = balance
	args[3] = oracleMap

	var argsSerialized []byte

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return err
	}

	_, err = proxyctx.Current.RouteCall(r.Reference, false, "SaveEthTx", argsSerialized, *PrototypeReference)
	if err != nil {
		return err
	}

	return nil
}

// ConfirmEthTx is proxy generated method
func (r *EthStore) ConfirmEthTx(oracleName string, ethTx string, amount uint) (uint, error) {
	var args [3]interface{}
	args[0] = oracleName
	args[1] = ethTx
	args[2] = amount

	var argsSerialized []byte

	ret := [2]interface{}{}
	var ret0 uint
	ret[0] = &ret0
	var ret1 *foundation.Error
	ret[1] = &ret1

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return ret0, err
	}

	res, err := proxyctx.Current.RouteCall(r.Reference, true, "ConfirmEthTx", argsSerialized, *PrototypeReference)
	if err != nil {
		return ret0, err
	}

	err = proxyctx.Current.Deserialize(res, &ret)
	if err != nil {
		return ret0, err
	}

	if ret1 != nil {
		return ret0, ret1
	}
	return ret0, nil
}

// ConfirmEthTxNoWait is proxy generated method
func (r *EthStore) ConfirmEthTxNoWait(oracleName string, ethTx string, amount uint) error {
	var args [3]interface{}
	args[0] = oracleName
	args[1] = ethTx
	args[2] = amount

	var argsSerialized []byte

	err := proxyctx.Current.Serialize(args, &argsSerialized)
	if err != nil {
		return err
	}

	_, err = proxyctx.Current.RouteCall(r.Reference, false, "ConfirmEthTx", argsSerialized, *PrototypeReference)
	if err != nil {
		return err
	}

	return nil
}
