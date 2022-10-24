/*
Copyright 2022 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package requirements

type MemoryValuesStore struct {
	values map[string]interface{}
}

func newMemoryValuesStore() *MemoryValuesStore {
	return &MemoryValuesStore{
		values: make(map[string]interface{}),
	}
}

func (m *MemoryValuesStore) Set(key string, value interface{}) {
	m.values[key] = value
}

func (m *MemoryValuesStore) Remove(key string) {
	delete(m.values, key)
}

func (m *MemoryValuesStore) Get(key string) (interface{}, bool) {
	v, ok := m.values[key]
	return v, ok
}
