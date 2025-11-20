-- Test data for Redfish API tests
-- Insert a test device into the devices table

INSERT OR REPLACE INTO devices (
    guid, 
    hostname, 
    tags, 
    mpsinstance, 
    connectionstatus, 
    mpsusername, 
    tenantid, 
    friendlyname, 
    dnssuffix, 
    deviceinfo,
    username,
    password,
    usetls,
    allowselfsigned
) VALUES (
    'test-system-1',
    'test-device-1.local',
    '[]',
    '',
    1,
    'admin',
    '',
    'Test System 1',
    'local',
    '{"BuildNumber":"3000","FWBuild":"1234","FWVersion":"16.0.0","FQDN":"test-device-1.local","IPConfiguration":{"IPAddress":"192.168.1.100","MACAddress":"00:11:22:33:44:55"},"OS":"Windows 10","SKU":"vPro","UUID":"12345678-1234-1234-1234-123456789012"}',
    'admin',
    '',
    0,
    1
);
