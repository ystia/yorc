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

package sizeutil

import (
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v3/helper/mathutil"
)

// ConvertToGB allows to convert a MB size as "42" or a human readable size as "42MB" or "42 KB" into GB
func ConvertToGB(size string) (int, error) {
	// Default size unit is MB
	mSize, err := strconv.Atoi(size)
	// Not an int value, so maybe a human readable size: we try to retrieve bytes
	if err != nil {
		var bsize uint64
		bsize, err = humanize.ParseBytes(size)
		if err != nil {
			return 0, errors.Errorf("Can't convert size to bytes value: %v", err)
		}
		gSize := float64(bsize) / humanize.GByte
		gSize = mathutil.Round(gSize, 0, 0)
		return int(gSize), nil
	}

	gSize := float64(mSize) / 1000
	gSize = mathutil.Round(gSize, 0, 0)
	return int(gSize), nil
}
