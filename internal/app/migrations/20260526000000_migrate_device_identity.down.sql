/*********************************************************************
* Copyright (c) Intel Corporation 2026
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

DROP INDEX IF EXISTS idx_devices_id;
ALTER TABLE devices DROP COLUMN connectiontype;
ALTER TABLE devices DROP COLUMN producttype;
ALTER TABLE devices DROP COLUMN deleteddate;
ALTER TABLE devices DROP COLUMN isdeleted;
ALTER TABLE devices DROP COLUMN createddate;
ALTER TABLE devices DROP COLUMN id;
