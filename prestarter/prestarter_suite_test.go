package prestarter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPrestarter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Prestarter Suite")
}
