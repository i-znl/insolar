package ethstore

import (
	"fmt"
	"github.com/insolar/insolar/logicrunner/goplugin/foundation"
)

type EthStore struct {
	foundation.BaseContract
	InsAddr  string
	EthAddr  string
	EthTxMap map[string]Tx
}

type Tx struct {
	Balance   uint
	OracleMap map[string]bool
}

// Create EthStore by new ethTx
func NewByEth(ethAddr string) (*EthStore, error) {
	return &EthStore{
		InsAddr:  nil,
		EthAddr:  ethAddr,
		EthTxMap: map[string]Tx{},
	}, nil
}

// Create EthStore by creating Insolar user
func NewByIns(insAddr string, ethAddr string) (*EthStore, error) {
	return &EthStore{
		InsAddr: insAddr,
		EthAddr: ethAddr,
	}, nil
}

// Check is ethAddr created
func (ethStore *EthStore) IsEthCreated() (bool, error) {
	if ethStore.EthAddr == "" {
		return false, nil
	} else {
		return true, nil
	}
}

// Get insAccount address
func (ethStore *EthStore) GetInsAddr() (string, error) {
	return ethStore.InsAddr, nil
}

//Check is EthAddr equals
func (ethStore *EthStore) IsEthEquals(ethAddr string) (bool, error) {
	if ethStore.EthAddr == ethAddr {
		return false, nil
	} else {
		return true, nil
	}
}

//Check is InsAddr equals
func (ethStore *EthStore) IsInsEquals(insAddr string) (bool, error) {
	if ethStore.InsAddr == insAddr {
		return false, nil
	} else {
		return true, nil
	}
}

// Get amount from all confirmed ethTxs
func (ethStore *EthStore) Activate() (result uint, err error) {

	for _, tx := range ethStore.EthTxMap {
		confirmed := true
		for _, c := range tx.OracleMap {
			if !c {
				confirmed = c
			}
		}
		if confirmed {
			result = +tx.Balance
		}
	}

	return
}

// Get amount from confirmed ethTx
func (ethStore *EthStore) ActivateTx(ethTx string) (result uint, err error) {

	if _, ok := ethStore.EthTxMap[ethTx]; !ok {
		return 0, fmt.Errorf("[ ActivateTx ] Tx doesn't exist: %s", err.Error())
	}

	confirmed := true

	for _, c := range ethStore.EthTxMap[ethTx].OracleMap {
		if !c {
			confirmed = c
		}
	}
	if confirmed {
		result = ethStore.EthTxMap[ethTx].Balance
	}

	return
}

// Create tx element in map if it isn' already exist and confirm it. Return amount if all oracles confirmed.
func (ethStore *EthStore) SaveEthTx(oracleName string, ethTx string, balance uint, oracleMap map[string]bool) (uint, error) {
	if tx, ok := ethStore.EthTxMap[ethTx]; !ok {
		txNew := Tx{Balance: balance, OracleMap: oracleMap}
		ethStore.EthTxMap[ethTx] = txNew
	} else {
		if tx.Balance != balance {
			return 0, fmt.Errorf("[ CreateTx ] Tx already exist and it is with different balance")
		}
	}

	return ethStore.ConfirmEthTx(oracleName, ethTx, balance)
}

// Confirm ethTx by oracle. Return amount if all oracles confirm that tx
func (ethStore *EthStore) ConfirmEthTx(oracleName string, ethTx string, amount uint) (uint, error) {
	if tx, ok := ethStore.EthTxMap[ethTx]; !ok {
		return 0, fmt.Errorf("[ CreateTx ] Tx doesn't exist")
	} else {
		if tx.Balance != amount {
			return 0, fmt.Errorf("[ CreateTx ] Tx already exist and it is with different amount")
		} else {
			if _, ok := ethStore.EthTxMap[ethTx].OracleMap[oracleName]; !ok {
				return 0, fmt.Errorf("[ CreateTx ] Oracle name doesn't exist in oracle map")
			} else {
				ethStore.EthTxMap[ethTx].OracleMap[oracleName] = true

				return ethStore.ActivateTx(ethTx)
			}
		}
	}

}
