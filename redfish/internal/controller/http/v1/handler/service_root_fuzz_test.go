package v1

import (
	"reflect"
	"testing"
)

func FuzzValidateMetadataXML(f *testing.F) {
	seedInputs := []string{
		"",
		`<?xml version="1.0" encoding="UTF-8"?><root></root>`,
		`<root><child attr="1">value</child></root>`,
		`<root>`,
		`not-xml`,
		`<root>用戶🙂</root>`,
		`<root><![CDATA[test]]></root>`,
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, xmlData string) {
		err1 := validateMetadataXML(xmlData)
		err2 := validateMetadataXML(xmlData)

		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("non-deterministic XML validation result for %q: first=%v second=%v", xmlData, err1, err2)
		}
	})
}

func FuzzExtractServicesFromOpenAPIData(f *testing.F) {
	seedInputs := [][]byte{
		[]byte("paths:\n  /redfish/v1/Systems:\n    get: {}\n"),
		[]byte("paths:\n  /redfish/v1/SessionService:\n    get: {}\n  /redfish/v1/Systems/{ComputerSystemId}:\n    get: {}\n"),
		[]byte("paths:\n  /redfish/v1/odata:\n    get: {}\n  /redfish/v1/$metadata:\n    get: {}\n"),
		[]byte("not: yaml: :"),
		[]byte("{}"),
		[]byte("paths:\n  /redfish/v1/用戶🙂:\n    get: {}\n"),
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		services1, err1 := ExtractServicesFromOpenAPIData(data)
		services2, err2 := ExtractServicesFromOpenAPIData(data)

		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("non-deterministic OpenAPI parse error state: first=%v second=%v", err1, err2)
		}

		if !reflect.DeepEqual(services1, services2) {
			t.Fatalf("non-deterministic extracted services for input %q", string(data))
		}

		if len(services1) == 0 {
			t.Fatal("expected fallback/default services, got none")
		}
	})
}
