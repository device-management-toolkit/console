# Standalone administrator password management

## First-run keyring setup

With `AUTH_SECRET_BACKEND=keyring`, Console starts without prompting for
credentials. When both administrator entries are absent, it creates a
single-use setup token valid for 10 minutes and starts a setup API bound only
to `127.0.0.1`. The embedded UI opens at `/setup` with the token in the URL
fragment. The page reads the fragment into memory, immediately clears it with
`history.replaceState`, and submits the token and credentials in the HTTPS
request body. The setup token is not exchanged for a cookie, and setup does
not create a login session. After setup, sign in with the new credentials.

The loopback-only setup API provides:

* `GET /api/v1/auth/setup/status` to report whether setup is required.
* `POST /api/v1/auth/setup` to submit `token`, `username`, `password`, and
  `confirmPassword` as JSON.

Setup returns `409 SETUP_ALREADY_COMPLETED` when credentials already exist and
`403 SETUP_LOCAL_ONLY` for requests that do not originate from loopback. A
partially populated keyring remains fail-closed and requires the keyring
recovery procedure below.

For a headless installation, Console prints the short-lived setup token to the
CLI at startup. It is also written to a permissions-restricted
`<config-path>.auth-state.db.setup-token` file as a fallback. Send the JSON
request body through standard input or a protected file; do not put the token
or credentials in command-line arguments. The API does not log request bodies.
The token expires after 10 minutes and its file is removed after successful
setup.

When standalone authentication is enabled, an authenticated administrator can
open `/account/password` in the Console browser to change their password. The
form requires the current password and a matching new-password confirmation.
Passwords must be at least eight characters long and contain at least four
distinct characters.

The form submits to `POST /api/v1/authorize/password`. The endpoint requires an
authenticated session and independently verifies the current password. In
`keyring` mode, the new password is stored as a PBKDF2-HMAC-SHA256 hash in the
`device-management-toolkit` keyring service. In `config` mode, it is saved to the configured
Console config file.

Keyring hashes use
`$pbkdf2-sha256$<iterations>$<base64(salt)>$<base64(hash)>`.
The earlier development-only `$pbkdf2$` prefix is not accepted; installations
with those entries must use the keyring recovery procedure below.

If `AUTH_ADMIN_PASSWORD` is set, the endpoint rejects changes with HTTP 409.
Update the password in the deployment environment and restart Console instead;
an API change to the config file would otherwise be overridden on restart.

After a successful password change, Console increments the durable standalone
authentication version in the permissions-restricted auth-state database next
to the configured config file. Tokens carry that version and a keyed tag of the
current stored credential. Every protected request checks both against current
state, so earlier standalone login session tokens (including the changing session) are
rejected across clients and process restarts. AMT redirection tokens are separate short-lived device tokens and are not credential-bound. Login issuance and password
updates share a cross-process state lock; if the revocation state cannot be
persisted, the credential update is rolled back where possible and the change
is not reported as successful. The JWT signing-key storage and lifecycle are
unchanged.

## Forgotten keyring password

Console does not provide an unauthenticated password-reset link or a reset
command. To recover access:

1. Stop Console.
2. Using the operating system's keyring/credential manager, locate the
   `device-management-toolkit` service and delete **both** `admin-username` and
   `admin-password-hash` entries. Do not delete only one entry.
3. Restart Console with `AUTH_SECRET_BACKEND=keyring`.
4. When both entries are absent, complete the local setup page to create a
   new administrator username and password.

The exact keyring manager and interface vary by operating system. On Linux,
use the Secret Service-compatible keyring manager; on macOS, Keychain Access;
on Windows, Credential Manager. Ensure the entries belong to the Console
`device-management-toolkit` service before removing them.
