// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package internal

import (
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/storage"
	"github.com/ystia/yorc/v4/storage/types"
	"github.com/ystia/yorc/v4/tosca"
	"net/url"
	"path"
)

// StoreComplexType stores a TOSCA value into a given path.
//
// This value can be a literal or a complex value composed by maps and lists
func StoreComplexType(consulStore consulutil.ConsulStore, valuePath string, value interface{}) {
	storage.GetStore(types.StoreTypeDeployment).Set(valuePath, value)
	//v := reflect.ValueOf(value)
	//switch v.Kind() {
	//case reflect.Slice:
	//	// Store an empty string all we want is to have a the value type here
	//	consulStore.StoreConsulKeyAsStringWithFlags(valuePath, "", uint64(tosca.ValueAssignmentList))
	//	for i := 0; i < v.Len(); i++ {
	//		StoreComplexType(consulStore, path.Join(valuePath, strconv.Itoa(i)), v.Index(i).Interface())
	//	}
	//case reflect.Map:
	//	// Store an empty string all we want is to have a the value type here
	//	consulStore.StoreConsulKeyAsStringWithFlags(valuePath, "", uint64(tosca.ValueAssignmentMap))
	//	for _, key := range v.MapKeys() {
	//		StoreComplexType(consulStore, path.Join(valuePath, url.QueryEscape(fmt.Sprint(key.Interface()))), v.MapIndex(key).Interface())
	//	}
	//default:
	//	// Default flag is for literal
	//	consulStore.StoreConsulKeyAsString(valuePath, fmt.Sprint(v))
	//}
}

// StoreValueAssignment stores a TOSCA value assignment under a given path
func StoreValueAssignment(consulStore consulutil.ConsulStore, vaPrefix string, va *tosca.ValueAssignment) {
	if va == nil {
		return
	}
	//switch va.Type {
	//case tosca.ValueAssignmentLiteral:
	//	// Default flag is for literal
	//	consulStore.StoreConsulKeyAsString(vaPrefix, va.GetLiteral())
	//case tosca.ValueAssignmentFunction:
	//	consulStore.StoreConsulKeyAsStringWithFlags(vaPrefix, va.GetFunction().String(), uint64(tosca.ValueAssignmentFunction))
	//case tosca.ValueAssignmentList:
	//	StoreComplexType(consulStore, vaPrefix, va.GetList())
	//case tosca.ValueAssignmentMap:
	//	StoreComplexType(consulStore, vaPrefix, va.GetMap())
	//}

	storage.GetStore(types.StoreTypeDeployment).Set(vaPrefix, va)
}

func storeMapValueAssignment(consulStore consulutil.ConsulStore, prefix string, mapValueAssignment map[string]*tosca.ValueAssignment) {
	for name, value := range mapValueAssignment {
		StoreValueAssignment(consulStore, path.Join(prefix, url.QueryEscape(name)), value)
	}
}
