// Copyright 2014 ALTOROS
// Licensed under the AGPLv3, see LICENSE file for details.

package mock

import "testing"

func TestGenerateID(t *testing.T) {
	for i := 0; i < 10; i++ {
		if v := genID(); v != i {
			t.Errorf("ID at %d should be equal to %d", i, v)
		}
	}
}
