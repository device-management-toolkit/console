/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package integration_test

// import (
// 	. "github.com/Eun/go-hit"
// )

// const (
// 	// Attempts connection
// 	host       = "localhost:8181"
// 	healthPath = "http://" + host + "/healthz"
// 	attempts   = 20

// 	// HTTP REST
// 	baseRPSPath = "http://" + host + "/api/v1/admin"
// )

// func TestMain(m *testing.M) {
// 	err := healthCheck(attempts)
// 	if err != nil {
// 		log.Fatalf("Integration tests: host %s is not available: %s", host, err)
// 	}

// 	log.Printf("Integration tests: host %s is available", host)

// 	code := m.Run()
// 	os.Exit(code)
// }

// func healthCheck(attempts int) error {
// 	var err error

// 	for attempts > 0 {
// 		err = Do(Get(healthPath), Expect().Status().Equal(http.StatusOK))
// 		if err == nil {
// 			return nil
// 		}

// 		log.Printf("Integration tests: url %s is not available, attempts left: %d", healthPath, attempts)

// 		time.Sleep(time.Second)

// 		attempts--
// 	}

// 	return err
// }

// // HTTP GET: /domains.
// func TestHTTPDomain(t *testing.T) {
// 	Test(t,
// 		Description("Domains Success"),
// 		Get(baseRPSPath+"/domains/"),
// 		Expect().Status().Equal(http.StatusOK),
// 		Expect().Body().String().Contains(`[]`),
// 	)
// 	Test(t,
// 		Description("Domains - Does not exist"),
// 		Get(baseRPSPath+"/domains/dne"),
// 		Expect().Status().Equal(http.StatusNotFound),
// 		Expect().Body().String().Contains(`{"error":""}`),
// 	)
// 	body := `{
//     "profileName": "NewDomain",
//     "domainSuffix": "test.com",
//     "provisioningCert": "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",
//     "provisioningCertStorageFormat": "raw",
//     "provisioningCertPassword": "Intel123!",
//     "expirationDate": "12/25/2023"
// }`
// 	Test(t,
// 		Description("Domains  - Create Domain"),
// 		Post(baseRPSPath+"/domains"),
// 		Send().Body().String(body),
// 		Expect().Status().Equal(http.StatusCreated),
// 		Expect().Body().String().Contains(`{"profileName":"NewDomain","domainSuffix":"test.com","provisioningCertStorageFormat":"raw","tenantId":"}`),
// 	)
// }
