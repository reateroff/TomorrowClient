//go:build with_acme

package include

import (
	"github.com/tumgovic/todaycore/adapter/certificate"
	"github.com/tumgovic/todaycore/service/acme"
)

func registerACMECertificateProvider(registry *certificate.Registry) {
	acme.RegisterCertificateProvider(registry)
}
