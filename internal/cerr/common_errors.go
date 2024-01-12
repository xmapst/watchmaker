// Copyright 2022 Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package cerr

import (
	"reflect"

	"github.com/pkg/errors"
)

type ErrHelper struct {
	inner error
}

func FromErr(err error) *ErrHelper {
	return &ErrHelper{inner: err}
}

func (h *ErrHelper) Err() error {
	return h.inner
}

func NotType[expected any]() *ErrHelper {
	var exp expected
	return &ErrHelper{inner: errors.Errorf("expected type: %T", exp)}
}

func NotImpl[expected any]() *ErrHelper {
	var exp *expected
	return &ErrHelper{inner: errors.Errorf("not implement %v", reflect.TypeOf(exp).Elem())}
}

func NotFoundType[in any]() *ErrHelper {
	var i in
	return &ErrHelper{inner: errors.Errorf("not found type: %T", i)}
}

func NotInit[in any]() *ErrHelper {
	var i in
	return &ErrHelper{inner: errors.Errorf("not init %T", i)}
}

func NotFound(name string) *ErrHelper {
	return &ErrHelper{errors.Errorf("%s not found", name)}
}

func (h *ErrHelper) WrapInput(in any) *ErrHelper {
	return &ErrHelper{inner: errors.Wrapf(h.inner, "input type: %T, input value: %v", in, in)}
}

func (h *ErrHelper) WrapValue(in any) *ErrHelper {
	return &ErrHelper{inner: errors.Wrapf(h.inner, "input value: %v", in)}
}

func (h *ErrHelper) WrapName(name string) *ErrHelper {
	return &ErrHelper{inner: errors.Wrapf(h.inner, "%s", name)}
}

func (h *ErrHelper) WrapErr(err error) *ErrHelper {
	return &ErrHelper{inner: errors.Wrapf(h.inner, "err : %s", err)}
}

func (h ErrHelper) Wrapf(format string, args ...interface{}) *ErrHelper {
	return &ErrHelper{inner: errors.Wrapf(h.inner, format, args...)}
}

func (h *ErrHelper) WithStack() *ErrHelper {
	return &ErrHelper{inner: errors.WithStack(h.inner)}
}

var (
	ErrDuplicateEntity = errors.New("duplicate entity")
)
