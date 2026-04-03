// Copyright 2026 The A2A Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package a2apb

import _ "unsafe" // for go:linkname

// Both this package (a2apb/v1) and the legacy github.com/a2aproject/a2a-go/a2apb
// register the same "a2a.proto" file descriptor. The protobuf runtime panics on
// duplicate registrations by default.
//
// Package-level vars initialize before init() functions within a package, and
// init() functions run in lexical file name order (proto_compat.go > a2a.pb.go).
// This lets us:
//  1. Set the policy to "warn" via var init (before any init runs).
//  2. Let a2a.pb.go init() register a2a.proto under the relaxed policy.
//  3. Restore "panic" in this file's init() so other conflicts still surface.
//
//go:linkname _protoConflictPolicy google.golang.org/protobuf/reflect/protoregistry.conflictPolicy
var _protoConflictPolicy string

// Var initializers run before any init() in the package, setting "warn"
// before a2a.pb.go's init() calls RegisterFile.
var origPolicy = func() string {
	orig := _protoConflictPolicy
	_protoConflictPolicy = "warn"
	return orig
}()

func init() {
	_protoConflictPolicy = origPolicy
}
