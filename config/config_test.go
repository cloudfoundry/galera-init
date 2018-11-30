package config_test

import (
	"errors"
	"flag"
	"fmt"
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/service-config"

	"github.com/cloudfoundry/galera-init/config"
)

var _ = Describe("Config", func() {

	Describe("Validate", func() {
		var rootConfig config.Config
		var serviceConfig *service_config.ServiceConfig

		BeforeEach(func() {
			serviceConfig = service_config.New()
			flags := flag.NewFlagSet("galera-init", flag.ExitOnError)
			serviceConfig.AddFlags(flags)

			serviceConfig.AddDefaults(config.Config{
				Db: config.DBHelper{
					User: "root",
				},
			})

			flags.Parse([]string{
				"-configPath=../example-config.yml",
			})

			err := serviceConfig.Read(&rootConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		var setNestedFieldToEmpty func(obj interface{}, nestedFieldNames []string) error
		setNestedFieldToEmpty = func(obj interface{}, nestedFieldNames []string) error {

			s := reflect.ValueOf(obj).Elem()
			if s.Type().Kind() == reflect.Slice {
				if s.Len() == 0 {
					return errors.New("Trying to set nested property on empty slice")
				}
				s = s.Index(0)
			}

			currFieldName := nestedFieldNames[0]
			remainingFieldNames := nestedFieldNames[1:]
			field := s.FieldByName(currFieldName)
			if field.IsValid() == false {
				return errors.New(fmt.Sprintf("Field '%s' is not defined", currFieldName))
			}

			if len(remainingFieldNames) == 0 {
				fieldType := field.Type()
				field.Set(reflect.Zero(fieldType))
				return nil
			}
			return setNestedFieldToEmpty(field.Addr().Interface(), remainingFieldNames)
		}

		var setFieldToEmpty = func(fieldName string) error {
			return setNestedFieldToEmpty(&rootConfig, strings.Split(fieldName, "."))
		}

		var isRequiredField = func(fieldName string) func() {
			return func() {
				err := setFieldToEmpty(fieldName)
				Expect(err).NotTo(HaveOccurred())

				err = rootConfig.Validate()

				Expect(err).To(HaveOccurred())

				fieldParts := strings.Split(fieldName, ".")
				for _, fieldPart := range fieldParts {
					Expect(err.Error()).To(ContainSubstring(fieldPart))
				}
			}
		}

		var isOptionalField = func(fieldName string) func() {
			return func() {
				err := setFieldToEmpty(fieldName)
				Expect(err).NotTo(HaveOccurred())

				err = rootConfig.Validate()

				Expect(err).NotTo(HaveOccurred())
			}
		}

		It("does not return error on valid config", func() {
			err := rootConfig.Validate()

			Expect(err).NotTo(HaveOccurred())
		})

		Describe("Config", func() {
			It("returns an error if LogFileLocation is blank", isRequiredField("LogFileLocation"))
			It("returns an error if PidFile is blank", isRequiredField("PidFile"))
			It("returns an error if ChildPidFile is blank", isRequiredField("ChildPidFile"))
		})

		Describe("Upgrader", func() {
			It("returns an error if Upgrader.PackageVersionFile is blank", isRequiredField("Upgrader.PackageVersionFile"))
			It("returns an error if Upgrader.LastUpgradedVersionFile is blank", isRequiredField("Upgrader.LastUpgradedVersionFile"))
		})

		Describe("StartManager", func() {
			It("returns an error if Manager.StateFileLocation is blank", isRequiredField("Manager.StateFileLocation"))
			It("returns an error if Manager.ClusterIps is blank", isRequiredField("Manager.ClusterIps"))
			It("returns an error if Manager.ClusterProbeTimeout is blank", isRequiredField("Manager.ClusterProbeTimeout"))
		})

		Describe("DBHelper", func() {
			It("returns an error if Db.UpgradePath is blank", isRequiredField("Db.UpgradePath"))
			It("returns an error if Db.User is blank", isRequiredField("Db.User"))

			It("does not return an error if Db.Password is blank", isOptionalField("Db.Password"))
			It("does not return an error if Db.PreseededDatabases is blank", isOptionalField("Db.PreseededDatabases"))

			Describe("PreseededDatabase", func() {
				It("returns an error if Db.PreseededDatabases.DBName is blank", isRequiredField("Db.PreseededDatabases.DBName"))
				It("returns an error if Db.PreseededDatabases.User is blank", isRequiredField("Db.PreseededDatabases.User"))

				It("does not an error if Db.PreseededDatabases.Password is blank", isOptionalField("Db.PreseededDatabases.Password"))
			})
		})
	})
})
