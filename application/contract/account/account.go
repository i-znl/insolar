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

package account

import (
	"encoding/json"
	"fmt"
	"github.com/insolar/insolar/application/proxy/account"
	"github.com/insolar/insolar/core"
	"github.com/insolar/insolar/logicrunner/goplugin/foundation"
)

// Account - wallet of member with ethHash
type Account struct {
	foundation.BaseContract
	Balance   uint
	EthHash   string
	SecretMap map[string]uint
}

var INSATTR_Call_API = true

// Call method for authorized calls
func (a *Account) Call(rootDomain core.RecordRef, method string, params []byte, seed []byte, sign []byte) (interface{}, error) {

	switch method {
	case "GetBalance":
		return a.GetBalance()
	case "Transfer":
		return a.transfer(params)
	case "SecretTransfer":
		return a.secretTransfer(params)
	case "ApplySecret":
		return a.applySecret(params)
	}
	return nil, &foundation.Error{S: "Unknown method"}
}

//
//// New creates new account
//func New(accountJson []byte) (account *Account, err error) {
//	if err = json.Unmarshal(accountJson, account); err != nil {
//		return nil, fmt.Errorf("[ New ] Can't Unmarshal account: %s", err.Error())
//	}
//	return
//}

// New creates new account
func New(ethHash string, balance uint) (account *Account, err error) {
	return &Account{
			EthHash: ethHash,
			Balance: balance,
		},
		nil
}

func (a *Account) ReceiveTransfer(amount uint) (interface{}, error) {
	a.Balance = a.Balance + amount

	return nil, nil
}

func (a *Account) ReceiveSecretTransfer(amount uint, secret string) (interface{}, error) {
	a.SecretMap[secret] = amount

	return nil, nil
}

func (a *Account) GetBalance() (uint, error) {
	return a.Balance, nil
}

func (a *Account) transfer(params []byte) (interface{}, error) {
	transfer := struct {
		amount uint   `json:"amount"`
		to     string `json:"to"`
	}{}

	if err := json.Unmarshal(params, &transfer); err != nil {
		return nil, fmt.Errorf("[ transfer ] Can't unmarshal params: %s", err.Error())
	}
	toRef, err := core.NewRefFromBase58(transfer.to)
	if err != nil {
		return nil, fmt.Errorf("[ transfer ] Failed to parse 'toRef' param: %s", err.Error())
	}
	if a.GetReference() == *toRef {
		return nil, fmt.Errorf("[ transfer ] Recipient must be different from the sender")
	}

	toAccount, err := account.GetImplementationFrom(*toRef)
	if err != nil {
		return nil, fmt.Errorf("[ transfer ] Can't get implementation: %s", err.Error())
	}

	a.Balance = a.Balance - transfer.amount

	return toAccount.ReceiveTransfer(transfer.amount)
}

func (a *Account) secretTransfer(params []byte) (interface{}, error) {
	secretTransfer := struct {
		amount uint   `json:"amount"`
		to     string `json:"to"`
		secret string `json:"secret"`
	}{}

	if err := json.Unmarshal(params, &secretTransfer); err != nil {
		return nil, fmt.Errorf("[ secretTransfer ] Can't unmarshal params: %s", err.Error())
	}
	toRef, err := core.NewRefFromBase58(secretTransfer.to)
	if err != nil {
		return nil, fmt.Errorf("[ secretTransfer ] Failed to parse 'toRef' param: %s", err.Error())
	}
	if a.GetReference() == *toRef {
		return nil, fmt.Errorf("[ secretTransfer ] Recipient must be different from the sender")
	}

	toAccount, err := account.GetImplementationFrom(*toRef)
	if err != nil {
		return nil, fmt.Errorf("[ secretTransfer ] Can't get implementation: %s", err.Error())
	}

	a.Balance = a.Balance - secretTransfer.amount

	return toAccount.ReceiveSecretTransfer(secretTransfer.amount, secretTransfer.secret)
}

func (a *Account) applySecret(params []byte) (interface{}, error) {
	var secret string

	if err := json.Unmarshal(params, &secret); err != nil {
		return nil, fmt.Errorf("[ applySecret ] Can't unmarshal params: %s", err.Error())
	}

	return a.ReceiveTransfer(a.SecretMap[secret])
}

func (a *Account) AddToBalance(amount uint) error {
	a.Balance = +amount

	return nil
}
