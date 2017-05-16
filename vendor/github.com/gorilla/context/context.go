// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package context

import (
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	mutex sync.RWMutex
	data  = make(map[*url.URL]map[interface{}]interface{})
	datat = make(map[*url.URL]int64)
)

// Set stores a value for a given key in a given request.
func Set(r *http.Request, key, val interface{}) {
	mutex.Lock()
	if data[r.URL] == nil {
		data[r.URL] = make(map[interface{}]interface{})
		datat[r.URL] = time.Now().Unix()
	}
	data[r.URL][key] = val
	mutex.Unlock()
}

// Get returns a value stored for a given key in a given request.
func Get(r *http.Request, key interface{}) interface{} {
	mutex.RLock()
	if ctx := data[r.URL]; ctx != nil {
		value := ctx[key]
		mutex.RUnlock()
		return value
	}
	mutex.RUnlock()
	return nil
}

// GetOk returns stored value and presence state like multi-value return of map access.
func GetOk(r *http.Request, key interface{}) (interface{}, bool) {
	mutex.RLock()
	if _, ok := data[r.URL]; ok {
		value, ok := data[r.URL][key]
		mutex.RUnlock()
		return value, ok
	}
	mutex.RUnlock()
	return nil, false
}

// GetAll returns all stored values for the request as a map. Nil is returned for invalid requests.
func GetAll(r *http.Request) map[interface{}]interface{} {
	mutex.RLock()
	if context, ok := data[r.URL]; ok {
		result := make(map[interface{}]interface{}, len(context))
		for k, v := range context {
			result[k] = v
		}
		mutex.RUnlock()
		return result
	}
	mutex.RUnlock()
	return nil
}

// GetAllOk returns all stored values for the request as a map and a boolean value that indicates if
// the request was registered.
func GetAllOk(r *http.Request) (map[interface{}]interface{}, bool) {
	mutex.RLock()
	context, ok := data[r.URL]
	result := make(map[interface{}]interface{}, len(context))
	for k, v := range context {
		result[k] = v
	}
	mutex.RUnlock()
	return result, ok
}

// Delete removes a value stored for a given key in a given request.
func Delete(r *http.Request, key interface{}) {
	mutex.Lock()
	if data[r.URL] != nil {
		delete(data[r.URL], key)
	}
	mutex.Unlock()
}

// Clear removes all values stored for a given request.
//
// This is usually called by a handler wrapper to clean up request
// variables at the end of a request lifetime. See ClearHandler().
func Clear(r *http.Request) {
	mutex.Lock()
	clear(r.URL)
	mutex.Unlock()
}

// clear is Clear without the lock.
func clear(u *url.URL) {
	delete(data, u)
	delete(datat, u)
}

// Purge removes request data stored for longer than maxAge, in seconds.
// It returns the amount of requests removed.
//
// If maxAge <= 0, all request data is removed.
//
// This is only used for sanity check: in case context cleaning was not
// properly set some request data can be kept forever, consuming an increasing
// amount of memory. In case this is detected, Purge() must be called
// periodically until the problem is fixed.
func Purge(maxAge int) int {
	mutex.Lock()
	count := 0
	if maxAge <= 0 {
		count = len(data)
		data = make(map[*url.URL]map[interface{}]interface{})
		datat = make(map[*url.URL]int64)
	} else {
		min := time.Now().Unix() - int64(maxAge)
		for u := range data {
			if datat[u] < min {
				clear(u)
				count++
			}
		}
	}
	mutex.Unlock()
	return count
}

// ClearHandler wraps an http.Handler and clears request values at the end
// of a request lifetime.
func ClearHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer Clear(r)
		h.ServeHTTP(w, r)
	})
}
