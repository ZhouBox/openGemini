/*
Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.

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

package bufferpool_test

import (
	"runtime/debug"
	"testing"

	"github.com/openGemini/openGemini/lib/bufferpool"
)

func TestBufferPool(t *testing.T) {
	debug.SetGCPercent(-1)
	defer debug.SetGCPercent(100)

	b := []byte{1, 2, 3, 4}
	bufferpool.Put(b)

	b2 := bufferpool.Get()

	if cap(b) != cap(b2) {
		t.Fatalf("failed, exp: %+v; got: %+v", cap(b), cap(b2))
	}

	b3 := make([]byte, 32*1024*1024+1)
	bufferpool.Put(b3)

	b4 := bufferpool.Get()

	if cap(b3) != cap(b4) {
		t.Fatalf("failed, exp: %+v; got: %+v", cap(b3), cap(b4))
	}
}
