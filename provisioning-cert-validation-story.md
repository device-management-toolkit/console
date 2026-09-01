# [Feature]: Validate the provisioning certificate (PFX) against the Intel AMT SDK rules at domain profile upload

**Labels:** `enhancement`, `domains`, `parity`, `user-experience`
**Related:** #910 (CIRA/TLS PEM load-path hardening — separate scope, unchanged by this story)

## Describe the feature

When a domain profile is created or updated (`POST` / `PATCH /api/v1/admin/domains`), Console only checks two things about the uploaded provisioning PFX: that it decrypts with the supplied password, and that the leaf certificate has not expired (`domains.DecryptAndCheckCertExpiration`). Everything else the Intel AMT firmware will verify during remote configuration is left unchecked, so a bad certificate is accepted at upload time and only fails later — on the device, as an opaque activation error.

The Intel AMT SDK (*Setup and Configuration Using PKI → PKI Certificate Verification Methods*) lists exactly what the device verifies:

> 1. The certificate is an SSL Server Certificate.
> 2. The certificate contains a designated OID or designated OU.
> 3. The certificate chain of trust ends with a root CA that has its hash pre-installed in the Intel AMT device.
> 4. The certificate identifier (the CN field in the Certificate Subject or DNS entry in Subject Alternative Name) is validated: *DHCP option 15 (or DHCPv6 option 24) is a suffix of `<CN>`.*

with two documented exceptions to (4):

> - **Wildcard Certificate** – A certificate signed for `*.some.domain` will be able to provision … as long as it is a suffix of the SCA FQDN.
> - **.com/.net** – Configuration will be successful if the `<CN>` and DHCP/PKI DNS Suffix ends with .com or .net and has the same "second level" domain (i.e. `csa.ftl.intel.com` can configure platforms in `dev.intel.com` domain). Releases 4.1/5.1 and later include extended support for Top Level Domains.

This story implements every one of those checks (plus the basic certificate-health checks a PFX should satisfy) so a domain profile that Console accepts is one AMT will accept.

## What Console validates today vs. what is missing

| # | Check | AMT SDK | RPS (legacy) | Console today | Status |
|---|---|---|---|---|---|
| 1 | Leaf is an SSL Server certificate (EKU contains `serverAuth`) | required | not checked | not checked | **missing** |
| 2 | Leaf EKU contains Intel AMT OID `2.16.840.1.113741.1.2.3` **or** Subject OU is exactly `Intel(R) Client Setup Certificate` (case-sensitive) | required | not checked | not checked | **missing** |
| 3 | PFX contains a self-signed root CA and the leaf chains to it through any included intermediates | required | `rootCertChecker` (root present) | not checked (`pkcs12.Decode` discards the chain) | **missing** (regression vs RPS) |
| 4 | Domain suffix matches the certificate identifier (CN **or** SAN DNS) per the SDK suffix rule, wildcard rule and `.com/.net` / TLD-depth extension | required | `domainSuffixChecker` (last-two-labels approximation, CN only) | not checked | **missing** (regression vs RPS) |
| 5 | Leaf `notAfter` > now | required | `expirationChecker` | `DecryptAndCheckCertExpiration` | present |
| 6 | Leaf `notBefore` ≤ now | implied | not checked | not checked | **missing** |
| 7 | Private key present in the PFX (the SCA needs it for the TLS handshake with the device) | required | implicit | not checked (`pkcs12.Decode` returns it, never inspected) | **missing** |
| 8 | RSA key ≥ 2048 bits | best practice | not checked | not checked | **missing** |
| 9 | Password decrypts the PFX | — | `passwordChecker` | `DecryptAndCheckCertExpiration` | present |

## Steps to reproduce (today)

1. Open **Domains → Add New** and upload any of the following PFX files with the correct password:
   - a leaf certificate with no `serverAuth` EKU;
   - a leaf certificate without the Intel AMT OID and without the `Intel(R) Client Setup Certificate` OU;
   - a PFX that contains only the leaf (no intermediate, no root);
   - a certificate whose CN/SAN is `intel.vprodemo.com` with domain suffix `vprodomain.com`;
   - a certificate whose `notBefore` is in the future;
   - a PFX that contains the certificate but no private key;
   - a leaf with a 1024-bit RSA key.
2. Observe that every one of them is saved as a valid domain profile.
3. Activate a device in ACM with that profile: activation fails on the device with a generic PKI/auth error and nothing in Console points at the certificate.

## Expected behavior

`POST` and `PATCH /api/v1/admin/domains` reject the request with **HTTP 400** and a specific, human-readable message when the uploaded PFX fails any of the checks below. Nothing is written to the database or to Vault when validation fails. A domain profile that Console accepts is one the Intel AMT firmware will accept for remote configuration.

### Acceptance criteria

**Certificate content**

- [ ] **AC1 – Server certificate.** The leaf certificate's Extended Key Usage contains `serverAuth` (`1.3.6.1.5.5.7.3.1`). Otherwise reject: *"Provisioning certificate is not an SSL server certificate"*.
- [ ] **AC2 – AMT marker.** The leaf certificate either has `2.16.840.1.113741.1.2.3` in its EKU list **or** has an OU in the Subject equal to `Intel(R) Client Setup Certificate` (exact, case-sensitive). Otherwise reject: *"Provisioning certificate does not contain the Intel AMT OID or OU"*.
- [ ] **AC3 – Chain to a root.** The PFX is decoded with `pkcs12.DecodeChain`; the bundle must contain at least one self-signed certificate with `IsCA: true` / `BasicConstraintsValid: true` and `KeyUsageCertSign`, and the leaf must verify to that root using only certificates from the PFX (`x509.Certificate.Verify` with the PFX intermediates/roots as pools, `ExtKeyUsageAny`). Otherwise reject: *"Provisioning certificate does not contain a root certificate"* or *"Provisioning certificate chain does not verify to the included root"*. (Matching the root against the device's pre-installed hash list is out of scope — it is device-specific.)
- [ ] **AC4 – Validity window.** `notBefore ≤ now < notAfter` for the leaf. Reject with *"Uploaded certificate has expired"* (existing message, unchanged) or *"Provisioning certificate is not yet valid"*.
- [ ] **AC5 – Private key.** The PFX must contain the private key matching the leaf's public key. Otherwise reject: *"Provisioning certificate does not contain a private key"*.
- [ ] **AC6 – Key strength.** RSA keys must be ≥ 2048 bits (ECDSA keys, if present, ≥ P-256). Otherwise reject: *"Provisioning certificate key is too weak (minimum RSA 2048)"*.

**Domain suffix (SDK rule 4)**

- [ ] **AC7 – Identifiers.** The set of certificate identifiers is the Subject CN **plus every DNS entry in the Subject Alternative Name**. The suffix is accepted if it matches *any* identifier.
- [ ] **AC8 – Suffix rule.** `domainSuffix` is accepted if it is a label-boundary suffix of an identifier (equal to it, or preceded by a `.`), and the matched part has at least two labels (a bare TLD never matches). Comparison is case-insensitive, whitespace-trimmed, trailing root dot ignored.
- [ ] **AC9 – Wildcard rule.** For an identifier `*.base`, the wildcard label is stripped and the suffix is accepted if it is `base`, a label-boundary suffix of `base` (≥ 2 labels), or any name under `base`.
- [ ] **AC10 – .com/.net and TLD-depth extension.** If the suffix's TLD is in the SDK table, the suffix is also accepted when identifier and suffix share the same last *depth* labels:
  `com 2, net 2, org 2, gov 2, edu 2, arpa 3` and ccTLDs `de 2, ch 2, dk 2, cz 2, cl 2, ua 2`, `fr 3, cn 3, nl 3, br 3, mx 3, uk 3, pl 3, tw 3, ca 3, fi 3, be 3, ru 3, se 3, ar 3, es 3, no 3, at 3, in 3, tr 3, ro 3, hu 3, nz 3, pt 3, il 3, gr 3, co 3, ie 3, za 3, th 3, sg 3, hk 3, lt 3, id 3, hr 3, ee 3, bg 3`.
  Private / unlisted TLDs (`.local`, `.internal`, `.corp`, …) get **no** extension — only AC8/AC9 apply.
- [ ] **AC11 – Rejection message.** A suffix that matches nothing is rejected with *"FQDN not associated with provisioning certificate"* (the RPS message, so the Sample Web UI shows the same text as before).
- [ ] **AC12 – Update path.** `PATCH` runs the same validation whenever a new `provisioningCert` is supplied. A `PATCH` that changes only `domainSuffix` (no new cert) re-validates the new suffix against the **stored** certificate.

Worked examples for AC7–AC10 (identifier `intel.vprodemo.com`, standard cert):

| domainSuffix | Result | Rule |
|---|---|---|
| `vprodemo.com` | accept | AC8 (suffix of identifier) |
| `intel.vprodemo.com` | accept | AC8 (equal) |
| `amt.vprodemo.com` | accept | AC10 (`.com`, same second level) |
| `xyz.vprodemo.com` | accept | AC10 (`.com`, same second level) |
| `vprodomain.com` | reject | no rule matches |
| `com` | reject | bare TLD |

Identifier `server.east.company.local` (private TLD):

| domainSuffix | Result |
|---|---|
| `east.company.local` | accept (AC8) |
| `company.local` | accept (AC8 — SDK says "a suffix of `<CN>`") |
| `mkgt.east.company.local` | reject (AC10 does not apply to `.local`) |
| `west.company.local` | reject |

Identifier `*.east.company.local`:

| domainSuffix | Result |
|---|---|
| `east.company.local`, `mkgt.east.company.local`, `company.local` | accept (AC9) |
| `west.company.local` | reject |

> **Decision recorded:** with AC10, `amt.vprodemo.com` and `xyz.vprodemo.com` are both accepted for a certificate whose only identifier is `intel.vprodemo.com`. That is what the SDK specifies and what AMT ≥ 4.1 does; the certificate carries no information that could distinguish the two. Rejecting sibling sub-domains on public TLDs would refuse configurations that provision successfully.

**Non-functional**

- [ ] **AC13 – API compatibility.** No request/response shape changes. The only new behaviour is additional `400` reasons on `POST`/`PATCH /api/v1/admin/domains`. Existing stored domains are not re-validated retroactively.
- [ ] **AC14 – Error mapping.** Each rejection is a typed error in `internal/usecase/domains` (following the `CertExpirationError` / `CertPasswordError` pattern) mapped to `400` in `internal/controller/httpapi/v1/error.go`, with the friendly message in both `error` and `message` fields.
- [ ] **AC15 – Tests.** Table-driven unit tests for every AC (positive and negative), including the worked examples above, using generated test PFX fixtures (standard, wildcard, SAN-only, missing-root, missing-key, weak-key, no-EKU, no-OID/OU, not-yet-valid). Existing `FuzzDecryptAndCheckCertExpiration` extended or complemented for the new decode path.
- [ ] **AC16 – Docs.** Update the Postman collection descriptions for the domain requests with the new error responses, and add a short "provisioning certificate requirements" note to the domains section of the docs site (link to the SDK page).

## Implementation notes (non-binding)

- Replace `pkcs12.Decode` with `pkcs12.DecodeChain` in `internal/usecase/domains/usecase.go` so the CA certificates are available for AC3.
- Keep `DecryptAndCheckCertExpiration` as the entry point (it is exported and fuzzed) or introduce a single `ValidateProvisioningCert(pfx, password, domainSuffix) (*x509.Certificate, error)` that runs the checks in the order: decrypt → private key → validity → key strength → serverAuth → AMT OID/OU → chain → suffix. Fail fast on the first error.
- The suffix logic (AC7–AC10) belongs in its own file with the TLD-depth table as a package-level map; it is pure and easy to test in isolation.
- All checks are on the **domain PFX path only**; the Console TLS/CIRA certificate load paths remain the subject of #910.

## Additional context

- **Why it matters:** AMT reports a PKI failure during remote configuration with very little detail. Every one of these conditions is detectable from the PFX at upload time, turning a field-debugging session into a `400` with a clear message.
- **RPS parity:** RPS validated password, expiry, root presence and (approximately) the domain suffix. Console currently only does the first two — this story restores parity and then aligns the suffix rule with the SDK instead of RPS's last-two-labels shortcut, which was wrong for private TLDs and for ccTLDs registered at the third level (e.g. `intel.co.uk`).
- **References:**
  - Intel AMT SDK – *PKI Certificate Verification Methods* (`AMT_Implementation_and_Reference_Guide/WordDocuments/pkicertificateverificationmethods.htm`)
  - Intel AMT SDK – *Prerequisites for Remote Configuration* (OID / OU requirement)
  - Intel white paper – *Remote Configuration Certificate Selection* (standard vs wildcard vs multi-level domain behaviour)
  - RPS `src/routes/admin/domains/domain.ts` (`passwordChecker`, `domainSuffixChecker`, `expirationChecker`, `rootCertChecker`)
