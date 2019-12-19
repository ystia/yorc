// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consul

import (
	"context"
	"github.com/stretchr/testify/require"
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/consul/testutil"

	"github.com/ystia/yorc/v4/storage/encoding"
)

// Foo is just some struct for common tests.
type Foo struct {
	Bar        string
	privateBar string
}

type privateFoo struct {
	Bar        string
	privateBar string
}

func handleGetError(t *testing.T, err error, found bool) {
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Error("No value was found, but should have been")
	}
}

// TestStore tests if reading from, writing to and deleting from the store works properly.
// A struct is used as value. See TestTypes() for a test that is simpler but tests all types.
func testStore(t *testing.T, srv1 *testutil.TestServer) {
	store := &consulStore{encoding.JSON}
	key := strconv.FormatInt(rand.Int63(), 10)
	ctx := context.Background()
	// Initially the key shouldn't exist
	found, err := store.Get(key, new(Foo))
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Error("A value was found, but no value was expected")
	}

	// Deleting a non-existing key-value pair should NOT lead to an error
	err = store.Delete(ctx, key, false)
	if err != nil {
		t.Error(err)
	}

	// Store an object
	val := Foo{
		Bar: "baz",
	}
	err = store.Set(ctx, key, val)
	if err != nil {
		t.Error(err)
	}

	// Storing it again should not lead to an error but just overwrite it
	err = store.Set(ctx, key, val)
	if err != nil {
		t.Error(err)
	}

	// Retrieve the object
	expected := val
	actualPtr := new(Foo)
	found, err = store.Get(key, actualPtr)
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Error("No value was found, but should have been")
	}
	actual := *actualPtr
	if actual != expected {
		t.Errorf("Expected: %v, but was: %v", expected, actual)
	}

	// Delete
	err = store.Delete(ctx, key, false)
	if err != nil {
		t.Error(err)
	}
	// Key-value pair shouldn't exist anymore
	found, err = store.Get(key, new(Foo))
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Error("A value was found, but no value was expected")
	}

	// Tree handling
	keypath1 := "one"
	keypath2 := "one/two"
	keypath3 := "one/two/one"
	keypath4 := "one/two/two"
	keypath5 := "one/two/three"
	err = store.Set(ctx, keypath1, val)
	require.NoError(t, err)

	err = store.Set(ctx, keypath2, val)
	require.NoError(t, err)

	err = store.Set(ctx, keypath3, val)
	require.NoError(t, err)

	err = store.Set(ctx, keypath4, val)
	require.NoError(t, err)

	err = store.Set(ctx, keypath5, val)
	require.NoError(t, err)

	// Check sub-keys
	keys, err := store.Keys(keypath1)
	require.NoError(t, err)
	require.Equal(t, 1, len(keys))
	require.Contains(t, keys, keypath2)

	keys, err = store.Keys(keypath2)
	require.NoError(t, err)
	require.Equal(t, 3, len(keys))
	require.Contains(t, keys, keypath3)
	require.Contains(t, keys, keypath4)
	require.Contains(t, keys, keypath5)

	// Delete recursively tree
	store.Delete(ctx, keypath1, true)
	keys, err = store.Keys(keypath1)
	require.NoError(t, err)
	require.Nil(t, keys)

	keys, err = store.Keys(keypath2)
	require.NoError(t, err)
	require.Nil(t, keys)
}

func testTypes(t *testing.T, srv1 *testutil.TestServer) {
	store := &consulStore{encoding.JSON}
	ctx := context.Background()
	boolVar := true
	// Omit byte
	// Omit error - it's a Go builtin type but marshalling and then unmarshalling doesn't lead to equal objects
	floatVar := 1.2
	intVar := 1
	runeVar := '⚡'
	stringVar := "foo"

	structVar := Foo{
		Bar: "baz",
	}
	structWithPrivateFieldVar := Foo{
		Bar:        "baz",
		privateBar: "privBaz",
	}
	// The differing expected var for structWithPrivateFieldVar
	structWithPrivateFieldExpectedVar := Foo{
		Bar: "baz",
	}
	privateStructVar := privateFoo{
		Bar: "baz",
	}
	privateStructWithPrivateFieldVar := privateFoo{
		Bar:        "baz",
		privateBar: "privBaz",
	}
	// The differing expected var for privateStructWithPrivateFieldVar
	privateStructWithPrivateFieldExpectedVar := privateFoo{
		Bar: "baz",
	}

	sliceOfBool := []bool{true, false}
	sliceOfByte := []byte("foo")
	// Omit slice of float
	sliceOfInt := []int{1, 2}
	// Omit slice of rune
	sliceOfString := []string{"foo", "bar"}

	sliceOfSliceOfString := [][]string{{"foo", "bar"}}

	sliceOfStruct := []Foo{{Bar: "baz"}}
	sliceOfPrivateStruct := []privateFoo{{Bar: "baz"}}

	testVals := []struct {
		subTestName string
		val         interface{}
		expected    interface{}
		testGet     func(*testing.T, *consulStore, string, interface{})
	}{
		{"bool", boolVar, boolVar, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new(bool)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if actual != expected {
				t.Errorf("Expected: %v, but was: %v", expected, actual)
			}
		}},
		{"float", floatVar, floatVar, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new(float64)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if actual != expected {
				t.Errorf("Expected: %v, but was: %v", expected, actual)
			}
		}},
		{"int", intVar, intVar, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new(int)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if actual != expected {
				t.Errorf("Expected: %v, but was: %v", expected, actual)
			}
		}},
		{"rune", runeVar, runeVar, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new(rune)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if actual != expected {
				t.Errorf("Expected: %v, but was: %v", expected, actual)
			}
		}},
		{"string", stringVar, stringVar, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new(string)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if actual != expected {
				t.Errorf("Expected: %v, but was: %v", expected, actual)
			}
		}},
		{"struct", structVar, structVar, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new(Foo)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if actual != expected {
				t.Errorf("Expected: %v, but was: %v", expected, actual)
			}
		}},
		{"struct with private field", structWithPrivateFieldVar, structWithPrivateFieldExpectedVar, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new(Foo)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if actual != expected {
				t.Errorf("Expected: %v, but was: %v", expected, actual)
			}
		}},
		{"private struct", privateStructVar, privateStructVar, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new(privateFoo)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if actual != expected {
				t.Errorf("Expected: %v, but was: %v", expected, actual)
			}
		}},
		{"private struct with private field", privateStructWithPrivateFieldVar, privateStructWithPrivateFieldExpectedVar, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new(privateFoo)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if actual != expected {
				t.Errorf("Expected: %v, but was: %v", expected, actual)
			}
		}},
		{"slice of bool", sliceOfBool, sliceOfBool, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new([]bool)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("testTypes() = %v, want %v", actual, expected)
			}
		}},
		{"slice of byte", sliceOfByte, sliceOfByte, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new([]byte)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("testTypes() = %v, want %v", actual, expected)
			}
		}},
		{"slice of int", sliceOfInt, sliceOfInt, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new([]int)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("testTypes() = %v, want %v", actual, expected)
			}
		}},
		{"slice of string", sliceOfString, sliceOfString, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new([]string)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("testTypes() = %v, want %v", actual, expected)
			}
		}},
		{"slice of slice of string", sliceOfSliceOfString, sliceOfSliceOfString, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new([][]string)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("testTypes() = %v, want %v", actual, expected)
			}
		}},
		{"slice of struct", sliceOfStruct, sliceOfStruct, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new([]Foo)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("testTypes() = %v, want %v", actual, expected)
			}
		}},
		{"slice of private struct", sliceOfPrivateStruct, sliceOfPrivateStruct, func(t *testing.T, store *consulStore, key string, expected interface{}) {
			actualPtr := new([]privateFoo)
			found, err := store.Get(key, actualPtr)
			handleGetError(t, err, found)
			actual := *actualPtr
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("testTypes() = %v, want %v", actual, expected)
			}
		}},
	}

	for _, testVal := range testVals {
		t.Run(testVal.subTestName, func(t2 *testing.T) {
			key := strconv.FormatInt(rand.Int63(), 10)
			err := store.Set(ctx, key, testVal.val)
			if err != nil {
				t.Error(err)
			}
			testVal.testGet(t, store, key, testVal.expected)
		})
	}
}
