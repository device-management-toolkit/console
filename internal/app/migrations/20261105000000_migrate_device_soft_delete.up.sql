/*********************************************************************
* Copyright (c) Intel Corporation 2026
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

-- Soft delete (issues #843, #1382). DELETE /api/v1/devices/:guid flags the
-- row instead of removing it; every read excludes flagged rows, and adding a
-- device with the same GUID restores the record. deleteddate is RFC3339Nano
-- UTC, written only by the soft-delete operation.
ALTER TABLE devices ADD COLUMN isdeleted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE devices ADD COLUMN deleteddate TEXT NOT NULL DEFAULT '';
