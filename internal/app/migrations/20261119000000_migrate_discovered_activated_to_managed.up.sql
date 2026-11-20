/*********************************************************************
* Copyright (c) Intel Corporation 2026
* SPDX-License-Identifier: Apache-2.0
**********************************************************************/

-- A device found by discovery now stays Discovered whatever its activation state.
-- Before, a discovered=true row already in ACM/CCM was listed as Managed. Keep those
-- rows Managed across the upgrade. Both copies of the flag are updated: the column
-- is what the lists filter on, and the deviceinfo JSON is what the next update reads
-- back and would otherwise write over the column again.
UPDATE devices
SET discovered = FALSE,
    deviceinfo = REPLACE(deviceinfo, '"discovered":true', '"discovered":false')
WHERE discovered = TRUE
  AND currentmode IS NOT NULL
  AND currentmode <> ''
  AND LOWER(currentmode) <> 'not activated';
