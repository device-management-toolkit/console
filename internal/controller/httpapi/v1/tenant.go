package v1

import (
	"reflect"

	"github.com/gin-gonic/gin"
)

const tenantHeaderName = "x-tenant-id"

func tenantIDFromHeader(c *gin.Context) string {
	return c.GetHeader(tenantHeaderName)
}

func setTenantID(target any, tenantID string) {
	if tenantID == "" || target == nil {
		return
	}

	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return
	}

	field := value.Elem().FieldByName("TenantID")
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.String {
		return
	}

	field.SetString(tenantID)
}