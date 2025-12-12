/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package domains

import "github.com/device-management-toolkit/console/pkg/consoleerrors"

type CertStoreError struct {
	Console consoleerrors.InternalError
}

func (e CertStoreError) Error() string {
	return e.Console.Error()
}

func (e CertStoreError) Wrap(call, function string, err error) error {
	_ = e.Console.Wrap(call, function, err)
	e.Console.Message = "certificate store operation failed"

	return e
}
