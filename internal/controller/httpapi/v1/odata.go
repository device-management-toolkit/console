/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package v1

type OData struct {
	Top   int  `form:"$top,default=25"`
	Skip  int  `form:"$skip"`
	Count bool `form:"$count"`
}
