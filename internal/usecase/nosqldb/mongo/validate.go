package mongo

import "regexp"

// Inline barrier for user-controlled identifiers reaching bson.M filters.
// MUST be called inline at each guard site — wrapping it breaks static-analysis
// barrier recognition. CodeQL's go/sql-injection flags these bson.M sinks as
// false positives; dismiss in GHAS, don't silence the query.
var identifierRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// domainSuffixRegex permits the dot for DNS suffixes; same inline rule.
var domainSuffixRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
