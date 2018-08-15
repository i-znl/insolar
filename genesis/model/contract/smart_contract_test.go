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

package contract

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/insolar/insolar/genesis/mock/storage"
	"github.com/insolar/insolar/genesis/model/class"
	"github.com/insolar/insolar/genesis/model/object"
	"github.com/stretchr/testify/assert"
)

type mockComposite struct {
	interfaceKeyIdx int
}

func GetTestIntarfaceKey(idx int) string {
	return "mockComposite" + strconv.Itoa(idx)
}

func newMockComposite(idx int) mockComposite {
	return mockComposite{
		interfaceKeyIdx: idx,
	}
}

func (mc *mockComposite) GetInterfaceKey() string {
	return GetTestIntarfaceKey(mc.interfaceKeyIdx)
}

func TestBaseCompositeCollection_GetInterfaceKey(t *testing.T) {
	compCollection := NewBaseCompositeCollection()
	assert.Equal(t, class.CompositeCollectionID, compCollection.GetInterfaceKey())
}

func TestBaseCompositeCollection_Add(t *testing.T) {
	compCollection := NewBaseCompositeCollection()

	numEl := 10
	for i := 0; i < numEl; i++ {
		mc := newMockComposite(i)
		compCollection.Add(&mc)
		assert.Len(t, compCollection.storage, i+1)
		assert.Equal(t, compCollection.storage[i].GetInterfaceKey(), GetTestIntarfaceKey(i))
	}

}

func TestBaseCompositeCollection_Add_SameInterfaceKeys(t *testing.T) {
	compCollection := NewBaseCompositeCollection()
	testIdx := 77
	mc := newMockComposite(testIdx)
	compCollection.Add(&mc)
	compCollection.Add(&mc)

	assert.Len(t, compCollection.storage, 2)
	assert.Equal(t, compCollection.storage[0].GetInterfaceKey(), GetTestIntarfaceKey(testIdx))
	assert.Equal(t, compCollection.storage[1].GetInterfaceKey(), GetTestIntarfaceKey(testIdx))
}

func TestBaseCompositeCollection_GetList(t *testing.T) {
	compCollection := NewBaseCompositeCollection()
	assert.Len(t, compCollection.GetList(), 0)

	numEl := 10
	for i := 0; i < numEl; i++ {
		mc := newMockComposite(i)
		compCollection.Add(&mc)
	}

	assert.Len(t, compCollection.GetList(), numEl)
}

func TestNewBaseCompositeCollection(t *testing.T) {
	compCollection := NewBaseCompositeCollection()
	assert.Len(t, compCollection.storage, 0)

}

type mockProxy struct {
	reference object.Reference
}

func (p *mockProxy) GetClassID() string {
	return "mockProxy"
}

func (p *mockProxy) GetReference() object.Reference {
	return p.reference
}

func (p *mockProxy) SetReference(reference object.Reference) {
	p.reference = reference
}

type mockChildProxy struct {
	mockProxy
	parent object.Parent
}

func (c *mockChildProxy) GetClassID() string {
	return "mockChild"
}

func (c *mockChildProxy) GetParent() object.Parent {
	return c.parent
}

type mockParent struct {
	ContextStorage storage.Storage
}

func (p *mockParent) GetClassID() string {
	return "mockParent"
}

func (p *mockParent) GetChildStorage() storage.Storage {
	return nil
}

func (p *mockParent) AddChild(child object.Child) (string, error) {
	return "", nil
}

func (p *mockParent) GetChild(key string) (object.Child, error) {
	return nil, nil
}

func (p *mockParent) GetContext() []string {
	return []string{}
}

func (p *mockParent) GetContextStorage() storage.Storage {
	return p.ContextStorage
}

type BaseComposite struct{}

func (bc *BaseComposite) GetInterfaceKey() string {
	return "BaseComposite"
}

func (bc *BaseComposite) GetClassID() string {
	return "BaseComposite"
}

func (bc *BaseComposite) GetParent() object.Parent {
	return nil
}

func (bc *BaseComposite) GetReference() object.Reference {
	return nil
}

type BaseCompositeNotChild struct{}

func (bc *BaseCompositeNotChild) GetInterfaceKey() string {
	return "BaseCompositeNotChild"
}

func (bc *BaseCompositeNotChild) GetClassID() string {
	return "BaseCompositeNotChild"
}

func (bc *BaseCompositeNotChild) GetReference() object.Reference {
	return nil
}

func (bc *BaseComposite) SetReference(reference object.Reference) {}

type BaseCompositeFactory struct{}

func (bcf *BaseCompositeFactory) SetReference(reference object.Reference) {
}

func (bcf *BaseCompositeFactory) GetReference() object.Reference {
	return nil
}

func (bcf *BaseCompositeFactory) GetParent() object.Parent {
	return nil
}

func (bcf *BaseCompositeFactory) GetClassID() string {
	return "BaseComposite"
}

func (bcf *BaseCompositeFactory) GetInterfaceKey() string {
	return "BaseComposite"
}

func (bcf *BaseCompositeFactory) Create(parent object.Parent) (object.Composite, error) {
	return &BaseComposite{}, nil
}

type BaseCompositeFactoryWithError struct{}

func (bcf *BaseCompositeFactoryWithError) GetClassID() string {
	return "BaseCompositeFactoryWithError_ID"
}

func (bcf *BaseCompositeFactoryWithError) SetReference(reference object.Reference) {
}

func (bcf *BaseCompositeFactoryWithError) GetReference() object.Reference {
	return nil
}

func (bcf *BaseCompositeFactoryWithError) GetParent() object.Parent {
	return nil
}

func (bcf *BaseCompositeFactoryWithError) GetInterfaceKey() string {
	return "BaseCompositeFactoryWithError_ID"
}

func (bcf *BaseCompositeFactoryWithError) Create(parent object.Parent) (object.Composite, error) {
	return nil, fmt.Errorf("composite factory create error")
}

type BaseCompositeNotChildFactory struct{}

func (bcf *BaseCompositeNotChildFactory) GetClassID() string {
	return "BaseCompositeNotChildFactory_ID"
}

func (bcf *BaseCompositeNotChildFactory) SetReference(reference object.Reference) {
}

func (bcf *BaseCompositeNotChildFactory) GetReference() object.Reference {
	return nil
}

func (bcf *BaseCompositeNotChildFactory) GetParent() object.Parent {
	return nil
}

func (bcf *BaseCompositeNotChildFactory) GetInterfaceKey() string {
	return "BaseCompositeNotChildFactory"
}

func (bcf *BaseCompositeNotChildFactory) Create(parent object.Parent) (object.Composite, error) {
	return &BaseCompositeNotChild{}, nil
}

func TestNewBaseSmartContract(t *testing.T) {
	parent := &mockParent{}
	childStorage := storage.NewMapStorage()
	sc := NewBaseSmartContract(parent)

	assert.Equal(t, &BaseSmartContract{
		CompositeMap: make(map[string]object.Reference),
		ChildStorage: childStorage,
		Parent:       parent,
	}, sc)
}

func TestSmartContract_GetClassID(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)

	classID := sc.GetClassID()

	assert.Equal(t, class.SmartContractID, classID)
}

func TestSmartContract_CreateComposite(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)
	compositeFactory := BaseCompositeFactory{}

	composite, err := sc.CreateComposite(&compositeFactory)

	assert.Len(t, sc.CompositeMap, 1)
	assert.Len(t, sc.ChildStorage.GetKeys(), 1)
	compositeRecord := sc.ChildStorage.GetKeys()[0]
	compositeInChildStorage, _ := sc.ChildStorage.Get(compositeRecord)
	assert.Equal(t, compositeInChildStorage, composite)
	assert.NoError(t, err)
}

func TestSmartContract_CreateComposite_Error(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)
	compositeFactory := BaseCompositeFactory{}
	// Add to CompositeMap and ChildStorage prepared item
	sc.CreateComposite(&compositeFactory)

	res, err := sc.CreateComposite(&compositeFactory)

	assert.Nil(t, res)
	assert.EqualError(t, err, "delegate with name BaseComposite already exist")
	// CompositeMap and ChildStorage contains only one prepared item
	assert.Len(t, sc.CompositeMap, 1)
	assert.Len(t, sc.ChildStorage.GetKeys(), 1)
}

func TestSmartContract_CreateComposite_NotChild(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)
	compositeFactory := BaseCompositeNotChildFactory{}

	res, err := sc.CreateComposite(&compositeFactory)

	assert.Nil(t, res)
	assert.EqualError(t, err, "composite is not a Child")
	// CompositeMap and ChildStorage contains zero items
	assert.Len(t, sc.CompositeMap, 0)
	assert.Len(t, sc.ChildStorage.GetKeys(), 0)
}

func TestSmartContract_CreateComposite_CreateError(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)
	errorFactory := BaseCompositeFactoryWithError{}

	_, err := sc.CreateComposite(&errorFactory)
	assert.EqualError(t, err, "composite factory create error")
}

func TestSmartContract_GetComposite(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)
	compositeFactory := BaseCompositeFactory{}
	composite, err := sc.CreateComposite(&compositeFactory)

	assert.NoError(t, err)

	res, err := sc.GetComposite(compositeFactory.GetInterfaceKey(), compositeFactory.GetClassID())

	assert.NoError(t, err)
	assert.Equal(t, composite, res)

}

func TestSmartContract_GetComposite_Error(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)

	compositeFactory := BaseCompositeFactory{}

	res, err := sc.GetComposite(compositeFactory.GetInterfaceKey(), compositeFactory.GetClassID())

	assert.Nil(t, res)
	assert.EqualError(t, err, "delegate with name BaseComposite does not exist")
}

func TestSmartContract_GetOrCreateComposite_Get(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)
	composite := &BaseComposite{}
	compositeFactory := &BaseCompositeFactory{}

	res, err := sc.GetOrCreateComposite(compositeFactory)

	assert.NoError(t, err)
	assert.Equal(t, composite, res)
}

func TestSmartContract_GetOrCreateComposite_Create(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)
	composite := &BaseComposite{}
	compositeFactory := &BaseCompositeFactory{}

	assert.Len(t, sc.CompositeMap, 0)
	res, err := sc.GetOrCreateComposite(compositeFactory)

	assert.NoError(t, err)
	assert.Len(t, sc.CompositeMap, 1)
	assert.Equal(t, composite, res)

	res, err = sc.GetOrCreateComposite(compositeFactory)

	assert.NoError(t, err)
	assert.Len(t, sc.CompositeMap, 1)
	assert.Equal(t, composite, res)

}

func TestSmartContract_GetChildStorage(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)

	res := sc.GetChildStorage()

	assert.Equal(t, sc.ChildStorage, res)
}

func TestSmartContract_AddChild(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)
	child := &mockChildProxy{}

	res, err := sc.AddChild(child)

	assert.NoError(t, err)
	assert.Len(t, sc.ChildStorage.GetKeys(), 1)
	assert.Equal(t, sc.ChildStorage.GetKeys()[0], res)
}

func TestSmartContract_GetChild(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)
	child := &mockChildProxy{}
	key, _ := sc.AddChild(child)

	res, err := sc.GetChild(key)

	assert.NoError(t, err)
	assert.Equal(t, child, res)
}

func TestSmartContract_GetChild_Error(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)

	res, err := sc.GetChild("someKey")

	assert.Nil(t, res)
	assert.EqualError(t, err, "object with record someKey does not exist")
}

func TestSmartContract_GetContextStorage(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)

	res := sc.GetContextStorage()

	assert.Equal(t, sc.ContextStorage, res)
}

func TestSmartContract_GetContext(t *testing.T) {
	parent := &mockParent{}
	contextStorage := storage.NewMapStorage()
	sc := NewBaseSmartContract(parent)
	sc.ContextStorage = contextStorage

	res := sc.GetContext()

	assert.Equal(t, contextStorage.GetKeys(), res)
}

func TestSmartContract_GetParent(t *testing.T) {
	parent := &mockParent{}
	sc := NewBaseSmartContract(parent)

	res := sc.GetParent()

	assert.Equal(t, sc.Parent, res)
}

func TestSmartContract_GetResolver(t *testing.T) {
	parent := &mockParent{}
	sc := BaseSmartContract{
		CompositeMap: make(map[string]object.Reference),
		ChildStorage: storage.NewMapStorage(),
		Parent:       parent,
	}
	assert.Nil(t, sc.resolver)
	sc.GetResolver()

	assert.NotNil(t, sc.resolver)
}

func TestSmartContract_GetResolver_Twice(t *testing.T) {
	parent := &mockParent{}
	sc := BaseSmartContract{
		CompositeMap: make(map[string]object.Reference),
		ChildStorage: storage.NewMapStorage(),
		Parent:       parent,
	}
	sc.GetResolver()
	assert.NotNil(t, sc.resolver)

	sc.GetResolver()

	assert.NotNil(t, sc.resolver)
}
