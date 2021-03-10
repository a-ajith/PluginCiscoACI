//(C) Copyright [2020] Hewlett Packard Enterprise Development LP
//
//Licensed under the Apache License, Version 2.0 (the "License"); you may
//not use this file except in compliance with the License. You may obtain
//a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//License for the specific language governing permissions and limitations
// under the License.

package db

import (
	"context"
	"fmt"
)

var ctx = context.Background()

// Create will create a new entry in DB for the value with the given table and key
func (c *Client) Create(table, key string, data interface{}) (err error) {
	err = c.pool.SetNX(ctx, generateKey(table, key), data, 0).Err()
	if err != nil {
		return fmt.Errorf(
			"Creating new entry for value %v in table %s with key %s failed: %v",
			data, table, key, err,
		)
	}
	return
}

func generateKey(table, key string) string {
	return fmt.Sprintf("%s:%s", table, key)
}
