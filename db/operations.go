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
	"errors"
	"fmt"

	"github.com/go-redis/redis"
)

var (
	// ctx = context.Background()

	// ErrorNotFound is for identifing not found error
	// Using this will enabling errors.Is() function
	ErrorNotFound = errors.New("Not Found in DB")
)

// Create will create a new entry in DB for the value with the given table and key
func (c *Client) Create(table, key string, data interface{}) (err error) {
	err = c.pool.SetNX(generateKey(table, key), data, 0).Err()
	if err != nil {
		return fmt.Errorf(
			"Creating new entry for value %v in table %s with key %s failed: %v",
			data, table, key, err,
		)
	}
	return
}

// GetAllKeys will collect all the keys of provided table
func (c *Client) GetAllKeys(table string) (allKeys *[]string, err error) {
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = c.pool.Scan(cursor, generateKey(table, "*"), 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch all keys of table %s: %s", table, err.Error())
		}
		*allKeys = append(*allKeys, keys...)
		if cursor == 0 {
			break
		}
	}
	return
}

// Get will collect the data associated with the given key from the given table
func (c *Client) Get(table, key string) (val string, err error) {
	val, err = c.pool.Get(generateKey(table, key)).Result()
	switch err {
	case redis.Nil:
		return "", fmt.Errorf(
			"%w: %s",
			ErrorNotFound,
			fmt.Sprintf("Data with key %s not found in table %s", key, table),
		)
	case nil:
		return val, nil
	default:
		return "", fmt.Errorf("unable to complete the operation: %s", err.Error())
	}
}

func generateKey(table, key string) string {
	return fmt.Sprintf("%s:%s", table, key)
}
