/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package tequilapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexpCorsPolicy_AllowedOrigin_ReturnsSameOrDefaultOrigin(t *testing.T) {
	var policy = RegexpCorsPolicy{
		DefaultTrustedOrigin:  "default",
		AllowedOriginSuffixes: []string{"^full-match$", "end-match$"},
	}

	tests := []struct {
		requested       string
		expectedAllowed string
	}{
		{
			requested:       "full-match",
			expectedAllowed: "full-match",
		},
		{
			requested:       "a-full-match",
			expectedAllowed: "default",
		},
		{
			requested:       "end-match",
			expectedAllowed: "end-match",
		},
		{
			requested:       "the-end-match",
			expectedAllowed: "the-end-match",
		},
	}

	for _, tt := range tests {
		allowed := policy.AllowedOrigin(tt.requested)
		assert.Equal(t, tt.expectedAllowed, allowed)
	}
}
