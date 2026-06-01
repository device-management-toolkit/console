/*********************************************************************
* Copyright (c) Intel Corporation 2026
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

-- Device identity & lifecycle columns (issue #843).
-- All TEXT columns are NOT NULL DEFAULT '' so the ALTER backfills existing
-- rows with a non-NULL value (the modernc sqlite driver cannot scan NULL into
-- a Go string). `id` is an app-generated surrogate key; the partial unique
-- index excludes the backfilled empty values on pre-existing rows.
-- createddate: server-set insert timestamp. isdeleted/deleteddate: logical-
-- deletion flag + server-set timestamp (column + plumbing only; soft-delete
-- behavior lands in a separate PR). producttype: manageability SKU (vPro/ISM).
-- connectiontype: CIRA/Direct.
ALTER TABLE devices ADD COLUMN id TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN createddate TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN isdeleted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE devices ADD COLUMN deleteddate TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN producttype TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN connectiontype TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_id ON devices (id) WHERE id <> '';
