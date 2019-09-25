/*
 * Copyright 2019 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package z

// KeyToHash interprets the type of key and converts it to a uint64 hash.
func KeyToHash(key interface{}) uint64 {
	switch k := key.(type) {
	case uint64:
		return k
	case string:
		return MemHashString(k)
	case []byte:
		return MemHash(k)
	case byte:
		return MemHash([]byte{k})
	case int:
		return uint64(k)
	case int32:
		return uint64(k)
	case uint32:
		return uint64(k)
	case int64:
		return uint64(k)
	default:
		panic("Key type not supported")
	}
}
